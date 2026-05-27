import { Database } from "bun:sqlite";
import { randomUUID } from "node:crypto";
import type { RegisteredApp } from "./types";

export class EventQueue {
  private db: Database;
  private epoch: string;

  constructor(dataDir: string) {
    this.db = new Database(`${dataDir}/pool-events.db`);
    this.db.exec("PRAGMA journal_mode=WAL");
    this.db.exec("PRAGMA synchronous=NORMAL");
    this.db.exec("PRAGMA busy_timeout=5000");
    this.epoch = randomUUID();
    this.migrate();
  }

  private migrate() {
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS events (
        id TEXT PRIMARY KEY,
        channel TEXT NOT NULL,
        offset_num INTEGER NOT NULL,
        payload TEXT NOT NULL,
        sender TEXT NOT NULL,
        timestamp INTEGER NOT NULL,
        created_at TEXT NOT NULL DEFAULT (datetime('now'))
      )
    `);

    this.db.exec(`
      CREATE TABLE IF NOT EXISTS subscriptions (
        subscriber_id TEXT NOT NULL,
        channel TEXT NOT NULL,
        last_acked INTEGER NOT NULL DEFAULT 0,
        PRIMARY KEY (subscriber_id, channel)
      )
    `);

    this.db.exec(`
      CREATE TABLE IF NOT EXISTS channel_offsets (
        channel TEXT PRIMARY KEY,
        next_offset INTEGER NOT NULL DEFAULT 1
      )
    `);

    this.db.exec(`
      CREATE TABLE IF NOT EXISTS apps (
        app_id TEXT PRIMARY KEY,
        app TEXT NOT NULL,
        token TEXT NOT NULL UNIQUE,
        emit_channels TEXT NOT NULL DEFAULT '[]',
        listen_channels TEXT NOT NULL DEFAULT '[]',
        registered_at TEXT NOT NULL DEFAULT (datetime('now'))
      )
    `);

    this.db.exec("CREATE INDEX IF NOT EXISTS idx_events_channel_offset ON events (channel, offset_num)");
    this.db.exec("CREATE INDEX IF NOT EXISTS idx_apps_token ON apps (token)");
  }

  getEpoch(): string {
    return this.epoch;
  }

  pushEvent(channel: string, payload: unknown, sender: string): { id: string; offset: number; timestamp: number } {
    const id = randomUUID();
    const timestamp = Date.now();

    const offset = this.db.transaction(() => {
      const row = this.db.query<{ next_offset: number }, [string]>(
        "SELECT next_offset FROM channel_offsets WHERE channel = ?",
      ).get(channel);

      const currentOffset = row?.next_offset ?? 1;

      if (row) {
        this.db.run("UPDATE channel_offsets SET next_offset = ? WHERE channel = ?", [currentOffset + 1, channel]);
      } else {
        this.db.run("INSERT INTO channel_offsets (channel, next_offset) VALUES (?, ?)", [channel, currentOffset + 1]);
      }

      this.db.run(
        "INSERT INTO events (id, channel, offset_num, payload, sender, timestamp) VALUES (?, ?, ?, ?, ?, ?)",
        [id, channel, currentOffset, JSON.stringify(payload), sender, timestamp],
      );

      return currentOffset;
    })();

    return { id, offset, timestamp };
  }

  getEventsAfter(channel: string, afterOffset: number, limit = 100): Array<{ id: string; channel: string; offset: number; payload: unknown; sender: string; timestamp: number }> {
    const rows = this.db.query<
      { id: string; channel: string; offset_num: number; payload: string; sender: string; timestamp: number },
      [string, number, number]
    >(
      "SELECT id, channel, offset_num, payload, sender, timestamp FROM events WHERE channel = ? AND offset_num > ? ORDER BY offset_num ASC LIMIT ?",
    ).all(channel, afterOffset, limit);

    return rows.map((row) => ({
      id: row.id,
      channel: row.channel,
      offset: row.offset_num,
      payload: JSON.parse(row.payload),
      sender: row.sender,
      timestamp: row.timestamp,
    }));
  }

  ack(subscriberId: string, channel: string, offset: number) {
    this.db.run(
      "INSERT INTO subscriptions (subscriber_id, channel, last_acked) VALUES (?, ?, ?) ON CONFLICT (subscriber_id, channel) DO UPDATE SET last_acked = excluded.last_acked",
      [subscriberId, channel, offset],
    );
  }

  getLastAcked(subscriberId: string, channel: string): number {
    const row = this.db.query<{ last_acked: number }, [string, string]>(
      "SELECT last_acked FROM subscriptions WHERE subscriber_id = ? AND channel = ?",
    ).get(subscriberId, channel);

    return row?.last_acked ?? 0;
  }

  registerApp(app: string, secret: string, emitChannels: string[], listenChannels: string[]): RegisteredApp {
    const existing = this.db.query<{ app_id: string; token: string; registered_at: string }, [string]>(
      "SELECT app_id, token, registered_at FROM apps WHERE app = ?",
    ).get(app);

    if (existing) {
      this.db.run(
        "UPDATE apps SET emit_channels = ?, listen_channels = ? WHERE app_id = ?",
        [JSON.stringify(emitChannels), JSON.stringify(listenChannels), existing.app_id],
      );

      return {
        app_id: existing.app_id,
        app,
        token: existing.token,
        emit_channels: emitChannels,
        listen_channels: listenChannels,
        registered_at: existing.registered_at,
      };
    }

    const appId = randomUUID();
    const token = randomUUID();
    const registeredAt = new Date().toISOString();

    this.db.run(
      "INSERT INTO apps (app_id, app, token, emit_channels, listen_channels, registered_at) VALUES (?, ?, ?, ?, ?, ?)",
      [appId, app, token, JSON.stringify(emitChannels), JSON.stringify(listenChannels), registeredAt],
    );

    return { app_id: appId, app, token, emit_channels: emitChannels, listen_channels: listenChannels, registered_at: registeredAt };
  }

  getAppByToken(token: string): RegisteredApp | null {
    const row = this.db.query<
      { app_id: string; app: string; token: string; emit_channels: string; listen_channels: string; registered_at: string },
      [string]
    >(
      "SELECT app_id, app, token, emit_channels, listen_channels, registered_at FROM apps WHERE token = ?",
    ).get(token);

    if (!row) return null;

    return {
      app_id: row.app_id,
      app: row.app,
      token: row.token,
      emit_channels: JSON.parse(row.emit_channels),
      listen_channels: JSON.parse(row.listen_channels),
      registered_at: row.registered_at,
    };
  }

  cleanup(olderThanMs = 7 * 24 * 60 * 60 * 1000) {
    const cutoff = Date.now() - olderThanMs;
    this.db.run("DELETE FROM events WHERE timestamp < ?", [cutoff]);
  }
}
