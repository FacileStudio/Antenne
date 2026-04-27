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
  source: "webhook" | "website" | "imap" | "test" | "system";
  providerId: string | null;
  providerName: string;
  providerType: string;
  event: string;
  message: string | null;
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

  return entry;
}

export function updateEventDeliveries(
  eventId: string,
  deliveries: DeliveryAttempt[],
): void {
  const entry = entries.find((e) => e.id === eventId);
  if (entry) {
    entry.deliveries = deliveries;
  }
}

export function getEvents(options?: {
  providerId?: string;
  targetId?: string;
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

  const reversed = [...result].reverse();
  return options?.limit ? reversed.slice(0, options.limit) : reversed;
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
