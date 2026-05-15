import { timingSafeEqual as nodeTimingSafeEqual } from "node:crypto";
import type { WebhookProviderDefinition } from "../types";

interface PlumeDocument {
  id: number;
  name: string;
  status: string;
  file_name: string;
  url?: string | null;
  sequential?: boolean;
  created_at?: string | null;
  updated_at?: string | null;
}

interface PlumeSigner {
  id: number;
  name: string;
  email: string;
  role: string;
  status?: string | null;
  order_num?: number;
  signing_url?: string | null;
  signed_at?: string | null;
  viewed_at?: string | null;
}

interface PlumeEventPayload {
  event_id?: string;
  event_type: string;
  occurred_at?: string;
  document: PlumeDocument;
  signer?: PlumeSigner | null;
}

async function hmacSha256Hex(secret: string, data: string): Promise<string> {
  const key = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"],
  );
  const sig = await crypto.subtle.sign("HMAC", key, new TextEncoder().encode(data));
  return [...new Uint8Array(sig)].map((b) => b.toString(16).padStart(2, "0")).join("");
}

function timingSafeEqual(a: string, b: string): boolean {
  const ab = Buffer.from(a);
  const bb = Buffer.from(b);
  if (ab.length !== bb.length) return false;
  return nodeTimingSafeEqual(ab, bb);
}

function escapeHtml(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}

function parsePayload(payload: unknown): PlumeEventPayload | null {
  const candidate = payload as Record<string, unknown> | null | undefined;
  if (!candidate) return null;
  if (typeof candidate.event_type !== "string") return null;
  if (typeof candidate.document !== "object" || candidate.document === null) return null;
  return candidate as unknown as PlumeEventPayload;
}

function actionVerb(eventType: string): string {
  switch (eventType) {
    case "document.created":
      return "created";
    case "document.sent":
      return "sent for signing";
    case "document.completed":
      return "completed";
    case "document.declined":
      return "declined";
    case "document.deleted":
      return "deleted";
    case "signer.added":
      return "added a signer to";
    case "signer.viewed":
      return "opened";
    case "signer.signed":
      return "signed";
    case "signer.declined":
      return "declined";
    case "signer.reminded":
      return "was reminded about";
    case "test.ping":
      return "test event from";
    default:
      return eventType;
  }
}

function emoji(eventType: string): string {
  switch (eventType) {
    case "document.completed":
    case "signer.signed":
      return "✅";
    case "document.declined":
    case "signer.declined":
      return "❌";
    case "document.sent":
      return "📨";
    case "signer.viewed":
      return "👀";
    case "signer.reminded":
      return "🔔";
    case "document.deleted":
      return "🗑️";
    case "document.created":
    case "signer.added":
      return "📝";
    case "test.ping":
      return "🛎️";
    default:
      return "ℹ️";
  }
}

export const plumeWebhookProvider: WebhookProviderDefinition = {
  type: "plume",
  label: "Plume",
  defaultPath: "/webhook/plume",
  defaultSecretHeader: "x-plume-signature-256",

  async authenticate(req, rawBody, settings) {
    if (!settings.secretValue) return true;
    const header = req.headers.get(settings.secretHeader ?? "x-plume-signature-256");
    if (!header) return false;
    const expected = "sha256=" + (await hmacSha256Hex(settings.secretValue, rawBody));
    return timingSafeEqual(header, expected);
  },

  formatMessage(payload) {
    const parsed = parsePayload(payload);
    if (!parsed) return null;

    const icon = emoji(parsed.event_type);
    const verb = actionVerb(parsed.event_type);
    const docName = parsed.document.name || parsed.document.file_name || "document";

    const parts: string[] = [];

    if (parsed.signer && parsed.event_type.startsWith("signer.")) {
      parts.push(
        `${icon} <b>${escapeHtml(parsed.signer.name)}</b> ${escapeHtml(verb)} <b>${escapeHtml(docName)}</b>`,
      );
    } else {
      parts.push(`${icon} <b>${escapeHtml(docName)}</b> ${escapeHtml(verb)}`);
    }

    if (parsed.signer?.email) {
      parts.push(`Signer: ${escapeHtml(parsed.signer.email)}`);
    }
    if (parsed.document.url) {
      parts.push(`<a href="${escapeHtml(parsed.document.url)}">Open document</a>`);
    }

    return parts.join("<br/>");
  },

  describeEvent(payload) {
    const parsed = parsePayload(payload);
    if (!parsed) return "unknown plume event";
    const subject = parsed.signer?.name ?? parsed.document.name ?? "plume";
    return `${parsed.event_type}: ${subject}`;
  },

  buildEvent(payload) {
    const parsed = parsePayload(payload);
    if (!parsed) return {};

    const docName = parsed.document.name || parsed.document.file_name || "document";
    const occurredAt =
      parsed.occurred_at ??
      parsed.signer?.signed_at ??
      parsed.signer?.viewed_at ??
      parsed.document.updated_at ??
      null;

    let status: string | null = null;
    switch (parsed.event_type) {
      case "document.completed":
      case "signer.signed":
        status = "completed";
        break;
      case "document.declined":
      case "signer.declined":
        status = "declined";
        break;
      case "document.sent":
      case "signer.added":
      case "signer.viewed":
      case "signer.reminded":
        status = "pending";
        break;
      case "document.created":
        status = "draft";
        break;
      case "document.deleted":
        status = "deleted";
        break;
    }

    return {
      eventType: `plume.${parsed.event_type}`,
      eventTitle: docName,
      eventUrl: parsed.document.url ?? parsed.signer?.signing_url ?? null,
      occurredAt,
      actor: parsed.signer
        ? {
            id: String(parsed.signer.id),
            name: parsed.signer.name,
            email: parsed.signer.email,
          }
        : null,
      subject: {
        type: "contract",
        id: String(parsed.document.id),
        name: docName,
        url: parsed.document.url ?? null,
      },
      normalizedPayload: {
        eventId: parsed.event_id ?? null,
        eventType: parsed.event_type,
        documentId: parsed.document.id,
        documentName: docName,
        documentStatus: parsed.document.status,
        sequential: parsed.document.sequential ?? false,
        signerId: parsed.signer?.id ?? null,
        signerName: parsed.signer?.name ?? null,
        signerEmail: parsed.signer?.email ?? null,
        signerRole: parsed.signer?.role ?? null,
        signerStatus: parsed.signer?.status ?? null,
        signerOrderNum: parsed.signer?.order_num ?? null,
        signedAt: parsed.signer?.signed_at ?? null,
        viewedAt: parsed.signer?.viewed_at ?? null,
        status,
      },
      containsPii: true,
    };
  },
};
