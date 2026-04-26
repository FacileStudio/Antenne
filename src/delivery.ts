import {
  isDiscordDeliveryTarget,
  isHttpDeliveryTarget,
  isMatrixDeliveryTarget,
} from "./delivery-config";
import { createDefaultDeliveryTarget } from "./delivery-config";
import type {
  DeliveryConfig,
  DeliveryTarget,
  DiscordDeliveryTarget,
  HttpDeliveryTarget,
  MatrixDeliveryTarget,
} from "./types";

let txnId = 0;

function htmlToPlainText(message: string) {
  return message
    .replaceAll(/<br\s*\/?>/gi, "\n")
    .replaceAll(/<\/p>/gi, "\n\n")
    .replaceAll(/<[^>]*>/g, "")
    .trim();
}

async function requestMatrix(config: MatrixDeliveryTarget, method: string, path: string, body?: unknown) {
  const base = config.homeserver.endsWith("/") ? config.homeserver : `${config.homeserver}/`;
  const url = new URL(path, base);

  const res = await fetch(url, {
    method,
    headers: {
      Authorization: `Bearer ${config.accessToken}`,
      "Content-Type": "application/json",
      "User-Agent": "Nook/1.0 uptime-bot",
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Matrix API ${method} ${path} -> HTTP ${res.status}${text ? ` ${text}` : ""}`);
  }
}

async function sendToMatrixRoom(config: MatrixDeliveryTarget, roomId: string, message: string) {
  txnId += 1;
  const txn = `${Date.now()}-${txnId}`;
  const path = `/_matrix/client/v3/rooms/${encodeURIComponent(roomId)}/send/m.room.message/${encodeURIComponent(txn)}`;

  await requestMatrix(config, "PUT", path, {
    msgtype: "m.text",
    body: htmlToPlainText(message),
    format: "org.matrix.custom.html",
    formatted_body: message,
  });
}

async function sendMatrixAlert(config: MatrixDeliveryTarget, message: string) {
  const results = await Promise.allSettled(config.roomIds.map((roomId) => sendToMatrixRoom(config, roomId, message)));

  for (const result of results) {
    if (result.status === "rejected") {
      console.error(`Failed to send Matrix alert: ${result.reason}`);
    }
  }
}

async function sendDiscordAlert(config: DiscordDeliveryTarget, message: string) {
  const res = await fetch(config.webhookUrl, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "User-Agent": "Nook/1.0 uptime-bot",
    },
    body: JSON.stringify({
      content: htmlToPlainText(message),
    }),
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Discord webhook -> HTTP ${res.status}${text ? ` ${text}` : ""}`);
  }
}

async function sendHttpAlert(config: HttpDeliveryTarget, message: string) {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    "User-Agent": "Nook/1.0 uptime-bot",
  };

  if (config.secretHeader && config.secretValue) {
    headers[config.secretHeader] = config.secretValue;
  }

  const res = await fetch(config.url, {
    method: "POST",
    headers,
    body: JSON.stringify({
      source: "nook",
      message,
      text: htmlToPlainText(message),
      sentAt: new Date().toISOString(),
    }),
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(`HTTP delivery webhook -> HTTP ${res.status}${text ? ` ${text}` : ""}`);
  }
}

export function hasMatrixDelivery(config: MatrixDeliveryTarget): boolean {
  return Boolean(config.enabled && config.homeserver && config.accessToken && config.roomIds.length > 0);
}

export function hasDiscordDelivery(config: DiscordDeliveryTarget): boolean {
  return Boolean(config.enabled && config.webhookUrl);
}

export function hasHttpDelivery(config: HttpDeliveryTarget): boolean {
  return Boolean(config.enabled && config.url);
}

export function hasAnyDelivery(config: DeliveryConfig): boolean {
  return config.items.some((target) => {
    if (isMatrixDeliveryTarget(target)) {
      return hasMatrixDelivery(target);
    }

    if (isDiscordDeliveryTarget(target)) {
      return hasDiscordDelivery(target);
    }

    return hasHttpDelivery(target);
  });
}

export function hasConfiguredMatrixDelivery(config: DeliveryConfig): boolean {
  return config.items.some((target) => isMatrixDeliveryTarget(target) && hasMatrixDelivery(target));
}

export function loadDeliveryConfigFromEnv(): DeliveryConfig {
  const items: DeliveryTarget[] = [];
  const roomIdsRaw = process.env.MATRIX_ROOM_IDS ?? process.env.MATRIX_ROOM_ID;
  const roomIds = roomIdsRaw
    ? roomIdsRaw
      .split(",")
      .map((id) => id.trim())
      .filter(Boolean)
    : [];

  const matrix = createDefaultDeliveryTarget("matrix") as MatrixDeliveryTarget;
  matrix.enabled = process.env.MATRIX_ENABLED !== "false";
  matrix.homeserver = process.env.MATRIX_HOMESERVER ?? "";
  matrix.accessToken = process.env.MATRIX_ACCESS_TOKEN ?? "";
  matrix.roomIds = roomIds;
  items.push(matrix);

  const discord = createDefaultDeliveryTarget("discord") as DiscordDeliveryTarget;
  discord.enabled = process.env.DISCORD_ENABLED === "true" || Boolean(process.env.DISCORD_WEBHOOK_URL);
  discord.webhookUrl = process.env.DISCORD_WEBHOOK_URL ?? "";
  items.push(discord);

  const webhook = createDefaultDeliveryTarget("webhook") as HttpDeliveryTarget;
  webhook.enabled = process.env.HTTP_WEBHOOK_ENABLED === "true" || Boolean(process.env.HTTP_WEBHOOK_URL);
  webhook.url = process.env.HTTP_WEBHOOK_URL ?? "";
  webhook.secretHeader = process.env.HTTP_WEBHOOK_SECRET_HEADER ?? "x-nook-secret";
  webhook.secretValue = process.env.HTTP_WEBHOOK_SECRET ?? null;
  items.push(webhook);

  return { items };
}

export async function sendAlert(config: DeliveryConfig, message: string) {
  if (!hasAnyDelivery(config)) {
    throw new Error("No delivery channel is configured.");
  }

  const tasks: Promise<void>[] = [];

  for (const target of config.items) {
    if (isMatrixDeliveryTarget(target) && hasMatrixDelivery(target)) {
      tasks.push(sendMatrixAlert(target, message));
      continue;
    }

    if (isDiscordDeliveryTarget(target) && hasDiscordDelivery(target)) {
      tasks.push(sendDiscordAlert(target, message));
      continue;
    }

    if (isHttpDeliveryTarget(target) && hasHttpDelivery(target)) {
      tasks.push(sendHttpAlert(target, message));
    }
  }

  const results = await Promise.allSettled(tasks);
  const failures = results.filter((result) => result.status === "rejected");

  for (const failure of failures) {
    console.error(`Failed to send alert: ${failure.reason}`);
  }

  if (failures.length === tasks.length) {
    throw new Error("All configured delivery channels failed.");
  }
}
