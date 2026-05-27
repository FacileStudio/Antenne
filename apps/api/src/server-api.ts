import { hasConfiguredMatrixDelivery } from "./delivery";
import { getEvents, getGlobalStats, getProviderStats, getTargetStats, onEventPushed, getEventById, type DeliveryAttempt } from "./event-log";
import { createJsonResponse } from "./server-responses";
import type { AlertContext, AppSettings, DashboardSessionConfig, DeliveryTarget, RuntimeStatus } from "./types";
import type { ProviderSettings } from "./webhooks/types";

const REDACTED = "••••••••";

const DELIVERY_SENSITIVE_FIELDS: Record<string, string[]> = {
  matrix: ["accessToken", "refreshToken", "password"],
  discord: ["webhookUrl"],
  webhook: ["secretValue"],
  smtp: ["password"],
  ntfy: ["accessToken"],
  perception: ["secretValue"],
};

const PROVIDER_SENSITIVE_FIELDS: Record<string, string[]> = {
  "uptime-kuma": ["secretValue"],
  docuseal: ["secretValue"],
  dokploy: ["secretValue"],
  git: ["secretValue"],
  vikunja: ["secretValue"],
  ardoise: ["secretValue"],
  sablier: ["secretValue"],
  perception: ["secretValue"],
  opus: ["secretValue"],
  vision: ["secretValue"],
  plume: ["secretValue"],
  nuage: ["secretValue"],
  imap: ["password"],
};

function redactSettings(settings: AppSettings): AppSettings {
  const clone = structuredClone(settings);

  for (const item of clone.delivery.items) {
    const fields = DELIVERY_SENSITIVE_FIELDS[item.type];
    if (!fields) continue;
    for (const field of fields) {
      if ((item as unknown as Record<string, unknown>)[field]) {
        (item as unknown as Record<string, unknown>)[field] = REDACTED;
      }
    }
  }

  for (const item of clone.providers.items) {
    const fields = PROVIDER_SENSITIVE_FIELDS[item.type];
    if (!fields) continue;
    for (const field of fields) {
      if ((item as unknown as Record<string, unknown>)[field]) {
        (item as unknown as Record<string, unknown>)[field] = REDACTED;
      }
    }
  }

  return clone;
}

function mergeSecrets(incoming: AppSettings, current: AppSettings): AppSettings {
  if (Array.isArray(incoming.delivery?.items) && Array.isArray(current.delivery?.items)) {
    const currentDeliveryById = new Map<string, DeliveryTarget>();
    for (const item of current.delivery.items) {
      currentDeliveryById.set(item.id, item);
    }

    for (const item of incoming.delivery.items) {
      const fields = DELIVERY_SENSITIVE_FIELDS[item.type];
      if (!fields) continue;
      const match = currentDeliveryById.get(item.id);
      if (!match) continue;
      for (const field of fields) {
        if ((item as unknown as Record<string, unknown>)[field] === REDACTED) {
          (item as unknown as Record<string, unknown>)[field] = (match as unknown as Record<string, unknown>)[field];
        }
      }
    }
  }

  if (Array.isArray(incoming.providers?.items) && Array.isArray(current.providers?.items)) {
    const currentProviderById = new Map<string, ProviderSettings>();
    for (const item of current.providers.items) {
      currentProviderById.set(item.id, item);
    }

    for (const item of incoming.providers.items) {
      const fields = PROVIDER_SENSITIVE_FIELDS[item.type];
      if (!fields) continue;
      const match = currentProviderById.get(item.id);
      if (!match) continue;
      for (const field of fields) {
        if ((item as unknown as Record<string, unknown>)[field] === REDACTED) {
          (item as unknown as Record<string, unknown>)[field] = (match as unknown as Record<string, unknown>)[field];
        }
      }
    }
  }

  return incoming;
}

function escapeHtml(value: string) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}

export type AppServerOptions = {
  getSettings: () => AppSettings;
  updateSettings: (input: unknown) => Promise<AppSettings>;
  sendConfiguredAlert: (message: string, context?: AlertContext) => Promise<DeliveryAttempt[]>;
  sendTestAlertToTarget: (targetId: string) => Promise<void>;
  testProviderConnection: (providerId: string) => Promise<void>;
  getRuntimeStatus: () => RuntimeStatus;
  replayEventToTarget: (eventId: string, targetId: string) => Promise<void>;
  session: DashboardSessionConfig;
};

async function parseJson(req: Request) {
  try {
    return await req.json();
  } catch {
    throw new Error("Invalid JSON body.");
  }
}

export async function handleDashboardApiRequest(req: Request, url: URL, options: AppServerOptions) {
  if (url.pathname === "/api/session" && req.method === "GET") {
    return createJsonResponse({
      authenticated: true,
      passwordRequired: Boolean(options.session.password),
      username: options.session.username,
    });
  }

  if (url.pathname === "/api/settings" && req.method === "GET") {
    const settings = options.getSettings();
    return createJsonResponse({
      settings: redactSettings(settings),
      runtime: options.getRuntimeStatus(),
      matrixConfigured: hasConfiguredMatrixDelivery(settings.delivery),
    });
  }

  if (url.pathname === "/api/settings" && req.method === "PUT") {
    try {
      const body = await parseJson(req);
      const merged = mergeSecrets(body as AppSettings, options.getSettings());
      const settings = await options.updateSettings(merged);
      console.log("[admin] settings updated");
      return createJsonResponse({
        settings: redactSettings(settings),
        runtime: options.getRuntimeStatus(),
        matrixConfigured: hasConfiguredMatrixDelivery(settings.delivery),
      });
    } catch (error) {
      return createJsonResponse({ error: error instanceof Error ? error.message : "failed to save" }, { status: 400 });
    }
  }

  if (url.pathname === "/api/test-alert" && req.method === "POST") {
    const settings = options.getSettings();
    const label = settings.providers.items[0]?.name ?? "dashboard";

    try {
      await options.sendConfiguredAlert(`🧪 <b>Nook test alert</b><br/>Dashboard config for <b>${escapeHtml(label)}</b> is live.`);
      return createJsonResponse({ ok: true });
    } catch (error) {
      return createJsonResponse({ error: error instanceof Error ? error.message : "failed to send alert" }, { status: 400 });
    }
  }

  if (url.pathname === "/api/test-delivery-target" && req.method === "POST") {
    try {
      const body = await parseJson(req) as { targetId?: string };
      if (!body.targetId) {
        return createJsonResponse({ error: "targetId is required" }, { status: 400 });
      }

      await options.sendTestAlertToTarget(body.targetId);
      return createJsonResponse({ ok: true });
    } catch (error) {
      return createJsonResponse({ error: error instanceof Error ? error.message : "failed to test target" }, { status: 400 });
    }
  }

  if (url.pathname === "/api/test-provider-connection" && req.method === "POST") {
    try {
      const body = await parseJson(req) as { providerId?: string };
      if (!body.providerId) {
        return createJsonResponse({ error: "providerId is required" }, { status: 400 });
      }

      await options.testProviderConnection(body.providerId);
      return createJsonResponse({ ok: true });
    } catch (error) {
      return createJsonResponse({ error: error instanceof Error ? error.message : "failed to test provider" }, { status: 400 });
    }
  }

  if (url.pathname === "/api/events/stream" && req.method === "GET") {
    const encoder = new TextEncoder();
    const stream = new ReadableStream({
      start(controller) {
        const cleanup = onEventPushed((entry) => {
          try {
            controller.enqueue(encoder.encode(`data: ${JSON.stringify(entry)}\n\n`));
          } catch {}
        });
        req.signal.addEventListener("abort", () => {
          cleanup();
          try { controller.close(); } catch {}
        });
      },
    });
    return new Response(stream, {
      headers: {
        "Content-Type": "text/event-stream",
        "Cache-Control": "no-cache",
        Connection: "keep-alive",
      },
    });
  }

  if (url.pathname === "/api/events" && req.method === "GET") {
    const limit = Number(url.searchParams.get("limit")) || 50;
    const offset = Number(url.searchParams.get("offset")) || 0;
    const q = url.searchParams.get("q") || undefined;
    const source = url.searchParams.get("source") || undefined;
    const { events, total } = getEvents({ limit, offset, q, source });
    return createJsonResponse({ events, total, stats: getGlobalStats() });
  }

  if (url.pathname.startsWith("/api/events/provider/") && req.method === "GET") {
    const providerId = url.pathname.replace("/api/events/provider/", "");
    const limit = Number(url.searchParams.get("limit")) || 50;
    const offset = Number(url.searchParams.get("offset")) || 0;
    const { events, total } = getEvents({ providerId, limit, offset });
    return createJsonResponse({ events, total, stats: getProviderStats(providerId) });
  }

  if (url.pathname.startsWith("/api/events/target/") && req.method === "GET") {
    const targetId = url.pathname.replace("/api/events/target/", "");
    const limit = Number(url.searchParams.get("limit")) || 50;
    const offset = Number(url.searchParams.get("offset")) || 0;
    const { events, total } = getEvents({ targetId, limit, offset });
    return createJsonResponse({ events, total, stats: getTargetStats(targetId) });
  }

  const replayMatch = url.pathname.match(/^\/api\/events\/([^/]+)\/replay$/);
  if (replayMatch && req.method === "POST") {
    const eventId = replayMatch[1]!;
    const body = await parseJson(req) as { targetId?: string };
    if (!body.targetId) {
      return createJsonResponse({ error: "targetId is required" }, { status: 400 });
    }
    try {
      await options.replayEventToTarget(eventId, body.targetId);
      return createJsonResponse({ ok: true });
    } catch (error) {
      return createJsonResponse({ error: error instanceof Error ? error.message : "replay failed" }, { status: 400 });
    }
  }

  return createJsonResponse({ error: "not found" }, { status: 404 });
}
