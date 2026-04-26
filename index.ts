import { createImapManager } from "./src/imap";
import { createMonitorManager } from "./src/monitor";
import { hasAnyDelivery, hasMatrixDelivery, sendAlert } from "./src/delivery";
import { createAppServer, createDashboardSessionConfig } from "./src/server";
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
const defaults = await loadDefaultSettings();
const persistedSettings = await loadPersistedSettings(settingsPath);
const initialSettings = persistedSettings ?? defaults;
const store = new SettingsStore(defaults, initialSettings, settingsPath, Boolean(persistedSettings));
const runtimeStatus = createInitialRuntimeStatus(settingsPath, Boolean(persistedSettings));
const session = createDashboardSessionConfig();

async function sendConfiguredAlert(message: string) {
  const settings = store.getSettings();
  if (!hasAnyDelivery(settings.delivery)) {
    throw new Error("No delivery channel configured. Fill Matrix, Discord, or HTTP webhook settings first.");
  }

  await sendAlert(settings.delivery, message);
}

const monitorManager = createMonitorManager(async (message) => {
  try {
    await sendConfiguredAlert(message);
  } catch (error) {
    console.error(`[matrix] ${error instanceof Error ? error.message : String(error)}`);
  }
});
const imapManager = createImapManager(async (message) => {
  try {
    await sendConfiguredAlert(message);
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
console.log(`[startup] room(s): ${settings.delivery.matrix.roomIds.join(", ") || "none configured"}`);
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
    getRuntimeStatus: () => ({ ...runtimeStatus }),
    session,
  }),
});

console.log(`[startup] listening on ${serverPort}`);
