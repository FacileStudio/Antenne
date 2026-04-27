<script setup lang="ts">
import { ref, onMounted } from "vue";
import { Activity, RefreshCw } from "lucide-vue-next";

import EventTimeline from "../components/EventTimeline.vue";
import type { EventLogEntry, EventLogStats } from "../types";
import { useDashboardState } from "../composables/useDashboardState";

const { loadEvents } = useDashboardState();

const events = ref<EventLogEntry[]>([]);
const stats = ref<EventLogStats | null>(null);
const loading = ref(false);

async function refresh() {
  loading.value = true;
  try {
    const data = await loadEvents({ limit: 100 });
    events.value = data.events;
    stats.value = data.stats;
  } finally {
    loading.value = false;
  }
}

onMounted(() => void refresh());
</script>

<template>
  <div class="activity-page">
    <div v-if="stats" class="stats-row">
      <div class="stat-card panel">
        <span class="stat-value">{{ stats.totalEvents }}</span>
        <span class="stat-label">Total events</span>
      </div>
      <div class="stat-card panel">
        <span class="stat-value success">{{ stats.deliverySuccessCount }}</span>
        <span class="stat-label">Delivered</span>
      </div>
      <div class="stat-card panel">
        <span class="stat-value error">{{ stats.deliveryErrorCount }}</span>
        <span class="stat-label">Failed</span>
      </div>
    </div>

    <div class="panel timeline-panel">
      <div class="timeline-header">
        <h2>Event log</h2>
        <button class="ghost action-button" type="button" :disabled="loading" @click="refresh">
          <RefreshCw :size="14" :class="{ spinning: loading }" />
        </button>
      </div>
      <EventTimeline :events="events" />
    </div>
  </div>
</template>

<style scoped>
.activity-page {
  display: grid;
  gap: var(--space-5);
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--space-3);
}

.stat-card {
  display: grid;
  gap: 0.25rem;
  text-align: center;
  padding: var(--space-4);
}

.stat-value {
  font-family: "Poppins", sans-serif;
  font-size: 2rem;
  font-weight: 700;
  letter-spacing: -0.03em;
}

.stat-value.success { color: #1CDB81; }
.stat-value.error { color: rgb(255 131 131); }

.stat-label {
  font-size: 0.85rem;
  color: var(--nook-muted);
}

.timeline-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-3);
}

.timeline-header h2 {
  margin: 0;
  font-size: 1.15rem;
}

.action-button {
  display: inline-flex;
  align-items: center;
  padding: 0.5rem;
}

@keyframes spin { to { transform: rotate(360deg); } }
.spinning { animation: spin 0.7s linear infinite; }

@media (max-width: 640px) {
  .stats-row {
    grid-template-columns: 1fr;
  }
}
</style>
