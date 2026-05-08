import type { WebhookProviderDefinition } from "../types";

interface VikunjaDoer {
  id?: number;
  name?: string;
  username?: string;
  email?: string;
}

interface VikunjaTask {
  id?: number;
  title?: string;
  done?: boolean;
  priority?: number;
  due_date?: string | null;
  project_id?: number;
  identifier?: string;
}

interface VikunjaProject {
  id?: number;
  title?: string;
  description?: string;
}

interface VikunjaTeam {
  id?: number;
  name?: string;
}

interface VikunjaComment {
  id?: number;
  comment?: string;
  author?: VikunjaDoer;
}

interface VikunjaRelation {
  task_id?: number;
  other_task_id?: number;
  relation_kind?: string;
}

interface VikunjaPayload {
  event_name?: string;
  time?: string;
  data?: {
    task?: VikunjaTask;
    doer?: VikunjaDoer;
    project?: VikunjaProject;
    team?: VikunjaTeam;
    comment?: VikunjaComment;
    assignee?: VikunjaDoer;
    relation?: VikunjaRelation;
    member?: VikunjaDoer;
    [key: string]: unknown;
  };
}

function escapeHtml(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}

function getDoerName(doer?: VikunjaDoer): string {
  return doer?.name ?? doer?.username ?? doer?.email ?? "someone";
}


function formatTaskEvent(payload: VikunjaPayload, verb: string): string {
  const task = payload.data?.task;
  const doer = payload.data?.doer;
  const title = escapeHtml(task?.title ?? "untitled");
  const actor = escapeHtml(getDoerName(doer));
  const ref = task?.identifier ? `<code>${escapeHtml(task.identifier)}</code>` : (task?.id ? `<code>#${task.id}</code>` : "");
  return `📋 <b>[Vikunja]</b> ${actor} ${verb} task ${ref} <b>${title}</b>`;
}

function formatProjectEvent(payload: VikunjaPayload, verb: string): string {
  const project = payload.data?.project;
  const doer = payload.data?.doer;
  const title = escapeHtml(project?.title ?? "untitled");
  const actor = escapeHtml(getDoerName(doer));
  return `📁 <b>[Vikunja]</b> ${actor} ${verb} project <b>${title}</b>`;
}

function formatTeamEvent(payload: VikunjaPayload, verb: string): string {
  const team = payload.data?.team;
  const doer = payload.data?.doer;
  const name = escapeHtml(team?.name ?? "unnamed");
  const actor = escapeHtml(getDoerName(doer));
  return `👥 <b>[Vikunja]</b> ${actor} ${verb} team <b>${name}</b>`;
}

function formatVikunjaAlert(payload: VikunjaPayload): string | null {
  const event = payload.event_name;
  if (!event) return null;

  const doer = payload.data?.doer;
  const actor = escapeHtml(getDoerName(doer));

  if (event === "task.created") return formatTaskEvent(payload, "created");
  if (event === "task.updated") {
    const done = payload.data?.task?.done;
    if (done === true) return formatTaskEvent(payload, "completed");
    if (done === false) return formatTaskEvent(payload, "reopened");
    return formatTaskEvent(payload, "updated");
  }
  if (event === "task.deleted") return formatTaskEvent(payload, "deleted");

  if (event === "task.assignee.created") {
    const assignee = escapeHtml(getDoerName(payload.data?.assignee));
    const task = payload.data?.task;
    const ref = task?.identifier ? ` <code>${escapeHtml(task.identifier)}</code>` : "";
    const title = escapeHtml(task?.title ?? "untitled");
    return `👤 <b>[Vikunja]</b> ${actor} assigned ${assignee} to${ref} <b>${title}</b>`;
  }

  if (event === "task.assignee.deleted") {
    const assignee = escapeHtml(getDoerName(payload.data?.assignee));
    const task = payload.data?.task;
    const ref = task?.identifier ? ` <code>${escapeHtml(task.identifier)}</code>` : "";
    const title = escapeHtml(task?.title ?? "untitled");
    return `👤 <b>[Vikunja]</b> ${actor} unassigned ${assignee} from${ref} <b>${title}</b>`;
  }

  if (event === "task.comment.created" || event === "task.comment.edited") {
    const task = payload.data?.task;
    const ref = task?.identifier ? ` <code>${escapeHtml(task.identifier)}</code>` : "";
    const title = escapeHtml(task?.title ?? "untitled");
    const verb = event === "task.comment.created" ? "commented on" : "edited a comment on";
    return `💬 <b>[Vikunja]</b> ${actor} ${verb}${ref} <b>${title}</b>`;
  }

  if (event === "task.comment.deleted") {
    const task = payload.data?.task;
    const ref = task?.identifier ? ` <code>${escapeHtml(task.identifier)}</code>` : "";
    const title = escapeHtml(task?.title ?? "untitled");
    return `💬 <b>[Vikunja]</b> ${actor} deleted a comment on${ref} <b>${title}</b>`;
  }

  if (event === "task.relation.created" || event === "task.relation.deleted") {
    const task = payload.data?.task;
    const ref = task?.identifier ? ` <code>${escapeHtml(task.identifier)}</code>` : "";
    const title = escapeHtml(task?.title ?? "untitled");
    const verb = event === "task.relation.created" ? "added a relation to" : "removed a relation from";
    return `🔗 <b>[Vikunja]</b> ${actor} ${verb}${ref} <b>${title}</b>`;
  }

  if (event === "task.attachment.created" || event === "task.attachment.deleted") {
    const task = payload.data?.task;
    const ref = task?.identifier ? ` <code>${escapeHtml(task.identifier)}</code>` : "";
    const title = escapeHtml(task?.title ?? "untitled");
    const verb = event === "task.attachment.created" ? "attached a file to" : "removed an attachment from";
    return `📎 <b>[Vikunja]</b> ${actor} ${verb}${ref} <b>${title}</b>`;
  }

  if (event === "task.reminder.fired") return formatTaskEvent(payload, "reminder fired for");
  if (event === "task.overdue" || event === "tasks.overdue") return formatTaskEvent(payload, "is overdue:");

  if (event === "project.created") return formatProjectEvent(payload, "created");
  if (event === "project.updated") return formatProjectEvent(payload, "updated");
  if (event === "project.deleted") return formatProjectEvent(payload, "deleted");

  if (event === "project.shared.user" || event === "project.shared.team") {
    const project = payload.data?.project;
    const title = escapeHtml(project?.title ?? "untitled");
    const target = event === "project.shared.team" ? "a team" : "a user";
    return `🔗 <b>[Vikunja]</b> ${actor} shared project <b>${title}</b> with ${target}`;
  }

  if (event === "team.created") return formatTeamEvent(payload, "created");
  if (event === "team.deleted") return formatTeamEvent(payload, "deleted");

  if (event === "team.member.added" || event === "team.member.removed") {
    const team = payload.data?.team;
    const name = escapeHtml(team?.name ?? "unnamed");
    const member = escapeHtml(getDoerName(payload.data?.member));
    const verb = event === "team.member.added" ? "added" : "removed";
    return `👥 <b>[Vikunja]</b> ${actor} ${verb} ${member} in team <b>${name}</b>`;
  }

  return `📋 <b>[Vikunja]</b> ${actor} triggered <code>${escapeHtml(event)}</code>`;
}

function parsePayload(payload: unknown): VikunjaPayload {
  return (payload ?? {}) as VikunjaPayload;
}

function describeEvent(payload: unknown): string {
  const parsed = parsePayload(payload);
  return parsed.event_name ?? "unknown event";
}

function getSubjectType(event: string): "task" | "unknown" {
  return event.startsWith("task.") || event === "tasks.overdue" ? "task" : "unknown";
}

export const vikunjaWebhookProvider: WebhookProviderDefinition = {
  type: "vikunja",
  label: "Vikunja",
  defaultPath: "/webhook/vikunja",
  defaultSecretHeader: "x-vikunja-signature",
  formatMessage(payload) {
    return formatVikunjaAlert(parsePayload(payload));
  },
  describeEvent(payload) {
    return describeEvent(payload);
  },
  buildEvent(payload) {
    const parsed = parsePayload(payload);
    const event = parsed.event_name ?? "vikunja.event";
    const task = parsed.data?.task;
    const project = parsed.data?.project;
    const doer = parsed.data?.doer;
    const subjectName = task?.title ?? project?.title ?? parsed.data?.team?.name ?? event;
    return {
      eventType: event,
      eventTitle: subjectName,
      occurredAt: parsed.time ?? null,
      actor: {
        id: doer?.id ? String(doer.id) : null,
        name: doer?.name ?? doer?.username ?? null,
        email: doer?.email ?? null,
      },
      subject: {
        type: getSubjectType(event),
        id: task?.identifier ?? (task?.id ? String(task.id) : project?.id ? String(project.id) : null),
        name: subjectName,
        url: null,
      },
      normalizedPayload: {
        event,
        taskId: task?.id ?? null,
        taskIdentifier: task?.identifier ?? null,
        taskTitle: task?.title ?? null,
        taskDone: task?.done ?? null,
        priority: task?.priority ?? null,
        dueDate: task?.due_date ?? null,
        projectId: task?.project_id ?? project?.id ?? null,
        projectTitle: project?.title ?? null,
        assignee: payload && typeof payload === "object" ? parsed.data?.assignee?.username ?? parsed.data?.assignee?.name ?? parsed.data?.assignee?.email ?? null : null,
      },
      containsPii: Boolean(doer?.email || parsed.data?.assignee?.email),
    };
  },
};
