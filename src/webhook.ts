import type { MatrixConfig, UptimeKumaPayload } from "./types";
import { sendAlert } from "./matrix";

const webhookSecret = process.env.WEBHOOK_SECRET ?? null;

function formatUptimeKumaAlert(payload: UptimeKumaPayload): string | null {
  const monitor = payload.monitor;
  const heartbeat = payload.heartbeat;
  const name = monitor?.name ?? monitor?.url ?? "unknown";
  const status = heartbeat?.status;
  const reason = heartbeat?.msg ?? payload.msg ?? "";

  if (status === 1) {
    return `✅ <b>[External] ${name}</b> is back UP` + (reason ? `<br/><i>${reason}</i>` : "");
  }

  if (status === 0) {
    return `🔴 <b>[External] ${name}</b> is DOWN — ${reason}` + (monitor?.url ? `<br/><code>${monitor.url}</code>` : "");
  }

  if (payload.msg) {
    return `ℹ️ <b>[External] ${name}</b> — ${payload.msg}`;
  }

  return null;
}

function createWebhookHandler(matrix: MatrixConfig) {
  return async (req: Request): Promise<Response> => {
    const url = new URL(req.url);

    if (url.pathname === "/health") {
      return new Response("ok", { status: 200 });
    }

    if (req.method !== "POST" || url.pathname !== "/webhook/uptime-kuma") {
      return new Response("not found", { status: 404 });
    }

    if (webhookSecret) {
      const provided = req.headers.get("x-webhook-secret");
      if (provided !== webhookSecret) {
        return new Response("unauthorized", { status: 401 });
      }
    }

    let payload: UptimeKumaPayload;
    try {
      payload = await req.json() as UptimeKumaPayload;
    } catch {
      return new Response("bad request", { status: 400 });
    }

    const message = formatUptimeKumaAlert(payload);
    if (message) {
      console.log(`[webhook] received alert: ${payload.msg ?? "no msg"}`);
      await sendAlert(matrix, message);
    }

    return new Response("ok", { status: 200 });
  };
}

export function startWebhookServer(matrix: MatrixConfig, port: number) {
  Bun.serve({
    port,
    fetch: createWebhookHandler(matrix),
  });
  console.log(`[webhook] listening on port ${port}`);
}
