import { redactEnvelopeRawPayload } from "./event-envelope";
import { createSignedEventHeaders } from "./event-signing";
import type {
  DeliveryMessage,
  HttpDeliveryTarget,
  PerceptionDeliveryTarget,
} from "./types";

function buildEnvelopeBody(delivery: DeliveryMessage, includeRawPayload: boolean) {
  return JSON.stringify(includeRawPayload ? delivery.envelope : redactEnvelopeRawPayload(delivery.envelope));
}

export async function sendHttpAlert(config: HttpDeliveryTarget, delivery: DeliveryMessage) {
  const body = buildEnvelopeBody(delivery, config.includeRawPayload);
  const res = await fetch(config.url, {
    method: "POST",
    headers: createSignedEventHeaders(config.secretValue, delivery.envelope.id, body),
    body,
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(`HTTP delivery webhook -> HTTP ${res.status}${text ? ` ${text}` : ""}`);
  }
}

export async function sendPerceptionEvent(config: PerceptionDeliveryTarget, delivery: DeliveryMessage) {
  const body = buildEnvelopeBody(delivery, config.includeRawPayload);
  const res = await fetch(config.url, {
    method: "POST",
    headers: createSignedEventHeaders(config.secretValue, delivery.envelope.id, body),
    body,
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Perception -> HTTP ${res.status}${text ? ` ${text}` : ""}`);
  }
}

export function hasHttpDelivery(config: HttpDeliveryTarget): boolean {
  return Boolean(config.enabled && config.url);
}

export function hasPerceptionDelivery(config: PerceptionDeliveryTarget): boolean {
  return Boolean(config.enabled && config.url && config.secretValue);
}
