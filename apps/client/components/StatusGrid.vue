<script setup lang="ts">
defineProps<{
  items: Array<{ label: string; value: string; pulse?: boolean }>;
  columns?: number;
}>();
</script>

<template>
  <div class="grid gap-4" :class="{
    'grid-cols-2 md:grid-cols-3 lg:grid-cols-5': !columns || columns === 5,
    'grid-cols-2 md:grid-cols-3': columns === 3,
    'grid-cols-2': columns === 2,
    'grid-cols-1 md:grid-cols-4': columns === 4,
  }">
    <div
      v-for="item in items"
      :key="item.label"
      class="flex flex-col gap-1 rounded-xl bg-card px-4 py-3 ring-1 ring-foreground/10"
    >
      <span class="text-xs text-muted-foreground">{{ item.label }}</span>
      <span class="text-2xl font-semibold tabular-nums flex items-center gap-2">
        <template v-if="item.pulse">
          <span class="relative flex h-2 w-2">
            <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-green-400 opacity-75"></span>
            <span class="relative inline-flex h-2 w-2 rounded-full bg-green-500"></span>
          </span>
        </template>
        {{ item.value }}
      </span>
    </div>
  </div>
</template>
