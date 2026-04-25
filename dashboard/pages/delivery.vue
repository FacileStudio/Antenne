<script setup lang="ts">
import SectionCard from "../components/SectionCard.vue";
import RoomIdEditor from "../components/RoomIdEditor.vue";
import { useDashboardState } from "../composables/useDashboardState";

const { settings } = useDashboardState();
</script>

<template>
  <SectionCard eyebrow="Delivery" title="Targets">
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
</template>

<style scoped>
.delivery-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
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

@media (max-width: 1120px) {
  .delivery-grid {
    grid-template-columns: 1fr;
  }
}
</style>
