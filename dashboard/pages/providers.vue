<script setup lang="ts">
import SectionCard from "../components/SectionCard.vue";
import ProviderListEditor from "../components/ProviderListEditor.vue";
import { useDashboardState } from "../composables/useDashboardState";

const { settings } = useDashboardState();
</script>

<template>
  <SectionCard eyebrow="Providers" title="Sources">
    <div class="provider-controls">
      <label class="field">
        <span>Request timeout in seconds</span>
        <input v-model.number="settings.timeout" type="number" min="1" step="1">
      </label>

      <label class="field">
        <span>Webhook listener port</span>
        <input v-model.number="settings.providers.port" type="number" min="1" step="1">
      </label>
    </div>

    <ProviderListEditor v-model="settings.providers.items" />
  </SectionCard>
</template>

<style scoped>
.provider-controls {
  display: grid;
  gap: var(--space-3);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-bottom: var(--space-4);
}

@media (max-width: 720px) {
  .provider-controls {
    grid-template-columns: 1fr;
  }
}
</style>
