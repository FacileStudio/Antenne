import type {
  ImapProviderSettings,
  ProviderSettings,
  ProviderType,
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
    host: overrides.host?.trim() || "",
    port: overrides.port ?? 993,
    secure: overrides.secure ?? true,
    username: overrides.username?.trim() || "",
    password: overrides.password?.trim() || null,
    pollInterval: overrides.pollInterval ?? 60,
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
    url: overrides.url?.trim() || "https://",
    interval: overrides.interval ?? 60,
  };
}

export function createDefaultProvider(type: ProviderType, overrides: Partial<ProviderSettings> = {}): ProviderSettings {
  if (type === "website") {
    return createDefaultWebsiteProvider(overrides as Partial<WebsiteProviderSettings>);
  }

  if (type === "imap") {
    return createDefaultImapProvider(overrides as Partial<ImapProviderSettings>);
  }

  return createDefaultWebhookProvider(type, overrides as Partial<WebhookProviderSettings>);
}
