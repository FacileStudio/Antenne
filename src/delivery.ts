import type {
  DeliveryConfig,
  DiscordDeliveryConfig,
  HttpDeliveryConfig,
  MatrixDeliveryConfig,
} from "./types";

let txnId = 0;

function htmlToPlainText(message: string) {
  return message
    .replaceAll(/<br\s*\/?>/gi, "\n")
    .replaceAll(/<\/p>/gi, "\n\n")
    .replaceAll(/<[^>]*>/g, "")
    .trim();
}

async function requestMatrix(config: MatrixDeliveryConfig, method: string, path: string, body?: unknown) {
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

async function sendToMatrixRoom(config: MatrixDeliveryConfig, roomId: string, message: string) {
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

async function sendMatrixAlert(config: MatrixDeliveryConfig, message: string) {
  const results = await Promise.allSettled(config.roomIds.map((roomId) => sendToMatrixRoom(config, roomId, message)));

  for (const result of results) {
    if (result.status === "rejected") {
      console.error(`Failed to send Matrix alert: ${result.reason}`);
    }
  }
}

async function sendDiscordAlert(config: DiscordDeliveryConfig, message: string) {
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

async function sendHttpAlert(config: HttpDeliveryConfig, message: string) {
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

export function hasMatrixDelivery(config: MatrixDeliveryConfig): boolean {
  return Boolean(config.enabled && config.homeserver && config.accessToken && config.roomIds.length > 0);
}

export function hasDiscordDelivery(config: DiscordDeliveryConfig): boolean {
  return Boolean(config.enabled && config.webhookUrl);
}

export function hasHttpDelivery(config: HttpDeliveryConfig): boolean {
  return Boolean(config.enabled && config.url);
}

export function hasAnyDelivery(config: DeliveryConfig): boolean {
  return hasMatrixDelivery(config.matrix) || hasDiscordDelivery(config.discord) || hasHttpDelivery(config.webhook);
}

export function loadDeliveryConfigFromEnv(): DeliveryConfig {
  const roomIdsRaw = process.env.MATRIX_ROOM_IDS ?? process.env.MATRIX_ROOM_ID;
  const roomIds = roomIdsRaw
    ? roomIdsRaw
      .split(",")
      .map((id) => id.trim())
      .filter(Boolean)
    : [];

  return {
    matrix: {
      enabled: process.env.MATRIX_ENABLED !== "false",
      homeserver: process.env.MATRIX_HOMESERVER ?? "",
      accessToken: process.env.MATRIX_ACCESS_TOKEN ?? "",
      roomIds,
    },
    discord: {
      enabled: process.env.DISCORD_ENABLED === "true" || Boolean(process.env.DISCORD_WEBHOOK_URL),
      webhookUrl: process.env.DISCORD_WEBHOOK_URL ?? "",
    },
    webhook: {
      enabled: process.env.HTTP_WEBHOOK_ENABLED === "true" || Boolean(process.env.HTTP_WEBHOOK_URL),
      url: process.env.HTTP_WEBHOOK_URL ?? "",
      secretHeader: process.env.HTTP_WEBHOOK_SECRET_HEADER ?? "x-nook-secret",
      secretValue: process.env.HTTP_WEBHOOK_SECRET ?? null,
    },
  };
}

export async function sendAlert(config: DeliveryConfig, message: string) {
  if (!hasAnyDelivery(config)) {
    throw new Error("No delivery channel is configured.");
  }

  const tasks: Promise<void>[] = [];

  if (hasMatrixDelivery(config.matrix)) {
    tasks.push(sendMatrixAlert(config.matrix, message));
  }

  if (hasDiscordDelivery(config.discord)) {
    tasks.push(sendDiscordAlert(config.discord, message));
  }

  if (hasHttpDelivery(config.webhook)) {
    tasks.push(sendHttpAlert(config.webhook, message));
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
