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
const matrixSessionCache = new Map<string, MatrixSession>();
const matrixSessionInFlight = new Map<string, Promise<MatrixSession>>();

type MatrixSession = {
  accessToken: string;
  refreshToken: string | null;
  userId: string | null;
  deviceId: string | null;
  expiresAt: number | null;
};

class MatrixApiError extends Error {
  readonly errcode: string | null;
  readonly softLogout: boolean;
  readonly status: number;

  constructor(message: string, options: { errcode?: string | null; softLogout?: boolean; status: number }) {
    super(message);
    this.name = "MatrixApiError";
    this.errcode = options.errcode ?? null;
    this.softLogout = options.softLogout ?? false;
    this.status = options.status;
  }
}

function htmlToPlainText(message: string) {
  return message
    .replaceAll(/<br\s*\/?>/gi, "\n")
    .replaceAll(/<\/p>/gi, "\n\n")
    .replaceAll(/<[^>]*>/g, "")
    .trim();
}

function getMatrixBaseUrl(homeserver: string) {
  return homeserver.endsWith("/") ? homeserver : `${homeserver}/`;
}

function getMatrixSessionFromConfig(config: MatrixDeliveryTarget): MatrixSession {
  const cached = matrixSessionCache.get(config.id);

  if (cached) {
    if (config.accessToken && config.accessToken !== cached.accessToken) {
      cached.accessToken = config.accessToken;
    }

    if (config.refreshToken && config.refreshToken !== cached.refreshToken) {
      cached.refreshToken = config.refreshToken;
    }

    if (config.userId && config.userId !== cached.userId) {
      cached.userId = config.userId;
    }

    if (config.deviceId && config.deviceId !== cached.deviceId) {
      cached.deviceId = config.deviceId;
    }

    return cached;
  }

  const session = {
    accessToken: config.accessToken,
    refreshToken: config.refreshToken,
    userId: config.userId || null,
    deviceId: config.deviceId,
    expiresAt: null,
  };
  matrixSessionCache.set(config.id, session);
  return session;
}

function hasMatrixPasswordAuth(config: MatrixDeliveryTarget) {
  return Boolean(config.userId && config.password);
}

function isMatrixUnknownTokenError(error: unknown) {
  return error instanceof MatrixApiError && error.errcode === "M_UNKNOWN_TOKEN";
}

async function requestMatrix(
  homeserver: string,
  accessToken: string,
  method: string,
  path: string,
  body?: unknown,
  headers?: Record<string, string>,
) {
  const base = getMatrixBaseUrl(homeserver);
  const url = new URL(path, base);

  const res = await fetch(url, {
    method,
    headers: {
      Authorization: `Bearer ${accessToken}`,
      "Content-Type": "application/json",
      "User-Agent": "Nook/1.0 uptime-bot",
      ...(headers ?? {}),
    } as Record<string, string>,
    body: body === undefined ? undefined : JSON.stringify(body),
  });

  if (!res.ok) {
    const text = await res.text();
    let errcode: string | null = null;
    let softLogout = false;

    try {
      const payload = JSON.parse(text) as { errcode?: string; soft_logout?: boolean };
      errcode = payload.errcode ?? null;
      softLogout = payload.soft_logout ?? false;
    } catch {}

    throw new MatrixApiError(
      `Matrix API ${method} ${path} -> HTTP ${res.status}${text ? ` ${text}` : ""}`,
      { errcode, softLogout, status: res.status },
    );
  }

  if (res.status === 204) {
    return null;
  }

  const text = await res.text();
  if (!text) {
    return null;
  }

  return JSON.parse(text) as unknown;
}

async function loginMatrix(config: MatrixDeliveryTarget): Promise<MatrixSession> {
  if (!config.userId || !config.password) {
    throw new Error(`Matrix target \`${config.name}\` has no password login configured.`);
  }

  const loginUser = config.userId.startsWith("@") && config.userId.includes(":")
    ? config.userId.slice(1, config.userId.indexOf(":"))
    : config.userId;

  const response = await fetch(new URL("/_matrix/client/v3/login", getMatrixBaseUrl(config.homeserver)), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "User-Agent": "Nook/1.0 uptime-bot",
    },
    body: JSON.stringify({
      type: "m.login.password",
      identifier: {
        type: "m.id.user",
        user: loginUser,
      },
      password: config.password,
      device_id: config.deviceId ?? undefined,
      initial_device_display_name: "Nook",
      refresh_token: true,
    }),
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(`Matrix login failed -> HTTP ${response.status}${text ? ` ${text}` : ""}`);
  }

  const payload = await response.json() as {
    access_token: string;
    refresh_token?: string;
    user_id?: string;
    device_id?: string;
    expires_in_ms?: number;
  };

  const session = {
    accessToken: payload.access_token,
    refreshToken: payload.refresh_token ?? null,
    userId: payload.user_id ?? config.userId,
    deviceId: payload.device_id ?? config.deviceId,
    expiresAt: typeof payload.expires_in_ms === "number" ? Date.now() + payload.expires_in_ms : null,
  };

  matrixSessionCache.set(config.id, session);
  return session;
}

async function refreshMatrixSession(config: MatrixDeliveryTarget, session: MatrixSession): Promise<MatrixSession> {
  if (!session.refreshToken) {
    throw new Error(`Matrix target \`${config.name}\` has no refresh token to renew.`);
  }

  const payload = await fetch(new URL("/_matrix/client/v3/refresh", getMatrixBaseUrl(config.homeserver)), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "User-Agent": "Nook/1.0 uptime-bot",
    },
    body: JSON.stringify({
      refresh_token: session.refreshToken,
    }),
  });

  if (!payload.ok) {
    const text = await payload.text();
    let errcode: string | null = null;
    let softLogout = false;

    try {
      const parsed = JSON.parse(text) as { errcode?: string; soft_logout?: boolean };
      errcode = parsed.errcode ?? null;
      softLogout = parsed.soft_logout ?? false;
    } catch {}

    throw new MatrixApiError(
      `Matrix token refresh failed -> HTTP ${payload.status}${text ? ` ${text}` : ""}`,
      { errcode, softLogout, status: payload.status },
    );
  }

  const next = await payload.json() as {
    access_token: string;
    refresh_token?: string;
    expires_in_ms?: number;
  };

  const refreshed = {
    accessToken: next.access_token,
    refreshToken: next.refresh_token ?? session.refreshToken,
    userId: session.userId,
    deviceId: session.deviceId,
    expiresAt: typeof next.expires_in_ms === "number" ? Date.now() + next.expires_in_ms : null,
  };

  matrixSessionCache.set(config.id, refreshed);
  return refreshed;
}

async function ensureMatrixSession(config: MatrixDeliveryTarget, forceRefresh = false): Promise<MatrixSession> {
  const current = getMatrixSessionFromConfig(config);

  if (!forceRefresh && current.accessToken && (!current.expiresAt || current.expiresAt > Date.now() + 15_000)) {
    return current;
  }

  const existing = matrixSessionInFlight.get(config.id);
  if (existing) {
    return await existing;
  }

  const nextPromise = (async () => {
    const latest = getMatrixSessionFromConfig(config);

    if (!forceRefresh && latest.accessToken && (!latest.expiresAt || latest.expiresAt > Date.now() + 15_000)) {
      return latest;
    }

    if (latest.refreshToken) {
      try {
        return await refreshMatrixSession(config, latest);
      } catch (error) {
        if (!hasMatrixPasswordAuth(config)) {
          throw error;
        }
      }
    }

    if (hasMatrixPasswordAuth(config)) {
      return await loginMatrix(config);
    }

    if (latest.accessToken && !forceRefresh) {
      return latest;
    }

    throw new Error(`Matrix target \`${config.name}\` needs either an access token or password login.`);
  })();

  matrixSessionInFlight.set(config.id, nextPromise);

  try {
    return await nextPromise;
  } finally {
    matrixSessionInFlight.delete(config.id);
  }
}

async function sendToMatrixRoom(config: MatrixDeliveryTarget, roomId: string, message: string) {
  let session = await ensureMatrixSession(config);
  txnId += 1;
  const txn = `${Date.now()}-${txnId}`;
  const path = `/_matrix/client/v3/rooms/${encodeURIComponent(roomId)}/send/m.room.message/${encodeURIComponent(txn)}`;

  try {
    await requestMatrix(config.homeserver, session.accessToken, "PUT", path, {
      msgtype: "m.text",
      body: htmlToPlainText(message),
      format: "org.matrix.custom.html",
      formatted_body: message,
    });
  } catch (error) {
    if (!isMatrixUnknownTokenError(error)) {
      throw error;
    }

    session = await ensureMatrixSession(config, true);
    await requestMatrix(config.homeserver, session.accessToken, "PUT", path, {
      msgtype: "m.text",
      body: htmlToPlainText(message),
      format: "org.matrix.custom.html",
      formatted_body: message,
    });
  }
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
  return Boolean(
    config.enabled
    && config.homeserver
    && config.roomIds.length > 0
    && (config.accessToken || config.refreshToken || hasMatrixPasswordAuth(config)),
  );
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
  matrix.refreshToken = process.env.MATRIX_REFRESH_TOKEN ?? null;
  matrix.userId = process.env.MATRIX_USER_ID ?? "";
  matrix.password = process.env.MATRIX_PASSWORD ?? null;
  matrix.deviceId = process.env.MATRIX_DEVICE_ID ?? matrix.deviceId;
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
