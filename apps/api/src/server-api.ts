import { hasConfiguredMatrixDelivery } from "./delivery";
import { getEvents, getGlobalStats, getProviderStats, getTargetStats, type DeliveryAttempt } from "./event-log";
import { createJsonResponse } from "./server-responses";
import type { AlertContext, AppSettings, DashboardSessionConfig, RuntimeStatus } from "./types";

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
    return createJsonResponse({
      settings: options.getSettings(),
      runtime: options.getRuntimeStatus(),
      matrixConfigured: hasConfiguredMatrixDelivery(options.getSettings().delivery),
    });
  }

  if (url.pathname === "/api/settings" && req.method === "PUT") {
    try {
      const body = await parseJson(req);
      const settings = await options.updateSettings(body);
      return createJsonResponse({
        settings,
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

  if (url.pathname === "/api/events" && req.method === "GET") {
    const limit = Number(url.searchParams.get("limit")) || 50;
    return createJsonResponse({ events: getEvents({ limit }), stats: getGlobalStats() });
  }

  if (url.pathname.startsWith("/api/events/provider/") && req.method === "GET") {
    const providerId = url.pathname.replace("/api/events/provider/", "");
    const limit = Number(url.searchParams.get("limit")) || 50;
    return createJsonResponse({ events: getEvents({ providerId, limit }), stats: getProviderStats(providerId) });
  }

  if (url.pathname.startsWith("/api/events/target/") && req.method === "GET") {
    const targetId = url.pathname.replace("/api/events/target/", "");
    const limit = Number(url.searchParams.get("limit")) || 50;
    return createJsonResponse({ events: getEvents({ targetId, limit }), stats: getTargetStats(targetId) });
  }

  return createJsonResponse({ error: "not found" }, { status: 404 });
}
