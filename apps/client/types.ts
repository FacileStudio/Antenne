export type WebhookProviderType = "uptime-kuma" | "docuseal" | "dokploy" | "git" | "vikunja" | "charles" | "sablier";
export type ProviderType = WebhookProviderType | "imap" | "website";
export type DeliveryTargetType = "matrix" | "discord" | "webhook" | "smtp" | "ntfy" | "perception";

export interface DeliveryTargetBase {
  id: string;
  type: DeliveryTargetType;
  name: string;
  enabled: boolean;
  providerIds: string[];
  providerTags: string[];
}

export interface MatrixTarget extends DeliveryTargetBase {
  type: "matrix";
  homeserver: string;
  accessToken: string;
  refreshToken: string | null;
  userId: string;
  password: string | null;
  deviceId: string | null;
  roomIds: string[];
}

export interface DiscordTarget extends DeliveryTargetBase {
  type: "discord";
  webhookUrl: string;
}

export interface WebhookTarget extends DeliveryTargetBase {
  type: "webhook";
  url: string;
  secretValue: string | null;
  includeRawPayload: boolean;
}

export interface SmtpTarget extends DeliveryTargetBase {
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

export interface NtfyTarget extends DeliveryTargetBase {
  type: "ntfy";
  serverUrl: string;
  topic: string;
  accessToken: string | null;
  priority: number | null;
  tags: string[];
  clickUrl: string | null;
}

export interface PerceptionTarget extends DeliveryTargetBase {
  type: "perception";
  url: string;
  secretValue: string | null;
  includeRawPayload: boolean;
}

export type DeliveryTarget = MatrixTarget | DiscordTarget | WebhookTarget | SmtpTarget | NtfyTarget | PerceptionTarget;

export interface WebsiteProviderSettings {
  id: string;
  type: "website";
  name: string;
  enabled: boolean;
  tags: string[];
  url: string;
  interval: number;
}

export interface WebhookProviderSettings {
  id: string;
  type: WebhookProviderType;
  name: string;
  enabled: boolean;
  tags: string[];
  path: string;
  secretHeader: string | null;
  secretValue: string | null;
}

export interface ImapProviderSettings {
  id: string;
  type: "imap";
  name: string;
  enabled: boolean;
  tags: string[];
  host: string;
  port: number;
  secure: boolean;
  username: string;
  password: string | null;
  pollInterval: number;
  includeEmailBody: boolean;
}

export type ProviderSettings = WebsiteProviderSettings | WebhookProviderSettings | ImapProviderSettings;

export interface AppSettings {
  delivery: {
    items: DeliveryTarget[];
  };
  system: {
    events: {
      startup: boolean;
      providerEnabled: boolean;
      providerDisabled: boolean;
      providerAdded: boolean;
      providerRemoved: boolean;
      deliveryFailure: boolean;
    };
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

export interface DeliveryAttempt {
  targetId: string;
  targetName: string;
  targetType: string;
  status: "success" | "error";
  error?: string;
  durationMs: number;
}

export interface EventLogEntry {
  id: string;
  timestamp: string;
  source: "webhook" | "website" | "imap" | "test" | "system";
  providerId: string | null;
  providerName: string;
  providerType: string;
  event: string;
  message: string | null;
  envelope?: unknown;
  deliveries: DeliveryAttempt[];
}

export interface EventLogStats {
  totalEvents: number;
  lastEventAt: string | null;
  deliverySuccessCount: number;
  deliveryErrorCount: number;
}

export const providerTypes = ["website", "imap", "charles", "docuseal", "dokploy", "git", "sablier", "uptime-kuma", "vikunja"] as const satisfies ProviderType[];
export const deliveryTargetTypes = ["matrix", "discord", "webhook", "smtp", "ntfy", "perception"] as const satisfies DeliveryTargetType[];
export const SYSTEM_PROVIDER_ID = "__nook_system__";

function createId() {
  return globalThis.crypto.randomUUID();
}

function createMatrixDeviceId() {
  return `NOOK${createId().replaceAll("-", "").slice(0, 12).toUpperCase()}`;
}

export function getProviderLabel(type: ProviderType) {
  if (type === "website") return "Website";
  if (type === "imap") return "IMAP";
  if (type === "docuseal") return "DocuSeal";
  if (type === "dokploy") return "Dokploy";
  if (type === "git") return "Git";
  if (type === "vikunja") return "Vikunja";
  if (type === "charles") return "Charles";
  if (type === "sablier") return "Sablier";
  return "Uptime Kuma";
}

export function getDeliveryTargetLabel(type: DeliveryTargetType) {
  if (type === "matrix") return "Matrix";
  if (type === "discord") return "Discord webhook";
  if (type === "smtp") return "SMTP";
  if (type === "ntfy") return "ntfy";
  if (type === "perception") return "Perception";
  return "HTTP webhook";
}

export function createDefaultProvider(type: ProviderType): ProviderSettings {
  if (type === "website") {
    return {
      id: createId(),
      type,
      name: "Website",
      enabled: true,
      tags: [],
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
      tags: [],
      host: "",
      port: 993,
      secure: true,
      username: "",
      password: "",
      pollInterval: 60,
      includeEmailBody: false,
    };
  }

  return {
    id: createId(),
    type,
    name: getProviderLabel(type),
    enabled: true,
    tags: [],
    path: type === "dokploy" ? "/webhook/dokploy" : type === "docuseal" ? "/webhook/docuseal" : type === "git" ? "/webhook/git" : type === "vikunja" ? "/webhook/vikunja" : type === "charles" ? "/webhook/charles" : type === "sablier" ? "/webhook/sablier" : "/webhook/uptime-kuma",
    secretHeader: type === "dokploy" ? "x-dokploy-secret" : type === "docuseal" ? "x-docuseal-secret" : type === "git" ? "x-hub-signature-256" : type === "vikunja" ? "x-vikunja-signature" : type === "charles" ? "x-charles-signature-256" : type === "sablier" ? "x-sablier-signature" : "x-webhook-secret",
    secretValue: "",
  };
}

export function createDefaultDeliveryTarget(type: DeliveryTargetType): DeliveryTarget {
  if (type === "matrix") {
    return {
      id: createId(),
      type,
      name: "Matrix",
      enabled: true,
      providerIds: [],
      providerTags: [],
      homeserver: "https://matrix.org",
      accessToken: "",
      refreshToken: "",
      userId: "",
      password: "",
      deviceId: createMatrixDeviceId(),
      roomIds: [],
    };
  }

  if (type === "discord") {
    return {
      id: createId(),
      type,
      name: "Discord webhook",
      enabled: true,
      providerIds: [],
      providerTags: [],
      webhookUrl: "",
    };
  }

  if (type === "smtp") {
    return {
      id: createId(),
      type,
      name: "SMTP",
      enabled: true,
      providerIds: [],
      providerTags: [],
      host: "",
      port: 587,
      secure: true,
      username: "",
      password: null,
      fromAddress: "",
      toAddresses: [],
      subjectPrefix: "[Nook]",
    };
  }

  if (type === "ntfy") {
    return {
      id: createId(),
      type,
      name: "ntfy",
      enabled: true,
      providerIds: [],
      providerTags: [],
      serverUrl: "https://ntfy.sh",
      topic: "",
      accessToken: "",
      priority: 3,
      tags: ["rotating_light"],
      clickUrl: "",
    };
  }

  if (type === "perception") {
    return {
      id: createId(),
      type,
      name: "Perception",
      enabled: true,
      providerIds: [],
      providerTags: [],
      url: "",
      secretValue: "",
      includeRawPayload: false,
    };
  }

  return {
    id: createId(),
    type,
    name: "HTTP webhook",
    enabled: true,
    providerIds: [],
    providerTags: [],
    url: "",
    secretValue: "",
    includeRawPayload: false,
  };
}
