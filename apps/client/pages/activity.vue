<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue";

import ActivityChart from "../components/ActivityChart.vue";
import ActivityOverview from "../components/ActivityOverview.vue";
import EventTimeline from "../components/EventTimeline.vue";
import type { EventLogEntry, EventLogStats } from "../types";
import { useDashboardState } from "../composables/useDashboardState";

const { loadEvents, notifySuccess, notifyError, settings } = useDashboardState();

const PAGE_SIZE = 50;

const events = ref<EventLogEntry[]>([]);
const stats = ref<EventLogStats | null>(null);
const total = ref(0);
const serverOffset = ref(0);
const loading = ref(false);
const loadingMore = ref(false);
const searchQuery = ref("");
const sourceFilter = ref("");
let searchTimer: ReturnType<typeof setTimeout> | null = null;
let eventSource: EventSource | null = null;

const hasMore = computed(() => serverOffset.value < total.value);

async function fetchEvents(options: { append?: boolean } = {}) {
  const append = options.append ?? false;
  if (append) {
    loadingMore.value = true;
  } else {
    loading.value = true;
  }

  try {
    const data = await loadEvents({
      limit: PAGE_SIZE,
      offset: append ? serverOffset.value : 0,
      q: searchQuery.value || undefined,
      source: sourceFilter.value || undefined,
    });

    if (append) {
      events.value = [...events.value, ...data.events];
    } else {
      events.value = data.events;
    }
    serverOffset.value = append ? serverOffset.value + data.events.length : data.events.length;
    total.value = data.total;
    stats.value = data.stats;
  } finally {
    loading.value = false;
    loadingMore.value = false;
  }
}

function onSearchInput() {
  if (searchTimer) clearTimeout(searchTimer);
  searchTimer = setTimeout(() => void fetchEvents(), 300);
}

watch(sourceFilter, () => void fetchEvents());

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
        if (searchQuery.value) return;
        if (sourceFilter.value && entry.source !== sourceFilter.value) return;
        events.value = [entry, ...events.value];
        total.value += 1;
      } catch {}
    };
    eventSource.onerror = () => {
      eventSource?.close();
      eventSource = null;
    };
  } catch {}
}

onMounted(() => {
  void fetchEvents();
  connectSSE();
});

onUnmounted(() => {
  eventSource?.close();
  eventSource = null;
  if (searchTimer) clearTimeout(searchTimer);
});
</script>

<template>
  <div class="grid gap-8">
    <div class="grid gap-5">
      <ActivityOverview :events="events" :stats="stats" :loading="loading" @refresh="fetchEvents()" />
      <ActivityChart :events="events" />
    </div>

    <div class="grid gap-5">
      <div>
        <p class="m-0 mb-1.5 text-muted-foreground text-xs">Recent feed</p>
        <h2 class="m-0 text-xl font-semibold">Activity stream</h2>
      </div>

      <div class="flex gap-2.5">
        <input v-model="searchQuery" class="!flex-1 min-w-0 !w-auto" type="text" placeholder="Search events..." @input="onSearchInput">
        <select v-model="sourceFilter" class="!w-auto flex-none">
          <option value="">All sources</option>
          <option value="webhook">Webhook</option>
          <option value="website">Website</option>
          <option value="imap">Inbox</option>
          <option value="rss">RSS</option>
          <option value="system">System</option>
          <option value="test">Test</option>
        </select>
      </div>

      <EventTimeline :events="events" @replay="handleReplay" />

      <button
        v-if="hasMore"
        class="w-full min-h-10 rounded-xl border border-dashed border-border text-sm text-muted-foreground hover:bg-muted hover:text-foreground transition-colors disabled:opacity-50 disabled:pointer-events-none"
        type="button"
        :disabled="loadingMore"
        @click="fetchEvents({ append: true })"
      >
        {{ loadingMore ? "Loading\u2026" : "Load more" }}
      </button>
    </div>
  </div>
</template>
