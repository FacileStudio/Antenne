import type { NookEventEnvelopeInput } from "../event-envelope";

export type WebhookProviderType = "uptime-kuma" | "docuseal" | "dokploy" | "git" | "vikunja" | "charles" | "sablier" | "perception" | "opus" | "vision" | "plume";
export type ProviderType = WebhookProviderType | "imap" | "website" | "rss";

export interface BaseProviderSettings {
  id: string;
  type: ProviderType;
  name: string;
  enabled: boolean;
  tags: string[];
}

export interface WebhookProviderSettings extends BaseProviderSettings {
  type: WebhookProviderType;
  path: string;
  secretHeader: string | null;
  secretValue: string | null;
}

export interface ImapProviderSettings extends BaseProviderSettings {
  type: "imap";
  host: string;
  port: number;
  secure: boolean;
  username: string;
  password: string | null;
  pollInterval: number;
  includeEmailBody: boolean;
}

export interface WebsiteProviderSettings extends BaseProviderSettings {
  type: "website";
  url: string;
  interval: number;
}

export interface RssProviderSettings extends BaseProviderSettings {
  type: "rss";
  feedUrl: string;
  pollInterval: number;
  includeContent: boolean;
}

export type ProviderSettings = WebhookProviderSettings | ImapProviderSettings | WebsiteProviderSettings | RssProviderSettings;

export interface ProvidersConfigInput {
  port: number;
  items: ProviderSettings[];
}

export interface RawWebhookConfig {
  enabled?: boolean;
  port?: number;
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

export interface DocuSealDocumentPayload {
  name?: string | null;
  url?: string | null;
}

export interface DocuSealEventPayload {
  event_type?: string;
  timestamp?: string;
  data?: {
    id?: number;
    email?: string | null;
    name?: string | null;
    phone?: string | null;
    role?: string | null;
    external_id?: string | null;
    decline_reason?: string | null;
    status?: string | null;
    opened_at?: string | null;
    completed_at?: string | null;
    declined_at?: string | null;
    submission?: {
      id?: number;
      status?: string | null;
      url?: string | null;
      audit_log_url?: string | null;
      combined_document_url?: string | null;
    } | null;
    template?: {
      id?: number;
      name?: string | null;
      external_id?: string | null;
      folder_name?: string | null;
    } | null;
    documents?: DocuSealDocumentPayload[] | null;
    slug?: string | null;
    source?: string | null;
    archived_at?: string | null;
  } | null;
}

export interface DokployWebhookPayload {
  title?: string | null;
  message?: string | null;
  timestamp?: string | null;
  projectName?: string | null;
  applicationName?: string | null;
  applicationType?: string | null;
  buildLink?: string | null;
  status?: string | null;
  type?: string | null;
  [key: string]: unknown;
}

export interface CharlesWebhookPayload {
  id?: string;
  event?: string;
  createdAt?: string;
  source?: string;
  actor?: {
    id?: string;
    email?: string | null;
    role?: string | null;
  } | null;
  data?: {
    entity?: string;
    entityId?: string;
    changedFields?: string[];
    logoUrl?: string;
    record?: {
      id?: string;
      email?: string | null;
      firstName?: string | null;
      lastName?: string | null;
      [key: string]: unknown;
    } | null;
    [key: string]: unknown;
  } | null;
}

export interface WebhookDispatchContext {
  providerId: string;
  providerName: string;
  providerType: WebhookProviderType;
  path: string;
}

export interface WebhookProviderDefinition {
  type: WebhookProviderType;
  label: string;
  defaultPath: string;
  defaultSecretHeader: string;
  authenticate?: (req: Request, rawBody: string, settings: WebhookProviderSettings) => boolean | Promise<boolean>;
  formatMessage: (payload: unknown, context: WebhookDispatchContext) => string | null;
  describeEvent: (payload: unknown) => string;
  buildEvent?: (payload: unknown, context: WebhookDispatchContext) => Partial<NookEventEnvelopeInput>;
}

export interface ConfiguredWebhookProvider {
  definition: WebhookProviderDefinition;
  settings: WebhookProviderSettings;
}

export const webhookProviderTypes = ["uptime-kuma", "docuseal", "dokploy", "git", "vikunja", "charles", "sablier", "perception", "opus", "vision", "plume"] as const satisfies WebhookProviderType[];
export const providerTypes = [...webhookProviderTypes, "imap", "website", "rss"] as const satisfies ProviderType[];

export function isWebhookProviderType(value: unknown): value is WebhookProviderType {
  return typeof value === "string" && webhookProviderTypes.includes(value as WebhookProviderType);
}

export function isProviderType(value: unknown): value is ProviderType {
  return typeof value === "string" && providerTypes.includes(value as ProviderType);
}

export function isWebhookProviderSettings(provider: ProviderSettings): provider is WebhookProviderSettings {
  return provider.type === "uptime-kuma" || provider.type === "docuseal" || provider.type === "dokploy" || provider.type === "git" || provider.type === "vikunja" || provider.type === "charles" || provider.type === "sablier" || provider.type === "perception" || provider.type === "opus" || provider.type === "vision" || provider.type === "plume";
}

export function isImapProviderSettings(provider: ProviderSettings): provider is ImapProviderSettings {
  return provider.type === "imap";
}

export function isWebsiteProviderSettings(provider: ProviderSettings): provider is WebsiteProviderSettings {
  return provider.type === "website";
}

export function isRssProviderSettings(provider: ProviderSettings): provider is RssProviderSettings {
  return provider.type === "rss";
}
