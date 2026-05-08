import type { ProviderSettings } from "./webhooks/types";

import type { NookEventEnvelope } from "./event-envelope";

export interface SiteState {
  up: boolean;
  lastChange: Date;
  consecutiveFailures: number;
  inFlight: boolean;
}

export type DeliveryTargetType = "matrix" | "discord" | "webhook" | "smtp" | "ntfy" | "perception";

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
  secretValue: string | null;
  includeRawPayload: boolean;
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

export interface PerceptionDeliveryTarget extends DeliveryTargetBase {
  type: "perception";
  url: string;
  secretValue: string | null;
  includeRawPayload: boolean;
}

export type DeliveryTarget = MatrixDeliveryTarget | DiscordDeliveryTarget | HttpDeliveryTarget | SmtpDeliveryTarget | NtfyDeliveryTarget | PerceptionDeliveryTarget;

export interface DeliveryConfig {
  items: DeliveryTarget[];
}

export interface SystemEventSettings {
  startup: boolean;
  providerEnabled: boolean;
  providerDisabled: boolean;
  providerAdded: boolean;
  providerRemoved: boolean;
  deliveryFailure: boolean;
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

export interface DashboardSessionConfig {
  enabled: boolean;
  username: string;
  password: string | null;
  cookieName: string;
}

export const SYSTEM_PROVIDER_ID = "__nook_system__";

export type AlertContext = {
  providerId: string;
  providerName: string;
  providerType: string;
  event: string;
  providerTags?: string[];
  eventType?: string;
  eventTitle?: string | null;
  eventUrl?: string | null;
  actor?: {
    id?: string | null;
    name?: string | null;
    email?: string | null;
  } | null;
  subject?: {
    type?: "task" | "contract" | "client" | "deployment" | "repository" | "email" | "service" | "schedule" | "system" | "unknown";
    id?: string | null;
    name?: string | null;
    url?: string | null;
  } | null;
  normalizedPayload?: Record<string, unknown> | null;
  rawPayload?: unknown | null;
  occurredAt?: string | null;
  containsPii?: boolean;
  rawPayloadAllowed?: boolean;
};

export interface DeliveryMessage {
  message: string;
  envelope: NookEventEnvelope;
}

export interface RuntimeStatus {
  serverPort: number;
  settingsPath: string;
  usingPersistedSettings: boolean;
  dashboardBuilt: boolean;
  lastReloadedAt: string;
}
