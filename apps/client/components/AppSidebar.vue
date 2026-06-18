<script setup lang="ts">
import { Icon } from "@iconify/vue";
import SpaceSwitcher from "./SpaceSwitcher.vue";

const route = useRoute();

defineProps<{
  sections: ReadonlyArray<{ href: string; label: string; icon: string; activePaths?: string[] }>;
}>();

function isActive(section: { href: string; activePaths?: string[] }) {
  if (section.activePaths) {
    return section.activePaths.some((p) => route.path === p || route.path.startsWith(p + "/"));
  }
  return route.path === section.href || route.path.startsWith(section.href + "/");
}
</script>

<template>
  <aside class="sticky top-0 flex h-screen w-60 shrink-0 flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground">
    <div class="flex items-center gap-3 px-5 pt-8 pb-6">
      <Icon icon="solar:translation-2-bold-duotone" class="shrink-0 text-foreground" :width="28" :height="28" aria-hidden="true" />
      <span class="font-heading text-2xl font-bold">Nook</span>
    </div>

    <div class="px-3 pb-3">
      <SpaceSwitcher />
    </div>

    <nav class="flex flex-1 flex-col gap-1 px-3" aria-label="Navigation">
      <NuxtLink
        v-for="section in sections"
        :key="section.href"
        :to="section.href"
        class="flex items-center gap-3 rounded-md px-3 py-2.5 text-sm transition-colors"
        :class="isActive(section)
          ? 'bg-sidebar-primary text-sidebar-primary-foreground font-medium'
          : 'text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground'"
      >
        <Icon :icon="section.icon" class="shrink-0" :width="16" :height="16" aria-hidden="true" />
        {{ section.label }}
      </NuxtLink>
    </nav>

    <div class="border-t border-sidebar-border p-4">
      <slot name="footer" />
    </div>
  </aside>
</template>
