import { sendAlertToTarget } from "./delivery";
import { updateEventDeliveryRetry } from "./event-log";
import type { DeliveryTarget, DeliveryMessage } from "./types";

interface RetryEntry {
  eventId: string;
  target: DeliveryTarget;
  delivery: DeliveryMessage;
  targetId: string;
  attempt: number;
  nextRetryAt: number;
}

const DELAYS = [5_000, 30_000, 300_000];
const MAX_ATTEMPTS = 3;

let queue: RetryEntry[] = [];
let timer: Timer | null = null;

export function enqueueRetry(eventId: string, target: DeliveryTarget, delivery: DeliveryMessage) {
  queue.push({
    eventId,
    target,
    delivery,
    targetId: target.id,
    attempt: 1,
    nextRetryAt: Date.now() + DELAYS[0]!,
  });
}

async function processQueue() {
  const now = Date.now();
  const ready = queue.filter((entry) => entry.nextRetryAt <= now);

  for (const entry of ready) {
    queue = queue.filter((e) => e !== entry);

    try {
      await sendAlertToTarget(entry.target, entry.delivery);
      updateEventDeliveryRetry(entry.eventId, entry.targetId, "success");
      console.log(`[retry] ${entry.target.name} succeeded on attempt ${entry.attempt}`);
    } catch {
      if (entry.attempt < MAX_ATTEMPTS) {
        const nextDelay = DELAYS[entry.attempt] ?? DELAYS[DELAYS.length - 1]!;
        queue.push({
          ...entry,
          attempt: entry.attempt + 1,
          nextRetryAt: now + nextDelay,
        });
        console.log(`[retry] ${entry.target.name} failed attempt ${entry.attempt}, retrying in ${nextDelay / 1000}s`);
      } else {
        updateEventDeliveryRetry(entry.eventId, entry.targetId, "error");
        console.error(`[retry] ${entry.target.name} exhausted all ${MAX_ATTEMPTS} attempts`);
      }
    }
  }
}

export function startRetryProcessor() {
  if (timer) return;
  timer = setInterval(() => void processQueue(), 5_000);
}

export function getRetryQueueSize() {
  return queue.length;
}
