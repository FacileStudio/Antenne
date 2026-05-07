import { reactive } from "vue";

import {
  createDefaultDeliveryTarget,
  createDefaultProvider,
  SYSTEM_PROVIDER_ID,
  type AppSettings,
  type EventLogEntry,
  type EventLogStats,
  type RuntimeStatus,
} from "../types";

type SettingsResponse = {
  settings: AppSettings;
  runtime: RuntimeStatus;
  matrixConfigured: boolean;
};

type SessionResponse = {
  authenticated: boolean;
  passwordRequired: boolean;
  username: string;
};

type Toast = {
  id: number;
  type: "success" | "error";
  message: string;
};

let nextToastId = 1;
const toastTimers = new Map<number, ReturnType<typeof setTimeout>>();

const defaultSettings = (): AppSettings => {
  const providers = [
    createDefaultProvider("website"),
    createDefaultProvider("uptime-kuma"),
    createDefaultProvider("docuseal"),
  ];
  const providerIds = [SYSTEM_PROVIDER_ID, ...providers.map((provider) => provider.id)];

  return {
    delivery: {
      items: [
        { ...createDefaultDeliveryTarget("matrix"), providerIds },
        { ...createDefaultDeliveryTarget("discord"), providerIds },
        { ...createDefaultDeliveryTarget("webhook"), providerIds },
      ],
    },
    system: {
      events: {
        startup: true,
        providerEnabled: false,
      },
    },
    timeout: 10,
    providers: {
      port: 9090,
      items: providers,
    },
  };
};

function cloneFromPayload(target: AppSettings, next: AppSettings) {
  target.delivery.items = next.delivery.items.map((item) => ({ ...item }));
  target.system.events.startup = next.system.events.startup;
  target.system.events.providerEnabled = next.system.events.providerEnabled;
  target.timeout = next.timeout;
  target.providers.port = next.providers.port;
  target.providers.items = next.providers.items.map((provider) => ({ ...provider }));
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
  });

  if (res.status === 401) {
    if (import.meta.client) {
      window.location.href = "/dashboard";
    }

    throw new Error("Unauthorized");
  }

  if (!res.ok) {
    const payload = await res.json().catch(() => ({ error: `HTTP ${res.status}` })) as { error?: string };
    throw new Error(payload.error ?? `HTTP ${res.status}`);
  }

  return await res.json() as T;
}

export function useDashboardState() {
  const settings = useState("dashboard-settings", () => reactive(defaultSettings()));
  const runtime = useState<RuntimeStatus | null>("dashboard-runtime", () => null);
  const matrixConfigured = useState("dashboard-matrix-configured", () => false);
  const passwordRequired = useState("dashboard-password-required", () => false);
  const loading = useState("dashboard-loading", () => true);
  const saving = useState("dashboard-saving", () => false);
  const testing = useState("dashboard-testing", () => false);
  const error = useState("dashboard-error", () => "");
  const loaded = useState("dashboard-loaded", () => false);
  const toasts = useState<Toast[]>("dashboard-toasts", () => []);

  function dismissToast(id: number) {
    toasts.value = toasts.value.filter((toast) => toast.id !== id);

    const timer = toastTimers.get(id);
    if (timer) {
      clearTimeout(timer);
      toastTimers.delete(id);
    }
  }

  function pushToast(type: Toast["type"], message: string, duration = 2600) {
    const id = nextToastId++;
    toasts.value = [...toasts.value, { id, type, message }];

    const timer = setTimeout(() => {
      dismissToast(id);
    }, duration);

    toastTimers.set(id, timer);
  }

  function notifySuccess(message: string) {
    pushToast("success", message);
  }

  function notifyError(message: string) {
    pushToast("error", message, 3200);
  }

  async function loadSettings(force = false) {
    if (loaded.value && !force) {
      return;
    }

    loading.value = true;
    error.value = "";

    try {
      const [payload, session] = await Promise.all([
        apiFetch<SettingsResponse>("/api/settings"),
        apiFetch<SessionResponse>("/api/session"),
      ]);
      cloneFromPayload(settings.value, payload.settings);
      runtime.value = payload.runtime;
      matrixConfigured.value = payload.matrixConfigured;
      passwordRequired.value = session.passwordRequired;
      loaded.value = true;
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : "Failed to load settings";
      notifyError(error.value);
    } finally {
      loading.value = false;
    }
  }

  async function saveSettings() {
    saving.value = true;
    error.value = "";

    try {
      const payload = await apiFetch<SettingsResponse>("/api/settings", {
        method: "PUT",
        body: JSON.stringify(settings.value),
      });
      cloneFromPayload(settings.value, payload.settings);
      runtime.value = payload.runtime;
      matrixConfigured.value = payload.matrixConfigured;
      notifySuccess("Settings saved. Runtime reloaded.");
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : "Failed to save settings";
      notifyError(error.value);
    } finally {
      saving.value = false;
    }
  }

  async function sendTestAlert() {
    testing.value = true;
    error.value = "";

    try {
      await apiFetch<{ ok: boolean }>("/api/test-alert", {
        method: "POST",
        body: JSON.stringify({}),
      });
      notifySuccess("Test alert sent.");
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : "Failed to send test alert";
      notifyError(error.value);
    } finally {
      testing.value = false;
    }
  }

  async function logout() {
    try {
      await apiFetch<{ ok: boolean }>("/api/logout", {
        method: "POST",
        body: JSON.stringify({}),
      });
    } finally {
      window.location.href = "/dashboard";
    }
  }

  async function testDeliveryTarget(targetId: string) {
    await apiFetch<{ ok: boolean }>("/api/test-delivery-target", {
      method: "POST",
      body: JSON.stringify({ targetId }),
    });
  }

  async function testProviderConnection(providerId: string) {
    await apiFetch<{ ok: boolean }>("/api/test-provider-connection", {
      method: "POST",
      body: JSON.stringify({ providerId }),
    });
  }

  function exportConfig() {
    const data = JSON.stringify(settings.value, null, 2);
    const blob = new Blob([data], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "nook-config.json";
    a.click();
    URL.revokeObjectURL(url);
  }

  async function importConfig(file: File) {
    const text = await file.text();
    let parsed: unknown;
    try {
      parsed = JSON.parse(text);
    } catch {
      throw new Error("Invalid JSON file.");
    }

    saving.value = true;
    error.value = "";
    try {
      const payload = await apiFetch<SettingsResponse>("/api/settings", {
        method: "PUT",
        body: JSON.stringify(parsed),
      });
      cloneFromPayload(settings.value, payload.settings);
      runtime.value = payload.runtime;
      matrixConfigured.value = payload.matrixConfigured;
      notifySuccess("Config imported and saved.");
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : "Failed to import config";
      notifyError(error.value);
      throw cause;
    } finally {
      saving.value = false;
    }
  }

  async function loadEvents(options?: { providerId?: string; targetId?: string; limit?: number }) {
    let path = "/api/events";
    const params = new URLSearchParams();
    if (options?.limit) params.set("limit", String(options.limit));

    if (options?.providerId) {
      path = `/api/events/provider/${options.providerId}`;
    } else if (options?.targetId) {
      path = `/api/events/target/${options.targetId}`;
    }

    const query = params.toString();
    return await apiFetch<{ events: EventLogEntry[]; stats: EventLogStats }>(`${path}${query ? `?${query}` : ""}`);
  }

  return {
    settings: settings.value,
    runtime,
    matrixConfigured,
    passwordRequired,
    loading,
    saving,
    testing,
    error,
    toasts,
    notifyError,
    notifySuccess,
    dismissToast,
    loadSettings,
    saveSettings,
    exportConfig,
    importConfig,
    sendTestAlert,
    testDeliveryTarget,
    testProviderConnection,
    loadEvents,
    logout,
  };
}
