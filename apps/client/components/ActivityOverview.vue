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
  <section class="rounded-xl bg-card ring-1 ring-foreground/10 p-5 grid gap-5">
    <div class="flex items-center justify-between gap-3 flex-wrap">
      <h2 class="m-0 text-xl font-semibold">Stats</h2>
      <div class="flex items-center gap-2.5">
        <span class="inline-flex items-center gap-1.5 rounded-full bg-muted px-2.5 py-1 text-xs text-muted-foreground ring-1 ring-foreground/10">
          <Icon icon="solar:clock-circle-linear" :width="13" :height="13" aria-hidden="true" />
          Last event {{ lastEventLabel }}
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

    <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
      <div class="flex flex-col gap-1 rounded-xl bg-muted/50 px-4 py-3">
        <span class="text-2xl font-bold tabular-nums">{{ stats?.totalEvents ?? events.length }}</span>
        <span class="text-xs text-muted-foreground">Tracked</span>
      </div>
      <div class="flex flex-col gap-1 rounded-xl bg-muted/50 px-4 py-3">
        <span class="text-2xl font-bold tabular-nums text-green-700">{{ stats?.deliverySuccessCount ?? 0 }}</span>
        <span class="text-xs text-muted-foreground">Success</span>
      </div>
      <div class="flex flex-col gap-1 rounded-xl bg-muted/50 px-4 py-3">
        <span class="text-2xl font-bold tabular-nums text-red-600">{{ stats?.deliveryErrorCount ?? 0 }}</span>
        <span class="text-xs text-muted-foreground">Failed</span>
      </div>
      <div class="flex flex-col gap-1 rounded-xl bg-muted/50 px-4 py-3">
        <span class="text-2xl font-bold tabular-nums">{{ successRate }}%</span>
        <span class="text-xs text-muted-foreground">Hit rate</span>
      </div>
    </div>
  </section>
</template>
