<script setup lang="ts">
import { computed } from "vue";
import { Icon } from "@iconify/vue";

import type { EventLogEntry, EventLogStats } from "../types";

const props = defineProps<{
  events: EventLogEntry[];
  stats: EventLogStats | null;
  loading: boolean;
}>();

defineEmits<{
  refresh: [];
}>();

const lastEventLabel = computed(() => {
  if (!props.stats?.lastEventAt) {
    return "No events yet";
  }

  return new Date(props.stats.lastEventAt).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
});

const successRate = computed(() => {
  if (!props.stats) {
    return 0;
  }

  const total = props.stats.deliverySuccessCount + props.stats.deliveryErrorCount;
  if (total === 0) {
    return 100;
  }

  return Math.round((props.stats.deliverySuccessCount / total) * 100);
});
</script>

<template>
  <section class="overview-card panel">
    <div class="overview-copy">
      <h2>Stats</h2>
    </div>

    <div class="overview-meta">
      <div class="meta-pill">
        <Icon icon="solar:clock-circle-bold" width="14" />
        <span>Last event {{ lastEventLabel }}</span>
      </div>
      <button class="ghost action-button refresh-button" type="button" :disabled="loading" @click="$emit('refresh')">
        <Icon icon="solar:refresh-bold" width="14" :class="{ spinning: loading }" />
        <span>{{ loading ? "Refreshing..." : "Refresh feed" }}</span>
      </button>
    </div>

    <div class="stats-grid">
      <div class="stat-tile">
        <span class="stat-value">{{ stats?.totalEvents ?? events.length }}</span>
        <span class="stat-label">Tracked</span>
      </div>
      <div class="stat-tile">
        <span class="stat-value success">{{ stats?.deliverySuccessCount ?? 0 }}</span>
        <span class="stat-label">Success</span>
      </div>
      <div class="stat-tile">
        <span class="stat-value error">{{ stats?.deliveryErrorCount ?? 0 }}</span>
        <span class="stat-label">Failed</span>
      </div>
      <div class="stat-tile">
        <span class="stat-value">{{ successRate }}%</span>
        <span class="stat-label">Hit rate</span>
      </div>
    </div>
  </section>
</template>

<style scoped>
.overview-card {
  display: grid;
  gap: 1.4rem;
  padding: 1.5rem;
  background: var(--nook-surface);
}

.overview-copy {
  display: grid;
  gap: 0.45rem;
}

.overview-eyebrow {
  margin: 0;
  font-size: 0.78rem;
  color: var(--nook-muted);
}

h2 {
  margin: 0;
  font-size: 1.55rem;
}

.overview-text {
  margin: 0;
  max-width: 34rem;
  color: var(--nook-muted);
  line-height: 1.55;
}

.overview-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: var(--space-3);
  flex-wrap: wrap;
}

.meta-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.5rem 0.8rem;
  border-radius: 999px;
  border: 1px solid var(--nook-border);
  background: rgb(255 255 255 / 0.03);
  color: var(--nook-muted);
  font-size: 0.84rem;
}

.refresh-button {
  min-height: 2.8rem;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.9rem;
}

.stat-tile {
  display: grid;
  gap: 0.3rem;
  padding: 1rem 1.05rem;
  border-radius: 1.15rem;
  border: 1px solid var(--nook-border);
  background: rgb(255 255 255 / 0.025);
}

.stat-value {
  font-family: "Poppins", sans-serif;
  font-size: 1.7rem;
  font-weight: 700;
  letter-spacing: -0.04em;
}

.stat-value.success {
  color: #1cdb81;
}

.stat-value.error {
  color: rgb(255 131 131);
}

.stat-label {
  color: var(--nook-muted);
  font-size: 0.82rem;
}

.spinning {
  animation: spin 0.7s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 960px) {
  .stats-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .stats-grid {
    grid-template-columns: 1fr;
  }
}
</style>
