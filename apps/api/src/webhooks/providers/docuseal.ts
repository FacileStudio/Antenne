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
      `<b>[DocuSeal] ${escapeHtml(templateName)}</b> completed`,
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
      `<b>[DocuSeal] ${escapeHtml(templateName)}</b> declined`,
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
      `<b>[DocuSeal] ${escapeHtml(templateName)}</b> ${verb}`,
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
  type: "docuseal",
  label: "DocuSeal",
  defaultPath: "/webhook/docuseal",
  defaultSecretHeader: "x-docuseal-secret",
  formatMessage(payload) {
    return formatDocuSealAlert(parsePayload(payload));
  },
  describeEvent(payload) {
    return parsePayload(payload).event_type ?? "unknown";
  },
  buildEvent(payload) {
    const parsed = parsePayload(payload);
    const data = parsed.data;
    const templateName = pickTemplateName(parsed);
    return {
      eventType: parsed.event_type ?? "docuseal.event",
      eventTitle: templateName,
      eventUrl: data?.submission?.url ?? null,
      occurredAt: parsed.timestamp ?? data?.completed_at ?? data?.declined_at ?? data?.opened_at ?? null,
      actor: {
        id: data?.id ? String(data.id) : null,
        name: data?.name ?? data?.role ?? null,
        email: data?.email ?? null,
      },
      subject: {
        type: "contract",
        id: data?.submission?.id ? String(data.submission.id) : data?.id ? String(data.id) : null,
        name: templateName,
        url: data?.submission?.url ?? null,
      },
      normalizedPayload: {
        eventType: parsed.event_type ?? null,
        submissionId: data?.submission?.id ?? data?.id ?? null,
        submissionStatus: data?.submission?.status ?? data?.status ?? null,
        templateId: data?.template?.id ?? null,
        templateName,
        completedAt: data?.completed_at ?? null,
        declinedAt: data?.declined_at ?? null,
        openedAt: data?.opened_at ?? null,
        status: data?.submission?.status ?? data?.status
          ? (data?.submission?.status ?? data?.status ?? null)
          : parsed.event_type === "form.completed" || parsed.event_type === "submission.completed"
            ? "completed"
            : parsed.event_type === "form.declined" || parsed.event_type === "submission.declined"
              ? "declined"
              : parsed.event_type === "form.started" || parsed.event_type === "form.viewed"
                ? "pending"
                : null,
        completedDate: data?.completed_at ?? null,
      },
      containsPii: true,
    };
  },
};
