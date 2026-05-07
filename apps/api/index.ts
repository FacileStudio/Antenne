import { createImapManager, testImapConnection } from "./src/imap";
import { createMonitorManager } from "./src/monitor";
import { DeliveryError, getDeliveryRouteSummary, hasAnyDelivery, sendAlert, sendAlertToTarget } from "./src/delivery";
import { isMatrixDeliveryTarget } from "./src/delivery-config";
import { createAppServer, createDashboardSessionConfig } from "./src/server";
import { initEventLog, pushEvent, updateEventDeliveries, type DeliveryAttempt } from "./src/event-log";
import { isImapProviderSettings, isWebhookProviderSettings, isWebsiteProviderSettings } from "./src/webhooks/types";
import {
  createInitialRuntimeStatus,
  getSettingsPath,
  loadDefaultSettings,
  loadPersistedSettings,
  SettingsStore,
} from "./src/settings";
import { SYSTEM_PROVIDER_ID, type AppSettings } from "./src/types";

const settingsPath = getSettingsPath();
const eventLogPath = settingsPath.replace(/\.json$/, "-events.json");
await initEventLog(eventLogPath);
const defaults = await loadDefaultSettings();
const persistedSettings = await loadPersistedSettings(settingsPath);
const initialSettings = persistedSettings ?? defaults;
const store = new SettingsStore(defaults, initialSettings, settingsPath, Boolean(persistedSettings));
const runtimeStatus = createInitialRuntimeStatus(settingsPath, Boolean(persistedSettings));
const session = createDashboardSessionConfig();

type AlertContext = {
  providerId: string;
  providerName: string;
  providerType: string;
  event: string;
};

function getAlertSource(context?: AlertContext) {
  if (context?.providerType === "website") return "website";
  if (context?.providerType === "imap") return "imap";
  if (context?.providerType === "system") return "system";
  return context ? "webhook" : "system";
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
  const route = getDeliveryRouteSummary(settings.delivery, providerId);
  const source = getAlertSource(context);
  const sourceLabel = describeAlertSource(context, providerId);
  const eventLabel = context?.event ?? message.slice(0, 80);

  const entry = pushEvent({
    source,
    providerId,
    providerName: context?.providerName ?? "Nook",
    providerType: context?.providerType ?? "system",
    event: eventLabel,
    message,
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
    const deliveries = await sendAlert(settings.delivery, message, providerId);
    updateEventDeliveries(entry.id, deliveries);
    const successCount = deliveries.filter((delivery) => delivery.status === "success").length;
    const errorCount = deliveries.length - successCount;
    console.log(`[delivery] \`${eventLabel}\` from ${sourceLabel}: ${successCount} delivered, ${errorCount} failed, ${route.skippedByFilterCount} filtered; targets: ${describeTargets(deliveries)}`);
    return deliveries;
  } catch (error) {
    if (error instanceof DeliveryError) {
      updateEventDeliveries(entry.id, error.attempts);
      console.error(`[delivery] \`${eventLabel}\` from ${sourceLabel}: all selected targets failed; targets: ${describeTargets(error.attempts)}`);
    }

    throw error;
  }
}

function getNewlyEnabledProviders(previous: AppSettings, next: AppSettings) {
  const previousEnabled = new Map(previous.providers.items.map((provider) => [provider.id, provider.enabled]));
  return next.providers.items.filter((provider) => provider.enabled && previousEnabled.get(provider.id) === false);
}

async function sendTestAlertToTarget(targetId: string) {
  const settings = store.getSettings();
  const target = settings.delivery.items.find((item) => item.id === targetId);
  if (!target) {
    throw new Error("Delivery target not found.");
  }

  await sendAlertToTarget(target, `🧪 <b>Nook target test</b><br/>Target <b>${target.name}</b> is configured.`);
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

function applySettings() {
  const settings = store.getSettings();
  runtimeStatus.usingPersistedSettings = store.isUsingPersistedSettings();
  runtimeStatus.lastReloadedAt = new Date().toISOString();
  monitorManager.apply(settings);
  imapManager.apply(settings);
}

async function updateSettings(input: unknown) {
  const previousSettings = store.getSettings();
  const settings = await store.update(input);
  applySettings();

  if (settings.system.events.providerEnabled) {
    const newlyEnabledProviders = getNewlyEnabledProviders(previousSettings, settings);
    for (const provider of newlyEnabledProviders) {
      try {
        await sendConfiguredAlert(
          `<b>Provider enabled</b><br/><b>${provider.name}</b> is now active.`,
          {
            providerId: SYSTEM_PROVIDER_ID,
            providerName: "Nook",
            providerType: "system",
            event: `provider enabled: ${provider.name}`,
          },
        );
      } catch (error) {
        console.error(`[settings] Failed to send provider-enabled notification: ${error instanceof Error ? error.message : String(error)}`);
      }
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

Bun.serve({
  port: serverPort,
  fetch: createAppServer({
    getSettings: () => store.getSettings(),
    updateSettings,
    sendConfiguredAlert,
    sendTestAlertToTarget,
    testProviderConnection,
    getRuntimeStatus: () => ({ ...runtimeStatus }),
    session,
  }),
});

console.log(`[startup] listening on ${serverPort}`);
