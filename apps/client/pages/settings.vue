<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import { Icon } from "@iconify/vue";

import SectionCard from "../components/SectionCard.vue";
import { useDashboardState } from "../composables/useDashboardState";

const { runtime, settings, saving, exportConfig, importConfig, notifyError } = useDashboardState();

const activeTab = ref<"general" | "pool" | "events" | "advanced">("general");
const importInput = ref<HTMLInputElement | null>(null);

type PoolStats = {
  epoch: number;
  connections: number;
  apps: Array<{ app_id: string; app: string; channels: string[] }>;
};

const poolStats = ref<PoolStats | null>(null);
let poolStatsInterval: ReturnType<typeof setInterval> | null = null;

async function loadPoolStats() {
  try {
    const res = await fetch("/api/pool/stats");
    if (res.ok) {
      poolStats.value = await res.json() as PoolStats;
    }
  } catch {}
}

onMounted(() => {
  loadPoolStats();
  poolStatsInterval = setInterval(loadPoolStats, 10000);
});

onUnmounted(() => {
  if (poolStatsInterval) clearInterval(poolStatsInterval);
});

const poolSecret = computed({
  get: () => settings.pool?.secret ?? "",
  set: (value: string) => {
    if (!settings.pool) settings.pool = {};
    settings.pool.secret = value || undefined;
  },
});

const poolSecretSource = computed(() => {
  if (settings.pool?.secret && settings.pool.secret !== "••••••••") return "Dashboard config";
  return "Environment variable or default";
});

const poolLogActivity = computed({
  get: () => settings.pool?.logActivity !== false,
  set: (value: boolean) => {
    if (!settings.pool) settings.pool = {};
    settings.pool.logActivity = value;
  },
});

const APP_ICONS: Record<string, string> = {
  Opus: "solar:widget-2-bold-duotone",
  Sablier: "solar:hourglass-bold-duotone",
  Ardoise: "solar:bill-list-bold-duotone",
  Plume: "solar:pen-new-square-bold-duotone",
  Glouton: "solar:chef-hat-bold-duotone",
  Vision: "solar:panorama-bold-duotone",
};

function appIcon(name: string): string {
  return APP_ICONS[name] ?? "solar:server-square-bold-duotone";
}

async function handleImport(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0];
  if (!file) return;
  try {
    await importConfig(file);
  } catch {
  } finally {
    if (importInput.value) importInput.value.value = "";
  }
}

const poolInfoItems = computed(() => {
  const items: Array<{ label: string; value: string }> = [];
  if (poolStats.value) {
    const epoch = String(poolStats.value.epoch);
    items.push(
      { label: "Epoch", value: epoch.length > 8 ? epoch.slice(0, 8) : epoch },
      { label: "Total connections", value: String(poolStats.value.connections) },
    );
  }
  return items;
});

const infoItems = computed(() => [
  {
    label: "Runtime port",
    value: runtime.value?.serverPort ? String(runtime.value.serverPort) : "Unknown",
  },
  {
    label: "Settings file",
    value: runtime.value?.settingsPath ?? "Unknown",
  },
  {
    label: "Source",
    value: runtime.value?.usingPersistedSettings ? "Dashboard override" : "Env defaults",
  },
  {
    label: "Reloaded",
    value: runtime.value?.lastReloadedAt ?? "Unknown",
  },
]);

const tabs = [
  { key: "general" as const, label: "General", icon: "solar:settings-linear" },
  { key: "pool" as const, label: "Pool", icon: null },
  { key: "events" as const, label: "Events", icon: "solar:bell-linear" },
  { key: "advanced" as const, label: "Advanced", icon: "solar:tuning-2-linear" },
];
</script>

<template>
  <div class="space-y-8 px-4 py-4 md:px-8 md:py-5">
    <div class="tab-bar">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        class="tab-button"
        :class="activeTab === tab.key ? 'tab-active' : 'tab-inactive'"
        @click="activeTab = tab.key"
      >
        <Icon v-if="tab.icon" :icon="tab.icon" :width="16" :height="16" aria-hidden="true" />
        <svg v-else xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="16" height="16" class="inline-block" aria-hidden="true">
          <path fill="currentColor" d="M10.31 17.344c.767-.876 1.151-1.314 1.625-1.342q.065-.004.13 0c.474.028.858.466 1.625 1.342c1.67 1.906 2.505 2.858 2.271 3.68q-.03.107-.074.206C15.543 22 14.362 22 12 22s-3.543 0-3.887-.77a2 2 0 0 1-.074-.206c-.234-.822.6-1.774 2.27-3.68M14.5 12.5a2.5 2.5 0 1 1-5 0a2.5 2.5 0 0 1 5 0"/>
          <path fill="currentColor" fill-rule="evenodd" d="M12 8.035c-2.697 0-4.884 2.151-4.884 4.806a4.75 4.75 0 0 0 1.43 3.398a.68.68 0 0 1 0 .97a.706.706 0 0 1-.986 0a6.1 6.1 0 0 1-1.84-4.368c0-3.413 2.812-6.18 6.28-6.18s6.279 2.767 6.279 6.18a6.1 6.1 0 0 1-1.84 4.369a.706.706 0 0 1-.986 0a.68.68 0 0 1 0-.971a4.75 4.75 0 0 0 1.43-3.398c0-2.655-2.186-4.806-4.883-4.806" clip-rule="evenodd" opacity=".7"/>
          <path fill="currentColor" fill-rule="evenodd" d="M12 4.373c-4.752 0-8.605 3.791-8.605 8.468c0 2.338.963 4.454 2.52 5.987a.68.68 0 0 1 0 .97a.706.706 0 0 1-.986 0A9.73 9.73 0 0 1 2 12.842C2 7.406 6.477 3 12 3s10 4.406 10 9.84a9.73 9.73 0 0 1-2.929 6.959a.706.706 0 0 1-.987 0a.68.68 0 0 1 0-.971a8.37 8.37 0 0 0 2.52-5.987c0-4.677-3.852-8.468-8.604-8.468" clip-rule="evenodd" opacity=".4"/>
        </svg>
        {{ tab.label }}
      </button>
    </div>

    <div v-if="activeTab === 'general'">
      <SectionCard eyebrow="Settings" title="System">
        <div class="settings-grid">
          <label class="field">
            <span>Request timeout in seconds</span>
            <input v-model.number="settings.timeout" type="number" min="1" step="1">
          </label>

          <label class="field">
            <span>Webhook listener port</span>
            <input v-model.number="settings.providers.port" type="number" min="1" step="1">
          </label>
        </div>
      </SectionCard>
    </div>

    <div v-else-if="activeTab === 'pool'" class="tab-content-stack">
      <SectionCard eyebrow="Pool" title="Secret">
        <div class="settings-grid">
          <label class="field">
            <span>Secret</span>
            <input v-model="poolSecret" type="password" placeholder="pool">
          </label>
        </div>
        <p class="field-hint">Source: {{ poolSecretSource }}</p>
      </SectionCard>

      <SectionCard eyebrow="Pool" title="Activity Log">
        <label class="toggle event-toggle">
          <input v-model="poolLogActivity" type="checkbox">
          <span>Log pool events to activity feed</span>
        </label>
        <p class="field-hint">When enabled, pool connection and sync events appear in the activity log.</p>
      </SectionCard>

      <SectionCard eyebrow="Pool" title="Connected Apps">
        <div v-if="!poolStats?.apps?.length" class="connected-apps-empty">
          <p class="field-hint">No apps connected</p>
        </div>
        <div v-else class="connected-apps-list">
          <div v-for="app in poolStats.apps" :key="app.app_id" class="connected-app">
            <div class="connected-app-info">
              <div class="connected-app-header">
                <span class="connected-app-dot" />
                <Icon :icon="appIcon(app.app)" :width="18" :height="18" class="connected-app-icon" aria-hidden="true" />
                <strong>{{ app.app }}</strong>
              </div>
              <span class="connected-app-channels">{{ app.channels.length }} channel{{ app.channels.length === 1 ? "" : "s" }}</span>
            </div>
          </div>
        </div>
      </SectionCard>

      <SectionCard v-if="poolInfoItems.length" eyebrow="Pool" title="Info">
        <div class="status-list">
          <div v-for="item in poolInfoItems" :key="item.label" class="status-row">
            <span class="status-label">{{ item.label }}</span>
            <span class="status-value" :title="item.value">{{ item.value }}</span>
          </div>
        </div>
      </SectionCard>
    </div>

    <div v-else-if="activeTab === 'events'">
      <SectionCard eyebrow="Settings" title="System Events">
        <div class="event-settings">
          <label class="toggle event-toggle">
            <input v-model="settings.system.events.startup" type="checkbox">
            <span>Send startup events</span>
          </label>

          <label class="toggle event-toggle">
            <input v-model="settings.system.events.providerEnabled" type="checkbox">
            <span>Send events when a provider gets enabled</span>
          </label>

          <label class="toggle event-toggle">
            <input v-model="settings.system.events.providerDisabled" type="checkbox">
            <span>Send events when a provider gets disabled</span>
          </label>

          <label class="toggle event-toggle">
            <input v-model="settings.system.events.providerAdded" type="checkbox">
            <span>Send events when a provider is added</span>
          </label>

          <label class="toggle event-toggle">
            <input v-model="settings.system.events.providerRemoved" type="checkbox">
            <span>Send events when a provider is removed</span>
          </label>

          <label class="toggle event-toggle">
            <input v-model="settings.system.events.deliveryFailure" type="checkbox">
            <span>Send events when a delivery target fails</span>
          </label>
        </div>
        <p class="field-hint">
          Targets can opt into these Nook system events from their provider filter inside Delivery.
        </p>
      </SectionCard>
    </div>

    <div v-else-if="activeTab === 'advanced'" class="tab-content-stack">
      <SectionCard eyebrow="Config" title="Export / Import">
        <div class="config-actions">
          <div class="config-action">
            <p class="field-hint">Download the current config as a <code>nook-config.json</code> file.</p>
            <button class="btn" @click="exportConfig">Export config</button>
          </div>
          <div class="config-action">
            <p class="field-hint">Import a previously exported <code>nook-config.json</code>. This overwrites your current settings.</p>
            <input ref="importInput" type="file" accept=".json,application/json" class="file-input" :disabled="saving" @change="handleImport">
          </div>
        </div>
      </SectionCard>

      <SectionCard eyebrow="Runtime" title="Status">
        <div class="status-list">
          <div v-for="item in infoItems" :key="item.label" class="status-row">
            <span class="status-label">{{ item.label }}</span>
            <span class="status-value" :title="item.value">{{ item.value }}</span>
          </div>
        </div>
      </SectionCard>
    </div>
  </div>
</template>

<style scoped>
.tab-bar {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  border-bottom: 1px solid var(--nook-border);
  padding-bottom: 1rem;
}

.tab-button {
  display: inline-flex;
  height: 2.25rem;
  align-items: center;
  gap: 0.5rem;
  border-radius: var(--radius-md);
  padding: 0 1rem;
  font-size: 0.875rem;
  font-weight: 500;
  transition: color 150ms, background-color 150ms;
  border: none;
}

.tab-active {
  background: var(--foreground);
  color: var(--background);
}

.tab-active:hover {
  background: var(--foreground);
  border-color: transparent;
}

.tab-inactive {
  color: var(--muted-foreground);
}

.tab-inactive:hover {
  background: var(--muted);
  color: var(--foreground);
  border-color: transparent;
}

.tab-content-stack {
  display: grid;
  gap: var(--space-5);
}

.settings-grid {
  display: grid;
  gap: var(--space-3);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  max-width: 36rem;
}

.event-settings {
  display: grid;
  gap: 1rem;
  max-width: 36rem;
}

.event-toggle {
  padding: 0.75rem 1rem;
  border: 1px solid var(--nook-border);
  border-radius: var(--radius-xl);
  background: oklch(0 0 0 / 2.5%);
}

.field-hint {
  margin: 0;
  color: var(--nook-muted);
  font-size: 0.84rem;
  line-height: 1.45;
}

.config-actions {
  display: grid;
  gap: var(--space-4);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.config-action {
  display: grid;
  gap: var(--space-3);
  align-content: start;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.5rem 1rem;
  border: 1px solid var(--nook-border);
  border-radius: var(--radius-xl);
  background: oklch(0 0 0 / 2.5%);
  color: var(--nook-fg);
  font-size: 0.875rem;
  cursor: pointer;
  transition: background 0.15s;
}

.btn:hover {
  background: oklch(0 0 0 / 6%);
}

.file-input {
  font-size: 0.875rem;
  color: var(--nook-fg);
  cursor: pointer;
}

.file-input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.connected-apps-empty {
  padding: var(--space-4);
  text-align: center;
}

.connected-apps-list {
  display: grid;
  gap: 0.75rem;
}

.connected-app {
  display: flex;
  align-items: center;
  padding: 0.875rem 1rem;
  border: 1px solid var(--nook-border);
  border-radius: var(--radius-xl);
  background: var(--nook-surface-strong);
}

.connected-app-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  gap: var(--space-3);
}

.connected-app-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.connected-app-icon {
  color: var(--nook-muted);
  flex-shrink: 0;
}

.connected-app-dot {
  width: 0.5rem;
  height: 0.5rem;
  border-radius: 50%;
  background: var(--success);
  flex-shrink: 0;
}

.connected-app-channels {
  color: var(--nook-muted);
  font-size: 0.84rem;
  white-space: nowrap;
}

.status-list {
  display: grid;
  gap: 0;
  border: 1px solid var(--nook-border);
  border-radius: var(--radius-xl);
  overflow: hidden;
}

.status-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
  padding: 0.75rem 1rem;
  min-width: 0;
}

.status-row + .status-row {
  border-top: 1px solid var(--nook-border);
}

.status-label {
  color: var(--nook-muted);
  font-size: 0.84rem;
  flex-shrink: 0;
}

.status-value {
  font-size: 0.875rem;
  font-weight: 500;
  text-align: right;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

@media (max-width: 720px) {
  .config-actions {
    grid-template-columns: 1fr;
  }

  .settings-grid {
    grid-template-columns: 1fr;
  }

  .tab-bar {
    flex-wrap: wrap;
  }
}
</style>
