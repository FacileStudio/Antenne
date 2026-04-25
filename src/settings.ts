import { mkdir } from "node:fs/promises";

import { loadConfig } from "./config";
import { loadMatrixConfigFromEnv } from "./matrix";
import type { AppSettings, RuntimeStatus } from "./types";
import type { WebhookConfig, WebhookProviderId, WebhookProviderSettings } from "./webhooks/types";

const defaultSettingsPath = process.env.NOOK_SETTINGS_PATH ?? "/data/nook.json";
const defaultServerPort = Number(process.env.PORT ?? process.env.WEBHOOK_PORT ?? 9090);
const webhookProviderIds = ["uptime-kuma", "docuseal"] as const satisfies WebhookProviderId[];

function cloneSettings(settings: AppSettings): AppSettings {
  return structuredClone(settings);
}

function isPositiveNumber(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value) && value > 0;
}

function normalizeRoomIds(value: unknown): string[] {
  if (Array.isArray(value)) {
    return value.map((item) => String(item).trim()).filter(Boolean);
  }

  if (typeof value === "string") {
    return value.split(",").map((item) => item.trim()).filter(Boolean);
  }

  return [];
}

function normalizeString(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function normalizeWebhookProvider(
  input: unknown,
  fallback: WebhookProviderSettings,
): WebhookProviderSettings {
  const candidate = input && typeof input === "object" ? input as Partial<WebhookProviderSettings> : {};

  return {
    enabled: typeof candidate.enabled === "boolean" ? candidate.enabled : fallback.enabled,
    path: normalizeString(candidate.path) || fallback.path,
    secretHeader: normalizeString(candidate.secretHeader) || fallback.secretHeader,
    secretValue: normalizeString(candidate.secretValue) || null,
  };
}

function normalizeWebhookConfig(input: unknown, fallback: WebhookConfig): WebhookConfig {
  const candidate = input && typeof input === "object" ? input as Partial<WebhookConfig> : {};
  const providersInput = candidate.providers && typeof candidate.providers === "object"
    ? candidate.providers as Partial<Record<WebhookProviderId, unknown>>
    : {};

  const providers = Object.fromEntries(
    webhookProviderIds.map((providerId) => [
      providerId,
      normalizeWebhookProvider(providersInput[providerId], fallback.providers[providerId]),
    ]),
  ) as Record<WebhookProviderId, WebhookProviderSettings>;

  const webhook: WebhookConfig = {
    enabled: Object.values(providers).some((provider) => provider.enabled),
    port: isPositiveNumber(candidate.port) ? Math.trunc(candidate.port) : fallback.port,
    providers,
  };

  if (!Number.isInteger(webhook.port)) {
    throw new Error("Webhook port must be an integer.");
  }

  const enabledPaths = new Set<string>();
  for (const provider of Object.values(webhook.providers)) {
    if (!provider.enabled) {
      continue;
    }

    if (!provider.path.startsWith("/")) {
      throw new Error("Webhook paths must start with `/`.");
    }

    if (enabledPaths.has(provider.path)) {
      throw new Error(`Webhook path collision detected for \`${provider.path}\`.`);
    }

    enabledPaths.add(provider.path);
  }

  return webhook;
}

export function normalizeAppSettings(input: unknown, fallback: AppSettings): AppSettings {
  const candidate = input && typeof input === "object" ? input as Partial<AppSettings> : {};
  const matrix = candidate.matrix && typeof candidate.matrix === "object"
    ? candidate.matrix as Partial<AppSettings["matrix"]>
    : {};
  const rawSites = Array.isArray(candidate.sites) ? candidate.sites : fallback.sites;
  const sites = rawSites.map((site, index) => {
    const parsed = site && typeof site === "object" ? site as Partial<AppSettings["sites"][number]> : {};
    const url = normalizeString(parsed.url);
    const interval = isPositiveNumber(parsed.interval) ? parsed.interval : fallback.sites[index]?.interval ?? fallback.timeout;
    const name = normalizeString(parsed.name) || url;

    if (!url) {
      throw new Error(`Site #${index + 1} must define a URL.`);
    }

    if (!isPositiveNumber(interval)) {
      throw new Error(`Site \`${url}\` must define a positive interval.`);
    }

    return {
      name,
      url,
      interval,
    };
  });

  const timeout = isPositiveNumber(candidate.timeout) ? candidate.timeout : fallback.timeout;
  if (!isPositiveNumber(timeout)) {
    throw new Error("Timeout must be a positive number.");
  }

  return {
    matrix: {
      homeserver: "homeserver" in matrix ? normalizeString(matrix.homeserver) : fallback.matrix.homeserver,
      accessToken: "accessToken" in matrix ? normalizeString(matrix.accessToken) : fallback.matrix.accessToken,
      roomIds: "roomIds" in matrix ? normalizeRoomIds(matrix.roomIds) : fallback.matrix.roomIds,
    },
    timeout,
    sites,
    webhook: normalizeWebhookConfig(candidate.webhook, fallback.webhook),
  };
}

export async function loadDefaultSettings(): Promise<AppSettings> {
  const { config } = await loadConfig();
  return {
    matrix: loadMatrixConfigFromEnv(),
    timeout: config.timeout,
    sites: config.sites,
    webhook: config.webhook,
  };
}

export async function loadPersistedSettings(path = defaultSettingsPath): Promise<AppSettings | null> {
  const file = Bun.file(path);
  if (!(await file.exists())) {
    return null;
  }

  const text = await file.text();
  if (!text.trim()) {
    return null;
  }

  return JSON.parse(text) as AppSettings;
}

export async function saveSettings(path: string, settings: AppSettings) {
  const directory = path.includes("/") ? path.slice(0, path.lastIndexOf("/")) : ".";
  await mkdir(directory, { recursive: true });
  await Bun.write(path, `${JSON.stringify(settings, null, 2)}\n`);
}

export function createInitialRuntimeStatus(settingsPath: string, usingPersistedSettings: boolean): RuntimeStatus {
  return {
    serverPort: defaultServerPort,
    settingsPath,
    usingPersistedSettings,
    dashboardBuilt: false,
    lastReloadedAt: new Date().toISOString(),
  };
}

export function getSettingsPath() {
  return defaultSettingsPath;
}

export function getServerPort() {
  return defaultServerPort;
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
