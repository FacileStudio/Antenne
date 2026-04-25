<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";

type WebhookProviderId = "uptime-kuma" | "docuseal";

interface SiteConfig {
  name: string;
  url: string;
  interval: number;
}

interface MatrixConfig {
  enabled: boolean;
  homeserver: string;
  accessToken: string;
  roomIds: string[];
}

interface DiscordDeliveryConfig {
  enabled: boolean;
  webhookUrl: string;
}

interface HttpDeliveryConfig {
  enabled: boolean;
  url: string;
  secretHeader: string | null;
  secretValue: string | null;
}

interface WebhookProviderSettings {
  enabled: boolean;
  path: string;
  secretHeader: string | null;
  secretValue: string | null;
}

interface AppSettings {
  delivery: {
    matrix: MatrixConfig;
    discord: DiscordDeliveryConfig;
    webhook: HttpDeliveryConfig;
  };
  timeout: number;
  sites: SiteConfig[];
  webhook: {
    enabled: boolean;
    port: number;
    providers: Record<WebhookProviderId, WebhookProviderSettings>;
  };
}

interface RuntimeStatus {
  serverPort: number;
  settingsPath: string;
  usingPersistedSettings: boolean;
  dashboardBuilt: boolean;
  lastReloadedAt: string;
}

const loading = ref(true);
const saving = ref(false);
const testing = ref(false);
const error = ref("");
const success = ref("");
const roomIdDraft = ref("");
const runtime = ref<RuntimeStatus | null>(null);
const matrixConfigured = ref(false);
const newSite = reactive<SiteConfig>({
  name: "",
  url: "https://",
  interval: 60,
});
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
    success.value = "Settings saved. Runtime reloaded."
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
    success.value = "Test alert sent."
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

function resetNewSite() {
  newSite.name = "";
  newSite.url = "https://";
  newSite.interval = 60;
}

function addSiteFromDraft() {
  const url = newSite.url.trim();
  if (!url || url === "https://") {
    error.value = "Site URL is required.";
    success.value = "";
    return;
  }

  settings.sites.push({
    name: newSite.name.trim(),
    url,
    interval: newSite.interval,
  });
  resetNewSite();
  error.value = "";
}

function addBlankSite() {
  settings.sites.push({
    name: "",
    url: "https://",
    interval: 60,
  });
}

function removeSite(index: number) {
  settings.sites.splice(index, 1);
}

function moveSite(index: number, direction: -1 | 1) {
  const targetIndex = index + direction;
  if (targetIndex < 0 || targetIndex >= settings.sites.length) {
    return;
  }

  const [site] = settings.sites.splice(index, 1);
  settings.sites.splice(targetIndex, 0, site);
}

function duplicateSite(index: number) {
  const site = settings.sites[index];
  if (!site) {
    return;
  }

  settings.sites.splice(index + 1, 0, {
    ...site,
    name: site.name ? `${site.name} copy` : "",
  });
}

function addRoomId() {
  const roomId = roomIdDraft.value.trim();
  if (!roomId) {
    return;
  }

  if (!settings.delivery.matrix.roomIds.includes(roomId)) {
    settings.delivery.matrix.roomIds.push(roomId);
  }

  roomIdDraft.value = "";
}

function removeRoomId(index: number) {
  settings.delivery.matrix.roomIds.splice(index, 1);
}

const providerCards = computed(() => [
  {
    id: "uptime-kuma" as const,
    label: "Uptime Kuma",
  },
  {
    id: "docuseal" as const,
    label: "DocuSeal",
  },
]);

const sections = [
  { id: "overview", label: "Overview" },
  { id: "delivery", label: "Delivery" },
  { id: "webhooks", label: "Webhooks" },
  { id: "sites", label: "Sites" },
] as const;

onMounted(() => {
  void loadSettings();
});
</script>

<template>
  <main class="shell">
    <section class="hero">
      <div>
        <p class="eyebrow">Nook</p>
        <h1>Settings</h1>
      </div>

      <div class="hero-actions">
        <button class="ghost" type="button" @click="logout">Logout</button>
        <button class="secondary" type="button" :disabled="testing || loading" @click="sendTestAlert">
          {{ testing ? "Sending..." : "Send test alert" }}
        </button>
        <button class="primary" type="button" :disabled="saving || loading" @click="saveSettings">
          {{ saving ? "Saving..." : "Save settings" }}
        </button>
      </div>
    </section>

    <section v-if="runtime" class="status-grid">
      <article class="status-card">
        <span>Server port</span>
        <strong>{{ runtime.serverPort }}</strong>
      </article>
      <article class="status-card">
        <span>Settings file</span>
        <strong>{{ runtime.settingsPath }}</strong>
      </article>
      <article class="status-card">
        <span>Source</span>
        <strong>{{ runtime.usingPersistedSettings ? "Dashboard override" : "Env/YAML defaults" }}</strong>
      </article>
      <article class="status-card">
        <span>Matrix</span>
        <strong>{{ matrixConfigured ? "Configured" : "Missing pieces" }}</strong>
      </article>
    </section>

    <p v-if="error" class="banner error">{{ error }}</p>
    <p v-if="success" class="banner success">{{ success }}</p>

    <section v-if="loading" class="loading-card">
      Loading dashboard...
    </section>

    <template v-else>
      <section class="layout">
        <aside class="sidebar panel">
          <nav class="sidebar-nav" aria-label="Settings sections">
            <a
              v-for="section in sections"
              :key="section.id"
              class="sidebar-link"
              :href="`#${section.id}`"
            >
              {{ section.label }}
            </a>
          </nav>

          <div class="sidebar-meta">
            <div>
              <span>Source</span>
              <strong>{{ runtime?.usingPersistedSettings ? "Dashboard override" : "Env/YAML defaults" }}</strong>
            </div>
            <div>
              <span>Matrix</span>
              <strong>{{ matrixConfigured ? "Configured" : "Missing pieces" }}</strong>
            </div>
            <div>
              <span>Discord</span>
              <strong>{{ settings.delivery.discord.enabled && settings.delivery.discord.webhookUrl ? "Configured" : "Disabled" }}</strong>
            </div>
            <div>
              <span>Port</span>
              <strong>{{ runtime?.serverPort }}</strong>
            </div>
          </div>
        </aside>

        <div class="content">
          <section id="overview" class="panel section-panel">
            <div class="panel-head">
              <div>
                <p class="eyebrow">Overview</p>
                <h2>Runtime</h2>
              </div>
            </div>

            <div class="status-grid compact">
              <article class="status-card">
                <span>Server port</span>
                <strong>{{ runtime?.serverPort }}</strong>
              </article>
              <article class="status-card">
                <span>Settings file</span>
                <strong>{{ runtime?.settingsPath }}</strong>
              </article>
              <article class="status-card">
                <span>Dashboard bundle</span>
                <strong>{{ runtime?.dashboardBuilt ? "Built" : "Missing" }}</strong>
              </article>
              <article class="status-card">
                <span>Reloaded</span>
                <strong>{{ runtime?.lastReloadedAt }}</strong>
              </article>
            </div>
          </section>

          <section id="delivery" class="panel section-panel">
            <div class="panel-head">
              <div>
                <p class="eyebrow">Delivery</p>
                <h2>Targets</h2>
              </div>
            </div>

            <div class="delivery-grid">
              <article class="provider">
                <div class="provider-head">
                  <h3>Matrix</h3>
                  <label class="toggle">
                    <input v-model="settings.delivery.matrix.enabled" type="checkbox">
                    <span>{{ settings.delivery.matrix.enabled ? "Enabled" : "Disabled" }}</span>
                  </label>
                </div>

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
                  <div class="list-editor">
                    <div v-if="settings.delivery.matrix.roomIds.length > 0" class="chip-list">
                      <span
                        v-for="(roomId, index) in settings.delivery.matrix.roomIds"
                        :key="roomId"
                        class="chip"
                      >
                        <span>{{ roomId }}</span>
                        <button class="chip-remove" type="button" @click="removeRoomId(index)">Remove</button>
                      </span>
                    </div>

                    <div class="list-input">
                      <input
                        v-model="roomIdDraft"
                        type="text"
                        placeholder="!room:example.com"
                        @keydown.enter.prevent="addRoomId"
                      >
                      <button class="secondary" type="button" @click="addRoomId">Add room</button>
                    </div>
                  </div>
                </label>
              </article>

              <article class="provider">
                <div class="provider-head">
                  <h3>Discord webhook</h3>
                  <label class="toggle">
                    <input v-model="settings.delivery.discord.enabled" type="checkbox">
                    <span>{{ settings.delivery.discord.enabled ? "Enabled" : "Disabled" }}</span>
                  </label>
                </div>

                <label class="field">
                  <span>Webhook URL</span>
                  <input v-model="settings.delivery.discord.webhookUrl" type="url" placeholder="https://discord.com/api/webhooks/...">
                </label>
              </article>

              <article class="provider">
                <div class="provider-head">
                  <h3>HTTP(S) webhook</h3>
                  <label class="toggle">
                    <input v-model="settings.delivery.webhook.enabled" type="checkbox">
                    <span>{{ settings.delivery.webhook.enabled ? "Enabled" : "Disabled" }}</span>
                  </label>
                </div>

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
              </article>
            </div>

            <label class="field">
              <span>Timeout in seconds</span>
              <input v-model.number="settings.timeout" type="number" min="1" step="1">
            </label>
          </section>

          <section id="webhooks" class="panel section-panel">
            <div class="panel-head">
              <div>
                <p class="eyebrow">Webhooks</p>
                <h2>Providers</h2>
              </div>
            </div>

            <div class="providers">
              <article
                v-for="provider in providerCards"
                :key="provider.id"
                class="provider"
              >
                <div class="provider-head">
                  <h3>{{ provider.label }}</h3>
                  <label class="toggle">
                    <input v-model="settings.webhook.providers[provider.id].enabled" type="checkbox">
                    <span>{{ settings.webhook.providers[provider.id].enabled ? "Enabled" : "Disabled" }}</span>
                  </label>
                </div>

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
              </article>
            </div>
          </section>

          <section id="sites" class="panel section-panel">
            <div class="panel-head">
              <div>
                <p class="eyebrow">Monitoring</p>
                <h2>Sites</h2>
              </div>
              <button class="secondary" type="button" @click="addBlankSite">Add blank</button>
            </div>

            <article class="site-draft">
              <label class="field">
                <span>Name</span>
                <input v-model="newSite.name" type="text" placeholder="facile.studio">
              </label>

              <label class="field wide">
                <span>URL</span>
                <input v-model="newSite.url" type="url" placeholder="https://example.com">
              </label>

              <label class="field">
                <span>Interval</span>
                <input v-model.number="newSite.interval" type="number" min="1" step="1">
              </label>

              <button class="primary" type="button" @click="addSiteFromDraft">Add site</button>
            </article>

            <div v-if="settings.sites.length === 0" class="empty">No sites configured.</div>

            <div v-else class="sites">
              <article v-for="(site, index) in settings.sites" :key="`${site.url}-${index}`" class="site">
                <div class="site-header">
                  <div class="site-title">
                    <strong>{{ site.name || `Site ${index + 1}` }}</strong>
                    <span>{{ site.url || "No URL yet" }}</span>
                  </div>

                  <div class="site-actions">
                    <button class="ghost" type="button" :disabled="index === 0" @click="moveSite(index, -1)">Up</button>
                    <button class="ghost" type="button" :disabled="index === settings.sites.length - 1" @click="moveSite(index, 1)">Down</button>
                    <button class="ghost" type="button" @click="duplicateSite(index)">Duplicate</button>
                    <button class="danger" type="button" @click="removeSite(index)">Remove</button>
                  </div>
                </div>

                <label class="field">
                  <span>Name</span>
                  <input v-model="site.name" type="text" placeholder="facile.studio">
                </label>

                <label class="field wide">
                  <span>URL</span>
                  <input v-model="site.url" type="url" placeholder="https://example.com">
                </label>

                <label class="field">
                  <span>Interval</span>
                  <input v-model.number="site.interval" type="number" min="1" step="1">
                </label>
              </article>
            </div>
          </section>
        </div>
      </section>
    </template>
  </main>
</template>

<style scoped>
.shell {
  width: min(1100px, calc(100vw - 2rem));
  margin: 0 auto;
  padding: 2.5rem 0 4rem;
}

.hero {
  display: flex;
  gap: 1.5rem;
  justify-content: space-between;
  align-items: flex-end;
  margin-bottom: 1.5rem;
}

.eyebrow {
  margin: 0 0 0.5rem;
  text-transform: none;
  letter-spacing: normal;
  font-size: 0.75rem;
  color: var(--nook-muted);
}

.hero h1,
.panel h2,
.provider h3 {
  margin: 0;
}

.hero-actions {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.layout,
.providers,
.status-grid {
  display: grid;
  gap: 1rem;
}

.layout {
  grid-template-columns: 240px minmax(0, 1fr);
  align-items: start;
}

.providers {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.delivery-grid {
  display: grid;
  gap: 1rem;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  margin-bottom: 1rem;
}

.status-grid {
  grid-template-columns: repeat(4, minmax(0, 1fr));
  margin-bottom: 1rem;
}

.status-grid.compact {
  margin-bottom: 0;
}

.status-card,
.panel,
.loading-card,
.provider {
  border: 1px solid var(--nook-border);
  background: var(--nook-surface);
  border-radius: 1.4rem;
  box-shadow: inset 0 0 0 1px rgb(255 255 255 / 0.02);
}

.status-card {
  padding: 1rem 1.2rem;
}

.status-card span,
.summary span {
  display: block;
  color: var(--nook-muted);
  margin-bottom: 0.45rem;
}

.status-card strong,
.summary strong {
  font-size: 1rem;
}

.panel,
.provider,
.loading-card {
  padding: 1.2rem;
}

.panel {
  margin-bottom: 1rem;
}

.sidebar {
  position: sticky;
  top: 1.5rem;
}

.sidebar-nav {
  display: grid;
  gap: 0.4rem;
}

.sidebar-link {
  display: block;
  padding: 0.8rem 1rem;
  border-radius: 999px;
  border: 1px solid transparent;
  color: var(--nook-muted);
  text-decoration: none;
}

.sidebar-link:hover {
  border-color: var(--nook-border);
  background: var(--nook-surface-strong);
  color: var(--nook-text);
}

.sidebar-meta {
  display: grid;
  gap: 1rem;
  margin-top: 1.4rem;
  padding-top: 1.4rem;
  border-top: 1px solid var(--nook-border);
}

.sidebar-meta span {
  display: block;
  margin-bottom: 0.35rem;
  color: var(--nook-muted);
}

.content {
  min-width: 0;
}

.section-panel {
  scroll-margin-top: 1.5rem;
}

.panel-head,
.provider-head {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: center;
  margin-bottom: 1rem;
}

.field {
  display: grid;
  gap: 0.45rem;
  margin-bottom: 0.9rem;
}

.field span {
  color: var(--nook-muted);
  font-size: 0.95rem;
}

.list-editor {
  display: grid;
  gap: 0.75rem;
}

.list-input {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 0.75rem;
  align-items: center;
}

.chip-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.55rem;
}

.chip {
  display: inline-flex;
  align-items: center;
  gap: 0.55rem;
  padding: 0.45rem 0.55rem 0.45rem 0.9rem;
  border-radius: 999px;
  border: 1px solid var(--nook-border);
  background: var(--nook-surface-strong);
}

.chip-remove {
  padding: 0.3rem 0.6rem;
  font-size: 0.78rem;
}

input {
  width: 100%;
  border: 1px solid var(--nook-border);
  border-radius: 1.4rem;
  padding: 0.9rem 1.25rem;
  background: var(--nook-surface);
  color: inherit;
  outline: none;
  transition: border-color 150ms ease, background-color 150ms ease, box-shadow 150ms ease;
}

input:hover {
  background: var(--nook-surface-strong);
  border-color: rgb(255 255 255 / 0.14);
}

input:focus {
  border-color: rgb(255 255 255 / 0.22);
  background: var(--nook-surface-strong);
  box-shadow: inset 0 0 0 1px rgb(255 255 255 / 0.08);
}

button {
  border: 1px solid var(--nook-border);
  border-radius: 999px;
  padding: 0.85rem 1.15rem;
  font-weight: 700;
  cursor: pointer;
  transition: background-color 150ms ease, border-color 150ms ease, opacity 120ms ease;
}

button:hover {
  background: var(--nook-surface-strong);
  border-color: rgb(255 255 255 / 0.14);
}

button:disabled {
  cursor: default;
  opacity: 0.55;
}

.primary {
  background: var(--nook-text);
  color: var(--nook-bg);
  border-color: var(--nook-text);
}

.primary:hover {
  background: rgb(255 255 255 / 0.86);
  border-color: rgb(255 255 255 / 0.86);
}

.secondary {
  background: var(--nook-surface);
  color: var(--nook-text);
}

.ghost {
  background: transparent;
  color: var(--nook-muted);
}

.danger {
  background: transparent;
  color: var(--nook-text);
}

.toggle {
  display: inline-flex;
  gap: 0.6rem;
  align-items: center;
  color: var(--nook-muted);
}

.toggle input {
  width: auto;
  margin: 0;
}

.summary {
  display: grid;
  gap: 1rem;
}

.sites {
  display: grid;
  gap: 0.9rem;
}

.site-draft,
.site {
  display: grid;
  grid-template-columns: 1.1fr 2fr 0.7fr auto;
  gap: 0.8rem;
  align-items: end;
  padding: 1rem;
  border-radius: 1.1rem;
  background: var(--nook-surface-strong);
  border: 1px solid var(--nook-border);
}

.site {
  grid-template-columns: 1fr 1fr 2fr 0.7fr;
}

.site-header {
  grid-column: 1 / -1;
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: flex-start;
}

.site-title span {
  display: block;
  margin-top: 0.3rem;
  color: var(--nook-muted);
  word-break: break-all;
}

.site-actions {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.empty,
.loading-card,
.banner {
  color: var(--nook-muted);
}

.banner {
  padding: 0.95rem 1rem;
  border-radius: 1rem;
  margin: 0 0 1rem;
  border: 1px solid var(--nook-border);
}

.banner.error {
  background: var(--nook-surface);
  color: var(--nook-text);
}

.banner.success {
  background: var(--nook-surface);
  color: var(--nook-text);
}

@media (max-width: 960px) {
  .hero,
  .layout,
  .providers,
  .delivery-grid,
  .status-grid,
  .site-draft,
  .site {
    grid-template-columns: 1fr;
  }

  .hero {
    align-items: stretch;
  }

  .hero-actions {
    justify-content: flex-start;
  }

  .sidebar {
    position: static;
  }

  .list-input {
    grid-template-columns: 1fr;
  }

  .site-header {
    flex-direction: column;
  }

  .site-actions {
    justify-content: flex-start;
  }
}
</style>
