import { htmlToPlainText } from "./delivery-utils";
import type { MatrixDeliveryTarget } from "./types";

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

  constructor(message: string, errcode?: string | null) {
    super(message);
    this.name = "MatrixApiError";
    this.errcode = errcode ?? null;
  }
}

function getMatrixBaseUrl(homeserver: string) {
  return homeserver.endsWith("/") ? homeserver : `${homeserver}/`;
}

function getMatrixSessionFromConfig(config: MatrixDeliveryTarget): MatrixSession {
  const cached = matrixSessionCache.get(config.id);

  if (cached) {
    if (config.accessToken !== cached.accessToken) {
      cached.accessToken = config.accessToken;
    }

    if (config.refreshToken !== cached.refreshToken) {
      cached.refreshToken = config.refreshToken;
    }

    const nextUserId = config.userId || null;
    if (nextUserId !== cached.userId) {
      cached.userId = nextUserId;
    }

    const nextDeviceId = config.deviceId || null;
    if (nextDeviceId !== cached.deviceId) {
      cached.deviceId = nextDeviceId;
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
) {
  const url = new URL(path, getMatrixBaseUrl(homeserver));
  const res = await fetch(url, {
    method,
    headers: {
      Authorization: `Bearer ${accessToken}`,
      "Content-Type": "application/json",
      "User-Agent": "Nook/1.0 uptime-bot",
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  });

  if (!res.ok) {
    const text = await res.text();
    let errcode: string | null = null;

    try {
      const payload = JSON.parse(text) as { errcode?: string };
      errcode = payload.errcode ?? null;
    } catch {}

    throw new MatrixApiError(`Matrix API ${method} ${path} -> HTTP ${res.status}${text ? ` ${text}` : ""}`, errcode);
  }
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

  const response = await fetch(new URL("/_matrix/client/v3/refresh", getMatrixBaseUrl(config.homeserver)), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "User-Agent": "Nook/1.0 uptime-bot",
    },
    body: JSON.stringify({
      refresh_token: session.refreshToken,
    }),
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(`Matrix token refresh failed -> HTTP ${response.status}${text ? ` ${text}` : ""}`);
  }

  const next = await response.json() as {
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
  const body = {
    msgtype: "m.text",
    body: htmlToPlainText(message),
    format: "org.matrix.custom.html",
    formatted_body: message,
  };

  try {
    await requestMatrix(config.homeserver, session.accessToken, "PUT", path, body);
  } catch (error) {
    if (!isMatrixUnknownTokenError(error)) {
      throw error;
    }

    session = await ensureMatrixSession(config, true);
    await requestMatrix(config.homeserver, session.accessToken, "PUT", path, body);
  }
}

export async function sendMatrixAlert(config: MatrixDeliveryTarget, message: string) {
  const results = await Promise.allSettled(config.roomIds.map((roomId) => sendToMatrixRoom(config, roomId, message)));
  const failures = results.filter((result): result is PromiseRejectedResult => result.status === "rejected");

  for (const failure of failures) {
    console.error(`Failed to send Matrix alert: ${failure.reason}`);
  }

  if (failures.length > 0) {
    throw new Error(`Matrix delivery failed for ${failures.length}/${config.roomIds.length} rooms.`);
  }
}

export function hasMatrixDelivery(config: MatrixDeliveryTarget): boolean {
  return Boolean(
    config.enabled
    && config.homeserver
    && config.roomIds.length > 0
    && (config.accessToken || config.refreshToken || hasMatrixPasswordAuth(config)),
  );
}
