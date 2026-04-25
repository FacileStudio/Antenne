<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";

type WebhookProviderId = "uptime-kuma" | "docuseal";

interface SiteConfig {
  name: string;
  url: string;
  interval: number;
}

interface MatrixConfig {
  homeserver: string;
  accessToken: string;
  roomIds: string[];
}

interface WebhookProviderSettings {
  enabled: boolean;
  path: string;
  secretHeader: string | null;
  secretValue: string | null;
}

interface AppSettings {
  matrix: MatrixConfig;
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
const runtime = ref<RuntimeStatus | null>(null);
const matrixConfigured = ref(false);
const settings = reactive<AppSettings>({
  matrix: {
    homeserver: "",
    accessToken: "",
    roomIds: [],
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
  settings.matrix.homeserver = next.matrix.homeserver;
  settings.matrix.accessToken = next.matrix.accessToken;
  settings.matrix.roomIds = [...next.matrix.roomIds];
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
        matrix: {
          ...settings.matrix,
          roomIds: settings.matrix.roomIds,
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

function addSite() {
  settings.sites.push({
    name: "",
    url: "https://",
    interval: 60,
  });
}

function removeSite(index: number) {
  settings.sites.splice(index, 1);
}

const roomIdsText = computed({
  get: () => settings.matrix.roomIds.join(", "),
  set: (value: string) => {
    settings.matrix.roomIds = value
      .split(",")
      .map((item) => item.trim())
      .filter(Boolean);
  },
});

const providerCards = computed(() => [
  {
    id: "uptime-kuma" as const,
    label: "Uptime Kuma",
    tone: "teal",
  },
  {
    id: "docuseal" as const,
    label: "DocuSeal",
    tone: "orange",
  },
]);

onMounted(() => {
  void loadSettings();
});
</script>

<template>
  <main class="shell">
    <section class="hero">
      <div>
        <p class="eyebrow">Nook Control Room</p>
        <h1>Minimal dashboard. No YAML archaeology.</h1>
        <p class="lede">
          Edit Matrix, sites, and webhook providers here. Env config still works as bootstrap, but saved dashboard
          settings take over once you hit save.
        </p>
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
      <section class="grid two">
        <article class="panel">
          <div class="panel-head">
            <div>
              <p class="eyebrow">Matrix</p>
              <h2>Delivery</h2>
            </div>
          </div>

          <label class="field">
            <span>Homeserver</span>
            <input v-model="settings.matrix.homeserver" type="url" placeholder="https://matrix.example.com">
          </label>

          <label class="field">
            <span>Access token</span>
            <input v-model="settings.matrix.accessToken" type="password" placeholder="syt_xxx">
          </label>

          <label class="field">
            <span>Room IDs</span>
            <input v-model="roomIdsText" type="text" placeholder="!room:example.com,!ops:example.com">
          </label>

          <label class="field">
            <span>Timeout in seconds</span>
            <input v-model.number="settings.timeout" type="number" min="1" step="1">
          </label>
        </article>

        <article class="panel">
          <div class="panel-head">
            <div>
              <p class="eyebrow">Runtime</p>
              <h2>Server</h2>
            </div>
          </div>

          <div class="summary">
            <div>
              <span>Dashboard bundle</span>
              <strong>{{ runtime?.dashboardBuilt ? "Built" : "Missing" }}</strong>
            </div>
            <div>
              <span>Reloaded</span>
              <strong>{{ runtime?.lastReloadedAt }}</strong>
            </div>
            <div>
              <span>Webhook port</span>
              <strong>{{ settings.webhook.port }}</strong>
            </div>
          </div>
        </article>
      </section>

      <section class="panel">
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
            :class="provider.tone"
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

      <section class="panel">
        <div class="panel-head">
          <div>
            <p class="eyebrow">Monitoring</p>
            <h2>Sites</h2>
          </div>
          <button class="secondary" type="button" @click="addSite">Add site</button>
        </div>

        <div v-if="settings.sites.length === 0" class="empty">
          No sites configured. Webhook-only mode still works.
        </div>

        <div v-else class="sites">
          <article v-for="(site, index) in settings.sites" :key="`${site.url}-${index}`" class="site">
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

            <button class="danger" type="button" @click="removeSite(index)">Remove</button>
          </article>
        </div>
      </section>
    </template>
  </main>
</template>

<style scoped>
.shell {
  width: min(1100px, calc(100vw - 2rem));
  margin: 0 auto;
  padding: 2rem 0 4rem;
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
  text-transform: uppercase;
  letter-spacing: 0.22em;
  font-size: 0.75rem;
  color: rgba(245, 248, 250, 0.6);
}

.lede {
  max-width: 52rem;
  color: rgba(245, 248, 250, 0.72);
  line-height: 1.6;
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

.grid.two,
.providers,
.status-grid {
  display: grid;
  gap: 1rem;
}

.grid.two {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.providers {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.status-grid {
  grid-template-columns: repeat(4, minmax(0, 1fr));
  margin-bottom: 1rem;
}

.status-card,
.panel,
.loading-card,
.provider {
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(11, 17, 21, 0.84);
  backdrop-filter: blur(18px);
  border-radius: 1.4rem;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.22);
}

.status-card {
  padding: 1rem 1.2rem;
}

.status-card span,
.summary span {
  display: block;
  color: rgba(245, 248, 250, 0.6);
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
  color: rgba(245, 248, 250, 0.7);
  font-size: 0.95rem;
}

input {
  width: 100%;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 0.95rem;
  padding: 0.9rem 1rem;
  background: rgba(255, 255, 255, 0.05);
  color: inherit;
}

button {
  border: 0;
  border-radius: 999px;
  padding: 0.85rem 1.15rem;
  font-weight: 700;
  cursor: pointer;
  transition: transform 120ms ease, opacity 120ms ease;
}

button:hover {
  transform: translateY(-1px);
}

button:disabled {
  cursor: default;
  opacity: 0.55;
  transform: none;
}

.primary {
  background: linear-gradient(135deg, #4fa9af, #e0843c);
  color: #091013;
}

.secondary {
  background: rgba(255, 255, 255, 0.08);
  color: #f5f8fa;
}

.ghost {
  background: transparent;
  color: rgba(245, 248, 250, 0.76);
  border: 1px solid rgba(255, 255, 255, 0.12);
}

.danger {
  background: rgba(227, 98, 65, 0.14);
  color: #ffb49f;
}

.provider.teal {
  background:
    linear-gradient(180deg, rgba(73, 161, 170, 0.12), rgba(11, 17, 21, 0.92)),
    rgba(11, 17, 21, 0.92);
}

.provider.orange {
  background:
    linear-gradient(180deg, rgba(221, 129, 58, 0.12), rgba(11, 17, 21, 0.92)),
    rgba(11, 17, 21, 0.92);
}

.toggle {
  display: inline-flex;
  gap: 0.6rem;
  align-items: center;
  color: rgba(245, 248, 250, 0.76);
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

.site {
  display: grid;
  grid-template-columns: 1.1fr 2fr 0.7fr auto;
  gap: 0.8rem;
  align-items: end;
  padding: 1rem;
  border-radius: 1.1rem;
  background: rgba(255, 255, 255, 0.04);
}

.empty,
.loading-card,
.banner {
  color: rgba(245, 248, 250, 0.76);
}

.banner {
  padding: 0.95rem 1rem;
  border-radius: 1rem;
  margin: 0 0 1rem;
}

.banner.error {
  background: rgba(216, 78, 48, 0.12);
  color: #ffb8a4;
}

.banner.success {
  background: rgba(69, 160, 130, 0.14);
  color: #aae2cb;
}

@media (max-width: 960px) {
  .hero,
  .grid.two,
  .providers,
  .status-grid,
  .site {
    grid-template-columns: 1fr;
  }

  .hero {
    align-items: stretch;
  }

  .hero-actions {
    justify-content: flex-start;
  }
}
</style>
