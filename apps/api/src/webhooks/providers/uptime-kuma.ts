import type { UptimeKumaPayload, WebhookProviderDefinition } from "../types";

function formatUptimeKumaAlert(payload: UptimeKumaPayload): string | null {
  const monitor = payload.monitor;
  const heartbeat = payload.heartbeat;
  const name = monitor?.name ?? monitor?.url ?? "unknown";
  const status = heartbeat?.status;
  const reason = heartbeat?.msg ?? payload.msg ?? "";

  if (status === 1) {
    return `<b>[Uptime Kuma] ${name}</b> is back UP${reason ? `<br/><i>${reason}</i>` : ""}`;
  }

  if (status === 0) {
    return `<b>[Uptime Kuma] ${name}</b> is DOWN${reason ? ` — ${reason}` : ""}${monitor?.url ? `<br/><code>${monitor.url}</code>` : ""}`;
  }

  if (payload.msg) {
    return `<b>[Uptime Kuma] ${name}</b> — ${payload.msg}`;
  }

  return null;
}

function parsePayload(payload: unknown): UptimeKumaPayload {
  return (payload ?? {}) as UptimeKumaPayload;
}

export const uptimeKumaWebhookProvider: WebhookProviderDefinition = {
  type: "uptime-kuma",
  label: "Uptime Kuma",
  defaultPath: "/webhook/uptime-kuma",
  defaultSecretHeader: "x-webhook-secret",
  formatMessage(payload) {
    return formatUptimeKumaAlert(parsePayload(payload));
  },
  describeEvent(payload) {
    const parsed = parsePayload(payload);
    return parsed.msg ?? parsed.monitor?.name ?? parsed.monitor?.url ?? "unknown";
  },
};
