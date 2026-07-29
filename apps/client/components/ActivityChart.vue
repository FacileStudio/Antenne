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

const halfCount = computed(() => Math.ceil(maxCount.value / 2));

const peakBucket = computed(() =>
  buckets.value.reduce((best, bucket) => bucket.count > best.count ? bucket : best, buckets.value[0] ?? { key: "", label: "--", count: 0, errorCount: 0 }),
);

const recentCount = computed(() => buckets.value.slice(-6).reduce((sum, bucket) => sum + bucket.count, 0));

function barHeight(count: number) {
  return `${(count / maxCount.value) * 100}%`;
}

function errorHeight(bucket: Bucket) {
  if (bucket.count === 0 || bucket.errorCount === 0) {
    return "0%";
  }

  return `${(bucket.errorCount / bucket.count) * 100}%`;
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
        <span>{{ recentCount }} {{ recentCount === 1 ? "event" : "events" }} in 6h</span>
        <span>Peak {{ peakBucket.count }} at {{ peakBucket.label }}</span>
      </div>
    </div>

    <div class="chart-plot" aria-label="Hourly event volume for the last 24 hours">
      <div class="chart-gridline chart-gridline--max">
        <span class="chart-gridline-label">{{ maxCount }}</span>
      </div>
      <div v-if="halfCount < maxCount" class="chart-gridline chart-gridline--half">
        <span class="chart-gridline-label">{{ halfCount }}</span>
      </div>

      <div class="chart-bars">
        <div
          v-for="bucket in buckets"
          :key="bucket.key"
          class="chart-column"
        >
          <div
            v-if="bucket.count > 0"
            class="chart-bar"
            :style="{ height: barHeight(bucket.count) }"
          >
            <div v-if="bucket.errorCount > 0" class="chart-bar-error" :style="{ height: errorHeight(bucket) }" />
          </div>

          <div class="chart-tip" role="tooltip">
            <span class="chart-tip-time">{{ bucket.label }}</span>
            <span>{{ bucket.count }} {{ bucket.count === 1 ? "event" : "events" }}</span>
            <span v-if="bucket.errorCount > 0" class="chart-tip-error">{{ bucket.errorCount }} with errors</span>
          </div>
        </div>
      </div>
    </div>

    <div class="chart-labels">
      <span>{{ buckets[0]?.label }}</span>
      <span>{{ buckets[6]?.label }}</span>
      <span>{{ buckets[12]?.label }}</span>
      <span>{{ buckets[18]?.label }}</span>
      <span>{{ buckets[23]?.label }}</span>
    </div>

    <div class="chart-footer">
      <div class="chart-legend">
        <span class="legend-swatch legend-swatch--activity" />
        <span>Events</span>
      </div>
      <div class="chart-legend">
        <span class="legend-swatch legend-swatch--error" />
        <span>With errors</span>
      </div>
    </div>
  </section>
</template>

<style scoped>
.chart-card {
  --chart-bar: var(--success);
  --chart-error: var(--destructive);
  display: grid;
  gap: 1rem;
  padding: 1.5rem;
  background: var(--nook-surface);
  border-radius: var(--radius-xl);
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

.chart-plot {
  position: relative;
  min-height: 11rem;
  padding-top: 0.9rem;
}

.chart-gridline {
  position: absolute;
  left: 0;
  right: 0;
  border-top: 1px dashed color-mix(in oklab, var(--nook-border) 70%, transparent);
  pointer-events: none;
}

.chart-gridline--max {
  top: 0.9rem;
}

.chart-gridline--half {
  top: calc(0.9rem + (100% - 0.9rem) / 2);
}

.chart-gridline-label {
  position: absolute;
  right: 0;
  bottom: 2px;
  font-size: 0.68rem;
  color: var(--nook-muted);
  font-variant-numeric: tabular-nums;
}

.chart-bars {
  display: grid;
  grid-template-columns: repeat(24, minmax(0, 1fr));
  gap: 0.4rem;
  align-items: end;
  height: 100%;
  min-height: calc(11rem - 0.9rem);
  border-bottom: 1px solid var(--nook-border);
}

.chart-column {
  position: relative;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  height: 100%;
}

.chart-column:hover .chart-bar {
  filter: brightness(1.08);
}

.chart-bar {
  width: 100%;
  min-height: 3px;
  border-radius: 4px 4px 0 0;
  background: var(--chart-bar);
  position: relative;
  overflow: hidden;
}

.chart-bar-error {
  position: absolute;
  inset: auto 0 0;
  min-height: 2px;
  background: var(--chart-error);
}

.chart-tip {
  position: absolute;
  bottom: calc(100% + 6px);
  left: 50%;
  transform: translateX(-50%);
  display: none;
  flex-direction: column;
  gap: 0.1rem;
  padding: 0.4rem 0.6rem;
  background: var(--card);
  border: 1px solid var(--nook-border);
  border-radius: var(--radius-md);
  box-shadow: 0 4px 12px oklch(0 0 0 / 0.08);
  font-size: 0.74rem;
  white-space: nowrap;
  z-index: 5;
  pointer-events: none;
}

.chart-column:hover .chart-tip {
  display: flex;
}

.chart-column:nth-child(-n+3) .chart-tip {
  left: 0;
  transform: none;
}

.chart-column:nth-last-child(-n+3) .chart-tip {
  left: auto;
  right: 0;
  transform: none;
}

.chart-tip-time {
  color: var(--nook-muted);
}

.chart-tip-error {
  color: var(--chart-error);
}

.chart-labels {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
  color: var(--nook-muted);
  font-size: 0.76rem;
}

.chart-footer {
  display: flex;
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
  width: 0.7rem;
  height: 0.7rem;
  border-radius: 3px;
  flex: none;
}

.legend-swatch--activity {
  background: var(--chart-bar);
}

.legend-swatch--error {
  background: var(--chart-error);
}

@media (max-width: 720px) {
  .chart-header {
    display: grid;
    grid-template-columns: 1fr;
  }

  .chart-summary {
    text-align: left;
  }

  .chart-bars {
    gap: 0.25rem;
    min-height: 8rem;
  }

  .chart-plot {
    min-height: 9rem;
  }
}
</style>
