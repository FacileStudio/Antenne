import type { Server as BunServer, ServerWebSocket } from "bun";
import { EventQueue } from "./queue";
import { ConnectionRegistry } from "./registry";
import type {
  AckMessage,
  EventMessage,
  RegisteredApp,
  RegistrationRequest,
  SubscribeMessage,
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

export class PoolServer {
  private queue: EventQueue;
  private registry = new ConnectionRegistry();
  private poolSecret: string;

  constructor(options: { dataDir: string; secret?: string }) {
    this.queue = new EventQueue(options.dataDir);
    this.poolSecret = options.secret ?? process.env.POOL_SECRET ?? "pool";
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

    if (body.secret !== this.poolSecret) {
      return json({ error: "invalid secret" }, 403);
    }

    const emitChannels = body.events?.emit ?? ["*"];
    const listenChannels = body.events?.listen ?? ["*"];
    const registered = this.queue.registerApp(body.app, body.secret, emitChannels, listenChannels);

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
    this.registry.remove(app.app_id);
    console.log(`[pool] disconnected app=${app.app} id=${app.app_id}`);
  };

  private handleSubscribe(ws: ServerWebSocket<PoolWebSocketData>, app: RegisteredApp, msg: SubscribeMessage) {
    for (const [channel, opts] of Object.entries(msg.channels)) {
      this.registry.subscribe(app.app_id, channel);

      const lastOffset = opts.last_offset ?? 0;
      const missed = this.queue.getEventsAfter(channel, lastOffset);

      for (const event of missed) {
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
    }
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
  }

  private handleAck(app: RegisteredApp, msg: AckMessage) {
    this.queue.ack(app.app_id, msg.channel, msg.offset);
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
