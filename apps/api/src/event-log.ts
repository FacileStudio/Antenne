import { mkdir } from "node:fs/promises";
import type { NookEventEnvelope } from "./event-envelope";

export interface DeliveryAttempt {
  targetId: string;
  targetName: string;
  targetType: string;
  status: "success" | "error";
  error?: string;
  durationMs: number;
}

export interface EventLogEntry {
  id: string;
  timestamp: string;
  source: "webhook" | "website" | "imap" | "rss" | "test" | "system";
  providerId: string | null;
  providerName: string;
  providerType: string;
  event: string;
  message: string | null;
  envelope?: NookEventEnvelope;
  deliveries: DeliveryAttempt[];
}

export interface EventLogStats {
  totalEvents: number;
  lastEventAt: string | null;
  deliverySuccessCount: number;
  deliveryErrorCount: number;
}

const MAX_ENTRIES = 500;

let entries: EventLogEntry[] = [];
let persistPath: string | null = null;
let flushTimer: Timer | null = null;

type EventLogListener = (entry: EventLogEntry) => void;
const listeners: Set<EventLogListener> = new Set();

export function onEventPushed(listener: EventLogListener): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function scheduleFlush() {
  if (!persistPath) return;
  if (flushTimer) return;
  flushTimer = setTimeout(() => {
    flushTimer = null;
    void flushToDisk();
  }, 500);
}

async function flushToDisk() {
  if (!persistPath) return;
  try {
    const directory = persistPath.includes("/") ? persistPath.slice(0, persistPath.lastIndexOf("/")) : ".";
    await mkdir(directory, { recursive: true });
    await Bun.write(persistPath, JSON.stringify(entries));
  } catch (error) {
    console.error(`[event-log] failed to persist: ${error instanceof Error ? error.message : String(error)}`);
  }
}

export async function initEventLog(path: string) {
  persistPath = path;
  try {
    const file = Bun.file(path);
    if (await file.exists()) {
      const raw = await file.json();
      if (Array.isArray(raw)) {
        entries = raw.slice(-MAX_ENTRIES) as EventLogEntry[];
        console.log(`[event-log] loaded ${entries.length} entries from ${path}`);
        return;
      }
    }
  } catch (error) {
    console.error(`[event-log] failed to load: ${error instanceof Error ? error.message : String(error)}`);
  }
  entries = [];
}

function createId(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

export function pushEvent(
  partial: Omit<EventLogEntry, "id" | "timestamp">,
): EventLogEntry {
  const entry: EventLogEntry = {
    id: createId(),
    timestamp: new Date().toISOString(),
    ...partial,
  };

  entries.push(entry);

  if (entries.length > MAX_ENTRIES) {
    entries = entries.slice(entries.length - MAX_ENTRIES);
  }

  scheduleFlush();

  for (const listener of listeners) {
    try { listener(entry); } catch {}
  }

  return entry;
}

export function updateEventDeliveries(
  eventId: string,
  deliveries: DeliveryAttempt[],
): void {
  const entry = entries.find((e) => e.id === eventId);
  if (entry) {
    entry.deliveries = deliveries;
    scheduleFlush();
  }
}

export function updateEventDeliveryRetry(
  eventId: string,
  targetId: string,
  status: "success" | "error",
): void {
  const entry = entries.find((e) => e.id === eventId);
  if (!entry) return;

  const delivery = entry.deliveries.find((d) => d.targetId === targetId);
  if (delivery) {
    delivery.status = status;
    if (status === "success") {
      delete delivery.error;
    }
  }

  scheduleFlush();
}

export function getEvents(options?: {
  providerId?: string;
  targetId?: string;
  source?: string;
  q?: string;
  limit?: number;
}): EventLogEntry[] {
  let result = entries;

  if (options?.providerId) {
    result = result.filter((e) => e.providerId === options.providerId);
  }

  if (options?.targetId) {
    result = result.filter((e) =>
      e.deliveries.some((d) => d.targetId === options.targetId),
    );
  }

  if (options?.source) {
    result = result.filter((e) => e.source === options.source);
  }

  if (options?.q) {
    const query = options.q.toLowerCase();
    result = result.filter((e) =>
      e.event?.toLowerCase().includes(query) ||
      e.providerName?.toLowerCase().includes(query) ||
      e.message?.toLowerCase().includes(query)
    );
  }

  const reversed = [...result].reverse();
  return options?.limit ? reversed.slice(0, options.limit) : reversed;
}

export function getEventById(id: string): EventLogEntry | null {
  return entries.find((e) => e.id === id) ?? null;
}

export function getProviderStats(providerId: string): EventLogStats {
  const providerEvents = entries.filter((e) => e.providerId === providerId);
  return computeStats(providerEvents);
}

export function getTargetStats(targetId: string): EventLogStats {
  const targetEvents = entries.filter((e) =>
    e.deliveries.some((d) => d.targetId === targetId),
  );
  return computeStats(targetEvents);
}

export function getGlobalStats(): EventLogStats {
  return computeStats(entries);
}

function computeStats(events: EventLogEntry[]): EventLogStats {
  let deliverySuccessCount = 0;
  let deliveryErrorCount = 0;

  for (const event of events) {
    for (const d of event.deliveries) {
      if (d.status === "success") deliverySuccessCount++;
      else deliveryErrorCount++;
    }
  }

  return {
    totalEvents: events.length,
    lastEventAt: events.length > 0 ? events[events.length - 1]!.timestamp : null,
    deliverySuccessCount,
    deliveryErrorCount,
  };
}
