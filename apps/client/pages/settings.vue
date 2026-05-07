<script setup lang="ts">
import { computed, ref } from "vue";

import SectionCard from "../components/SectionCard.vue";
import StatusGrid from "../components/StatusGrid.vue";
import { useDashboardState } from "../composables/useDashboardState";

const { runtime, settings, saving, exportConfig, importConfig, notifyError } = useDashboardState();

const importInput = ref<HTMLInputElement | null>(null);

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
</script>

<template>
  <div class="settings-page">
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
      </div>
      <p class="field-hint">
        Targets can opt into these Nook system events from their provider filter inside Delivery.
      </p>
    </SectionCard>

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
      <StatusGrid :items="infoItems" />
    </SectionCard>
  </div>
</template>

<style scoped>
.settings-page {
  display: grid;
  gap: var(--space-5);
}

.settings-grid {
  display: grid;
  gap: var(--space-3);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.event-settings {
  display: grid;
  gap: 1rem;
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

@media (max-width: 720px) {
  .config-actions {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 720px) {
  .settings-grid {
    grid-template-columns: 1fr;
  }
}
</style>
