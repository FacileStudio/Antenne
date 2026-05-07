<script setup lang="ts">
import { onMounted, ref } from "vue";

import ActivityChart from "../components/ActivityChart.vue";
import ActivityOverview from "../components/ActivityOverview.vue";
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
    <div class="overview-grid">
      <ActivityOverview :events="events" :stats="stats" :loading="loading" @refresh="refresh" />
      <ActivityChart :events="events" />
    </div>

    <div class="timeline-panel">
      <div class="timeline-header">
        <div>
          <p class="timeline-eyebrow">Recent feed</p>
          <h2>Activity stream</h2>
        </div>
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

.overview-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.1fr) minmax(22rem, 0.9fr);
  gap: var(--space-3);
}

.timeline-panel {
  display: grid;
  gap: 1.2rem;
}

.timeline-header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.timeline-eyebrow {
  margin: 0 0 0.35rem;
  color: var(--nook-muted);
  font-size: 0.78rem;
}

.timeline-header h2 {
  margin: 0;
  font-size: 1.2rem;
}

@media (max-width: 980px) {
  .overview-grid {
    grid-template-columns: 1fr;
  }
}
</style>
