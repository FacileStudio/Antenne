import { createMonitorManager } from "./src/monitor";
import { hasMatrixConfig, sendAlert } from "./src/matrix";
import { createAppServer, createDashboardSessionConfig } from "./src/server";
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
  if (!hasMatrixConfig(settings.matrix)) {
    throw new Error("Matrix config incomplete. Fill homeserver, access token, and room IDs first.");
  }

  await sendAlert(settings.matrix, message);
}

const monitorManager = createMonitorManager(async (message) => {
  try {
    await sendConfiguredAlert(message);
  } catch (error) {
    console.error(`[matrix] ${error instanceof Error ? error.message : String(error)}`);
  }
});

function applySettings() {
  const settings = store.getSettings();
  runtimeStatus.usingPersistedSettings = store.isUsingPersistedSettings();
  runtimeStatus.lastReloadedAt = new Date().toISOString();
  monitorManager.apply(settings);
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
console.log(`[startup] room(s): ${settings.matrix.roomIds.join(", ") || "none configured"}`);
console.log(`[startup] monitoring ${settings.sites.length} site(s)`);
console.log(`[startup] webhook: ${settings.webhook.enabled ? "enabled" : "disabled"}`);

if (hasMatrixConfig(settings.matrix)) {
  try {
    await sendConfiguredAlert("🟢 <b>Nook started</b><br/>Dashboard and API ready.");
    console.log("[startup] Matrix startup message sent");
  } catch (error) {
    console.error(`[startup] Failed to send Matrix startup message: ${error instanceof Error ? error.message : String(error)}`);
  }
} else {
  console.warn("[startup] Matrix is not configured yet. Dashboard will still start.");
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
