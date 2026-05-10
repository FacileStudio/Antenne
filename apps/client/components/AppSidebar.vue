<script setup lang="ts">
import type { Component } from "vue";

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
      <svg class="h-8 w-8 shrink-0" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" aria-hidden="true">
          <path fill="#1CDB81" d="M10.31 17.344c.767-.876 1.151-1.314 1.625-1.342q.065-.004.13 0c.474.028.858.466 1.625 1.342c1.67 1.906 2.505 2.858 2.271 3.68q-.03.107-.074.206C15.543 22 14.362 22 12 22s-3.543 0-3.887-.77a2 2 0 0 1-.074-.206c-.234-.822.6-1.774 2.27-3.68M14.5 12.5a2.5 2.5 0 1 1-5 0a2.5 2.5 0 0 1 5 0"/>
          <path fill="#1CDB81" fill-rule="evenodd" d="M12 8.035c-2.697 0-4.884 2.151-4.884 4.806a4.75 4.75 0 0 0 1.43 3.398a.68.68 0 0 1 0 .97a.706.706 0 0 1-.986 0a6.1 6.1 0 0 1-1.84-4.368c0-3.413 2.812-6.18 6.28-6.18s6.279 2.767 6.279 6.18a6.1 6.1 0 0 1-1.84 4.369a.706.706 0 0 1-.986 0a.68.68 0 0 1 0-.971a4.75 4.75 0 0 0 1.43-3.398c0-2.655-2.186-4.806-4.883-4.806" clip-rule="evenodd" opacity=".7"/>
          <path fill="#1CDB81" fill-rule="evenodd" d="M12 4.373c-4.752 0-8.605 3.791-8.605 8.468c0 2.338.963 4.454 2.52 5.987a.68.68 0 0 1 0 .97a.706.706 0 0 1-.986 0A9.73 9.73 0 0 1 2 12.842C2 7.406 6.477 3 12 3s10 4.406 10 9.84a9.73 9.73 0 0 1-2.929 6.959a.706.706 0 0 1-.987 0a.68.68 0 0 1 0-.971a8.37 8.37 0 0 0 2.52-5.987c0-4.677-3.852-8.468-8.604-8.468" clip-rule="evenodd" opacity=".4"/>
        </svg>
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
