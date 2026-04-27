import { createImapManager, testImapConnection } from "./src/imap";
import { createMonitorManager } from "./src/monitor";
import { hasAnyDelivery, sendAlert, sendAlertToTarget } from "./src/delivery";
import { isMatrixDeliveryTarget } from "./src/delivery-config";
import { createAppServer, createDashboardSessionConfig } from "./src/server";
import { initEventLog, pushEvent, updateEventDeliveries } from "./src/event-log";
import { isImapProviderSettings, isWebhookProviderSettings, isWebsiteProviderSettings } from "./src/webhooks/types";
import {
  createInitialRuntimeStatus,
  getServerPort,
  getSettingsPath,
  loadDefaultSettings,
  loadPersistedSettings,
  SettingsStore,
} from "./src/settings";

const settingsPath = getSettingsPath();
const eventLogPath = settingsPath.replace(/\.json$/, "-events.json");
await initEventLog(eventLogPath);
const defaults = await loadDefaultSettings();
const persistedSettings = await loadPersistedSettings(settingsPath);
const initialSettings = persistedSettings ?? defaults;
const store = new SettingsStore(defaults, initialSettings, settingsPath, Boolean(persistedSettings));
const runtimeStatus = createInitialRuntimeStatus(settingsPath, Boolean(persistedSettings));
const session = createDashboardSessionConfig();

async function sendConfiguredAlert(message: string, context?: { providerId: string; providerName: string; providerType: string; event: string }) {
  const settings = store.getSettings();
  if (!hasAnyDelivery(settings.delivery)) {
    throw new Error("No delivery channel configured. Fill Matrix, Discord, or HTTP webhook settings first.");
  }

  const source = (context?.providerType === "website" ? "website" : context?.providerType === "imap" ? "imap" : context ? "webhook" : "system") as "webhook" | "website" | "imap" | "system";

  const entry = pushEvent({
    source,
    providerId: context?.providerId ?? null,
    providerName: context?.providerName ?? "system",
    providerType: context?.providerType ?? "system",
    event: context?.event ?? message.slice(0, 80),
    message,
    deliveries: [],
  });

  const deliveries = await sendAlert(settings.delivery, message, context?.providerId);
  updateEventDeliveries(entry.id, deliveries);
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
  const settings = await store.update(input);
  applySettings();
  return settings;
}

applySettings();

const serverPort = getServerPort();
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

if (hasAnyDelivery(settings.delivery)) {
  try {
    await sendConfiguredAlert("🟢 <b>Nook started</b><br/>Dashboard and API ready.");
    console.log("[startup] Startup notification sent");
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
