import { createHmac, timingSafeEqual } from "node:crypto";
import type { ArdoiseWebhookPayload, WebhookProviderDefinition, WebhookProviderSettings } from "../types";

function escapeHtml(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}

function toTitle(value: string) {
  return value
    .split(".")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
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

  const expected = `sha256=${createHmac("sha256", settings.secretValue).update(rawBody).digest("hex")}`;
  return equalsSignature(expected, signature);
}

function parsePayload(payload: unknown): ArdoiseWebhookPayload {
  return (payload ?? {}) as ArdoiseWebhookPayload;
}

function formatActor(payload: ArdoiseWebhookPayload) {
  return payload.actor?.email ?? payload.actor?.id ?? "someone";
}

function formatAdminMessage(payload: ArdoiseWebhookPayload) {
  const event = payload.event ?? "admin.event";
  const [, entity = "record", action = "updated"] = event.split(".");
  const actor = escapeHtml(formatActor(payload));
  const entityLabel = escapeHtml(entity);
  const entityId = payload.data?.entityId ?? payload.data?.record?.id;
  const name = [payload.data?.record?.firstName, payload.data?.record?.lastName].filter(Boolean).join(" ");
  const email = payload.data?.record?.email;
  const parts = [
    `<b>[Ardoise]</b> ${actor} ${escapeHtml(action)} ${entityLabel}`,
    entityId ? `ID: <code>${escapeHtml(String(entityId))}</code>` : null,
    name ? `Name: <b>${escapeHtml(name)}</b>` : null,
    email ? `Email: <code>${escapeHtml(email)}</code>` : null,
  ].filter(Boolean);

  return parts.join("<br/>");
}

function formatSettingsMessage(payload: ArdoiseWebhookPayload) {
  const event = payload.event ?? "settings.updated";
  if (event === "settings.logo.updated") {
    return `🖼️ <b>[Ardoise]</b> logo updated`;
  }

  const changedFields = Array.isArray(payload.data?.changedFields) ? payload.data.changedFields : [];
  const parts = [`⚙️ <b>[Ardoise]</b> settings updated`];

  if (changedFields.length > 0) {
    parts.push(`Fields: ${changedFields.map((field) => `<code>${escapeHtml(field)}</code>`).join(", ")}`);
  }

  return parts.join("<br/>");
}

export const ardoiseWebhookProvider: WebhookProviderDefinition = {
  type: "ardoise",
  label: "Ardoise",
  defaultPath: "/webhook/ardoise",
  defaultSecretHeader: "x-ardoise-signature-256",
  authenticate(req, rawBody, settings) {
    return verifySignature(rawBody, settings, req.headers.get(settings.secretHeader ?? "x-ardoise-signature-256"));
  },
  formatMessage(payload) {
    const parsed = parsePayload(payload);
    if (!parsed.event) {
      return "📡 <b>[Ardoise]</b> received an event";
    }

    if (parsed.event.startsWith("admin.")) {
      return formatAdminMessage(parsed);
    }

    if (parsed.event.startsWith("settings.")) {
      return formatSettingsMessage(parsed);
    }

    return `📡 <b>[Ardoise]</b> ${escapeHtml(toTitle(parsed.event))}`;
  },
  describeEvent(payload) {
    return parsePayload(payload).event ?? "ardoise event";
  },
  buildEvent(payload) {
    const parsed = parsePayload(payload);
    const entityType = parsed.data?.entity ?? parsed.event?.split(".")[1] ?? "client";
    const record = parsed.data?.record;
    const name = [record?.firstName, record?.lastName].filter(Boolean).join(" ") || null;
    return {
      eventType: parsed.event ?? "ardoise.event",
      eventTitle: parsed.event ? toTitle(parsed.event) : "Ardoise event",
      occurredAt: parsed.createdAt ?? null,
      actor: {
        id: parsed.actor?.id ?? null,
        email: parsed.actor?.email ?? null,
        name: parsed.actor?.role ?? null,
      },
      subject: {
        type: "client",
        id: parsed.data?.entityId ?? record?.id ?? parsed.id ?? null,
        name: name ?? record?.email ?? entityType,
        url: null,
      },
      normalizedPayload: {
        event: parsed.event ?? null,
        source: parsed.source ?? null,
        entity: entityType,
        entityId: parsed.data?.entityId ?? record?.id ?? null,
        changedFields: parsed.data?.changedFields ?? [],
        recordEmail: record?.email ?? null,
        status: (() => {
          const action = parsed.event?.split(".").pop();
          if (action === "create" || action === "created") return "created";
          if (action === "update" || action === "updated") return "updated";
          if (action === "delete" || action === "deleted") return "deleted";
          return null;
        })(),
      },
      containsPii: true,
    };
  },
};
