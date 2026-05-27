import type {
  ImapProviderSettings,
  ProviderSettings,
  ProviderType,
  RssProviderSettings,
  WebsiteProviderSettings,
  WebhookProviderSettings,
  WebhookProviderType,
} from "./webhooks/types";

function createProviderId() {
  return globalThis.crypto.randomUUID();
}

function toTitleCase(value: string) {
  return value
    .split("-")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

export function getProviderLabel(type: ProviderType) {
  if (type === "website") {
    return "Website";
  }

  if (type === "imap") {
    return "IMAP";
  }

  if (type === "rss") {
    return "RSS";
  }

  if (type === "docuseal") {
    return "DocuSeal";
  }

  if (type === "dokploy") {
    return "Dokploy";
  }

  if (type === "git") {
    return "Git";
  }

  if (type === "vikunja") {
    return "Vikunja";
  }

  if (type === "ardoise") {
    return "Ardoise";
  }

  if (type === "perception") {
    return "Perception";
  }

  if (type === "opus") {
    return "Opus";
  }

  if (type === "nuage") {
    return "Nuage";
  }

  return toTitleCase(type);
}

export function getWebhookDefaultPath(type: WebhookProviderType) {
  if (type === "uptime-kuma") {
    return "/webhook/uptime-kuma";
  }

  if (type === "docuseal") {
    return "/webhook/docuseal";
  }

  if (type === "git") {
    return "/webhook/git";
  }

  if (type === "vikunja") {
    return "/webhook/vikunja";
  }

  if (type === "ardoise") {
    return "/webhook/ardoise";
  }

  if (type === "sablier") {
    return "/webhook/sablier";
  }

  if (type === "perception") {
    return "/webhook/perception";
  }

  if (type === "opus") {
    return "/webhook/opus";
  }

  if (type === "nuage") {
    return "/webhook/nuage";
  }

  return "/webhook/dokploy";
}

export function getWebhookDefaultSecretHeader(type: WebhookProviderType) {
  if (type === "uptime-kuma") {
    return "x-webhook-secret";
  }

  if (type === "docuseal") {
    return "x-docuseal-secret";
  }

  if (type === "git") {
    return "x-hub-signature-256";
  }

  if (type === "vikunja") {
    return "x-vikunja-signature";
  }

  if (type === "ardoise") {
    return "x-ardoise-signature-256";
  }

  if (type === "sablier") {
    return "x-sablier-signature";
  }

  if (type === "perception") {
    return "x-perception-secret";
  }

  if (type === "opus") {
    return "x-opus-signature";
  }

  if (type === "nuage") {
    return "x-nuage-signature-256";
  }

  return "x-dokploy-secret";
}

export function createDefaultWebhookProvider(
  type: WebhookProviderType,
  overrides: Partial<WebhookProviderSettings> = {},
): WebhookProviderSettings {
  return {
    id: overrides.id ?? createProviderId(),
    type,
    name: overrides.name?.trim() || getProviderLabel(type),
    enabled: overrides.enabled ?? true,
    tags: overrides.tags?.map((tag) => tag.trim()).filter(Boolean) ?? [],
    path: overrides.path?.trim() || getWebhookDefaultPath(type),
    secretHeader: overrides.secretHeader?.trim() || getWebhookDefaultSecretHeader(type),
    secretValue: overrides.secretValue?.trim() || null,
  };
}

export function createDefaultImapProvider(
  overrides: Partial<ImapProviderSettings> = {},
): ImapProviderSettings {
  return {
    id: overrides.id ?? createProviderId(),
    type: "imap",
    name: overrides.name?.trim() || "IMAP Inbox",
    enabled: overrides.enabled ?? true,
    tags: overrides.tags?.map((tag) => tag.trim()).filter(Boolean) ?? [],
    host: overrides.host?.trim() || "",
    port: overrides.port ?? 993,
    secure: overrides.secure ?? true,
    username: overrides.username?.trim() || "",
    password: overrides.password?.trim() || null,
    pollInterval: overrides.pollInterval ?? 60,
    includeEmailBody: overrides.includeEmailBody ?? false,
  };
}

export function createDefaultWebsiteProvider(
  overrides: Partial<WebsiteProviderSettings> = {},
): WebsiteProviderSettings {
  return {
    id: overrides.id ?? createProviderId(),
    type: "website",
    name: overrides.name?.trim() || "Website",
    enabled: overrides.enabled ?? true,
    tags: overrides.tags?.map((tag) => tag.trim()).filter(Boolean) ?? [],
    url: overrides.url?.trim() || "https://",
    interval: overrides.interval ?? 60,
  };
}

export function createDefaultRssProvider(
  overrides: Partial<RssProviderSettings> = {},
): RssProviderSettings {
  return {
    id: overrides.id ?? createProviderId(),
    type: "rss",
    name: overrides.name?.trim() || "RSS Feed",
    enabled: overrides.enabled ?? true,
    tags: overrides.tags?.map((tag) => tag.trim()).filter(Boolean) ?? [],
    feedUrl: overrides.feedUrl?.trim() || "",
    pollInterval: overrides.pollInterval ?? 300,
    includeContent: overrides.includeContent ?? false,
  };
}

export function createDefaultProvider(type: ProviderType, overrides: Partial<ProviderSettings> = {}): ProviderSettings {
  if (type === "website") {
    return createDefaultWebsiteProvider(overrides as Partial<WebsiteProviderSettings>);
  }

  if (type === "imap") {
    return createDefaultImapProvider(overrides as Partial<ImapProviderSettings>);
  }

  if (type === "rss") {
    return createDefaultRssProvider(overrides as Partial<RssProviderSettings>);
  }

  return createDefaultWebhookProvider(type, overrides as Partial<WebhookProviderSettings>);
}
