<script setup lang="ts">
import { computed } from "vue";

import SectionCard from "../components/SectionCard.vue";
import StatusGrid from "../components/StatusGrid.vue";
import { useDashboardState } from "../composables/useDashboardState";

const { runtime, settings } = useDashboardState();

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
    value: runtime.value?.usingPersistedSettings ? "Dashboard override" : "Env/YAML defaults",
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
  min-height: 3rem;
  padding: 0.95rem 1rem;
  border: 1px solid var(--nook-border);
  border-radius: 1.1rem;
  background: rgb(255 255 255 / 0.025);
}

.field-hint {
  margin: 0;
  color: var(--nook-muted);
  font-size: 0.84rem;
  line-height: 1.45;
}

@media (max-width: 720px) {
  .settings-grid {
    grid-template-columns: 1fr;
  }
}
</style>
