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

function connectSSE() {
  try {
    eventSource = new EventSource("/api/events/stream");
    eventSource.onmessage = (event) => {
      try {
        const entry = JSON.parse(event.data) as EventLogEntry;
        events.value = [entry, ...events.value].slice(0, 100);
      } catch {}
    };
    eventSource.onerror = () => {
      eventSource?.close();
      eventSource = null;
    };
  } catch {}
}

onMounted(() => {
  void refresh();
  connectSSE();
});

onUnmounted(() => {
  eventSource?.close();
  eventSource = null;
});
</script>

<template>
  <div class="grid gap-5">
    <div class="grid grid-cols-[minmax(0,1.1fr)_minmax(22rem,0.9fr)] gap-3 max-lg:grid-cols-1">
      <ActivityOverview :events="events" :stats="stats" :loading="loading" @refresh="refresh" />
      <ActivityChart :events="events" />
    </div>

    <div class="grid gap-5">
      <div class="flex items-center gap-3">
        <div>
          <p class="m-0 mb-1.5 text-muted-foreground text-xs">Recent feed</p>
          <h2 class="m-0 text-xl font-semibold">Activity stream</h2>
        </div>
      </div>
      <div class="flex gap-2.5">
        <input v-model="searchQuery" class="flex-1 min-w-0 w-auto" type="text" placeholder="Search events...">
        <select v-model="sourceFilter" class="w-auto flex-none">
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
