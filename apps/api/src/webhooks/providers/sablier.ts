import type { WebhookProviderDefinition } from "../types";

interface SablierTimeEntry {
  id?: number;
  project_id?: number;
  project_name?: string;
  task_id?: number;
  task_name?: string;
  user_id?: number;
  user_email?: string;
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
  const totalMinutes = Math.floor(ms / 60000);
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

  const user = data.user_email ? ` <code>${escapeHtml(data.user_email)}</code>` : "";
  const project = data.project_name ? ` on <i>${escapeHtml(data.project_name)}</i>` : "";
  const task = data.task_name ? ` / <i>${escapeHtml(data.task_name)}</i>` : "";
  const desc = data.description ? `: ${escapeHtml(data.description)}` : "";

  if (event === "timer_started") {
    return `<b>[Sablier]</b> Timer started${user}${project}${task}${desc}`;
  }

  if (event === "timer_stopped") {
    const duration = formatDuration(data.started_at, data.stopped_at);
    return `<b>[Sablier]</b> Timer stopped${user}${project}${task}${desc}${duration}`;
  }

  return `<b>[Sablier]</b> <code>${escapeHtml(event)}</code>`;
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
  buildEvent(payload) {
    const parsed = parsePayload(payload);
    const data = parsed.data;
    return {
      eventType: parsed.event ?? "sablier.event",
      eventTitle: data?.task_name ?? data?.project_name ?? parsed.event ?? "Sablier event",
      occurredAt: data?.stopped_at ?? data?.started_at ?? null,
      actor: {
        id: data?.user_id ? String(data.user_id) : null,
        email: data?.user_email ?? null,
      },
      subject: {
        type: "schedule",
        id: data?.id ? String(data.id) : null,
        name: data?.task_name ?? data?.project_name ?? data?.description ?? null,
        url: null,
      },
      normalizedPayload: {
        event: parsed.event ?? null,
        timeEntryId: data?.id ?? null,
        projectId: data?.project_id ?? null,
        projectName: data?.project_name ?? null,
        taskId: data?.task_id ?? null,
        taskName: data?.task_name ?? null,
        startedAt: data?.started_at ?? null,
        stoppedAt: data?.stopped_at ?? null,
      },
      containsPii: true,
    };
  },
};
