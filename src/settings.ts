import { mkdir } from "node:fs/promises";

import { loadConfig } from "./config";
import { createDefaultProvider, createDefaultWebhookProvider, createDefaultWebsiteProvider } from "./provider-config";
import {
  createDefaultDeliveryTarget,
  isDiscordDeliveryTarget,
  isHttpDeliveryTarget,
  isMatrixDeliveryTarget,
  isSmtpDeliveryTarget,
} from "./delivery-config";
import { loadDeliveryConfigFromEnv } from "./delivery";
import type {
  AppSettings,
  DeliveryTarget,
  DiscordDeliveryTarget,
  HttpDeliveryTarget,
  MatrixDeliveryTarget,
  ProvidersConfig,
  RuntimeStatus,
  SmtpDeliveryTarget,
} from "./types";
import type {
  ImapProviderSettings,
  ProviderSettings,
  ProviderType,
  WebhookProviderSettings,
  WebsiteProviderSettings,
} from "./webhooks/types";
import {
  isImapProviderSettings,
  isProviderType,
  isWebhookProviderSettings,
  isWebsiteProviderSettings,
} from "./webhooks/types";

const defaultSettingsPath = process.env.NOOK_SETTINGS_PATH ?? "/data/nook.json";
const defaultServerPort = Number(process.env.PORT ?? process.env.WEBHOOK_PORT ?? 9090);

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

function normalizeSmtpAddresses(value: unknown): string[] {
  if (Array.isArray(value)) {
    return value.map((item) => String(item).trim()).filter(Boolean);
  }

  if (typeof value === "string") {
    return value.split(",").map((item) => item.trim()).filter(Boolean);
  }

  return [];
}

function normalizeDeliveryConfig(input: unknown, fallback: AppSettings["delivery"]): AppSettings["delivery"] {
  const candidate = input && typeof input === "object" ? input as {
    delivery?: unknown;
    matrix?: unknown;
    discord?: unknown;
    webhook?: unknown;
  } : {};
  const itemsContainer = candidate.delivery && typeof candidate.delivery === "object"
    ? candidate.delivery as { items?: unknown; matrix?: unknown; discord?: unknown; webhook?: unknown }
    : {};

  const rawItems = Array.isArray(itemsContainer.items)
    ? itemsContainer.items
    : null;

  if (rawItems) {
    return {
      items: rawItems.map((target) => normalizeDeliveryTarget(target)),
    };
  }

  const matrixInput = itemsContainer.matrix ?? candidate.matrix;
  const discordInput = itemsContainer.discord ?? candidate.discord;
  const webhookInput = itemsContainer.webhook ?? candidate.webhook;

  const legacyTargets = fallback.items.map((target) => {
    if (isMatrixDeliveryTarget(target)) {
      const matrix = matrixInput && typeof matrixInput === "object" ? matrixInput as Partial<typeof target> : {};
      return {
        ...target,
        enabled: typeof matrix.enabled === "boolean" ? matrix.enabled : target.enabled,
        homeserver: "homeserver" in matrix ? normalizeString(matrix.homeserver) : target.homeserver,
        accessToken: "accessToken" in matrix ? normalizeString(matrix.accessToken) : target.accessToken,
        roomIds: "roomIds" in matrix ? normalizeRoomIds(matrix.roomIds) : target.roomIds,
      };
    }

    if (isDiscordDeliveryTarget(target)) {
      const discord = discordInput && typeof discordInput === "object" ? discordInput as Partial<typeof target> : {};
      return {
        ...target,
        enabled: typeof discord.enabled === "boolean" ? discord.enabled : target.enabled,
        webhookUrl: "webhookUrl" in discord ? normalizeString(discord.webhookUrl) : target.webhookUrl,
      };
    }

    if (isSmtpDeliveryTarget(target)) {
      return target;
    }

    const webhook = webhookInput && typeof webhookInput === "object" ? webhookInput as Partial<typeof target> : {};
    return {
      ...target,
      enabled: typeof webhook.enabled === "boolean" ? webhook.enabled : target.enabled,
      url: "url" in webhook ? normalizeString(webhook.url) : target.url,
      secretHeader: "secretHeader" in webhook ? (normalizeString(webhook.secretHeader) || null) : target.secretHeader,
      secretValue: "secretValue" in webhook ? (normalizeString(webhook.secretValue) || null) : target.secretValue,
    };
  });

  return {
    items: legacyTargets,
  };
}

function normalizeString(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function normalizeBoolean(value: unknown, fallback: boolean): boolean {
  return typeof value === "boolean" ? value : fallback;
}

function normalizeProviderIds(value: unknown): string[] {
  if (Array.isArray(value)) {
    return value.map((item) => String(item).trim()).filter(Boolean);
  }

  return [];
}

function normalizeDeliveryTarget(input: unknown): DeliveryTarget {
  const candidate = input && typeof input === "object" ? input as Partial<DeliveryTarget> : {};
  const type = candidate.type;
  const providerIds = normalizeProviderIds(candidate.providerIds);

  if (type === "matrix") {
    const fallback = createDefaultDeliveryTarget("matrix") as MatrixDeliveryTarget;
    return {
      ...fallback,
      id: normalizeString(candidate.id) || fallback.id,
      name: normalizeString(candidate.name) || fallback.name,
      enabled: normalizeBoolean(candidate.enabled, fallback.enabled),
      providerIds,
      homeserver: "homeserver" in candidate ? normalizeString(candidate.homeserver) : fallback.homeserver,
      accessToken: "accessToken" in candidate ? normalizeString(candidate.accessToken) : fallback.accessToken,
      roomIds: "roomIds" in candidate ? normalizeRoomIds(candidate.roomIds) : fallback.roomIds,
    };
  }

  if (type === "discord") {
    const fallback = createDefaultDeliveryTarget("discord") as DiscordDeliveryTarget;
    return {
      ...fallback,
      id: normalizeString(candidate.id) || fallback.id,
      name: normalizeString(candidate.name) || fallback.name,
      enabled: normalizeBoolean(candidate.enabled, fallback.enabled),
      providerIds,
      webhookUrl: "webhookUrl" in candidate ? normalizeString(candidate.webhookUrl) : fallback.webhookUrl,
    };
  }

  if (type === "smtp") {
    const fallback = createDefaultDeliveryTarget("smtp") as SmtpDeliveryTarget;
    return {
      ...fallback,
      id: normalizeString(candidate.id) || fallback.id,
      name: normalizeString(candidate.name) || fallback.name,
      enabled: normalizeBoolean(candidate.enabled, fallback.enabled),
      providerIds,
      host: "host" in candidate ? normalizeString(candidate.host) : fallback.host,
      port: "port" in candidate ? normalizeInteger(candidate.port, fallback.port, "SMTP port") : fallback.port,
      secure: "secure" in candidate ? normalizeBoolean(candidate.secure, fallback.secure) : fallback.secure,
      username: "username" in candidate ? normalizeString(candidate.username) : fallback.username,
      password: "password" in candidate ? (normalizeString(candidate.password) || null) : fallback.password,
      fromAddress: "fromAddress" in candidate ? normalizeString(candidate.fromAddress) : fallback.fromAddress,
      toAddresses: "toAddresses" in candidate ? normalizeSmtpAddresses(candidate.toAddresses) : fallback.toAddresses,
      subjectPrefix: "subjectPrefix" in candidate ? normalizeString(candidate.subjectPrefix) : fallback.subjectPrefix,
    };
  }

  const fallback = createDefaultDeliveryTarget("webhook") as HttpDeliveryTarget;
  return {
    ...fallback,
    id: normalizeString(candidate.id) || fallback.id,
    name: normalizeString(candidate.name) || fallback.name,
    enabled: normalizeBoolean(candidate.enabled, fallback.enabled),
    providerIds,
    url: "url" in candidate ? normalizeString(candidate.url) : fallback.url,
    secretHeader: "secretHeader" in candidate ? (normalizeString(candidate.secretHeader) || null) : fallback.secretHeader,
    secretValue: "secretValue" in candidate ? (normalizeString(candidate.secretValue) || null) : fallback.secretValue,
  };
}

function normalizeInteger(value: unknown, fallback: number, fieldName: string): number {
  const nextValue = isPositiveNumber(value) ? Math.trunc(value) : fallback;
  if (!Number.isInteger(nextValue)) {
    throw new Error(`${fieldName} must be an integer.`);
  }

  return nextValue;
}

function normalizeWebsiteProvider(
  input: unknown,
  fallback?: WebsiteProviderSettings,
): WebsiteProviderSettings {
  const seed = fallback ?? createDefaultWebsiteProvider();
  const candidate = input && typeof input === "object" ? input as Partial<WebsiteProviderSettings> : {};
  const url = normalizeString(candidate.url) || seed.url;
  const interval = normalizeInteger(candidate.interval, seed.interval, `Provider \`${seed.name}\` interval`);

  if (!url) {
    throw new Error("Website providers must define a URL.");
  }

  return {
    ...seed,
    id: normalizeString(candidate.id) || seed.id,
    name: normalizeString(candidate.name) || seed.name,
    enabled: normalizeBoolean(candidate.enabled, seed.enabled),
    url,
    interval,
  };
}

function normalizeImapProvider(
  input: unknown,
  fallback?: ImapProviderSettings,
): ImapProviderSettings {
  const seed = fallback ?? createDefaultProvider("imap") as ImapProviderSettings;
  const candidate = input && typeof input === "object" ? input as Partial<ImapProviderSettings> : {};

  return {
    ...seed,
    id: normalizeString(candidate.id) || seed.id,
    name: normalizeString(candidate.name) || seed.name,
    enabled: normalizeBoolean(candidate.enabled, seed.enabled),
    host: normalizeString(candidate.host) || seed.host,
    port: normalizeInteger(candidate.port, seed.port, `Provider \`${seed.name}\` port`),
    secure: typeof candidate.secure === "boolean" ? candidate.secure : seed.secure,
    username: normalizeString(candidate.username) || seed.username,
    password: "password" in candidate ? (normalizeString(candidate.password) || null) : seed.password,
    pollInterval: normalizeInteger(candidate.pollInterval, seed.pollInterval, `Provider \`${seed.name}\` poll interval`),
  };
}

function normalizeWebhookProvider(
  input: unknown,
  fallback?: WebhookProviderSettings,
): WebhookProviderSettings {
  const type = input && typeof input === "object" && isProviderType((input as { type?: unknown }).type)
    ? (input as { type: ProviderType }).type
    : fallback?.type ?? "uptime-kuma";
  const seed = fallback ?? createDefaultProvider(type) as WebhookProviderSettings;
  const candidate = input && typeof input === "object" ? input as Partial<WebhookProviderSettings> : {};
  const path = normalizeString(candidate.path) || seed.path;

  if (!path.startsWith("/")) {
    throw new Error(`Webhook path for provider \`${normalizeString(candidate.name) || seed.name}\` must start with \`/\`.`);
  }

  return {
    ...seed,
    id: normalizeString(candidate.id) || seed.id,
    name: normalizeString(candidate.name) || seed.name,
    enabled: normalizeBoolean(candidate.enabled, seed.enabled),
    path,
    secretHeader: normalizeString(candidate.secretHeader) || seed.secretHeader,
    secretValue: "secretValue" in candidate ? (normalizeString(candidate.secretValue) || null) : seed.secretValue,
  };
}

function normalizeProvider(input: unknown, fallback?: ProviderSettings): ProviderSettings {
  const type = input && typeof input === "object" && isProviderType((input as { type?: unknown }).type)
    ? (input as { type: ProviderType }).type
    : fallback?.type ?? "website";

  if (type === "website") {
    return normalizeWebsiteProvider(input, fallback && isWebsiteProviderSettings(fallback) ? fallback : undefined);
  }

  if (type === "imap") {
    return normalizeImapProvider(input, fallback && isImapProviderSettings(fallback) ? fallback : undefined);
  }

  return normalizeWebhookProvider(input, fallback && isWebhookProviderSettings(fallback) ? fallback : undefined);
}

function extractLegacyProviders(input: unknown, fallback: AppSettings): ProviderSettings[] {
  const candidate = input && typeof input === "object" ? input as {
    providers?: unknown;
    webhook?: { providers?: unknown };
    sites?: unknown;
  } : {};

  if (candidate.providers && typeof candidate.providers === "object") {
    const providersContainer = candidate.providers as { items?: unknown };
    if (Array.isArray(providersContainer.items)) {
      return providersContainer.items.map((provider) => normalizeProvider(provider));
    }
  }

  const legacyWebhookProviders = candidate.webhook && typeof candidate.webhook === "object"
    ? (candidate.webhook as { providers?: unknown }).providers
    : undefined;

  if (Array.isArray(legacyWebhookProviders)) {
    return legacyWebhookProviders.map((provider) => normalizeProvider(provider));
  }

  if (legacyWebhookProviders && typeof legacyWebhookProviders === "object") {
    const providers = Object.entries(legacyWebhookProviders as Record<string, unknown>).map(([type, provider]) => {
      if (!isProviderType(type)) {
        return null;
      }

      return normalizeProvider({ ...(provider as object), type });
    }).filter(Boolean) as ProviderSettings[];

    const legacySites = Array.isArray(candidate.sites) ? candidate.sites : [];
    const websiteProviders = legacySites.map((site, index) => {
      const parsed = site && typeof site === "object" ? site as { name?: unknown; url?: unknown; interval?: unknown } : {};
      return normalizeProvider({
        type: "website",
        name: normalizeString(parsed.name) || `Website ${index + 1}`,
        url: normalizeString(parsed.url),
        interval: isPositiveNumber(parsed.interval) ? parsed.interval : fallback.timeout,
      });
    });

    return [...websiteProviders, ...providers];
  }

  if (Array.isArray(candidate.sites)) {
    return candidate.sites.map((site, index) => {
      const parsed = site && typeof site === "object" ? site as { name?: unknown; url?: unknown; interval?: unknown } : {};
      return normalizeProvider({
        type: "website",
        name: normalizeString(parsed.name) || `Website ${index + 1}`,
        url: normalizeString(parsed.url),
        interval: isPositiveNumber(parsed.interval) ? parsed.interval : fallback.timeout,
      });
    });
  }

  return fallback.providers.items.map((provider) => normalizeProvider(provider));
}

function normalizeProvidersConfig(input: unknown, fallback: AppSettings): ProvidersConfig {
  const candidate = input && typeof input === "object" ? input as {
    providers?: unknown;
    webhook?: { port?: unknown };
  } : {};

  const providersContainer = candidate.providers && typeof candidate.providers === "object"
    ? candidate.providers as { port?: unknown }
    : {};

  const items = extractLegacyProviders(input, fallback);
  const port = normalizeInteger(
    providersContainer.port ?? candidate.webhook?.port,
    fallback.providers.port,
    "Provider port",
  );

  const enabledPaths = new Set<string>();
  for (const provider of items) {
    if (!isWebhookProviderSettings(provider) || !provider.enabled) {
      continue;
    }

    if (enabledPaths.has(provider.path)) {
      throw new Error(`Provider path collision detected for \`${provider.path}\`.`);
    }

    enabledPaths.add(provider.path);
  }

  return {
    port,
    items,
  };
}

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

export async function loadDefaultSettings(): Promise<AppSettings> {
  const { config } = await loadConfig();
  return {
    delivery: loadDeliveryConfigFromEnv(),
    timeout: config.timeout,
    providers: config.providers,
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
