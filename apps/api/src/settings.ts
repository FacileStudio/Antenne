import { loadDeliveryConfigFromEnv } from "./delivery";
import { createDefaultWebhookProvider } from "./provider-config";
import { normalizeAppSettings } from "./settings-normalize";
import {
  createInitialRuntimeStatus,
  getServerPort,
  getSettingsPath,
  loadPersistedSettings,
  saveSettings,
} from "./settings-storage";
import { SYSTEM_PROVIDER_ID, type AppSettings } from "./types";
import type { ProvidersConfig } from "./types";
import type { WebhookProviderType } from "./webhooks/types";

function cloneSettings(settings: AppSettings): AppSettings {
  return structuredClone(settings);
}

export { createInitialRuntimeStatus, getServerPort, getSettingsPath, loadPersistedSettings };

function envBoolean(name: string): boolean | undefined {
  return process.env[name] === undefined ? undefined : process.env[name] === "true";
}

function envPositiveNumber(name: string, fallback: number) {
  const rawValue = process.env[name];
  if (!rawValue) {
    return fallback;
  }

  const value = Number(rawValue);
  return Number.isFinite(value) && value > 0 ? value : fallback;
}

function createEnvWebhookProvider(type: WebhookProviderType, envPrefix: string, fallbackEnabled = false) {
  const enabled = envBoolean(`${envPrefix}_WEBHOOK_ENABLED`) ?? fallbackEnabled;
  const path = process.env[`${envPrefix}_WEBHOOK_PATH`];
  const secretHeader = process.env[`${envPrefix}_WEBHOOK_SECRET_HEADER`];
  const secretValue = process.env[`${envPrefix}_WEBHOOK_SECRET`] ?? (type === "uptime-kuma" ? process.env.WEBHOOK_SECRET : undefined);

  if (!enabled && !path && !secretHeader && !secretValue) {
    return null;
  }

  return createDefaultWebhookProvider(type, {
    enabled,
    path,
    secretHeader,
    secretValue,
  });
}

function loadProvidersConfigFromEnv(): ProvidersConfig {
  const webhookEnabled = envBoolean("WEBHOOK_ENABLED") ?? false;
  const providers = [
    createEnvWebhookProvider("uptime-kuma", "UPTIME_KUMA", webhookEnabled),
    createEnvWebhookProvider("docuseal", "DOCUSEAL", webhookEnabled),
    createEnvWebhookProvider("dokploy", "DOKPLOY", webhookEnabled),
  ].filter((provider) => provider !== null);

  return {
    port: getServerPort(),
    items: providers,
  };
}

export async function loadDefaultSettings(): Promise<AppSettings> {
  const providers = loadProvidersConfigFromEnv();
  const delivery = loadDeliveryConfigFromEnv();
  const defaultProviderIds = [SYSTEM_PROVIDER_ID, ...providers.items.map((provider) => provider.id)];
  delivery.items = delivery.items.map((target) => ({
    ...target,
    providerIds: defaultProviderIds,
  }));

  return {
    delivery,
    system: {
      events: {
        startup: true,
        providerEnabled: false,
        providerDisabled: false,
        providerAdded: false,
        providerRemoved: false,
        deliveryFailure: false,
      },
    },
    timeout: envPositiveNumber("TIMEOUT", 10),
    providers,
  };
}

export class SettingsStore {
  private settings: AppSettings;
  private defaults: AppSettings;
  private readonly path: string;
  private usingPersistedSettings: boolean;

  constructor(defaults: AppSettings, initialSettings: AppSettings, path: string, usingPersistedSettings: boolean) {
    this.defaults = cloneSettings(defaults);
    this.settings = normalizeAppSettings(initialSettings, this.defaults);
    this.path = path;
    this.usingPersistedSettings = usingPersistedSettings;
  }

  getPath() {
    return this.path;
  }

  isUsingPersistedSettings() {
    return this.usingPersistedSettings;
  }

  getDefaults() {
    return cloneSettings(this.defaults);
  }

  getSettings() {
    return cloneSettings(this.settings);
  }

  async update(input: unknown) {
    const nextSettings = normalizeAppSettings(input, this.defaults);
    await saveSettings(this.path, nextSettings);
    this.settings = cloneSettings(nextSettings);
    this.usingPersistedSettings = true;
    return this.getSettings();
  }
}
