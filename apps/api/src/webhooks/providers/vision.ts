import { timingSafeEqual as nodeTimingSafeEqual } from "node:crypto";
import type { WebhookProviderDefinition } from "../types";

interface VisionSite {
  id: number;
  name: string;
  domain: string;
}

interface VisionPeriod {
  type: string;
  from: string;
  to: string;
}

interface VisionMetrics {
  total_pageviews: number;
  unique_visitors: number;
  views_per_visitor: number;
  prev_total_pageviews: number;
  prev_unique_visitors: number;
  pageviews_change_pct: number;
  visitors_change_pct: number;
}

interface VisionReportPayload {
  event_type: string;
  site: VisionSite;
  period: VisionPeriod;
  metrics: VisionMetrics;
  top_pages: { name: string; count: number }[];
  top_referrers: { name: string; count: number }[];
  top_countries: { name: string; count: number }[];
  top_browsers: { name: string; count: number }[];
  top_devices: { name: string; count: number }[];
}

async function hmacSha256Hex(secret: string, data: string): Promise<string> {
  const key = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"],
  );
  const sig = await crypto.subtle.sign("HMAC", key, new TextEncoder().encode(data));
  return [...new Uint8Array(sig)].map((b) => b.toString(16).padStart(2, "0")).join("");
}

function timingSafeEqual(a: string, b: string): boolean {
  const ab = Buffer.from(a);
  const bb = Buffer.from(b);
  if (ab.length !== bb.length) return false;
  return nodeTimingSafeEqual(ab, bb);
}

function parsePayload(payload: unknown): VisionReportPayload | null {
  const p = payload as Record<string, unknown>;
  if (!p?.event_type || !p?.site || !p?.metrics) return null;
  return p as unknown as VisionReportPayload;
}

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1);
}

function formatChange(pct: number): string {
  if (pct > 0) return `<span style="color:green">↑ +${pct.toFixed(1)}%</span>`;
  if (pct < 0) return `<span style="color:red">↓ ${pct.toFixed(1)}%</span>`;
  return `<span>0%</span>`;
}

function topList(items: { name: string; count: number }[], limit = 3): string {
  return items
    .slice(0, limit)
    .map((i) => `${i.name} (${i.count.toLocaleString()})`)
    .join(", ");
}

export const visionWebhookProvider: WebhookProviderDefinition = {
  type: "vision",
  label: "Vision",
  defaultPath: "/webhook/vision",
  defaultSecretHeader: "x-vision-signature-256",

  async authenticate(req, rawBody, settings) {
    if (!settings.secretValue) return true;
    const header = req.headers.get(settings.secretHeader ?? "x-vision-signature-256");
    if (!header) return false;
    const expected = "sha256=" + (await hmacSha256Hex(settings.secretValue, rawBody));
    return timingSafeEqual(header, expected);
  },

  formatMessage(payload) {
    const parsed = parsePayload(payload);
    if (!parsed) return null;

    const lines = [
      `<p><strong>\u{1F4CA} Vision Analytics \u2014 ${parsed.site.domain}</strong></p>`,
      `<p>${capitalize(parsed.period.type)} report (${parsed.period.from} \u2192 ${parsed.period.to})</p>`,
      `<p><strong>Pageviews:</strong> ${parsed.metrics.total_pageviews.toLocaleString()} (${formatChange(parsed.metrics.pageviews_change_pct)})</p>`,
      `<p><strong>Visitors:</strong> ${parsed.metrics.unique_visitors.toLocaleString()} (${formatChange(parsed.metrics.visitors_change_pct)})</p>`,
      `<p><strong>Views/Visitor:</strong> ${parsed.metrics.views_per_visitor.toFixed(1)}</p>`,
    ];

    if (parsed.top_pages?.length) {
      lines.push(`<p><strong>Top Pages:</strong> ${topList(parsed.top_pages)}</p>`);
    }

    if (parsed.top_referrers?.length) {
      lines.push(`<p><strong>Top Sources:</strong> ${topList(parsed.top_referrers)}</p>`);
    }

    return lines.join("\n");
  },

  describeEvent(payload) {
    const parsed = parsePayload(payload);
    if (!parsed) return "unknown vision event";
    return `${capitalize(parsed.period.type)} analytics report for ${parsed.site.domain}: ${parsed.metrics.total_pageviews.toLocaleString()} pageviews, ${parsed.metrics.unique_visitors.toLocaleString()} visitors`;
  },

  buildEvent(payload) {
    const parsed = parsePayload(payload);
    if (!parsed) return {};

    return {
      eventType: `vision.${parsed.event_type}`,
      eventTitle: `${capitalize(parsed.period.type)} Analytics \u2014 ${parsed.site.domain}`,
      subject: {
        type: "service",
        id: String(parsed.site.id),
        name: parsed.site.domain,
      },
      normalizedPayload: {
        period: parsed.period.type,
        siteDomain: parsed.site.domain,
        totalPageviews: parsed.metrics.total_pageviews,
        uniqueVisitors: parsed.metrics.unique_visitors,
        viewsPerVisitor: parsed.metrics.views_per_visitor,
        pageviewsChangePct: parsed.metrics.pageviews_change_pct,
        visitorsChangePct: parsed.metrics.visitors_change_pct,
        status: "delivered",
      },
      containsPii: false,
      rawPayloadAllowed: true,
    };
  },
};
