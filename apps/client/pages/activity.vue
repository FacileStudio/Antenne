<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";

import ActivityChart from "../components/ActivityChart.vue";
import ActivityOverview from "../components/ActivityOverview.vue";
import EventTimeline from "../components/EventTimeline.vue";
import type { EventLogEntry, EventLogStats } from "../types";
import { useDashboardState } from "../composables/useDashboardState";

const { loadEvents, notifySuccess, notifyError, settings } = useDashboardState();

const events = ref<EventLogEntry[]>([]);
const stats = ref<EventLogStats | null>(null);
const loading = ref(false);
const searchQuery = ref("");
const sourceFilter = ref("");
let eventSource: EventSource | null = null;

const filteredEvents = computed(() => {
  let result = events.value;
  if (sourceFilter.value) {
    result = result.filter((e) => e.source === sourceFilter.value);
  }
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase();
    result = result.filter((e) =>
      e.event?.toLowerCase().includes(q) ||
      e.providerName?.toLowerCase().includes(q) ||
      (e.message ?? "").toLowerCase().includes(q)
    );
  }
  return result;
});

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

async function handleReplay(eventId: string) {
  const targets = settings.delivery?.items ?? [];
  if (targets.length === 0) {
    notifyError("No delivery targets configured");
    return;
  }
  const target = targets.find((t) => t.enabled) ?? targets[0]!;
  try {
    await fetch(`/api/events/${eventId}/replay`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ targetId: target.id }),
    });
    notifySuccess("Event replayed");
  } catch {
    notifyError("Replay failed");
  }
}

onMounted(() => {
  void refresh();

  eventSource = new EventSource("/api/events/stream");
  eventSource.onmessage = (event) => {
    try {
      const entry = JSON.parse(event.data) as EventLogEntry;
      events.value = [entry, ...events.value].slice(0, 100);
    } catch {}
  };
});

onUnmounted(() => {
  eventSource?.close();
  eventSource = null;
});
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
      <div class="search-bar">
        <input v-model="searchQuery" type="text" placeholder="Search events..." class="search-input">
        <select v-model="sourceFilter" class="source-select">
          <option value="">All sources</option>
          <option value="webhook">Webhook</option>
          <option value="website">Website</option>
          <option value="imap">Inbox</option>
          <option value="system">System</option>
          <option value="test">Test</option>
        </select>
      </div>
      <EventTimeline :events="filteredEvents" @replay="handleReplay" />
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

.search-bar {
  display: flex;
  gap: 0.65rem;
}

.search-input {
  flex: 1;
  padding: 0.55rem 0.85rem;
  border: 1px solid var(--nook-border);
  border-radius: var(--radius-lg);
  background: var(--nook-surface);
  color: var(--nook-text);
  font-size: 0.88rem;
}

.search-input::placeholder {
  color: var(--nook-muted);
}

.source-select {
  padding: 0.55rem 0.85rem;
  border: 1px solid var(--nook-border);
  border-radius: var(--radius-lg);
  background: var(--nook-surface);
  color: var(--nook-text);
  font-size: 0.88rem;
}

@media (max-width: 980px) {
  .overview-grid {
    grid-template-columns: 1fr;
  }
}
</style>
