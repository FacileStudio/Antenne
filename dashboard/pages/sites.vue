<script setup lang="ts">
import SectionCard from "../components/SectionCard.vue";
import SiteListEditor from "../components/SiteListEditor.vue";
import { useDashboardState } from "../composables/useDashboardState";

const { settings, notifyError } = useDashboardState();

function setError(message: string) {
  notifyError(message);
}

function clearError() {
  return;
}
</script>

<template>
  <SectionCard eyebrow="Sites" title="Monitoring">
    <div class="timeout-card">
      <label class="field timeout-field">
        <span>Timeout in seconds</span>
        <input v-model.number="settings.timeout" type="number" min="1" step="1">
      </label>
      <p class="timeout-help">Applied to every site check and delivery request before Nook gives up.</p>
    </div>

    <SiteListEditor
      v-model="settings.sites"
      @error="setError"
      @clear-error="clearError"
    />
  </SectionCard>
</template>

<style scoped>
.timeout-card {
  display: flex;
  justify-content: space-between;
  gap: var(--space-4);
  align-items: end;
  padding: var(--space-4);
  border-radius: 1.25rem;
  border: 1px solid var(--nook-border);
  background: var(--nook-surface-strong);
  margin-bottom: var(--space-4);
}

.timeout-field {
  max-width: 18rem;
}

.timeout-help {
  margin: 0;
  color: var(--nook-muted);
}

@media (max-width: 720px) {
  .timeout-card {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
