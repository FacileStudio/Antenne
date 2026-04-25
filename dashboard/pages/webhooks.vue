<script setup lang="ts">
import { Eye, EyeOff } from "lucide-vue-next";
import { reactive } from "vue";

import SectionCard from "../components/SectionCard.vue";
import { useDashboardState } from "../composables/useDashboardState";
import type { WebhookProviderId } from "../types";

const { settings } = useDashboardState();
const revealedSecrets = reactive<Record<WebhookProviderId, boolean>>({
  "uptime-kuma": false,
  docuseal: false,
});

const providerCards = [
  {
    id: "uptime-kuma" as const satisfies WebhookProviderId,
    label: "Uptime Kuma",
  },
  {
    id: "docuseal" as const satisfies WebhookProviderId,
    label: "DocuSeal",
  },
];
</script>

<template>
  <SectionCard eyebrow="Webhooks" title="Providers">
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
            <div class="secret-field">
              <input
                v-model="settings.webhook.providers[provider.id].secretValue"
                :type="revealedSecrets[provider.id] ? 'text' : 'password'"
              >
              <button
                class="ghost action-button reveal-button"
                type="button"
                @click="revealedSecrets[provider.id] = !revealedSecrets[provider.id]"
              >
                <component :is="revealedSecrets[provider.id] ? EyeOff : Eye" :size="16" aria-hidden="true" />
                <span>{{ revealedSecrets[provider.id] ? "Hide" : "Show" }}</span>
              </button>
            </div>
          </label>
        </div>
      </article>
    </div>
  </SectionCard>
</template>

<style scoped>
.provider-grid {
  display: grid;
  grid-template-columns: 1fr;
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

.secret-field {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: var(--space-2);
  align-items: center;
}

.toggle {
  display: inline-flex;
  align-items: center;
  gap: 0.6rem;
  color: var(--nook-muted);
}

.reveal-button {
  min-height: 3.5rem;
}

@media (max-width: 720px) {
  .secret-field {
    grid-template-columns: 1fr;
  }
}
</style>
