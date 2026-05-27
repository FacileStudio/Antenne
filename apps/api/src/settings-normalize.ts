import type { AppSettings } from "./types";
import { normalizeDeliveryConfig } from "./settings-normalize-delivery";
import { normalizeProvidersConfig } from "./settings-normalize-providers";
import { isPositiveNumber, normalizeBoolean } from "./settings-normalize-shared";

export function normalizeAppSettings(input: unknown, fallback: AppSettings): AppSettings {
  const candidate = input && typeof input === "object" ? input as Partial<AppSettings> : {};
  const timeout = isPositiveNumber(candidate.timeout) ? candidate.timeout : fallback.timeout;
  if (!isPositiveNumber(timeout)) {
    throw new Error("Timeout must be a positive number.");
  }

  const pool = candidate.pool && typeof candidate.pool === "object"
    ? { secret: typeof candidate.pool.secret === "string" ? candidate.pool.secret : undefined }
    : fallback.pool;

  return {
    delivery: normalizeDeliveryConfig(input, fallback.delivery),
    system: {
      events: {
        startup: normalizeBoolean(candidate.system?.events?.startup, fallback.system.events.startup),
        providerEnabled: normalizeBoolean(candidate.system?.events?.providerEnabled, fallback.system.events.providerEnabled),
        providerDisabled: normalizeBoolean(candidate.system?.events?.providerDisabled, fallback.system.events.providerDisabled),
        providerAdded: normalizeBoolean(candidate.system?.events?.providerAdded, fallback.system.events.providerAdded),
        providerRemoved: normalizeBoolean(candidate.system?.events?.providerRemoved, fallback.system.events.providerRemoved),
        deliveryFailure: normalizeBoolean(candidate.system?.events?.deliveryFailure, fallback.system.events.deliveryFailure),
      },
    },
    timeout,
    providers: normalizeProvidersConfig(input, fallback),
    pool,
  };
}
