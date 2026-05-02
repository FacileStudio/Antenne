import { loadConfig } from "./config";
import { loadDeliveryConfigFromEnv } from "./delivery";
import { normalizeAppSettings } from "./settings-normalize";
import {
  createInitialRuntimeStatus,
  getServerPort,
  getSettingsPath,
  loadPersistedSettings,
  saveSettings,
} from "./settings-storage";
import type { AppSettings } from "./types";

function cloneSettings(settings: AppSettings): AppSettings {
  return structuredClone(settings);
}

export { createInitialRuntimeStatus, getServerPort, getSettingsPath, loadPersistedSettings };

export async function loadDefaultSettings(): Promise<AppSettings> {
  const { config } = await loadConfig();
  return {
    delivery: loadDeliveryConfigFromEnv(),
    system: {
      events: {
        startup: true,
        providerEnabled: false,
      },
    },
    timeout: config.timeout,
    providers: config.providers,
  };
}

export class SettingsStore {
  private settings: AppSettings;
  private defaults: AppSettings;
  private readonly path: string;
  private usingPersistedSettings: boolean;

  constructor(defaults: AppSettings, initialSettings: AppSettings, path: string, usingPersistedSettings: boolean) {
    this.defaults = cloneSettings(defaults);
    this.settings = normalizeAppSettings(initialSettings, this.defaults);
    this.path = path;
    this.usingPersistedSettings = usingPersistedSettings;
  }

  getPath() {
    return this.path;
  }

  isUsingPersistedSettings() {
    return this.usingPersistedSettings;
  }

  getDefaults() {
    return cloneSettings(this.defaults);
  }

  getSettings() {
    return cloneSettings(this.settings);
  }

  async update(input: unknown) {
    const nextSettings = normalizeAppSettings(input, this.defaults);
    await saveSettings(this.path, nextSettings);
    this.settings = cloneSettings(nextSettings);
    this.usingPersistedSettings = true;
    return this.getSettings();
  }
}
