import { mkdir } from "node:fs/promises";

import type { AppSettings, RuntimeStatus } from "./types";

const defaultSettingsPath = process.env.NOOK_SETTINGS_PATH ?? "/data/nook.json";
const defaultServerPort = Number(process.env.PORT ?? process.env.WEBHOOK_PORT ?? 9090);

export async function loadPersistedSettings(path = defaultSettingsPath): Promise<AppSettings | null> {
  const file = Bun.file(path);
  if (!(await file.exists())) {
    return null;
  }

  const text = await file.text();
  if (!text.trim()) {
    return null;
  }

  return JSON.parse(text) as AppSettings;
}

export async function saveSettings(path: string, settings: AppSettings) {
  const directory = path.includes("/") ? path.slice(0, path.lastIndexOf("/")) : ".";
  await mkdir(directory, { recursive: true });
  await Bun.write(path, `${JSON.stringify(settings, null, 2)}\n`);
}

export function createInitialRuntimeStatus(settingsPath: string, usingPersistedSettings: boolean): RuntimeStatus {
  return {
    serverPort: defaultServerPort,
    settingsPath,
    usingPersistedSettings,
    dashboardBuilt: false,
    lastReloadedAt: new Date().toISOString(),
  };
}

export function getSettingsPath() {
  return defaultSettingsPath;
}

export function getServerPort() {
  return defaultServerPort;
}
