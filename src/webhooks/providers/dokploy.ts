import type { DokployWebhookPayload, WebhookDispatchContext, WebhookProviderDefinition } from "../types";

function escapeHtml(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}

function formatDokployAlert(payload: DokployWebhookPayload, context: WebhookDispatchContext): string | null {
  const title = payload.title?.trim();
  const message = payload.message?.trim();
  const timestamp = payload.timestamp?.trim();
  const projectName = payload.projectName?.trim();
  const applicationName = payload.applicationName?.trim();

  if (!title && !message && !timestamp) {
    return null;
  }

  const heading = title || context.providerName || "Dokploy";
  const parts = [`🚀 <b>[Dokploy] ${escapeHtml(heading)}</b>`];

  if (projectName || applicationName) {
    const segments: string[] = [];
    if (projectName) segments.push(escapeHtml(projectName));
    if (applicationName) segments.push(escapeHtml(applicationName));
    parts.push(`Service: <b>${segments.join(" / ")}</b>`);
  }

  if (message) {
    parts.push(escapeHtml(message).replaceAll("\n", "<br/>"));
  }

  if (timestamp) {
    parts.push(`<code>${escapeHtml(timestamp)}</code>`);
  }

  return parts.join("<br/>");
}

function parsePayload(payload: unknown): DokployWebhookPayload {
  return (payload ?? {}) as DokployWebhookPayload;
}

export const dokployWebhookProvider: WebhookProviderDefinition = {
  type: "dokploy",
  label: "Dokploy",
  defaultPath: "/webhook/dokploy",
  defaultSecretHeader: "x-dokploy-secret",
  formatMessage(payload, context) {
    return formatDokployAlert(parsePayload(payload), context);
  },
  describeEvent(payload) {
    const parsed = parsePayload(payload);
    const label = parsed.applicationName?.trim() || parsed.projectName?.trim();
    const event = parsed.title ?? parsed.message ?? "unknown";
    return label ? `${event} (${label})` : event;
  },
};
