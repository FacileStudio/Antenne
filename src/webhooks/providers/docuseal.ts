import type { DocuSealDocumentPayload, DocuSealEventPayload, WebhookProviderDefinition } from "../types";

function escapeHtml(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}

function renderCode(value: string | number): string {
  return `<code>${escapeHtml(String(value))}</code>`;
}

function pickSubmitterName(payload: DocuSealEventPayload): string | null {
  const data = payload.data;
  return data?.name ?? data?.email ?? data?.role ?? null;
}

function pickTemplateName(payload: DocuSealEventPayload): string {
  return payload.data?.template?.name ?? payload.data?.name ?? "Untitled template";
}

function formatDocuments(documents: DocuSealDocumentPayload[] | null | undefined): string | null {
  if (!documents || documents.length === 0) {
    return null;
  }

  return documents
    .filter((document) => document.url)
    .map((document) => {
      const label = document.name ?? "document";
      return `• <a href="${escapeHtml(document.url as string)}">${escapeHtml(label)}</a>`;
    })
    .join("<br/>") || null;
}

function formatDocuSealAlert(payload: DocuSealEventPayload): string | null {
  const eventType = payload.event_type;
  const data = payload.data;

  if (!eventType || !data) {
    return null;
  }

  const submitter = pickSubmitterName(payload);
  const templateName = pickTemplateName(payload);
  const submissionId = data.submission?.id ?? data.id;
  const externalId = data.external_id;
  const submissionUrl = data.submission?.url;
  const documents = formatDocuments(data.documents);

  if (eventType === "form.completed") {
    const parts = [
      `✅ <b>[DocuSeal] ${escapeHtml(templateName)}</b> completed`,
      submitter ? `Signer: <b>${escapeHtml(submitter)}</b>` : null,
      submissionId ? `Submission: ${renderCode(submissionId)}` : null,
      externalId ? `External ID: ${renderCode(externalId)}` : null,
      submissionUrl ? `<a href="${escapeHtml(submissionUrl)}">Open submission</a>` : null,
      documents ? `Documents:<br/>${documents}` : null,
    ].filter(Boolean);

    return parts.join("<br/>");
  }

  if (eventType === "form.declined") {
    const parts = [
      `🟠 <b>[DocuSeal] ${escapeHtml(templateName)}</b> declined`,
      submitter ? `Signer: <b>${escapeHtml(submitter)}</b>` : null,
      data.decline_reason ? `Reason: <i>${escapeHtml(data.decline_reason)}</i>` : null,
      submissionId ? `Submission: ${renderCode(submissionId)}` : null,
      submissionUrl ? `<a href="${escapeHtml(submissionUrl)}">Open submission</a>` : null,
    ].filter(Boolean);

    return parts.join("<br/>");
  }

  if (eventType === "form.started" || eventType === "form.viewed") {
    const verb = eventType === "form.started" ? "started" : "viewed";
    const parts = [
      `ℹ️ <b>[DocuSeal] ${escapeHtml(templateName)}</b> ${verb}`,
      submitter ? `Signer: <b>${escapeHtml(submitter)}</b>` : null,
      submissionId ? `Submission: ${renderCode(submissionId)}` : null,
      submissionUrl ? `<a href="${escapeHtml(submissionUrl)}">Open submission</a>` : null,
    ].filter(Boolean);

    return parts.join("<br/>");
  }

  if (eventType.startsWith("template.")) {
    const verb = eventType.slice("template.".length);
    const parts = [
      `🧾 <b>[DocuSeal] Template ${escapeHtml(verb)}</b>`,
      `Name: <b>${escapeHtml(templateName)}</b>`,
      data.id ? `Template ID: ${renderCode(data.id)}` : null,
      data.slug ? `Slug: ${renderCode(data.slug)}` : null,
    ].filter(Boolean);

    return parts.join("<br/>");
  }

  if (eventType.startsWith("submission.")) {
    const verb = eventType.slice("submission.".length);
    const parts = [
      `📨 <b>[DocuSeal] Submission ${escapeHtml(verb)}</b>`,
      templateName ? `Template: <b>${escapeHtml(templateName)}</b>` : null,
      data.id ? `Submission ID: ${renderCode(data.id)}` : null,
    ].filter(Boolean);

    return parts.join("<br/>");
  }

  return `ℹ️ <b>[DocuSeal]</b> ${escapeHtml(eventType)}`;
}

function parsePayload(payload: unknown): DocuSealEventPayload {
  return (payload ?? {}) as DocuSealEventPayload;
}

export const docuSealWebhookProvider: WebhookProviderDefinition = {
  id: "docuseal",
  label: "DocuSeal",
  defaultPath: "/webhook/docuseal",
  defaultSecretHeader: "x-docuseal-secret",
  formatMessage(payload) {
    return formatDocuSealAlert(parsePayload(payload));
  },
  describeEvent(payload) {
    return parsePayload(payload).event_type ?? "unknown";
  },
};
