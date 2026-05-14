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

function envPositiveNumber(name: string, fallback: number) {
  const rawValue = process.env[name];
  if (!rawValue) {
    return fallback;
  }

  const value = Number(rawValue);
  return Number.isFinite(value) && value > 0 ? value : fallback;
}

export async function loadDefaultSettings(): Promise<AppSettings> {
  return {
    delivery: { items: [] },
    system: {
      events: {
        startup: true,
        providerEnabled: false,
        providerDisabled: false,
        providerAdded: false,
        providerRemoved: false,
        deliveryFailure: false,
      },
    },
    timeout: envPositiveNumber("TIMEOUT", 10),
    providers: {
      port: getServerPort(),
      items: [],
    },
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
