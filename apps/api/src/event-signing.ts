import { createHmac } from "node:crypto";

export function createSignedEventHeaders(secretValue: string | null, eventId: string, body: string) {
  const timestamp = String(Math.floor(Date.now() / 1000));
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    "User-Agent": "Nook/1.0 uptime-bot",
    "x-nook-event-id": eventId,
    "x-nook-timestamp": timestamp,
    "x-nook-schema-version": "1",
  };

  if (secretValue) {
    headers["x-nook-signature-256"] = `sha256=${createHmac("sha256", secretValue)
      .update(`${timestamp}.${eventId}.${body}`)
      .digest("hex")}`;
  }

  return headers;
}
