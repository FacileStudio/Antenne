import type { WebhookConfig, RawWebhookConfig } from "./webhooks/types";

export interface SiteConfig {
  name: string;
  url: string;
  interval: number;
}

export interface SiteState {
  up: boolean;
  lastChange: Date;
  consecutiveFailures: number;
  inFlight: boolean;
}

export interface MatrixConfig {
  homeserver: string;
  accessToken: string;
  roomIds: string[];
}

export interface AppSettings {
  matrix: MatrixConfig;
  timeout: number;
  sites: SiteConfig[];
  webhook: WebhookConfig;
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
