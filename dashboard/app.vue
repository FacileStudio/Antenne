<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { LogOut, Save, Send } from "lucide-vue-next";

import AppSidebar from "./components/AppSidebar.vue";
import RoomIdEditor from "./components/RoomIdEditor.vue";
import SectionCard from "./components/SectionCard.vue";
import SiteListEditor from "./components/SiteListEditor.vue";
import StatusGrid from "./components/StatusGrid.vue";
import type { AppSettings, RuntimeStatus, WebhookProviderId } from "./types";

const loading = ref(true);
const saving = ref(false);
const testing = ref(false);
const error = ref("");
const success = ref("");
const runtime = ref<RuntimeStatus | null>(null);
const matrixConfigured = ref(false);

const settings = reactive<AppSettings>({
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

function cloneFromPayload(next: AppSettings) {
  settings.delivery.matrix.enabled = next.delivery.matrix.enabled;
  settings.delivery.matrix.homeserver = next.delivery.matrix.homeserver;
  settings.delivery.matrix.accessToken = next.delivery.matrix.accessToken;
  settings.delivery.matrix.roomIds = [...next.delivery.matrix.roomIds];
  settings.delivery.discord.enabled = next.delivery.discord.enabled;
  settings.delivery.discord.webhookUrl = next.delivery.discord.webhookUrl;
  settings.delivery.webhook.enabled = next.delivery.webhook.enabled;
  settings.delivery.webhook.url = next.delivery.webhook.url;
  settings.delivery.webhook.secretHeader = next.delivery.webhook.secretHeader;
  settings.delivery.webhook.secretValue = next.delivery.webhook.secretValue ?? "";
  settings.timeout = next.timeout;
  settings.sites = next.sites.map((site) => ({ ...site }));
  settings.webhook.enabled = next.webhook.enabled;
  settings.webhook.port = next.webhook.port;
  settings.webhook.providers["uptime-kuma"] = { ...next.webhook.providers["uptime-kuma"] };
  settings.webhook.providers.docuseal = { ...next.webhook.providers.docuseal };
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

async function loadSettings() {
  loading.value = true;
  error.value = "";

  try {
    const payload = await apiFetch<{ settings: AppSettings; runtime: RuntimeStatus; matrixConfigured: boolean }>("/api/settings");
    cloneFromPayload(payload.settings);
    runtime.value = payload.runtime;
    matrixConfigured.value = payload.matrixConfigured;
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
    const payload = await apiFetch<{ settings: AppSettings; runtime: RuntimeStatus; matrixConfigured: boolean }>("/api/settings", {
      method: "PUT",
      body: JSON.stringify({
        ...settings,
        delivery: {
          ...settings.delivery,
          matrix: {
            ...settings.delivery.matrix,
            roomIds: settings.delivery.matrix.roomIds,
          },
        },
      }),
    });
    cloneFromPayload(payload.settings);
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

const sections = [
  { id: "delivery", label: "Delivery" },
  { id: "webhooks", label: "Webhooks" },
  { id: "sites", label: "Sites" },
  { id: "info", label: "Info" },
] as const;

const sidebarInfoItems = computed(() => [
  {
    label: "Delivery",
    value: [
      settings.delivery.matrix.enabled && settings.delivery.matrix.roomIds.length > 0 ? "Matrix" : null,
      settings.delivery.discord.enabled && settings.delivery.discord.webhookUrl ? "Discord" : null,
      settings.delivery.webhook.enabled && settings.delivery.webhook.url ? "HTTP" : null,
    ].filter(Boolean).join(", ") || "Not configured",
  },
  {
    label: "Sites",
    value: `${settings.sites.length}`,
  },
  {
    label: "Status",
    value: runtime.value?.usingPersistedSettings ? "Saved settings" : "Defaults",
  },
]);

const infoItems = computed(() => [
  {
    label: "Server port",
    value: runtime.value?.serverPort ? String(runtime.value.serverPort) : "Unknown",
  },
  {
    label: "Settings file",
    value: runtime.value?.settingsPath ?? "Unknown",
  },
  {
    label: "Source",
    value: runtime.value?.usingPersistedSettings ? "Dashboard override" : "Env/YAML defaults",
  },
  {
    label: "Reloaded",
    value: runtime.value?.lastReloadedAt ?? "Unknown",
  },
]);

const providerCards = computed(() => [
  {
    id: "uptime-kuma" as const satisfies WebhookProviderId,
    label: "Uptime Kuma",
  },
  {
    id: "docuseal" as const satisfies WebhookProviderId,
    label: "DocuSeal",
  },
]);

onMounted(() => {
  void loadSettings();
});
</script>

<template>
  <main class="shell">
    <header class="topbar">
      <div class="topbar-copy">
        <p class="eyebrow">Nook</p>
        <h1>Settings</h1>
      </div>

      <div class="topbar-actions">
        <button class="secondary action-button" type="button" :disabled="testing || loading" @click="sendTestAlert">
          <Send :size="16" aria-hidden="true" />
          <span>{{ testing ? "Sending..." : "Send test alert" }}</span>
        </button>
        <button class="primary action-button" type="button" :disabled="saving || loading" @click="saveSettings">
          <Save :size="16" aria-hidden="true" />
          <span>{{ saving ? "Saving..." : "Save settings" }}</span>
        </button>
      </div>
    </header>

    <p v-if="error" class="banner error">{{ error }}</p>
    <p v-if="success" class="banner success">{{ success }}</p>

    <section v-if="loading" class="panel loading-card">
      Loading dashboard...
    </section>

    <template v-else>
      <section class="layout">
        <AppSidebar :sections="sections" :info-items="sidebarInfoItems">
          <template #footer>
            <button class="ghost action-button sidebar-logout" type="button" @click="logout">
              <LogOut :size="16" aria-hidden="true" />
              <span>Logout</span>
            </button>
          </template>
        </AppSidebar>

        <div class="content">
          <SectionCard id="delivery" eyebrow="Delivery" title="Targets">
            <div class="delivery-grid">
              <article class="panel nested-panel">
                <div class="card-header">
                  <h3>Matrix</h3>
                  <label class="toggle">
                    <input v-model="settings.delivery.matrix.enabled" type="checkbox">
                    <span>Enabled</span>
                  </label>
                </div>

                <div class="field-list">
                  <label class="field">
                    <span>Homeserver</span>
                    <input v-model="settings.delivery.matrix.homeserver" type="url" placeholder="https://matrix.example.com">
                  </label>

                  <label class="field">
                    <span>Access token</span>
                    <input v-model="settings.delivery.matrix.accessToken" type="password" placeholder="syt_xxx">
                  </label>

                  <label class="field">
                    <span>Room IDs</span>
                    <RoomIdEditor v-model="settings.delivery.matrix.roomIds" />
                  </label>
                </div>
              </article>

              <article class="panel nested-panel">
                <div class="card-header">
                  <h3>Discord webhook</h3>
                  <label class="toggle">
                    <input v-model="settings.delivery.discord.enabled" type="checkbox">
                    <span>Enabled</span>
                  </label>
                </div>

                <div class="field-list">
                  <label class="field">
                    <span>Webhook URL</span>
                    <input v-model="settings.delivery.discord.webhookUrl" type="url" placeholder="https://discord.com/api/webhooks/...">
                  </label>
                </div>
              </article>

              <article class="panel nested-panel">
                <div class="card-header">
                  <h3>HTTP(S) webhook</h3>
                  <label class="toggle">
                    <input v-model="settings.delivery.webhook.enabled" type="checkbox">
                    <span>Enabled</span>
                  </label>
                </div>

                <div class="field-list">
                  <label class="field">
                    <span>Webhook URL</span>
                    <input v-model="settings.delivery.webhook.url" type="url" placeholder="https://example.com/hooks/nook">
                  </label>

                  <label class="field">
                    <span>Secret header</span>
                    <input v-model="settings.delivery.webhook.secretHeader" type="text" placeholder="x-nook-secret">
                  </label>

                  <label class="field">
                    <span>Secret value</span>
                    <input v-model="settings.delivery.webhook.secretValue" type="password">
                  </label>
                </div>
              </article>
            </div>

            <label class="field timeout-field">
              <span>Timeout in seconds</span>
              <input v-model.number="settings.timeout" type="number" min="1" step="1">
            </label>
          </SectionCard>

          <SectionCard id="webhooks" eyebrow="Webhooks" title="Providers">
            <div class="provider-grid">
              <article
                v-for="provider in providerCards"
                :key="provider.id"
                class="panel nested-panel"
              >
                <div class="card-header">
                  <h3>{{ provider.label }}</h3>
                  <label class="toggle">
                    <input v-model="settings.webhook.providers[provider.id].enabled" type="checkbox">
                    <span>Enabled</span>
                  </label>
                </div>

                <div class="field-list">
                  <label class="field">
                    <span>Path</span>
                    <input v-model="settings.webhook.providers[provider.id].path" type="text">
                  </label>

                  <label class="field">
                    <span>Secret header</span>
                    <input v-model="settings.webhook.providers[provider.id].secretHeader" type="text">
                  </label>

                  <label class="field">
                    <span>Secret value</span>
                    <input v-model="settings.webhook.providers[provider.id].secretValue" type="password">
                  </label>
                </div>
              </article>
            </div>
          </SectionCard>

          <SectionCard id="sites" eyebrow="Sites" title="Monitoring">
            <SiteListEditor
              v-model="settings.sites"
              @error="error = $event"
              @clear-error="error = ''"
            />
          </SectionCard>

          <SectionCard id="info" eyebrow="Info" title="Runtime">
            <StatusGrid :items="infoItems" />
          </SectionCard>
        </div>
      </section>
    </template>
  </main>
</template>

<style scoped>
.shell {
  width: min(1200px, calc(100vw - 2rem));
  margin: 0 auto;
  padding: var(--space-6) 0 var(--space-7);
}

.topbar {
  display: flex;
  justify-content: space-between;
  gap: var(--space-4);
  align-items: flex-end;
  margin-bottom: var(--space-5);
}

.topbar-copy .eyebrow {
  margin: 0 0 0.35rem;
  font-size: 0.75rem;
  color: var(--nook-muted);
}

.topbar-copy h1 {
  margin: 0;
}

.topbar-actions {
  display: flex;
  gap: var(--space-3);
  flex-wrap: wrap;
  justify-content: flex-end;
}

.layout {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  gap: var(--space-5);
  align-items: start;
}

.content {
  display: grid;
  gap: var(--space-5);
  min-width: 0;
}

.delivery-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--space-4);
}

.provider-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--space-4);
}

.nested-panel {
  display: grid;
  gap: var(--space-4);
  padding: var(--space-4);
  border-radius: 1.25rem;
  background: var(--nook-surface-strong);
}

.card-header {
  display: flex;
  justify-content: space-between;
  gap: var(--space-3);
  align-items: center;
}

.card-header h3 {
  margin: 0;
}

.field-list {
  display: grid;
  gap: var(--space-3);
}

.timeout-field {
  max-width: 18rem;
}

.toggle {
  display: inline-flex;
  align-items: center;
  gap: 0.6rem;
  color: var(--nook-muted);
}

.action-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
}

.sidebar-logout {
  width: 100%;
}

.banner {
  margin: 0 0 var(--space-4);
  padding: var(--space-3) var(--space-4);
  border-radius: 1rem;
  border: 1px solid var(--nook-border);
}

.banner.error,
.banner.success,
.loading-card {
  color: var(--nook-text);
}

@media (max-width: 1120px) {
  .delivery-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 960px) {
  .layout,
  .provider-grid {
    grid-template-columns: 1fr;
  }

  .topbar {
    flex-direction: column;
    align-items: stretch;
  }

  .topbar-actions {
    justify-content: flex-start;
  }
}
</style>
