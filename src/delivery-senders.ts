import { connect as connectNet } from "node:net";
import { connect as connectTls } from "node:tls";

import { htmlToPlainText } from "./delivery-utils";
import type {
  DiscordDeliveryTarget,
  HttpDeliveryTarget,
  NtfyDeliveryTarget,
  SmtpDeliveryTarget,
} from "./types";

function normalizeNtfyServerUrl(serverUrl: string) {
  return serverUrl.endsWith("/") ? serverUrl : `${serverUrl}/`;
}

type SmtpSocket = import("node:net").Socket | import("node:tls").TLSSocket;

function smtpCommand(socket: SmtpSocket, command: string): Promise<string> {
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

async function upgradeSocketToTls(socket: import("node:net").Socket, host: string) {
  return await new Promise<import("node:tls").TLSSocket>((resolve, reject) => {
    const tlsSocket = connectTls({
      socket,
      servername: host,
    });
    tlsSocket.once("secureConnect", () => resolve(tlsSocket));
    tlsSocket.once("error", reject);
  });
}

export function normalizeNtfyPriority(value: string | undefined): number | null {
  if (!value) {
    return 3;
  }

  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed < 1) {
    return 3;
  }

  return Math.trunc(parsed);
}

export async function sendDiscordAlert(config: DiscordDeliveryTarget, message: string) {
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

export async function sendHttpAlert(config: HttpDeliveryTarget, message: string) {
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

export async function sendNtfyAlert(config: NtfyDeliveryTarget, message: string) {
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

export async function sendSmtpAlert(config: SmtpDeliveryTarget, message: string) {
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

  let socket = await new Promise<SmtpSocket>((resolve, reject) => {
    rawSocket.once("error", reject);
    if (config.secure) {
      (rawSocket as import("node:tls").TLSSocket).once("secureConnect", () => resolve(rawSocket));
    } else {
      rawSocket.once("connect", () => resolve(rawSocket));
    }
  });

  try {
    await smtpCommand(socket, "");
    await smtpCommand(socket, "EHLO nook");

    if (!config.secure) {
      await smtpCommand(socket, "STARTTLS");
      socket = await upgradeSocketToTls(socket as import("node:net").Socket, config.host);
      await smtpCommand(socket, "EHLO nook");
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

export function hasDiscordDelivery(config: DiscordDeliveryTarget): boolean {
  return Boolean(config.enabled && config.webhookUrl);
}

export function hasHttpDelivery(config: HttpDeliveryTarget): boolean {
  return Boolean(config.enabled && config.url);
}

export function hasNtfyDelivery(config: NtfyDeliveryTarget): boolean {
  return Boolean(config.enabled && config.serverUrl && config.topic);
}
