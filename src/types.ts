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

export interface WebhookConfig {
  enabled: boolean;
  port: number;
}

export interface MatrixConfig {
  homeserver: string;
  accessToken: string;
  roomIds: string[];
}

export interface Config {
  sites: SiteConfig[];
  timeout: number;
  webhook: WebhookConfig;
}

export interface RawSiteOptions {
  name?: string;
  url?: string;
  every?: number;
  interval?: number;
}

export type RawSiteConfig = string | RawSiteOptions;

export interface RawWebhookConfig {
  enabled?: boolean;
  port?: number;
}

export interface RawConfig {
  timeout?: number;
  every?: number;
  interval?: number;
  sites?: RawSiteConfig[];
  webhook?: RawWebhookConfig;
}

export interface UptimeKumaPayload {
  heartbeat?: {
    monitorID?: number;
    status?: number;
    time?: string;
    msg?: string;
    ping?: number | null;
    important?: boolean;
    duration?: number;
  } | null;
  monitor?: {
    id?: number;
    name?: string;
    url?: string;
    type?: string;
    hostname?: string | null;
  } | null;
  msg?: string;
}
