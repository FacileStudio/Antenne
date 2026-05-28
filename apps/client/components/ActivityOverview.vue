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

const recentCount = computed(() => {
  if (!props.events.length) return 0;
  const sixHoursAgo = Date.now() - 6 * 60 * 60 * 1000;
  return props.events.filter((e) => new Date(e.timestamp).getTime() > sixHoursAgo).length;
});
</script>

<template>
  <section class="grid gap-5">
    <div class="flex items-center justify-between gap-3 flex-wrap">
      <div>
        <p class="m-0 mb-1 text-xs text-muted-foreground uppercase tracking-wider">Overview</p>
        <h2 class="m-0 text-xl font-semibold">Stats</h2>
      </div>
      <div class="flex items-center gap-2.5">
        <span class="inline-flex items-center gap-1.5 rounded-full bg-muted px-2.5 py-1 text-xs text-muted-foreground ring-1 ring-foreground/10">
          <Icon icon="solar:clock-circle-linear" :width="13" :height="13" aria-hidden="true" />
          {{ lastEventLabel }}
        </span>
        <button
          class="ghost action-button h-8"
          type="button"
          :disabled="loading"
          @click="$emit('refresh')"
        >
          <Icon icon="solar:refresh-linear" :width="14" :height="14" :class="{ 'animate-spin': loading }" aria-hidden="true" />
          <span>{{ loading ? "Refreshing..." : "Refresh" }}</span>
        </button>
      </div>
    </div>

    <div class="stat-grid">
      <div class="stat-card">
        <span class="stat-label">Tracked</span>
        <span class="stat-value">{{ stats?.totalEvents ?? events.length }}</span>
      </div>

      <div class="stat-card">
        <span class="stat-label">Success</span>
        <span class="stat-value text-green-600">{{ stats?.deliverySuccessCount ?? 0 }}</span>
      </div>

      <div class="stat-card">
        <span class="stat-label">Failed</span>
        <span class="stat-value text-red-500">{{ stats?.deliveryErrorCount ?? 0 }}</span>
      </div>

      <div class="stat-card">
        <span class="stat-label">Hit rate</span>
        <span class="stat-value">{{ successRate }}%</span>
      </div>

      <div class="stat-card">
        <span class="stat-label">Recent (6h)</span>
        <span class="stat-value stat-value-live">
          <span class="pulse-dot">
            <span class="pulse-ring" />
            <span class="pulse-core" />
          </span>
          {{ recentCount }}
        </span>
      </div>
    </div>
  </section>
</template>

<style scoped>
.stat-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;
}

@media (min-width: 768px) {
  .stat-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (min-width: 1024px) {
  .stat-grid {
    grid-template-columns: repeat(5, minmax(0, 1fr));
  }
}

.stat-card {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  padding: 0.875rem 1rem;
  border-radius: var(--radius-xl, 0.75rem);
  background: var(--color-card, oklch(0.985 0 0));
  ring: 1px solid;
  border: 1px solid oklch(0 0 0 / 0.06);
}

.stat-label {
  font-size: 0.75rem;
  color: var(--nook-muted, oklch(0.556 0 0));
}

.stat-value {
  font-size: 1.5rem;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  line-height: 1.2;
}

.stat-value-live {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.pulse-dot {
  position: relative;
  display: flex;
  height: 0.5rem;
  width: 0.5rem;
}

.pulse-ring {
  position: absolute;
  display: inline-flex;
  height: 100%;
  width: 100%;
  border-radius: 9999px;
  background-color: oklch(0.72 0.19 149);
  opacity: 0.75;
  animation: ping 1s cubic-bezier(0, 0, 0.2, 1) infinite;
}

.pulse-core {
  position: relative;
  display: inline-flex;
  height: 0.5rem;
  width: 0.5rem;
  border-radius: 9999px;
  background-color: oklch(0.62 0.17 149);
}

@keyframes ping {
  75%, 100% {
    transform: scale(2);
    opacity: 0;
  }
}
</style>
