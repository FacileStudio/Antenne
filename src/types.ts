import type { ProviderSettings, RawWebhookConfig } from "./webhooks/types";

export interface SiteState {
  up: boolean;
  lastChange: Date;
  consecutiveFailures: number;
  inFlight: boolean;
}

export type DeliveryTargetType = "matrix" | "discord" | "webhook" | "smtp";

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

export type DeliveryTarget = MatrixDeliveryTarget | DiscordDeliveryTarget | HttpDeliveryTarget | SmtpDeliveryTarget;

export interface DeliveryConfig {
  items: DeliveryTarget[];
}

export interface ProvidersConfig {
  port: number;
  items: ProviderSettings[];
}

export interface AppSettings {
  delivery: DeliveryConfig;
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
  token: string;
}

export interface RuntimeStatus {
  serverPort: number;
  settingsPath: string;
  usingPersistedSettings: boolean;
  dashboardBuilt: boolean;
  lastReloadedAt: string;
}
