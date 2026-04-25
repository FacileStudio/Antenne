export type WebhookProviderId = "uptime-kuma" | "docuseal";

export interface SiteConfig {
  name: string;
  url: string;
  interval: number;
}

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

export interface WebhookProviderSettings {
  enabled: boolean;
  path: string;
  secretHeader: string | null;
  secretValue: string | null;
}

export interface AppSettings {
  delivery: {
    matrix: MatrixConfig;
    discord: DiscordDeliveryConfig;
    webhook: HttpDeliveryConfig;
  };
  timeout: number;
  sites: SiteConfig[];
  webhook: {
    enabled: boolean;
    port: number;
    providers: Record<WebhookProviderId, WebhookProviderSettings>;
  };
}

export interface RuntimeStatus {
  serverPort: number;
  settingsPath: string;
  usingPersistedSettings: boolean;
  dashboardBuilt: boolean;
  lastReloadedAt: string;
}
