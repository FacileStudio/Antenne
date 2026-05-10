<script setup lang="ts">
import type { Component } from "vue";
import { Icon } from "@iconify/vue";

const route = useRoute();

defineProps<{
  sections: ReadonlyArray<{ href: string; label: string; icon: Component; activePaths?: string[] }>;
}>();

function isActive(section: { href: string; activePaths?: string[] }) {
  if (section.activePaths) {
    return section.activePaths.some((p) => route.path === p || route.path.startsWith(p + "/"));
  }
  return route.path === section.href || route.path.startsWith(section.href + "/");
}
</script>

<template>
  <aside class="sticky top-0 flex h-screen w-60 shrink-0 flex-col border-r border-border bg-background">
    <div class="flex items-center gap-3 px-5 pt-8 pb-6">
      <Icon icon="solar:translation-2-bold-duotone" class="shrink-0 text-[#1CDB81]" :width="32" :height="32" aria-hidden="true" />
      <span class="font-heading text-2xl font-bold tracking-tight">Nook</span>
    </div>

    <nav class="flex flex-1 flex-col gap-1 px-3" aria-label="Navigation">
      <NuxtLink
        v-for="section in sections"
        :key="section.href"
        :to="section.href"
        class="flex items-center gap-3 rounded-md px-3 py-2.5 text-sm transition-colors"
        :class="isActive(section)
          ? 'bg-foreground text-background font-medium'
          : 'text-muted-foreground hover:bg-muted hover:text-foreground'"
      >
        <component :is="section.icon" :size="16" class="shrink-0" aria-hidden="true" />
        {{ section.label }}
      </NuxtLink>
    </nav>

    <div class="border-t border-border p-4">
      <slot name="footer" />
    </div>
  </aside>
</template>
