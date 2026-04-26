<script setup lang="ts">
import type { Component } from "vue";

const route = useRoute();

defineProps<{
  sections: ReadonlyArray<{ href: string; label: string; icon: Component }>;
}>();
</script>

<template>
  <aside class="sidebar">
    <nav class="sidebar-nav" aria-label="Settings sections">
      <NuxtLink
        v-for="section in sections"
        :key="section.href"
        class="sidebar-link"
        :class="{ active: route.path === section.href }"
        :to="section.href"
      >
        <component :is="section.icon" :size="18" aria-hidden="true" />
        <span>{{ section.label }}</span>
      </NuxtLink>
    </nav>

    <div class="sidebar-footer">
      <slot name="footer" />
    </div>
  </aside>
</template>

<style scoped>
.sidebar {
  display: grid;
  align-content: start;
  gap: var(--space-5);
  padding: var(--space-6) var(--space-4);
  position: sticky;
  top: 0;
  height: 100vh;
  border-right: 1px solid var(--nook-border);
}

.sidebar-nav {
  display: grid;
  gap: 0.45rem;
}

.sidebar-link {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  min-height: 3rem;
  padding: 0.85rem 1.15rem;
  border-radius: 9999px;
  border: 1px solid transparent;
  color: var(--nook-muted);
  text-decoration: none;
  transition: background-color 150ms ease, border-color 150ms ease, color 150ms ease;
}

.sidebar-link:hover {
  background: var(--nook-surface-strong);
  border-color: var(--nook-border);
  color: var(--nook-text);
}

.sidebar-link.active {
  background: var(--nook-text);
  border-color: var(--nook-text);
  color: var(--nook-bg);
}

.sidebar-footer {
  margin-top: auto;
  padding-top: var(--space-5);
  border-top: 1px solid var(--nook-border);
}

@media (max-width: 960px) {
  .sidebar {
    position: static;
    height: auto;
    padding: var(--space-4);
    border-right: 0;
    border-bottom: 1px solid var(--nook-border);
  }
}
</style>
