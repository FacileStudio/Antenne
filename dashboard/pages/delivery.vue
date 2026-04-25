<script setup lang="ts">
import { computed } from "vue";

import SectionCard from "../components/SectionCard.vue";
import RoomIdEditor from "../components/RoomIdEditor.vue";
import { useDashboardState } from "../composables/useDashboardState";

const { settings } = useDashboardState();

const activeTargets = computed(() => {
  const targets: string[] = [];

  if (settings.delivery.matrix.enabled && settings.delivery.matrix.roomIds.length > 0) {
    targets.push("Matrix");
  }

  if (settings.delivery.discord.enabled && settings.delivery.discord.webhookUrl) {
    targets.push("Discord");
  }

  if (settings.delivery.webhook.enabled && settings.delivery.webhook.url) {
    targets.push("HTTP");
  }

  return targets;
});
</script>

<template>
  <SectionCard eyebrow="Delivery" title="Targets">
    <div class="delivery-overview">
      <div class="overview-copy">
        <p class="overview-title">Configure where alerts go.</p>
        <p class="overview-text">You can enable multiple targets at once. Nook sends the same alert to every active delivery target.</p>
      </div>

      <div class="overview-meta">
        <span class="meta-label">Active targets</span>
        <div v-if="activeTargets.length > 0" class="target-list">
          <span
            v-for="target in activeTargets"
            :key="target"
            class="target-chip"
          >
            {{ target }}
          </span>
        </div>
        <span v-else class="meta-empty">No active targets yet</span>
      </div>
    </div>
    <div class="delivery-grid">
      <article class="panel nested-panel matrix-panel" :class="{ inactive: !settings.delivery.matrix.enabled }">
        <div class="card-header">
          <div>
            <h3>Matrix</h3>
            <p class="card-help">Primary delivery for rooms and operator chatter.</p>
          </div>
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

      <article class="panel nested-panel" :class="{ inactive: !settings.delivery.discord.enabled }">
        <div class="card-header">
          <div>
            <h3>Discord webhook</h3>
            <p class="card-help">Send plain-text alerts into a Discord channel.</p>
          </div>
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

      <article class="panel nested-panel" :class="{ inactive: !settings.delivery.webhook.enabled }">
        <div class="card-header">
          <div>
            <h3>HTTP(S) webhook</h3>
            <p class="card-help">Post JSON to another service or automation endpoint.</p>
          </div>
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

  </SectionCard>
</template>

<style scoped>
.delivery-overview {
  display: flex;
  justify-content: space-between;
  gap: var(--space-4);
  align-items: flex-start;
  padding: var(--space-4);
  border-radius: 1.25rem;
  border: 1px solid var(--nook-border);
  background: var(--nook-surface-strong);
}

.overview-copy,
.overview-meta {
  display: grid;
  gap: 0.45rem;
}

.overview-title {
  margin: 0;
  font-weight: 700;
}

.overview-text,
.timeout-help,
.card-help,
.meta-empty,
.meta-label {
  margin: 0;
  color: var(--nook-muted);
}

.target-list {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.target-chip {
  display: inline-flex;
  align-items: center;
  min-height: 2rem;
  padding: 0.3rem 0.8rem;
  border-radius: 999px;
  border: 1px solid var(--nook-border);
  background: var(--nook-surface);
}

.delivery-grid {
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

.inactive {
  opacity: 0.76;
}

.card-header {
  display: flex;
  justify-content: space-between;
  gap: var(--space-3);
  align-items: flex-start;
}

.card-header h3 {
  margin: 0;
}

.field-list {
  display: grid;
  gap: var(--space-3);
}

.toggle {
  display: inline-flex;
  align-items: center;
  gap: 0.6rem;
  color: var(--nook-muted);
}

@media (max-width: 720px) {
  .delivery-overview {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
