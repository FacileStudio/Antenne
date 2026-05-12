import { createHmac, timingSafeEqual } from "node:crypto";
import type { WebhookProviderDefinition, WebhookProviderSettings } from "../types";

interface OpusTask {
  id?: string;
  number?: number | null;
  title?: string;
  status?: string | null;
  priority?: string | null;
  url?: string;
}

interface OpusActor {
  id?: string | null;
  name?: string | null;
}

interface OpusProject {
  id?: string;
  name?: string;
  workspaceId?: string;
}

interface OpusPayload {
  event?: string;
  timestamp?: string;
  integration?: { type?: string };
  project?: OpusProject;
  task?: OpusTask;
  actor?: OpusActor;
  data?: {
    title?: string;
    description?: string;
    oldStatus?: string;
    newStatus?: string;
    oldPriority?: string;
    newPriority?: string;
    oldTitle?: string;
    newTitle?: string;
    oldDescription?: string;
    newDescription?: string;
    comment?: string;
    priority?: string;
    status?: string;
    number?: number;
    [key: string]: unknown;
  };
}

function escapeHtml(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}

function equalsSignature(expected: string, received: string) {
  const expectedBuffer = Buffer.from(expected);
  const receivedBuffer = Buffer.from(received);
  if (expectedBuffer.length !== receivedBuffer.length) {
    return false;
  }
  return timingSafeEqual(expectedBuffer, receivedBuffer);
}

function verifySignature(rawBody: string, settings: WebhookProviderSettings, signature: string | null) {
  if (!settings.secretValue) {
    return true;
  }
  if (!signature) {
    return false;
  }
  const expected = createHmac("sha256", settings.secretValue).update(rawBody).digest("hex");
  return equalsSignature(expected, signature);
}

function parsePayload(payload: unknown): OpusPayload {
  return (payload ?? {}) as OpusPayload;
}

function getActorName(actor?: OpusActor): string {
  return actor?.name ?? "someone";
}

function getTaskRef(task?: OpusTask): string {
  if (!task) return "";
  if (task.number) return `<code>#${task.number}</code> `;
  return "";
}

function formatMessage(parsed: OpusPayload): string | null {
  const event = parsed.event;
  if (!event) return "📋 <b>[Opus]</b> received an event";

  const actor = escapeHtml(getActorName(parsed.actor));
  const task = parsed.task;
  const ref = getTaskRef(task);
  const title = escapeHtml(task?.title ?? "untitled");
  const project = parsed.project?.name ? ` in <b>${escapeHtml(parsed.project.name)}</b>` : "";

  if (event === "task.created") {
    return `📋 <b>[Opus]</b> ${actor} created task ${ref}<b>${title}</b>${project}`;
  }

  if (event === "task.status_changed") {
    const from = parsed.data?.oldStatus ? escapeHtml(parsed.data.oldStatus) : null;
    const to = parsed.data?.newStatus ? escapeHtml(parsed.data.newStatus) : null;
    const change = from && to ? ` from <code>${from}</code> to <code>${to}</code>` : "";
    return `🔄 <b>[Opus]</b> ${actor} changed status of ${ref}<b>${title}</b>${change}`;
  }

  if (event === "task.priority_changed") {
    const from = parsed.data?.oldPriority ? escapeHtml(parsed.data.oldPriority) : null;
    const to = parsed.data?.newPriority ? escapeHtml(parsed.data.newPriority) : null;
    const change = from && to ? ` from <code>${from}</code> to <code>${to}</code>` : "";
    return `🏷️ <b>[Opus]</b> ${actor} changed priority of ${ref}<b>${title}</b>${change}`;
  }

  if (event === "task.title_changed") {
    const oldTitle = parsed.data?.oldTitle ? escapeHtml(parsed.data.oldTitle) : null;
    const newTitle = parsed.data?.newTitle ? escapeHtml(parsed.data.newTitle) : null;
    if (oldTitle && newTitle) {
      return `✏️ <b>[Opus]</b> ${actor} renamed task ${ref}<b>${oldTitle}</b> → <b>${newTitle}</b>`;
    }
    return `✏️ <b>[Opus]</b> ${actor} renamed task ${ref}<b>${title}</b>`;
  }

  if (event === "task.description_changed") {
    return `📝 <b>[Opus]</b> ${actor} updated description of ${ref}<b>${title}</b>`;
  }

  if (event === "task.comment_created") {
    return `💬 <b>[Opus]</b> ${actor} commented on ${ref}<b>${title}</b>`;
  }

  return `📋 <b>[Opus]</b> ${actor} triggered <code>${escapeHtml(event)}</code> on ${ref}<b>${title}</b>`;
}

function mapStatus(event: string, data?: OpusPayload["data"]): string | null {
  if (event === "task.created") return "open";
  if (event === "task.status_changed") return data?.newStatus ?? null;
  return null;
}

export const opusWebhookProvider: WebhookProviderDefinition = {
  type: "opus",
  label: "Opus",
  defaultPath: "/webhook/opus",
  defaultSecretHeader: "x-opus-signature",
  authenticate(req, rawBody, settings) {
    return verifySignature(rawBody, settings, req.headers.get(settings.secretHeader ?? "x-opus-signature"));
  },
  formatMessage(payload) {
    return formatMessage(parsePayload(payload));
  },
  describeEvent(payload) {
    return parsePayload(payload).event ?? "opus event";
  },
  buildEvent(payload) {
    const parsed = parsePayload(payload);
    const event = parsed.event ?? "opus.event";
    const task = parsed.task;
    return {
      eventType: event,
      eventTitle: task?.title ?? event,
      eventUrl: task?.url ?? null,
      occurredAt: parsed.timestamp ?? null,
      actor: {
        id: parsed.actor?.id ?? null,
        name: parsed.actor?.name ?? null,
        email: null,
      },
      subject: {
        type: "task",
        id: task?.id ?? null,
        name: task?.title ?? event,
        url: task?.url ?? null,
      },
      normalizedPayload: {
        event,
        taskId: task?.id ?? null,
        taskNumber: task?.number ?? null,
        taskTitle: task?.title ?? null,
        projectId: parsed.project?.id ?? null,
        projectName: parsed.project?.name ?? null,
        status: mapStatus(event, parsed.data),
        priority: event === "task.priority_changed" ? (parsed.data?.newPriority ?? null) : (task?.priority ?? null),
      },
      containsPii: false,
    };
  },
};
