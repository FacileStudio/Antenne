import type { AppSettings } from "./types";
import { normalizeDeliveryConfig } from "./settings-normalize-delivery";
import { normalizeProvidersConfig } from "./settings-normalize-providers";
import { isPositiveNumber } from "./settings-normalize-shared";

export function normalizeAppSettings(input: unknown, fallback: AppSettings): AppSettings {
  const candidate = input && typeof input === "object" ? input as Partial<AppSettings> : {};
  const timeout = isPositiveNumber(candidate.timeout) ? candidate.timeout : fallback.timeout;
  if (!isPositiveNumber(timeout)) {
    throw new Error("Timeout must be a positive number.");
  }

  return {
    delivery: normalizeDeliveryConfig(input, fallback.delivery),
    timeout,
    providers: normalizeProvidersConfig(input, fallback),
  };
}
