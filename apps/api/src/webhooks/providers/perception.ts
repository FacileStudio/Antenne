import type { WebhookProviderDefinition } from "../types";

interface PerceptionAlert {
  type: string;
  severity: "info" | "warning" | "critical";
  title: string;
  message: string;
  entity_name?: string | null;
  metadata?: Record<string, unknown>;
}

interface PerceptionDigestPayload {
  source: "perception";
  event: "digest";
  count: number;
  alerts: PerceptionAlert[];
  generated_at: string;
}

const SEVERITY_EMOJI: Record<string, string> = {
  critical: "\u{1F534}",
  warning: "\u{1F7E0}",
  info: "\u{1F535}",
};

const TYPE_LABELS: Record<string, string> = {
  entity_velocity_spike: "Velocity Spike",
  service_health_degradation: "Health Degradation",
  client_silence: "Gone Quiet",
  actor_absence: "Actor Absent",
};

function formatAlert(alert: PerceptionAlert): string {
  const emoji = SEVERITY_EMOJI[alert.severity] ?? SEVERITY_EMOJI.info;
  const typeLabel = TYPE_LABELS[alert.type] ?? alert.type;
  const entity = alert.entity_name ? ` <code>${alert.entity_name}</code>` : "";
  return `${emoji} <b>${alert.title}</b>${entity}<br/><i>${alert.message}</i> <sub>[${typeLabel}]</sub>`;
}

function parsePayload(payload: unknown): PerceptionDigestPayload | null {
  const p = payload as Record<string, unknown>;
  if (p?.source !== "perception" || p?.event !== "digest") return null;
  return p as unknown as PerceptionDigestPayload;
}

export const perceptionWebhookProvider: WebhookProviderDefinition = {
  type: "perception",
  label: "Perception",
  defaultPath: "/webhook/perception",
  defaultSecretHeader: "x-perception-secret",

  formatMessage(payload) {
    const parsed = parsePayload(payload);
    if (!parsed || !parsed.alerts?.length) return null;

    const header = `<b>[Perception]</b> ${parsed.count} anomal${parsed.count === 1 ? "y" : "ies"} detected`;
    const body = parsed.alerts.map(formatAlert).join("<br/><br/>");
    return `${header}<br/><br/>${body}`;
  },

  describeEvent(payload) {
    const parsed = parsePayload(payload);
    if (!parsed) return "unknown perception event";
    return `${parsed.count} anomal${parsed.count === 1 ? "y" : "ies"} detected`;
  },

  buildEvent(payload) {
    const parsed = parsePayload(payload);
    if (!parsed) return {};

    const severities = parsed.alerts.map((a) => a.severity);
    const topSeverity = severities.includes("critical") ? "critical" : severities.includes("warning") ? "warning" : "info";

    return {
      eventType: "perception.digest",
      eventTitle: `${parsed.count} anomal${parsed.count === 1 ? "y" : "ies"}`,
      occurredAt: parsed.generated_at ?? null,
      normalizedPayload: {
        count: parsed.count,
        top_severity: topSeverity,
        types: [...new Set(parsed.alerts.map((a) => a.type))],
        entities: parsed.alerts
          .map((a) => a.entity_name)
          .filter(Boolean),
      },
    };
  },
};
