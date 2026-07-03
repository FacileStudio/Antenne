import type { Server as BunServer, ServerWebSocket } from "bun";
import { timingSafeEqual } from "node:crypto";
import { EventQueue } from "./queue";
import { ConnectionRegistry } from "./registry";
import type {
  AckMessage,
  EventMessage,
  RegisteredApp,
  RegistrationRequest,
  SubscribeMessage,
  SubscribedMessage,
  WelcomeMessage,
  ErrorMessage,
} from "./types";

interface PoolWebSocketData {
  app: RegisteredApp;
}

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

type PoolEventLogger = (event: { type: string; app: string; detail: string }) => void;

function safeSecretEqual(a: string, b: string): boolean {
  const bufA = Buffer.from(a);
  const bufB = Buffer.from(b);
  if (bufA.length !== bufB.length) return false;
  return timingSafeEqual(bufA, bufB);
}

export class PoolServer {
  private queue: EventQueue;
  private registry = new ConnectionRegistry();
  private poolSecret: string;
  private logger: PoolEventLogger | null = null;

  constructor(options: { dataDir: string; secret?: string }) {
    this.queue = new EventQueue(options.dataDir);
    this.poolSecret = options.secret ?? process.env.POOL_SECRET ?? "pool";
    if (this.poolSecret === "pool") {
      console.warn("[pool] using default secret 'pool' — set POOL_SECRET in production");
    }
  }

  onPoolEvent(logger: PoolEventLogger) {
    this.logger = logger;
  }

  async handleRegister(req: Request): Promise<Response> {
    let body: RegistrationRequest;
    try {
      body = await req.json() as RegistrationRequest;
    } catch {
      return json({ error: "invalid json" }, 400);
    }

    if (!body.app || typeof body.app !== "string") {
      return json({ error: "app is required" }, 400);
    }

    if (typeof body.secret !== "string" || !safeSecretEqual(body.secret, this.poolSecret)) {
      return json({ error: "invalid secret" }, 403);
    }

    const instanceId = typeof body.instance_id === "string" ? body.instance_id.trim() : "";
    const identity = instanceId ? `${body.app}:${instanceId}` : body.app;

    const emitChannels = body.events?.emit ?? ["*"];
    const listenChannels = body.events?.listen ?? ["*"];
    const registered = this.queue.registerApp(identity, body.secret, emitChannels, listenChannels);
    this.queue.touchApp(registered.app_id);

    console.log(`[pool] registered app=${registered.app} id=${registered.app_id}`);

    return json({ token: registered.token, app_id: registered.app_id });
  }

  handleUpgrade(req: Request, server: BunServer<PoolWebSocketData>): Response | undefined {
    const url = new URL(req.url);
    const token = url.searchParams.get("token");

    if (!token) {
      return json({ error: "token required" }, 401);
    }

    const app = this.queue.getAppByToken(token);
    if (!app) {
      return json({ error: "invalid token" }, 403);
    }

    const upgraded = server.upgrade(req, {
      data: { app },
    });

    if (!upgraded) {
      return json({ error: "upgrade failed" }, 500);
    }

    return undefined;
  }

  handleOpen = (ws: ServerWebSocket<PoolWebSocketData>) => {
    const { app } = ws.data;

    const stale = this.registry.get(app.app_id);
    if (stale && stale.ws !== ws) {
      this.registry.remove(app.app_id);
      (stale.ws as ServerWebSocket<PoolWebSocketData>).close(1000, "replaced by new connection");
    }

    this.registry.add({
      app_id: app.app_id,
      app: app.app,
      ws,
      channels: new Set(),
    });

    const welcome: WelcomeMessage = {
      type: "welcome",
      epoch: this.queue.getEpoch(),
      app_id: app.app_id,
    };

    ws.send(JSON.stringify(welcome));
    console.log(`[pool] connected app=${app.app} id=${app.app_id}`);
    this.logger?.({ type: "pool.connected", app: app.app, detail: `${app.app} joined the pool` });
  };

  handleMessage = (ws: ServerWebSocket<PoolWebSocketData>, message: string | Buffer) => {
    const raw = typeof message === "string" ? message : message.toString();
    const { app } = ws.data;

    let parsed: { type: string };
    try {
      parsed = JSON.parse(raw);
    } catch {
      this.sendError(ws, "PARSE_ERROR", "invalid json");
      return;
    }

    switch (parsed.type) {
      case "subscribe":
        this.handleSubscribe(ws, app, parsed as SubscribeMessage);
        break;
      case "event": {
        const evt = parsed as Record<string, unknown>;
        if (typeof evt.channel !== "string" || !evt.channel) {
          this.sendError(ws, "INVALID_EVENT", "channel is required");
          return;
        }
        this.handleEvent(ws, app, evt as unknown as EventMessage & { channel: string; payload: unknown });
        break;
      }
      case "ack":
        this.handleAck(app, parsed as AckMessage);
        break;
      case "ping":
        ws.send(JSON.stringify({ type: "pong" }));
        break;
      default:
        this.sendError(ws, "UNKNOWN_TYPE", `unknown message type: ${parsed.type}`);
    }
  };

  handleClose = (ws: ServerWebSocket<PoolWebSocketData>) => {
    const app = ws.data?.app;
    if (!app) return;
    // Only drop the registry entry if it still belongs to this socket —
    // a newer connection for the same app may have replaced it.
    if (this.registry.get(app.app_id)?.ws !== ws) return;
    this.registry.remove(app.app_id);
    console.log(`[pool] disconnected app=${app.app} id=${app.app_id}`);
    this.logger?.({ type: "pool.disconnected", app: app.app, detail: `${app.app} left the pool` });
  };

  private static readonly REPLAY_BATCH = 100;

  private handleSubscribe(ws: ServerWebSocket<PoolWebSocketData>, app: RegisteredApp, msg: SubscribeMessage) {
    const confirmed: Record<string, { offset: number }> = {};

    for (const [channel, opts] of Object.entries(msg.channels ?? {})) {
      this.registry.subscribe(app.app_id, channel);

      // Resume from the client's offset, or from the server-side ack cursor
      // when the client lost its in-memory offsets (e.g. after a restart).
      let cursor = Math.max(opts.last_offset ?? 0, this.queue.getLastAcked(app.app_id, channel));

      while (true) {
        const batch = this.queue.getEventsAfter(channel, cursor, PoolServer.REPLAY_BATCH);
        for (const event of batch.events) {
          if (event.sender === app.app) continue;
          const eventMsg: EventMessage = {
            type: "event",
            id: event.id,
            channel: event.channel,
            offset: event.offset,
            payload: event.payload,
            timestamp: event.timestamp,
            sender: event.sender,
          };
          ws.send(JSON.stringify(eventMsg));
        }
        cursor = batch.lastOffset;
        if (batch.scanned < PoolServer.REPLAY_BATCH) break;
      }

      confirmed[channel] = { offset: cursor };
    }

    const reply: SubscribedMessage = { type: "subscribed", channels: confirmed };
    ws.send(JSON.stringify(reply));
  }

  private handleEvent(ws: ServerWebSocket<PoolWebSocketData>, app: RegisteredApp, msg: { channel: string; payload: unknown }) {
    const { id, offset, timestamp } = this.queue.pushEvent(msg.channel, msg.payload, app.app);

    const eventMsg: EventMessage = {
      type: "event",
      id,
      channel: msg.channel,
      offset,
      payload: msg.payload,
      timestamp,
      sender: app.app,
    };

    const subscribers = this.registry.getSubscribers(msg.channel);
    for (const sub of subscribers) {
      if (sub.app_id === app.app_id) continue;
      const subWs = sub.ws as ServerWebSocket<PoolWebSocketData>;
      subWs.send(JSON.stringify(eventMsg));
    }

    const ack: AckMessage = { type: "ack", channel: msg.channel, offset };
    ws.send(JSON.stringify(ack));

    this.logger?.({ type: "pool.event", app: app.app, detail: `${app.app} → ${msg.channel} (${subscribers.length - 1} recipients)` });
  }

  private handleAck(app: RegisteredApp, msg: AckMessage) {
    this.queue.ack(app.app_id, msg.channel, msg.offset);
    this.queue.touchApp(app.app_id);
  }

  private sendError(ws: ServerWebSocket<PoolWebSocketData>, code: string, message: string) {
    const err: ErrorMessage = { type: "error", code, message };
    ws.send(JSON.stringify(err));
  }

  setSecret(secret: string) {
    this.poolSecret = secret;
  }

  cleanup() {
    this.queue.cleanup();
  }

  getStats() {
    return {
      epoch: this.queue.getEpoch(),
      connections: this.registry.size(),
      apps: this.registry.getAll().map((c) => ({ app_id: c.app_id, app: c.app, channels: Array.from(c.channels) })),
    };
  }
}
