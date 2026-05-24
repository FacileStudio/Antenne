import { timingSafeEqual as nodeTimingSafeEqual } from "node:crypto";
import type { WebhookProviderDefinition } from "../types";

interface NuageActor {
  id: number;
  email: string;
  name: string;
}

interface NuageWebhookPayload {
  event_type:
    | "file.uploaded"
    | "file.updated"
    | "file.deleted"
    | "file.shared"
    | "folder.created"
    | "folder.updated"
    | "folder.deleted"
    | "share.created"
    | "share.revoked";
  occurred_at: string;
  instance_name: string;
  actor: NuageActor;
  data: {
    file?: { id: number; name: string; mime_type: string; size: number };
    folder?: { id: number; name: string };
    share?: { id: number; shared_with_email?: string; permission: string };
  };
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

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}

function parsePayload(payload: unknown): NuageWebhookPayload | null {
  const candidate = payload as Record<string, unknown> | null | undefined;
  if (!candidate) return null;
  if (typeof candidate.event_type !== "string") return null;
  if (typeof candidate.actor !== "object" || candidate.actor === null) return null;
  if (typeof candidate.data !== "object" || candidate.data === null) return null;
  return candidate as unknown as NuageWebhookPayload;
}

function eventEmoji(eventType: string): string {
  switch (eventType) {
    case "file.uploaded":
      return "📁";
    case "file.updated":
      return "✏️";
    case "file.deleted":
      return "🗑️";
    case "file.shared":
      return "🔗";
    case "folder.created":
      return "📂";
    case "folder.updated":
      return "✏️";
    case "folder.deleted":
      return "🗑️";
    case "share.created":
      return "🔗";
    case "share.revoked":
      return "🚫";
    default:
      return "ℹ️";
  }
}

function subjectName(parsed: NuageWebhookPayload): string {
  return parsed.data.file?.name ?? parsed.data.folder?.name ?? "unknown";
}

export const nuageWebhookProvider: WebhookProviderDefinition = {
  type: "nuage",
  label: "Nuage",
  defaultPath: "/webhook/nuage",
  defaultSecretHeader: "x-nuage-signature-256",

  async authenticate(req, rawBody, settings) {
    if (!settings.secretValue) return true;
    const header = req.headers.get(settings.secretHeader ?? "x-nuage-signature-256");
    if (!header) return false;
    const expected = "sha256=" + (await hmacSha256Hex(settings.secretValue, rawBody));
    return timingSafeEqual(header, expected);
  },

  formatMessage(payload) {
    const parsed = parsePayload(payload);
    if (!parsed) return null;

    const icon = eventEmoji(parsed.event_type);
    const actorName = escapeHtml(parsed.actor.name);

    switch (parsed.event_type) {
      case "file.uploaded": {
        const file = parsed.data.file;
        if (!file) return null;
        return `${icon} <b>${actorName}</b> uploaded <b>${escapeHtml(file.name)}</b> (${formatSize(file.size)})`;
      }
      case "file.updated": {
        const file = parsed.data.file;
        if (!file) return null;
        return `${icon} <b>${actorName}</b> updated <b>${escapeHtml(file.name)}</b>`;
      }
      case "file.deleted": {
        const file = parsed.data.file;
        if (!file) return null;
        return `${icon} <b>${actorName}</b> deleted <b>${escapeHtml(file.name)}</b>`;
      }
      case "file.shared": {
        const file = parsed.data.file;
        const share = parsed.data.share;
        if (!file) return null;
        const target = share?.shared_with_email
          ? ` with ${escapeHtml(share.shared_with_email)}`
          : "";
        return `${icon} <b>${actorName}</b> shared <b>${escapeHtml(file.name)}</b>${target}`;
      }
      case "folder.created": {
        const folder = parsed.data.folder;
        if (!folder) return null;
        return `${icon} <b>${actorName}</b> created folder <b>${escapeHtml(folder.name)}</b>`;
      }
      case "folder.updated": {
        const folder = parsed.data.folder;
        if (!folder) return null;
        return `${icon} <b>${actorName}</b> updated folder <b>${escapeHtml(folder.name)}</b>`;
      }
      case "folder.deleted": {
        const folder = parsed.data.folder;
        if (!folder) return null;
        return `${icon} <b>${actorName}</b> deleted folder <b>${escapeHtml(folder.name)}</b>`;
      }
      case "share.created": {
        const share = parsed.data.share;
        const file = parsed.data.file;
        const fileName = file ? escapeHtml(file.name) : "a resource";
        const target = share?.shared_with_email
          ? ` with ${escapeHtml(share.shared_with_email)}`
          : "";
        return `${icon} <b>${actorName}</b> shared <b>${fileName}</b>${target}`;
      }
      case "share.revoked": {
        const share = parsed.data.share;
        const target = share?.shared_with_email
          ? ` for ${escapeHtml(share.shared_with_email)}`
          : "";
        return `${icon} <b>${actorName}</b> revoked share access${target}`;
      }
      default:
        return `ℹ️ <b>${actorName}</b> triggered <b>${escapeHtml(parsed.event_type)}</b>`;
    }
  },

  describeEvent(payload) {
    const parsed = parsePayload(payload);
    if (!parsed) return "unknown nuage event";
    return `${parsed.event_type}: ${subjectName(parsed)}`;
  },

  buildEvent(payload) {
    const parsed = parsePayload(payload);
    if (!parsed) return {};

    const name = subjectName(parsed);

    return {
      eventType: `nuage.${parsed.event_type}`,
      eventTitle: name,
      occurredAt: parsed.occurred_at,
      actor: {
        id: String(parsed.actor.id),
        name: parsed.actor.name,
        email: parsed.actor.email,
      },
      subject: {
        type: "unknown" as const,
        id: String(parsed.data.file?.id ?? parsed.data.folder?.id ?? parsed.data.share?.id ?? ""),
        name,
        url: null,
      },
      normalizedPayload: {
        eventType: parsed.event_type,
        instanceName: parsed.instance_name,
        actorId: parsed.actor.id,
        actorEmail: parsed.actor.email,
        actorName: parsed.actor.name,
        fileId: parsed.data.file?.id ?? null,
        fileName: parsed.data.file?.name ?? null,
        fileMimeType: parsed.data.file?.mime_type ?? null,
        fileSize: parsed.data.file?.size ?? null,
        folderId: parsed.data.folder?.id ?? null,
        folderName: parsed.data.folder?.name ?? null,
        shareId: parsed.data.share?.id ?? null,
        sharedWithEmail: parsed.data.share?.shared_with_email ?? null,
        sharePermission: parsed.data.share?.permission ?? null,
      },
      containsPii: true,
    };
  },
};
