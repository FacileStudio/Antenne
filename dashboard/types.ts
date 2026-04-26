export type WebhookProviderType = "uptime-kuma" | "docuseal" | "dokploy";
export type ProviderType = WebhookProviderType | "imap" | "website";

export interface MatrixConfig {
  enabled: boolean;
  homeserver: string;
  accessToken: string;
  roomIds: string[];
}

export interface DiscordDeliveryConfig {
  enabled: boolean;
  webhookUrl: string;
}

export interface HttpDeliveryConfig {
  enabled: boolean;
  url: string;
  secretHeader: string | null;
  secretValue: string | null;
}

export interface WebsiteProviderSettings {
  id: string;
  type: "website";
  name: string;
  enabled: boolean;
  url: string;
  interval: number;
}

export interface WebhookProviderSettings {
  id: string;
  type: WebhookProviderType;
  name: string;
  enabled: boolean;
  path: string;
  secretHeader: string | null;
  secretValue: string | null;
}

export interface ImapProviderSettings {
  id: string;
  type: "imap";
  name: string;
  enabled: boolean;
  host: string;
  port: number;
  secure: boolean;
  username: string;
  password: string | null;
  mailbox: string;
  pollInterval: number;
}

export type ProviderSettings = WebsiteProviderSettings | WebhookProviderSettings | ImapProviderSettings;

export interface AppSettings {
  delivery: {
    matrix: MatrixConfig;
    discord: DiscordDeliveryConfig;
    webhook: HttpDeliveryConfig;
  };
  timeout: number;
  providers: {
    port: number;
    items: ProviderSettings[];
  };
}

export interface RuntimeStatus {
  serverPort: number;
  settingsPath: string;
  usingPersistedSettings: boolean;
  dashboardBuilt: boolean;
  lastReloadedAt: string;
}

export const providerTypes = ["website", "imap", "uptime-kuma", "docuseal", "dokploy"] as const satisfies ProviderType[];

function createId() {
  return globalThis.crypto.randomUUID();
}

export function getProviderLabel(type: ProviderType) {
  if (type === "website") return "Website";
  if (type === "imap") return "IMAP";
  if (type === "docuseal") return "DocuSeal";
  if (type === "dokploy") return "Dokploy";
  return "Uptime Kuma";
}

export function createDefaultProvider(type: ProviderType): ProviderSettings {
  if (type === "website") {
    return {
      id: createId(),
      type,
      name: "Website",
      enabled: true,
      url: "https://",
      interval: 60,
    };
  }

  if (type === "imap") {
    return {
      id: createId(),
      type,
      name: "IMAP Inbox",
      enabled: true,
      host: "",
      port: 993,
      secure: true,
      username: "",
      password: "",
      mailbox: "INBOX",
      pollInterval: 60,
    };
  }

  return {
    id: createId(),
    type,
    name: getProviderLabel(type),
    enabled: true,
    path: type === "dokploy" ? "/webhook/dokploy" : type === "docuseal" ? "/webhook/docuseal" : "/webhook/uptime-kuma",
    secretHeader: type === "dokploy" ? "x-dokploy-secret" : type === "docuseal" ? "x-docuseal-secret" : "x-webhook-secret",
    secretValue: "",
  };
}
