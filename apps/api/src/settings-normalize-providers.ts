import { createDefaultProvider, createDefaultWebsiteProvider } from "./provider-config";
import type { AppSettings, ProvidersConfig } from "./types";
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
import {
  isPositiveNumber,
  normalizeBoolean,
  normalizeInteger,
  normalizeList,
  normalizeString,
} from "./settings-normalize-shared";

function normalizeWebsiteProvider(input: unknown, fallback?: WebsiteProviderSettings): WebsiteProviderSettings {
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
    tags: "tags" in candidate ? normalizeList(candidate.tags) : seed.tags,
    url,
    interval,
  };
}

function normalizeImapProvider(input: unknown, fallback?: ImapProviderSettings): ImapProviderSettings {
  const seed = fallback ?? createDefaultProvider("imap") as ImapProviderSettings;
  const candidate = input && typeof input === "object" ? input as Partial<ImapProviderSettings> : {};

  return {
    ...seed,
    id: normalizeString(candidate.id) || seed.id,
    name: normalizeString(candidate.name) || seed.name,
    enabled: normalizeBoolean(candidate.enabled, seed.enabled),
    tags: "tags" in candidate ? normalizeList(candidate.tags) : seed.tags,
    host: normalizeString(candidate.host) || seed.host,
    port: normalizeInteger(candidate.port, seed.port, `Provider \`${seed.name}\` port`),
    secure: typeof candidate.secure === "boolean" ? candidate.secure : seed.secure,
    username: normalizeString(candidate.username) || seed.username,
    password: "password" in candidate ? (normalizeString(candidate.password) || null) : seed.password,
    pollInterval: normalizeInteger(candidate.pollInterval, seed.pollInterval, `Provider \`${seed.name}\` poll interval`),
    includeEmailBody: normalizeBoolean(candidate.includeEmailBody, seed.includeEmailBody),
  };
}

function normalizeWebhookProvider(input: unknown, fallback?: WebhookProviderSettings): WebhookProviderSettings {
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
    tags: "tags" in candidate ? normalizeList(candidate.tags) : seed.tags,
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
    const providers = Object.entries(legacyWebhookProviders as Record<string, unknown>)
      .map(([type, provider]) => isProviderType(type) ? normalizeProvider({ ...(provider as object), type }) : null)
      .filter(Boolean) as ProviderSettings[];

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

export function normalizeProvidersConfig(input: unknown, fallback: AppSettings): ProvidersConfig {
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

  return { port, items };
}
