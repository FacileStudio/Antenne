import type { MatrixConfig } from "./types";

let txnId = 0;

async function request(config: MatrixConfig, method: string, path: string, body?: unknown) {
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

  if (res.status === 204) return null;

  const text = await res.text();
  return text ? JSON.parse(text) : null;
}

async function sendToRoom(config: MatrixConfig, roomId: string, message: string) {
  txnId += 1;
  const txn = `${Date.now()}-${txnId}`;
  const path = `/_matrix/client/v3/rooms/${encodeURIComponent(roomId)}/send/m.room.message/${encodeURIComponent(txn)}`;

  await request(config, "PUT", path, {
    msgtype: "m.text",
    body: message.replace(/<[^>]*>/g, ""),
    format: "org.matrix.custom.html",
    formatted_body: message,
  });
}

export async function sendAlert(config: MatrixConfig, message: string) {
  const results = await Promise.allSettled(
    config.roomIds.map((roomId) => sendToRoom(config, roomId, message)),
  );

  for (const result of results) {
    if (result.status === "rejected") {
      console.error(`Failed to send Matrix alert: ${result.reason}`);
    }
  }
}

export function loadMatrixConfig(): MatrixConfig {
  const homeserver = process.env.MATRIX_HOMESERVER;
  const accessToken = process.env.MATRIX_ACCESS_TOKEN;

  const roomIdsRaw = process.env.MATRIX_ROOM_IDS ?? process.env.MATRIX_ROOM_ID;
  if (!homeserver || !accessToken || !roomIdsRaw) {
    throw new Error(
      "Missing Matrix configuration. Set MATRIX_HOMESERVER, MATRIX_ACCESS_TOKEN, and MATRIX_ROOM_IDS (or MATRIX_ROOM_ID) in the environment.",
    );
  }

  const roomIds = roomIdsRaw
    .split(",")
    .map((id) => id.trim())
    .filter(Boolean);

  if (roomIds.length === 0) {
    throw new Error("MATRIX_ROOM_IDS must contain at least one room ID.");
  }

  return { homeserver, accessToken, roomIds };
}
