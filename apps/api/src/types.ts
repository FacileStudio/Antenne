import type { ProviderSettings, RawWebhookConfig } from "./webhooks/types";

export interface SiteState {
  up: boolean;
  lastChange: Date;
  consecutiveFailures: number;
  inFlight: boolean;
}

export type DeliveryTargetType = "matrix" | "discord" | "webhook" | "smtp" | "ntfy";

export interface DeliveryTargetBase {
  id: string;
  type: DeliveryTargetType;
  name: string;
  enabled: boolean;
  providerIds: string[];
}

export interface MatrixDeliveryTarget extends DeliveryTargetBase {
  type: "matrix";
  homeserver: string;
  accessToken: string;
  refreshToken: string | null;
  userId: string;
  password: string | null;
  deviceId: string | null;
  roomIds: string[];
}

export interface DiscordDeliveryTarget extends DeliveryTargetBase {
  type: "discord";
  webhookUrl: string;
}

export interface HttpDeliveryTarget extends DeliveryTargetBase {
  type: "webhook";
  url: string;
  secretHeader: string | null;
  secretValue: string | null;
}

export interface SmtpDeliveryTarget extends DeliveryTargetBase {
  type: "smtp";
  host: string;
  port: number;
  secure: boolean;
  username: string;
  password: string | null;
  fromAddress: string;
  toAddresses: string[];
  subjectPrefix: string;
}

export interface NtfyDeliveryTarget extends DeliveryTargetBase {
  type: "ntfy";
  serverUrl: string;
  topic: string;
  accessToken: string | null;
  priority: number | null;
  tags: string[];
  clickUrl: string | null;
}

export type DeliveryTarget = MatrixDeliveryTarget | DiscordDeliveryTarget | HttpDeliveryTarget | SmtpDeliveryTarget | NtfyDeliveryTarget;

export interface DeliveryConfig {
  items: DeliveryTarget[];
}

export interface SystemEventSettings {
  startup: boolean;
  providerEnabled: boolean;
}

export interface SystemSettings {
  events: SystemEventSettings;
}

export interface ProvidersConfig {
  port: number;
  items: ProviderSettings[];
}

export interface AppSettings {
  delivery: DeliveryConfig;
  system: SystemSettings;
  timeout: number;
  providers: ProvidersConfig;
}

export interface RawSiteOptions {
  name?: string;
  url?: string;
  every?: number;
  interval?: number;
}

export type RawSiteConfig = string | RawSiteOptions;

export interface RawConfig {
  timeout?: number;
  every?: number;
  interval?: number;
  sites?: RawSiteConfig[];
  webhook?: RawWebhookConfig;
}

export interface DashboardSessionConfig {
  enabled: boolean;
  username: string;
  password: string | null;
  cookieName: string;
}

export const SYSTEM_PROVIDER_ID = "__nook_system__";

export interface RuntimeStatus {
  serverPort: number;
  settingsPath: string;
  usingPersistedSettings: boolean;
  dashboardBuilt: boolean;
  lastReloadedAt: string;
}
