import { enqueueRetry, startRetryProcessor } from "./src/retry";
import { createImapManager, testImapConnection } from "./src/imap";
import { createMonitorManager } from "./src/monitor";
import { createRssManager } from "./src/rss";
import { DeliveryError, getDeliveryRouteSummary, hasAnyDelivery, sendAlert, sendAlertToTarget } from "./src/delivery";
import { isMatrixDeliveryTarget } from "./src/delivery-config";
import { buildNookEventEnvelope } from "./src/event-envelope";
import { createAppServer, createDashboardSessionConfig } from "./src/server";
import { initEventLog, pushEvent, updateEventDeliveries, getEventById, type DeliveryAttempt } from "./src/event-log";
import { isImapProviderSettings, isRssProviderSettings, isWebhookProviderSettings, isWebsiteProviderSettings } from "./src/webhooks/types";
import {
  createInitialRuntimeStatus,
  getSettingsPath,
  loadDefaultSettings,
  loadPersistedSettings,
  SettingsStore,
} from "./src/settings";
import { SYSTEM_PROVIDER_ID, type AppSettings } from "./src/types";
import type { AlertContext, DeliveryMessage } from "./src/types";
import { createHash } from "node:crypto";
import { PoolServer } from "./src/pool/server";

const recentEventHashes = new Map<string, number>();
const DEDUP_WINDOW_MS = 30_000;

function isDuplicateEvent(message: string, providerId: string): boolean {
  const now = Date.now();
  for (const [key, timestamp] of recentEventHashes) {
    if (now - timestamp > DEDUP_WINDOW_MS) recentEventHashes.delete(key);
  }

  const hash = createHash("sha256").update(`${providerId}\0${message}`).digest("hex");
  if (recentEventHashes.has(hash)) return true;
  recentEventHashes.set(hash, now);
  return false;
}

const settingsPath = getSettingsPath();
const eventLogPath = settingsPath.replace(/\.json$/, "-events.json");
await initEventLog(eventLogPath);
const defaults = await loadDefaultSettings();
const persistedSettings = await loadPersistedSettings(settingsPath);
const initialSettings = persistedSettings ?? defaults;
const store = new SettingsStore(defaults, initialSettings, settingsPath, Boolean(persistedSettings));
const runtimeStatus = createInitialRuntimeStatus(settingsPath, Boolean(persistedSettings));
const session = createDashboardSessionConfig();

function getAlertSource(context?: AlertContext) {
  if (context?.providerType === "website") return "website";
  if (context?.providerType === "imap") return "imap";
  if (context?.providerType === "rss") return "rss";
  if (context?.providerType === "system") return "system";
  return context ? "webhook" : "system";
}

function getProviderTags(settings: AppSettings, providerId: string) {
  if (providerId === SYSTEM_PROVIDER_ID) {
    return [];
  }

  return settings.providers.items.find((provider) => provider.id === providerId)?.tags ?? [];
}

function describeAlertSource(context: AlertContext | undefined, providerId: string) {
  const providerName = context?.providerName ?? "Nook";
  const providerType = context?.providerType ?? "system";
  return `${providerName} (${providerType}:${providerId})`;
}

function describeTargets(deliveries: DeliveryAttempt[]) {
  return deliveries.map((delivery) => `${delivery.targetName}(${delivery.targetType})`).join(", ") || "none";
}

function describeConfiguredTargets(targets: { name: string; type: string }[]) {
  return targets.map((target) => `${target.name}(${target.type})`).join(", ") || "none";
}

async function sendConfiguredAlert(message: string, context?: AlertContext): Promise<DeliveryAttempt[]> {
  const settings = store.getSettings();
  if (!hasAnyDelivery(settings.delivery)) {
    throw new Error("No delivery target configured. Add and enable at least one delivery target first.");
  }

  const providerId = context?.providerId ?? SYSTEM_PROVIDER_ID;
  const providerTags = context?.providerTags ?? getProviderTags(settings, providerId);
  const source = getAlertSource(context);
  const sourceLabel = describeAlertSource(context, providerId);
  const eventLabel = context?.event ?? message.slice(0, 80);

  if (isDuplicateEvent(message, providerId)) {
    console.log(`[delivery] deduplicated: ${eventLabel}`);
    return [];
  }

  const route = getDeliveryRouteSummary(settings.delivery, providerId, providerTags);
  const envelope = buildNookEventEnvelope({
    source,
    providerId,
    providerName: context?.providerName ?? "Nook",
    providerType: context?.providerType ?? "system",
    providerTags,
    eventType: context?.eventType ?? eventLabel,
    eventTitle: context?.eventTitle ?? eventLabel,
    eventUrl: context?.eventUrl ?? null,
    messageHtml: message,
    actor: context?.actor ?? null,
    subject: context?.subject ?? {
      type: context?.providerType === "system" || !context ? "system" : "unknown",
      id: null,
      name: context?.providerName ?? "Nook",
      url: null,
    },
    normalizedPayload: context?.normalizedPayload ?? {},
    rawPayload: context?.rawPayload ?? null,
    occurredAt: context?.occurredAt ?? null,
    containsPii: context?.containsPii ?? false,
    rawPayloadAllowed: context?.rawPayloadAllowed ?? false,
  });
  const delivery: DeliveryMessage = { message, envelope };

  const entry = pushEvent({
    source,
    providerId,
    providerName: context?.providerName ?? "Nook",
    providerType: context?.providerType ?? "system",
    event: eventLabel,
    message,
    envelope,
    deliveries: [],
  });

  if (route.selectedTargets.length === 0) {
    console.log(`[delivery] skipped \`${eventLabel}\` from ${sourceLabel}: no selected delivery targets`);
    updateEventDeliveries(entry.id, []);
    return [];
  }

  if (route.sendableTargets.length === 0) {
    console.warn(`[delivery] skipped \`${eventLabel}\` from ${sourceLabel}: selected targets are not fully configured (${describeConfiguredTargets(route.skippedUnconfiguredTargets)})`);
    updateEventDeliveries(entry.id, []);
    return [];
  }

  try {
    const deliveries = await sendAlert(settings.delivery, delivery, providerId, providerTags);
    updateEventDeliveries(entry.id, deliveries);
    const failedDeliveries = deliveries.filter((d) => d.status === "error");
    for (const failure of failedDeliveries) {
      const target = settings.delivery.items.find((t) => t.id === failure.targetId);
      if (target) {
        enqueueRetry(entry.id, target, delivery);
      }
    }
    const successCount = deliveries.filter((delivery) => delivery.status === "success").length;
    const errorCount = deliveries.length - successCount;
    console.log(`[delivery] \`${eventLabel}\` from ${sourceLabel}: ${successCount} delivered, ${errorCount} failed, ${route.skippedByFilterCount} filtered; targets: ${describeTargets(deliveries)}`);
    if (errorCount > 0 && settings.system.events.deliveryFailure && !notifyingDeliveryFailure) {
      const failedTargets = deliveries.filter((d) => d.status === "error").map((d) => d.targetName).join(", ");
      void notifyDeliveryFailure(`<b>Delivery failure</b><br/>${errorCount} target(s) failed for <b>${eventLabel}</b>: ${failedTargets}`);
    }
    return deliveries;
  } catch (error) {
    if (error instanceof DeliveryError) {
      updateEventDeliveries(entry.id, error.attempts);
      const failedAttempts = error.attempts.filter((a) => a.status === "error");
      for (const failure of failedAttempts) {
        const target = settings.delivery.items.find((t) => t.id === failure.targetId);
        if (target) {
          enqueueRetry(entry.id, target, delivery);
        }
      }
      console.error(`[delivery] \`${eventLabel}\` from ${sourceLabel}: all selected targets failed; targets: ${describeTargets(error.attempts)}`);
    }

    throw error;
  }
}

let notifyingDeliveryFailure = false;
async function notifyDeliveryFailure(message: string) {
  notifyingDeliveryFailure = true;
  try {
    await sendConfiguredAlert(message, {
      providerId: SYSTEM_PROVIDER_ID,
      providerName: "Nook",
      providerType: "system",
      event: "delivery failure",
      eventType: "system.delivery.failure",
      eventTitle: "Delivery failure",
      subject: { type: "system", id: null, name: "Nook", url: null },
    });
  } catch {
  } finally {
    notifyingDeliveryFailure = false;
  }
}

function getNewlyEnabledProviders(previous: AppSettings, next: AppSettings) {
  const previousEnabled = new Map(previous.providers.items.map((provider) => [provider.id, provider.enabled]));
  return next.providers.items.filter((provider) => provider.enabled && previousEnabled.get(provider.id) === false);
}

function getNewlyDisabledProviders(previous: AppSettings, next: AppSettings) {
  const nextEnabled = new Map(next.providers.items.map((provider) => [provider.id, provider.enabled]));
  return previous.providers.items.filter((provider) => provider.enabled && nextEnabled.get(provider.id) === false);
}

function getAddedProviders(previous: AppSettings, next: AppSettings) {
  const previousIds = new Set(previous.providers.items.map((provider) => provider.id));
  return next.providers.items.filter((provider) => !previousIds.has(provider.id));
}

function getRemovedProviders(previous: AppSettings, next: AppSettings) {
  const nextIds = new Set(next.providers.items.map((provider) => provider.id));
  return previous.providers.items.filter((provider) => !nextIds.has(provider.id));
}

async function sendTestAlertToTarget(targetId: string) {
  const settings = store.getSettings();
  const target = settings.delivery.items.find((item) => item.id === targetId);
  if (!target) {
    throw new Error("Delivery target not found.");
  }

  const message = `🧪 <b>Nook target test</b><br/>Target <b>${target.name}</b> is configured.`;
  await sendAlertToTarget(target, {
    message,
    envelope: buildNookEventEnvelope({
      source: "test",
      providerId: SYSTEM_PROVIDER_ID,
      providerName: "Nook",
      providerType: "system",
      eventType: "target test",
      eventTitle: "Nook target test",
      messageHtml: message,
      subject: {
        type: "system",
        id: target.id,
        name: target.name,
        url: null,
      },
      normalizedPayload: {
        targetId: target.id,
        targetType: target.type,
      },
    }),
  });
}

async function testProviderConnection(providerId: string) {
  const settings = store.getSettings();
  const provider = settings.providers.items.find((item) => item.id === providerId);
  if (!provider) {
    throw new Error("Provider not found.");
  }

  if (!isImapProviderSettings(provider)) {
    throw new Error("Connection testing is only supported for IMAP providers.");
  }

  await testImapConnection(provider);
}

const monitorManager = createMonitorManager(async (message, context) => {
  try {
    await sendConfiguredAlert(message, context);
  } catch (error) {
    console.error(`[monitor] ${error instanceof Error ? error.message : String(error)}`);
  }
});
const imapManager = createImapManager(async (message, context) => {
  try {
    await sendConfiguredAlert(message, context);
  } catch (error) {
    console.error(`[imap] ${error instanceof Error ? error.message : String(error)}`);
  }
});
const rssManager = createRssManager(async (message, context) => {
  try {
    await sendConfiguredAlert(message, context);
  } catch (error) {
    console.error(`[rss] ${error instanceof Error ? error.message : String(error)}`);
  }
});

async function replayEventToTarget(eventId: string, targetId: string) {
  const entry = getEventById(eventId);
  if (!entry || !entry.envelope) {
    throw new Error("Event not found or has no envelope.");
  }

  const settings = store.getSettings();
  const target = settings.delivery.items.find((t) => t.id === targetId);
  if (!target) {
    throw new Error("Delivery target not found.");
  }

  await sendAlertToTarget(target, {
    message: entry.message ?? "",
    envelope: entry.envelope,
  });
}

function applySettings() {
  const settings = store.getSettings();
  runtimeStatus.usingPersistedSettings = store.isUsingPersistedSettings();
  runtimeStatus.lastReloadedAt = new Date().toISOString();
  monitorManager.apply(settings);
  imapManager.apply(settings);
  rssManager.apply(settings);
}

async function updateSettings(input: unknown) {
  const previousSettings = store.getSettings();
  const settings = await store.update(input);
  applySettings();
  poolServer.setSecret(settings.pool?.secret || process.env.POOL_SECRET || "pool");

  async function fireSystemEvent(message: string, event: string, eventType: string) {
    try {
      await sendConfiguredAlert(message, {
        providerId: SYSTEM_PROVIDER_ID,
        providerName: "Nook",
        providerType: "system",
        event,
        eventType,
        subject: { type: "system", id: null, name: "Nook", url: null },
      });
    } catch (error) {
      console.error(`[settings] Failed to send system event \`${event}\`: ${error instanceof Error ? error.message : String(error)}`);
    }
  }

  if (settings.system.events.providerEnabled) {
    for (const provider of getNewlyEnabledProviders(previousSettings, settings)) {
      await fireSystemEvent(
        `<b>Provider enabled</b><br/><b>${provider.name}</b> is now active.`,
        `provider enabled: ${provider.name}`,
        "system.provider.enabled",
      );
    }
  }

  if (settings.system.events.providerDisabled) {
    for (const provider of getNewlyDisabledProviders(previousSettings, settings)) {
      await fireSystemEvent(
        `<b>Provider disabled</b><br/><b>${provider.name}</b> is now inactive.`,
        `provider disabled: ${provider.name}`,
        "system.provider.disabled",
      );
    }
  }

  if (settings.system.events.providerAdded) {
    for (const provider of getAddedProviders(previousSettings, settings)) {
      await fireSystemEvent(
        `<b>Provider added</b><br/><b>${provider.name}</b> (${provider.type}) was added.`,
        `provider added: ${provider.name}`,
        "system.provider.added",
      );
    }
  }

  if (settings.system.events.providerRemoved) {
    for (const provider of getRemovedProviders(previousSettings, settings)) {
      await fireSystemEvent(
        `<b>Provider removed</b><br/><b>${provider.name}</b> (${provider.type}) was removed.`,
        `provider removed: ${provider.name}`,
        "system.provider.removed",
      );
    }
  }

  return settings;
}

applySettings();

const serverPort = store.getSettings().providers.port;
const dashboardFile = Bun.file(new URL("./dashboard/.output/public/index.html", import.meta.url));
runtimeStatus.dashboardBuilt = await dashboardFile.exists();
runtimeStatus.serverPort = serverPort;

const settings = store.getSettings();
console.log("[startup] Nook booting");
console.log(`[startup] settings path: ${settingsPath}`);
console.log(`[startup] persisted settings: ${store.isUsingPersistedSettings() ? "yes" : "no"}`);
console.log(`[startup] dashboard: ${runtimeStatus.dashboardBuilt ? "built" : "not built"}`);
console.log(`[startup] server port: ${serverPort}`);
console.log(`[startup] delivery targets: ${settings.delivery.items.length}`);
console.log(`[startup] matrix rooms: ${settings.delivery.items.filter(isMatrixDeliveryTarget).flatMap((target) => target.roomIds).join(", ") || "none configured"}`);
console.log(`[startup] providers: ${settings.providers.items.length} total`);
console.log(`[startup] websites: ${settings.providers.items.filter(isWebsiteProviderSettings).length}`);
console.log(`[startup] imap: ${settings.providers.items.filter(isImapProviderSettings).length}`);
console.log(`[startup] rss: ${settings.providers.items.filter(isRssProviderSettings).length}`);
console.log(`[startup] webhooks: ${settings.providers.items.filter(isWebhookProviderSettings).length}`);

if (hasAnyDelivery(settings.delivery) && settings.system.events.startup) {
  try {
    const deliveries = await sendConfiguredAlert(
      "<b>Nook started</b><br/>Dashboard and API ready.",
      {
        providerId: SYSTEM_PROVIDER_ID,
        providerName: "Nook",
        providerType: "system",
        event: "nook started",
      },
    );
    if (deliveries.length > 0) {
      console.log("[startup] Startup notification delivered");
    } else {
      console.log("[startup] Startup notification skipped by provider filters");
    }
  } catch (error) {
    console.error(`[startup] Failed to send startup notification: ${error instanceof Error ? error.message : String(error)}`);
  }
} else {
  console.warn("[startup] No delivery channel is configured yet. Dashboard will still start.");
}

startRetryProcessor();

const dataDir = settingsPath.includes("/") ? settingsPath.slice(0, settingsPath.lastIndexOf("/")) : ".";
const poolSecret = store.getSettings().pool?.secret || process.env.POOL_SECRET || "pool";
const poolServer = new PoolServer({ dataDir, secret: poolSecret });

Bun.serve({
  port: serverPort,
  fetch: createAppServer({
    getSettings: () => store.getSettings(),
    updateSettings,
    sendConfiguredAlert,
    sendTestAlertToTarget,
    testProviderConnection,
    replayEventToTarget,
    getRuntimeStatus: () => ({ ...runtimeStatus }),
    session,
    poolServer,
  }),
  websocket: {
    open: poolServer.handleOpen,
    message: poolServer.handleMessage,
    close: poolServer.handleClose,
  },
});

console.log(`[startup] pool: enabled (epoch=${poolServer.getStats().epoch})`);

setInterval(() => poolServer.cleanup(), 24 * 60 * 60 * 1000);
console.log(`[startup] listening on ${serverPort}`);

if (!session.password) {
  console.warn("[security] NOOK_ADMIN_PASSWORD is not set — dashboard and API are open to anyone who can reach this server");
}

console.warn("[security] rate limiting trusts x-forwarded-for — ensure your reverse proxy overwrites this header");
