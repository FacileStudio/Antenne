import { connect as connectNet } from "node:net";
import { connect as connectTls } from "node:tls";

import {
  isDiscordDeliveryTarget,
  isHttpDeliveryTarget,
  isMatrixDeliveryTarget,
  isNtfyDeliveryTarget,
  isSmtpDeliveryTarget,
} from "./delivery-config";
import { createDefaultDeliveryTarget } from "./delivery-config";
import type { DeliveryAttempt } from "./event-log";
import type {
  DeliveryConfig,
  DeliveryTarget,
  DiscordDeliveryTarget,
  HttpDeliveryTarget,
  MatrixDeliveryTarget,
  NtfyDeliveryTarget,
  SmtpDeliveryTarget,
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

function normalizeNtfyServerUrl(serverUrl: string) {
  return serverUrl.endsWith("/") ? serverUrl : `${serverUrl}/`;
}

function normalizeNtfyPriority(value: string | undefined): number | null {
  if (!value) {
    return 3;
  }

  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed < 1) {
    return 3;
  }

  return Math.trunc(parsed);
}

async function sendNtfyAlert(config: NtfyDeliveryTarget, message: string) {
  const topic = encodeURIComponent(config.topic);
  const url = new URL(topic, normalizeNtfyServerUrl(config.serverUrl));
  const headers: Record<string, string> = {
    "Content-Type": "text/plain; charset=utf-8",
    "User-Agent": "Nook/1.0 uptime-bot",
    Title: "Nook Alert",
  };

  if (config.accessToken) {
    headers.Authorization = `Bearer ${config.accessToken}`;
  }

  if (config.priority !== null) {
    headers.Priority = String(config.priority);
  }

  if (config.tags.length > 0) {
    headers.Tags = config.tags.join(",");
  }

  if (config.clickUrl) {
    headers.Click = config.clickUrl;
  }

  const res = await fetch(url, {
    method: "POST",
    headers,
    body: htmlToPlainText(message),
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(`ntfy -> HTTP ${res.status}${text ? ` ${text}` : ""}`);
  }
}

function smtpCommand(socket: import("node:net").Socket, command: string): Promise<string> {
  return new Promise((resolve, reject) => {
    let response = "";
    const onData = (chunk: Buffer) => {
      response += chunk.toString();
      if (/^\d{3} /m.test(response)) {
        socket.off("data", onData);
        const code = parseInt(response.slice(0, 3), 10);
        if (code >= 400) {
          reject(new Error(`SMTP error: ${response.trim()}`));
        } else {
          resolve(response);
        }
      }
    };
    socket.on("data", onData);
    if (command) {
      socket.write(`${command}\r\n`);
    }
  });
}

async function sendSmtpAlert(config: SmtpDeliveryTarget, message: string) {
  const plainText = htmlToPlainText(message);
  const subject = config.subjectPrefix ? `${config.subjectPrefix} Nook Alert` : "Nook Alert";
  const boundary = `nook-${Date.now()}`;

  const body = [
    `From: ${config.fromAddress}`,
    `To: ${config.toAddresses.join(", ")}`,
    `Subject: ${subject}`,
    `MIME-Version: 1.0`,
    `Content-Type: multipart/alternative; boundary="${boundary}"`,
    ``,
    `--${boundary}`,
    `Content-Type: text/plain; charset=utf-8`,
    `Content-Transfer-Encoding: 8bit`,
    ``,
    plainText,
    `--${boundary}`,
    `Content-Type: text/html; charset=utf-8`,
    `Content-Transfer-Encoding: 8bit`,
    ``,
    message,
    `--${boundary}--`,
  ].join("\r\n");

  const rawSocket = config.secure
    ? connectTls({ host: config.host, port: config.port, servername: config.host })
    : connectNet({ host: config.host, port: config.port });

  const socket = await new Promise<import("node:net").Socket>((resolve, reject) => {
    rawSocket.once("error", reject);
    if (config.secure) {
      (rawSocket as import("node:tls").TLSSocket).once("secureConnect", () => resolve(rawSocket));
    } else {
      rawSocket.once("connect", () => resolve(rawSocket));
    }
  });

  try {
    await smtpCommand(socket, "");
    await smtpCommand(socket, `EHLO nook`);

    if (!config.secure) {
      try {
        await smtpCommand(socket, "STARTTLS");
      } catch {}
    }

    if (config.username && config.password) {
      const credentials = Buffer.from(`\0${config.username}\0${config.password}`).toString("base64");
      await smtpCommand(socket, `AUTH PLAIN ${credentials}`);
    }

    await smtpCommand(socket, `MAIL FROM:<${config.fromAddress}>`);
    for (const to of config.toAddresses) {
      await smtpCommand(socket, `RCPT TO:<${to}>`);
    }
    await smtpCommand(socket, "DATA");
    socket.write(`${body}\r\n.\r\n`);
    await smtpCommand(socket, "");
    await smtpCommand(socket, "QUIT").catch(() => {});
  } finally {
    socket.end();
    socket.destroy();
  }
}

export function hasSmtpDelivery(config: SmtpDeliveryTarget): boolean {
  return Boolean(config.enabled && config.host && config.fromAddress && config.toAddresses.length > 0);
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

export function hasNtfyDelivery(config: NtfyDeliveryTarget): boolean {
  return Boolean(config.enabled && config.serverUrl && config.topic);
}

export function hasAnyDelivery(config: DeliveryConfig): boolean {
  return config.items.some((target) => canSendToDeliveryTarget(target));
}

export function hasConfiguredMatrixDelivery(config: DeliveryConfig): boolean {
  return config.items.some((target) => isMatrixDeliveryTarget(target) && hasMatrixDelivery(target));
}

export function canSendToDeliveryTarget(target: DeliveryTarget): boolean {
  if (isMatrixDeliveryTarget(target)) {
    return hasMatrixDelivery(target);
  }

  if (isDiscordDeliveryTarget(target)) {
    return hasDiscordDelivery(target);
  }

  if (isSmtpDeliveryTarget(target)) {
    return hasSmtpDelivery(target);
  }

  if (isNtfyDeliveryTarget(target)) {
    return hasNtfyDelivery(target);
  }

  return hasHttpDelivery(target);
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
  matrix.providerIds = [];
  items.push(matrix);

  const discord = createDefaultDeliveryTarget("discord") as DiscordDeliveryTarget;
  discord.enabled = process.env.DISCORD_ENABLED === "true" || Boolean(process.env.DISCORD_WEBHOOK_URL);
  discord.webhookUrl = process.env.DISCORD_WEBHOOK_URL ?? "";
  discord.providerIds = [];
  items.push(discord);

  const webhook = createDefaultDeliveryTarget("webhook") as HttpDeliveryTarget;
  webhook.enabled = process.env.HTTP_WEBHOOK_ENABLED === "true" || Boolean(process.env.HTTP_WEBHOOK_URL);
  webhook.url = process.env.HTTP_WEBHOOK_URL ?? "";
  webhook.secretHeader = process.env.HTTP_WEBHOOK_SECRET_HEADER ?? "x-nook-secret";
  webhook.secretValue = process.env.HTTP_WEBHOOK_SECRET ?? null;
  webhook.providerIds = [];
  items.push(webhook);

  const ntfy = createDefaultDeliveryTarget("ntfy") as NtfyDeliveryTarget;
  ntfy.enabled = process.env.NTFY_ENABLED === "true" || Boolean(process.env.NTFY_TOPIC);
  ntfy.serverUrl = process.env.NTFY_SERVER_URL ?? "https://ntfy.sh";
  ntfy.topic = process.env.NTFY_TOPIC ?? "";
  ntfy.accessToken = process.env.NTFY_ACCESS_TOKEN ?? null;
  ntfy.priority = normalizeNtfyPriority(process.env.NTFY_PRIORITY);
  ntfy.tags = process.env.NTFY_TAGS
    ? process.env.NTFY_TAGS.split(",").map((tag) => tag.trim()).filter(Boolean)
    : ["rotating_light"];
  ntfy.clickUrl = process.env.NTFY_CLICK_URL ?? null;
  ntfy.providerIds = [];
  items.push(ntfy);

  return { items };
}

function targetAcceptsProvider(target: DeliveryTarget, providerId: string | null): boolean {
  if (!target.providerIds || target.providerIds.length === 0) return true;
  if (!providerId) return true;
  return target.providerIds.includes(providerId);
}

export async function sendAlert(config: DeliveryConfig, message: string, providerId?: string | null): Promise<DeliveryAttempt[]> {
  if (!hasAnyDelivery(config)) {
    throw new Error("No delivery channel is configured.");
  }

  const targets: DeliveryTarget[] = [];
  const tasks: Promise<void>[] = [];

  for (const target of config.items) {
    if (!targetAcceptsProvider(target, providerId ?? null)) continue;

    if (isMatrixDeliveryTarget(target) && hasMatrixDelivery(target)) {
      targets.push(target);
      tasks.push(sendMatrixAlert(target, message));
      continue;
    }

    if (isDiscordDeliveryTarget(target) && hasDiscordDelivery(target)) {
      targets.push(target);
      tasks.push(sendDiscordAlert(target, message));
      continue;
    }

    if (isSmtpDeliveryTarget(target) && hasSmtpDelivery(target)) {
      targets.push(target);
      tasks.push(sendSmtpAlert(target, message));
      continue;
    }

    if (isNtfyDeliveryTarget(target) && hasNtfyDelivery(target)) {
      targets.push(target);
      tasks.push(sendNtfyAlert(target, message));
      continue;
    }

    if (isHttpDeliveryTarget(target) && hasHttpDelivery(target)) {
      targets.push(target);
      tasks.push(sendHttpAlert(target, message));
    }
  }

  const startTimes = targets.map(() => Date.now());
  const results = await Promise.allSettled(tasks);
  const now = Date.now();

  const attempts: DeliveryAttempt[] = results.map((result, index) => {
    const target = targets[index]!;
    return {
      targetId: target.id,
      targetName: target.name,
      targetType: target.type,
      status: result.status === "fulfilled" ? "success" as const : "error" as const,
      error: result.status === "rejected" ? String(result.reason) : undefined,
      durationMs: now - startTimes[index]!,
    };
  });

  const failures = attempts.filter((a) => a.status === "error");
  for (const failure of failures) {
    console.error(`Failed to send alert: ${failure.error}`);
  }

  if (failures.length === tasks.length) {
    throw new Error("All configured delivery channels failed.");
  }

  return attempts;
}

export async function sendAlertToTarget(target: DeliveryTarget, message: string) {
  if (isMatrixDeliveryTarget(target) && hasMatrixDelivery(target)) {
    await sendMatrixAlert(target, message);
    return;
  }

  if (isDiscordDeliveryTarget(target) && hasDiscordDelivery(target)) {
    await sendDiscordAlert(target, message);
    return;
  }

  if (isSmtpDeliveryTarget(target) && hasSmtpDelivery(target)) {
    await sendSmtpAlert(target, message);
    return;
  }

  if (isNtfyDeliveryTarget(target) && hasNtfyDelivery(target)) {
    await sendNtfyAlert(target, message);
    return;
  }

  if (isHttpDeliveryTarget(target) && hasHttpDelivery(target)) {
    await sendHttpAlert(target, message);
    return;
  }

  throw new Error(`Target \`${target.name}\` is not fully configured.`);
}
