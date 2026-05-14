<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { Icon } from "@iconify/vue";

import EventTimeline from "../../components/EventTimeline.vue";
import { getProviderLabel } from "../../types";
import type { EventLogEntry, EventLogStats } from "../../types";
import { createTagToneMap, getTagTone } from "../../utils/provider-tags";
import { useDashboardState } from "../../composables/useDashboardState";

const route = useRoute();
const providerId = computed(() => route.params.id as string);

const { settings, loadSettings, loadEvents } = useDashboardState();

const events = ref<EventLogEntry[]>([]);
const stats = ref<EventLogStats | null>(null);
const loading = ref(false);

const provider = computed(() => settings.providers.items.find((p) => p.id === providerId.value));
const tagToneMap = computed(() => createTagToneMap(settings.providers.items.flatMap((item) => item.tags)));

async function refresh() {
  loading.value = true;
  try {
    const data = await loadEvents({ providerId: providerId.value, limit: 100 });
    events.value = data.events;
    stats.value = data.stats;
  } finally {
    loading.value = false;
  }
}

onMounted(async () => {
  await loadSettings();
  await refresh();
});
</script>

<template>
  <div class="detail-page">
    <NuxtLink to="/providers" class="back-link">
      <Icon icon="solar:arrow-left-linear" :width="16" :height="16" aria-hidden="true" />
      Back to providers
    </NuxtLink>

    <div v-if="provider" class="detail-header">
      <h2>{{ provider.name }}</h2>
      <span class="detail-badge">{{ getProviderLabel(provider.type) }}</span>
    </div>

    <div v-if="provider && provider.tags.length > 0" class="detail-tags">
      <span
        v-for="tag in provider.tags"
        :key="tag"
        class="tag-pill tag-pill--filled detail-tag"
        :data-tag-tone="getTagTone(tag, tagToneMap)"
      >
        {{ tag }}
      </span>
    </div>

    <div v-if="stats" class="stats-row">
      <div class="stat-card panel">
        <span class="stat-value">{{ stats.totalEvents }}</span>
        <span class="stat-label">Events</span>
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
          <Icon icon="solar:refresh-linear" :width="14" :height="14" :class="{ spinning: loading }" aria-hidden="true" />
        </button>
      </div>
      <EventTimeline :events="events" />
    </div>
  </div>
</template>

<style scoped>
.detail-page {
  display: grid;
  gap: var(--space-4);
}

.back-link {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  color: var(--nook-muted);
  text-decoration: none;
  font-size: 0.9rem;
  transition: color 150ms ease;
}

.back-link:hover {
  color: var(--nook-text);
}

.detail-header {
  display: flex;
  align-items: baseline;
  gap: var(--space-3);
}

.detail-header h2 {
  margin: 0;
}

.detail-badge {
  font-size: 0.75rem;
  padding: 0.1rem 0.5rem;
  border-radius: var(--radius-pill);
  border: 1px solid var(--nook-border);
  background: var(--nook-surface);
  color: var(--nook-muted);
}

.detail-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.55rem;
}

.detail-tag {
  min-height: 1.7rem;
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
  font-family: "Goga";
  font-size: 2rem;
  font-weight: 700;
  letter-spacing: 0;
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
  .stats-row { grid-template-columns: 1fr; }
}
</style>
