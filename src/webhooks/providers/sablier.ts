import type { WebhookProviderDefinition } from "../types";

interface SablierTimeEntry {
  id?: number;
  project_id?: number;
  user_id?: number;
  description?: string;
  started_at?: string | null;
  stopped_at?: string | null;
}

interface SablierPayload {
  event?: string;
  data?: SablierTimeEntry;
}

function formatDuration(startedAt?: string | null, stoppedAt?: string | null): string {
  if (!startedAt || !stoppedAt) return "";
  const ms = new Date(stoppedAt).getTime() - new Date(startedAt).getTime();
  const totalMinutes = Math.round(ms / 60000);
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  if (hours > 0 && minutes > 0) return ` (${hours}h ${minutes}m)`;
  if (hours > 0) return ` (${hours}h)`;
  return ` (${minutes}m)`;
}

function escapeHtml(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}

function formatSablierAlert(payload: SablierPayload): string | null {
  const event = payload.event;
  const data = payload.data;

  if (!event || !data) return null;

  const desc = data.description ? ` <i>${escapeHtml(data.description)}</i>` : "";

  if (event === "timer_started") {
    return `⏱ <b>[Sablier]</b> Timer started${desc}`;
  }

  if (event === "timer_stopped") {
    const duration = formatDuration(data.started_at, data.stopped_at);
    return `⏹ <b>[Sablier]</b> Timer stopped${desc}${duration}`;
  }

  return `⏱ <b>[Sablier]</b> <code>${escapeHtml(event)}</code>`;
}

function parsePayload(payload: unknown): SablierPayload {
  return (payload ?? {}) as SablierPayload;
}

export const sablierWebhookProvider: WebhookProviderDefinition = {
  type: "sablier",
  label: "Sablier",
  defaultPath: "/webhook/sablier",
  defaultSecretHeader: "x-sablier-signature",
  formatMessage(payload) {
    return formatSablierAlert(parsePayload(payload));
  },
  describeEvent(payload) {
    return parsePayload(payload).event ?? "unknown event";
  },
};
