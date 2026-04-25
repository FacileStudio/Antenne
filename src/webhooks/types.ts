export type WebhookProviderId = "uptime-kuma" | "docuseal";

export interface WebhookProviderSettings {
  enabled: boolean;
  path: string;
  secretHeader: string | null;
  secretValue: string | null;
}

export interface WebhookConfig {
  enabled: boolean;
  port: number;
  providers: Record<WebhookProviderId, WebhookProviderSettings>;
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

export interface WebhookDispatchContext {
  providerId: WebhookProviderId;
  path: string;
}

export interface WebhookProviderDefinition {
  id: WebhookProviderId;
  label: string;
  defaultPath: string;
  defaultSecretHeader: string;
  formatMessage: (payload: unknown) => string | null;
  describeEvent: (payload: unknown) => string;
}

export interface ConfiguredWebhookProvider {
  definition: WebhookProviderDefinition;
  settings: WebhookProviderSettings;
}
