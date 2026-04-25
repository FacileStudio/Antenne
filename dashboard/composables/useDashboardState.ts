import { reactive, ref } from "vue";

import type { AppSettings, RuntimeStatus } from "../types";

type SettingsResponse = {
  settings: AppSettings;
  runtime: RuntimeStatus;
  matrixConfigured: boolean;
};

const defaultSettings = (): AppSettings => ({
  delivery: {
    matrix: {
      enabled: true,
      homeserver: "",
      accessToken: "",
      roomIds: [],
    },
    discord: {
      enabled: false,
      webhookUrl: "",
    },
    webhook: {
      enabled: false,
      url: "",
      secretHeader: "x-nook-secret",
      secretValue: "",
    },
  },
  timeout: 10,
  sites: [],
  webhook: {
    enabled: true,
    port: 9090,
    providers: {
      "uptime-kuma": {
        enabled: true,
        path: "/webhook/uptime-kuma",
        secretHeader: "x-webhook-secret",
        secretValue: "",
      },
      docuseal: {
        enabled: true,
        path: "/webhook/docuseal",
        secretHeader: "x-docuseal-secret",
        secretValue: "",
      },
    },
  },
});

function cloneFromPayload(target: AppSettings, next: AppSettings) {
  target.delivery.matrix.enabled = next.delivery.matrix.enabled;
  target.delivery.matrix.homeserver = next.delivery.matrix.homeserver;
  target.delivery.matrix.accessToken = next.delivery.matrix.accessToken;
  target.delivery.matrix.roomIds = [...next.delivery.matrix.roomIds];
  target.delivery.discord.enabled = next.delivery.discord.enabled;
  target.delivery.discord.webhookUrl = next.delivery.discord.webhookUrl;
  target.delivery.webhook.enabled = next.delivery.webhook.enabled;
  target.delivery.webhook.url = next.delivery.webhook.url;
  target.delivery.webhook.secretHeader = next.delivery.webhook.secretHeader;
  target.delivery.webhook.secretValue = next.delivery.webhook.secretValue ?? "";
  target.timeout = next.timeout;
  target.sites = next.sites.map((site) => ({ ...site }));
  target.webhook.enabled = next.webhook.enabled;
  target.webhook.port = next.webhook.port;
  target.webhook.providers["uptime-kuma"] = { ...next.webhook.providers["uptime-kuma"] };
  target.webhook.providers.docuseal = { ...next.webhook.providers.docuseal };
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
  });

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
  const loading = useState("dashboard-loading", () => false);
  const saving = useState("dashboard-saving", () => false);
  const testing = useState("dashboard-testing", () => false);
  const error = useState("dashboard-error", () => "");
  const success = useState("dashboard-success", () => "");
  const loaded = useState("dashboard-loaded", () => false);

  async function loadSettings(force = false) {
    if (loaded.value && !force) {
      return;
    }

    loading.value = true;
    error.value = "";

    try {
      const payload = await apiFetch<SettingsResponse>("/api/settings");
      cloneFromPayload(settings.value, payload.settings);
      runtime.value = payload.runtime;
      matrixConfigured.value = payload.matrixConfigured;
      loaded.value = true;
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : "Failed to load settings";
    } finally {
      loading.value = false;
    }
  }

  async function saveSettings() {
    saving.value = true;
    error.value = "";
    success.value = "";

    try {
      const payload = await apiFetch<SettingsResponse>("/api/settings", {
        method: "PUT",
        body: JSON.stringify({
          ...settings.value,
          delivery: {
            ...settings.value.delivery,
            matrix: {
              ...settings.value.delivery.matrix,
              roomIds: settings.value.delivery.matrix.roomIds,
            },
          },
        }),
      });
      cloneFromPayload(settings.value, payload.settings);
      runtime.value = payload.runtime;
      matrixConfigured.value = payload.matrixConfigured;
      success.value = "Settings saved. Runtime reloaded.";
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : "Failed to save settings";
    } finally {
      saving.value = false;
    }
  }

  async function sendTestAlert() {
    testing.value = true;
    error.value = "";
    success.value = "";

    try {
      await apiFetch<{ ok: boolean }>("/api/test-alert", {
        method: "POST",
        body: JSON.stringify({}),
      });
      success.value = "Test alert sent.";
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : "Failed to send test alert";
    } finally {
      testing.value = false;
    }
  }

  async function logout() {
    await apiFetch<{ ok: boolean }>("/api/logout", {
      method: "POST",
      body: JSON.stringify({}),
    });
    window.location.href = "/dashboard";
  }

  return {
    settings: settings.value,
    runtime,
    matrixConfigured,
    loading,
    saving,
    testing,
    error,
    success,
    loadSettings,
    saveSettings,
    sendTestAlert,
    logout,
  };
}
