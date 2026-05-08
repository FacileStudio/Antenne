import { createHash } from "node:crypto";

import { htmlToPlainText } from "./delivery-utils";

export type NookEventSource = "webhook" | "website" | "imap" | "test" | "system";
export type NookSubjectType = "task" | "contract" | "client" | "deployment" | "repository" | "email" | "service" | "schedule" | "system" | "unknown";

export interface NookEventActor {
  id: string | null;
  name: string | null;
  email: string | null;
}

export interface NookEventSubject {
  type: NookSubjectType;
  id: string | null;
  name: string | null;
  url: string | null;
}

export interface NookEventEnvelope {
  schemaVersion: 1;
  id: string;
  occurredAt: string;
  emittedAt: string;
  source: NookEventSource;
  provider: {
    id: string | null;
    type: string;
    name: string;
    tags: string[];
  };
  event: {
    type: string;
    title: string | null;
    messageHtml: string | null;
    messageText: string | null;
    url: string | null;
  };
  actor: NookEventActor | null;
  subject: NookEventSubject;
  payload: {
    normalized: Record<string, unknown>;
    raw: unknown | null;
  };
  privacy: {
    containsPii: boolean;
    rawPayloadAllowed: boolean;
  };
}

export interface NookEventEnvelopeInput {
  id?: string | null;
  source: NookEventSource;
  providerId: string | null;
  providerType: string;
  providerName: string;
  providerTags?: string[];
  eventType: string;
  eventTitle?: string | null;
  messageHtml?: string | null;
  messageText?: string | null;
  eventUrl?: string | null;
  actor?: Partial<NookEventActor> | null;
  subject?: Partial<NookEventSubject> | null;
  normalizedPayload?: Record<string, unknown> | null;
  rawPayload?: unknown | null;
  occurredAt?: string | null;
  emittedAt?: string | null;
  containsPii?: boolean;
  rawPayloadAllowed?: boolean;
}

function compactString(value: unknown): string | null {
  return typeof value === "string" && value.trim() ? value.trim() : null;
}

function normalizeActor(actor?: Partial<NookEventActor> | null): NookEventActor | null {
  if (!actor) {
    return null;
  }

  const normalized = {
    id: compactString(actor.id),
    name: compactString(actor.name),
    email: compactString(actor.email),
  };

  return normalized.id || normalized.name || normalized.email ? normalized : null;
}

function normalizeSubject(subject?: Partial<NookEventSubject> | null): NookEventSubject {
  return {
    type: subject?.type ?? "unknown",
    id: compactString(subject?.id),
    name: compactString(subject?.name),
    url: compactString(subject?.url),
  };
}

function stableJson(value: unknown): string {
  if (value === null || typeof value !== "object") {
    return JSON.stringify(value);
  }

  if (Array.isArray(value)) {
    return `[${value.map((item) => stableJson(item)).join(",")}]`;
  }

  return `{${Object.entries(value as Record<string, unknown>)
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([key, item]) => `${JSON.stringify(key)}:${stableJson(item)}`)
    .join(",")}}`;
}

function createEnvelopeId(input: NookEventEnvelopeInput, occurredAt: string) {
  if (input.id) {
    return input.id;
  }

  const hash = createHash("sha256");
  hash.update(input.providerId ?? "");
  hash.update("\0");
  hash.update(input.providerType);
  hash.update("\0");
  hash.update(input.eventType);
  hash.update("\0");
  hash.update(occurredAt);
  hash.update("\0");
  hash.update(stableJson(input.normalizedPayload ?? input.rawPayload ?? input.messageText ?? input.messageHtml ?? ""));
  return `nook_${hash.digest("hex").slice(0, 32)}`;
}

export function buildNookEventEnvelope(input: NookEventEnvelopeInput): NookEventEnvelope {
  const emittedAt = input.emittedAt ?? new Date().toISOString();
  const occurredAt = input.occurredAt ?? emittedAt;
  const messageHtml = input.messageHtml ?? null;
  const messageText = input.messageText ?? (messageHtml ? htmlToPlainText(messageHtml) : null);
  const normalizedPayload = input.normalizedPayload ?? {};
  const rawPayloadAllowed = input.rawPayloadAllowed ?? false;

  return {
    schemaVersion: 1,
    id: createEnvelopeId(input, occurredAt),
    occurredAt,
    emittedAt,
    source: input.source,
    provider: {
      id: input.providerId,
      type: input.providerType,
      name: input.providerName,
      tags: input.providerTags ?? [],
    },
    event: {
      type: input.eventType,
      title: input.eventTitle ?? input.eventType,
      messageHtml,
      messageText,
      url: input.eventUrl ?? input.subject?.url ?? null,
    },
    actor: normalizeActor(input.actor),
    subject: normalizeSubject(input.subject),
    payload: {
      normalized: normalizedPayload,
      raw: rawPayloadAllowed ? input.rawPayload ?? null : null,
    },
    privacy: {
      containsPii: input.containsPii ?? false,
      rawPayloadAllowed,
    },
  };
}

export function redactEnvelopeRawPayload(envelope: NookEventEnvelope): NookEventEnvelope {
  return {
    ...envelope,
    payload: {
      ...envelope.payload,
      raw: null,
    },
    privacy: {
      ...envelope.privacy,
      rawPayloadAllowed: false,
    },
  };
}
