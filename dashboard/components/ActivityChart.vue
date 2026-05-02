<script setup lang="ts">
import { computed } from "vue";

import type { EventLogEntry } from "../types";

const props = defineProps<{
  events: EventLogEntry[];
}>();

type Bucket = {
  key: string;
  label: string;
  count: number;
  errorCount: number;
};

const buckets = computed<Bucket[]>(() => {
  const now = new Date();
  now.setMinutes(0, 0, 0);

  const items = Array.from({ length: 24 }, (_, index) => {
    const bucketDate = new Date(now);
    bucketDate.setHours(now.getHours() - (23 - index));
    return {
      key: bucketDate.toISOString(),
      label: bucketDate.toLocaleTimeString([], { hour: "2-digit" }),
      count: 0,
      errorCount: 0,
    };
  });

  const indexByHour = new Map(items.map((bucket, index) => [bucket.key.slice(0, 13), index]));

  for (const event of props.events) {
    const eventDate = new Date(event.timestamp);
    eventDate.setMinutes(0, 0, 0);
    const key = eventDate.toISOString().slice(0, 13);
    const index = indexByHour.get(key);
    if (index === undefined) {
      continue;
    }

    items[index]!.count += 1;
    if (event.deliveries.some((delivery) => delivery.status === "error")) {
      items[index]!.errorCount += 1;
    }
  }

  return items;
});

const maxCount = computed(() => Math.max(...buckets.value.map((bucket) => bucket.count), 1));

const peakBucket = computed(() =>
  buckets.value.reduce((best, bucket) => bucket.count > best.count ? bucket : best, buckets.value[0] ?? { key: "", label: "--", count: 0, errorCount: 0 }),
);

const recentCount = computed(() => buckets.value.slice(-6).reduce((sum, bucket) => sum + bucket.count, 0));

function barHeight(count: number) {
  return `${Math.max((count / maxCount.value) * 100, count > 0 ? 10 : 4)}%`;
}

function errorHeight(bucket: Bucket) {
  if (bucket.count === 0 || bucket.errorCount === 0) {
    return "0%";
  }

  return `${Math.max((bucket.errorCount / bucket.count) * 100, 18)}%`;
}
</script>

<template>
  <section class="chart-card panel">
    <div class="chart-header">
      <div>
        <p class="chart-eyebrow">Last 24 hours</p>
        <h3>Activity pulse</h3>
      </div>

      <div class="chart-summary">
        <span>{{ recentCount }} events in 6h</span>
        <span>Peak {{ peakBucket.count }} at {{ peakBucket.label }}</span>
      </div>
    </div>

    <div class="chart-bars" aria-label="Activity graph">
      <div
        v-for="bucket in buckets"
        :key="bucket.key"
        class="chart-column"
        :title="`${bucket.label}: ${bucket.count} events${bucket.errorCount ? `, ${bucket.errorCount} with errors` : ''}`"
      >
        <div class="chart-bar-track">
          <div class="chart-bar" :style="{ height: barHeight(bucket.count) }">
            <div v-if="bucket.errorCount > 0" class="chart-bar-error" :style="{ height: errorHeight(bucket) }" />
          </div>
        </div>
      </div>
    </div>

    <div class="chart-footer">
      <div class="chart-legend">
        <span class="legend-swatch legend-swatch--activity" />
        <span>Events</span>
      </div>
      <div class="chart-legend">
        <span class="legend-swatch legend-swatch--error" />
        <span>Error hit</span>
      </div>
      <div class="chart-labels">
        <span>{{ buckets[0]?.label }}</span>
        <span>{{ buckets[6]?.label }}</span>
        <span>{{ buckets[12]?.label }}</span>
        <span>{{ buckets[18]?.label }}</span>
        <span>{{ buckets[23]?.label }}</span>
      </div>
    </div>
  </section>
</template>

<style scoped>
.chart-card {
  display: grid;
  gap: 1.2rem;
  padding: 1.5rem;
  background: var(--nook-surface);
}

.chart-header {
  display: flex;
  justify-content: space-between;
  gap: var(--space-3);
  align-items: flex-start;
}

.chart-eyebrow {
  margin: 0 0 0.35rem;
  color: var(--nook-muted);
  font-size: 0.78rem;
}

h3 {
  margin: 0;
  font-size: 1.15rem;
}

.chart-summary {
  display: grid;
  gap: 0.25rem;
  text-align: right;
  color: var(--nook-muted);
  font-size: 0.84rem;
}

.chart-bars {
  display: grid;
  grid-template-columns: repeat(24, minmax(0, 1fr));
  gap: 0.45rem;
  align-items: end;
  min-height: 11rem;
}

.chart-column,
.chart-bar-track {
  height: 100%;
}

.chart-bar-track {
  border-radius: 999px;
  background: rgb(255 255 255 / 0.01);
  display: flex;
  align-items: flex-end;
  overflow: hidden;
}

.chart-bar {
  width: 100%;
  min-height: 0.25rem;
  border-radius: 999px;
  background: #1cdb81;
  position: relative;
  overflow: hidden;
}

.chart-bar-error {
  position: absolute;
  inset: auto 0 0;
  background: rgb(255 131 131 / 0.92);
}

.chart-footer {
  display: grid;
  grid-template-columns: auto auto 1fr;
  gap: 1rem;
  align-items: center;
}

.chart-legend {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  color: var(--nook-muted);
  font-size: 0.82rem;
}

.legend-swatch {
  width: 0.75rem;
  height: 0.75rem;
  border-radius: 999px;
  flex: none;
}

.legend-swatch--activity {
  background: #1cdb81;
}

.legend-swatch--error {
  background: rgb(255 131 131);
}

.chart-labels {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
  color: var(--nook-muted);
  font-size: 0.76rem;
}

@media (max-width: 720px) {
  .chart-header,
  .chart-footer {
    display: grid;
    grid-template-columns: 1fr;
  }

  .chart-summary {
    text-align: left;
  }

  .chart-bars {
    gap: 0.3rem;
    min-height: 9rem;
  }
}
</style>
