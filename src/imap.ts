import { connect as connectNet } from "node:net";
import { connect as connectTls, TLSSocket } from "node:tls";

import type { AppSettings } from "./types";
import { isImapProviderSettings, type ImapProviderSettings } from "./webhooks/types";

interface ImapEntryState {
  timerId: Timer;
  running: boolean;
  lastSeenUid: number | null;
  initialized: boolean;
}

interface ImapFetchResult {
  uid: number;
  subject: string | null;
  from: string | null;
  date: string | null;
}

const defaultImapMailbox = "INBOX";

class ImapSession {
  private socket: TLSSocket;
  private buffer = "";
  private lines: string[] = [];
  private waiters: Array<() => void> = [];
  private tagCounter = 0;

  private constructor(socket: TLSSocket) {
    this.socket = socket;
    this.socket.setEncoding("utf8");
    this.socket.on("data", (chunk: string) => {
      this.buffer += chunk;
      this.drainBuffer();
    });
  }

  static async connect(config: ImapProviderSettings) {
    const socket = await new Promise<TLSSocket>((resolve, reject) => {
      const onError = (error: Error) => reject(error);

      if (config.secure) {
        const tlsSocket = connectTls({
          host: config.host,
          port: config.port,
          servername: config.host,
        }, () => resolve(tlsSocket));
        tlsSocket.once("error", onError);
        return;
      }

      const netSocket = connectNet({
        host: config.host,
        port: config.port,
      }, () => resolve(netSocket as TLSSocket));
      netSocket.once("error", onError);
    });

    const session = new ImapSession(socket);
    await session.readGreeting();
    return session;
  }

  private drainBuffer() {
    while (true) {
      const separatorIndex = this.buffer.indexOf("\r\n");
      if (separatorIndex === -1) {
        return;
      }

      const line = this.buffer.slice(0, separatorIndex);
      this.buffer = this.buffer.slice(separatorIndex + 2);
      this.lines.push(line);
      const nextWaiter = this.waiters.shift();
      if (nextWaiter) {
        nextWaiter();
      }
    }
  }

  private async nextLine(): Promise<string> {
    while (this.lines.length === 0) {
      await new Promise<void>((resolve) => this.waiters.push(resolve));
    }

    return this.lines.shift() as string;
  }

  private async readGreeting() {
    const greeting = await this.nextLine();
    if (!greeting.startsWith("* OK")) {
      throw new Error(`IMAP greeting failed: ${greeting}`);
    }
  }

  private createTag() {
    this.tagCounter += 1;
    return `A${String(this.tagCounter).padStart(4, "0")}`;
  }

  private quote(value: string) {
    return `"${value.replaceAll("\\", "\\\\").replaceAll("\"", "\\\"")}"`;
  }

  private send(command: string) {
    this.socket.write(`${command}\r\n`);
  }

  async command(command: string): Promise<string[]> {
    const tag = this.createTag();
    this.send(`${tag} ${command}`);
    const lines: string[] = [];

    while (true) {
      const line = await this.nextLine();
      lines.push(line);
      if (line.startsWith(`${tag} `)) {
        if (!line.includes(" OK")) {
          throw new Error(`IMAP command failed: ${line}`);
        }

        return lines;
      }
    }
  }

  async login(username: string, password: string) {
    await this.command(`LOGIN ${this.quote(username)} ${this.quote(password)}`);
  }

  async selectMailbox(mailbox: string) {
    await this.command(`SELECT ${this.quote(mailbox)}`);
  }

  async searchUids() {
    const lines = await this.command("UID SEARCH ALL");
    const searchLine = lines.find((line) => line.startsWith("* SEARCH"));
    if (!searchLine) {
      return [];
    }

    return searchLine
      .replace("* SEARCH", "")
      .trim()
      .split(/\s+/)
      .map((part) => Number(part))
      .filter((part) => Number.isInteger(part) && part > 0);
  }

  async fetchHeaders(uid: number): Promise<ImapFetchResult> {
    const lines = await this.command(`UID FETCH ${uid} (UID BODY.PEEK[HEADER.FIELDS (FROM SUBJECT DATE)])`);
    const headerLines: string[] = [];
    let collectingHeaders = false;

    for (const line of lines) {
      if (line.startsWith("* ") && line.includes("{")) {
        collectingHeaders = true;
        continue;
      }

      if (!collectingHeaders) {
        continue;
      }

      if (line === ")" || line.startsWith("A")) {
        collectingHeaders = false;
        continue;
      }

      headerLines.push(line);
    }

    const headers = new Map<string, string>();
    let currentKey: string | null = null;

    for (const rawLine of headerLines) {
      if (/^\s/.test(rawLine) && currentKey) {
        headers.set(currentKey, `${headers.get(currentKey) ?? ""} ${rawLine.trim()}`.trim());
        continue;
      }

      const separatorIndex = rawLine.indexOf(":");
      if (separatorIndex === -1) {
        continue;
      }

      currentKey = rawLine.slice(0, separatorIndex).trim().toLowerCase();
      headers.set(currentKey, rawLine.slice(separatorIndex + 1).trim());
    }

    return {
      uid,
      subject: headers.get("subject") ?? null,
      from: headers.get("from") ?? null,
      date: headers.get("date") ?? null,
    };
  }

  async close() {
    try {
      await this.command("LOGOUT");
    } catch {
      return;
    } finally {
      this.socket.end();
      this.socket.destroy();
    }
  }
}

function escapeHtml(value: string) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}

function buildMessage(provider: ImapProviderSettings, message: ImapFetchResult) {
  const parts = [
    `📨 <b>[IMAP] ${escapeHtml(provider.name)}</b> new email`,
    `Mailbox: <code>${defaultImapMailbox}</code>`,
  ];

  if (message.from) {
    parts.push(`From: ${escapeHtml(message.from)}`);
  }

  if (message.subject) {
    parts.push(`Subject: <b>${escapeHtml(message.subject)}</b>`);
  }

  if (message.date) {
    parts.push(`Date: <code>${escapeHtml(message.date)}</code>`);
  }

  return parts.join("<br/>");
}

export interface ImapAlertContext {
  providerId: string;
  providerName: string;
  providerType: "imap";
  event: string;
}

export function createImapManager(sendAlert: (message: string, context: ImapAlertContext) => Promise<void>) {
  const entries = new Map<string, ImapEntryState>();

  async function pollProvider(provider: ImapProviderSettings) {
    const entry = entries.get(provider.id);
    if (!entry || entry.running || !provider.host || !provider.username || !provider.password) {
      return;
    }

    entry.running = true;

    let session: ImapSession | null = null;
    try {
      session = await ImapSession.connect(provider);
      await session.login(provider.username, provider.password);
      await session.selectMailbox(defaultImapMailbox);
      const uids = await session.searchUids();
      const latestUid = uids.length > 0 ? uids[uids.length - 1] ?? null : null;

      if (!entry.initialized) {
        entry.lastSeenUid = latestUid;
        entry.initialized = true;
        console.log(`[imap] initialized ${provider.name} at UID ${latestUid ?? "none"}`);
        return;
      }

      const newUids = uids.filter((uid) => uid > (entry.lastSeenUid ?? 0));
      for (const uid of newUids) {
        const message = await session.fetchHeaders(uid);
        await sendAlert(buildMessage(provider, message), {
          providerId: provider.id,
          providerName: provider.name,
          providerType: "imap",
          event: `new email: ${message.subject ?? "no subject"}`,
        });
        entry.lastSeenUid = uid;
      }

      if (latestUid !== null) {
        entry.lastSeenUid = latestUid;
      }
    } catch (error) {
      console.error(`[imap] ${provider.name}: ${error instanceof Error ? error.message : String(error)}`);
    } finally {
      entry.running = false;
      await session?.close();
    }
  }

  function stopAll() {
    for (const entry of entries.values()) {
      clearInterval(entry.timerId);
    }

    entries.clear();
  }

  function apply(settings: AppSettings) {
    stopAll();

    for (const provider of settings.providers.items.filter(isImapProviderSettings).filter((provider) => provider.enabled)) {
      const timerId = setInterval(() => void pollProvider(provider), provider.pollInterval * 1000);
      entries.set(provider.id, {
        timerId,
        running: false,
        lastSeenUid: null,
        initialized: false,
      });
      console.log(`[imap] polling ${provider.name} on ${provider.host}:${provider.port}/${defaultImapMailbox} every ${provider.pollInterval}s`);
      void pollProvider(provider);
    }
  }

  return {
    apply,
    stopAll,
  };
}

export async function testImapConnection(provider: ImapProviderSettings) {
  if (!provider.host || !provider.username || !provider.password) {
    throw new Error("IMAP host, username, and password are required.");
  }

  let session: ImapSession | null = null;
  try {
    session = await ImapSession.connect(provider);
    await session.login(provider.username, provider.password);
    await session.selectMailbox(defaultImapMailbox);
  } finally {
    await session?.close();
  }
}
