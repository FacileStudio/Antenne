<script setup lang="ts">
import { Icon } from "@iconify/vue";

const route = useRoute();

defineProps<{
  sections: ReadonlyArray<{ href: string; label: string; icon: string }>;
}>();
</script>

<template>
  <aside class="sidebar">
    <div class="sidebar-brand">
      <img class="brand-avatar" src="/nook-avatar.webp" alt="Nook" />
      <span class="brand-name">Nook</span>
    </div>

    <nav class="sidebar-nav" aria-label="Settings sections">
      <NuxtLink
        v-for="section in sections"
        :key="section.href"
        class="sidebar-link"
        :class="{ active: route.path === section.href }"
        :to="section.href"
      >
        <Icon :icon="section.icon" width="18" aria-hidden="true" />
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
  grid-template-rows: auto auto 1fr;
  gap: var(--space-5);
  padding: var(--space-6) var(--space-4);
  position: sticky;
  top: 0;
  height: 100vh;
  border-right: 1px solid var(--nook-border);
}

.sidebar-brand {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: 0 0.35rem;
}

.brand-avatar {
  width: 3rem;
  height: 3rem;
  border-radius: 50%;
  object-fit: cover;
  border: 2px solid var(--nook-border);
  flex: none;
}

.brand-name {
  font-family: "Goga", "Helvetica Neue", Helvetica, Arial, sans-serif;
  font-size: 2.2rem;
  font-weight: 700;
  letter-spacing: -0.04em;
  line-height: 1;
}

.sidebar-nav {
  display: grid;
  gap: 0.45rem;
}

.sidebar-link {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  height: 2rem;
  padding: 0 0.75rem;
  border-radius: var(--radius-lg);
  border: 1px solid transparent;
  color: var(--nook-muted);
  text-decoration: none;
  transition: all 200ms;
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
  align-self: end;
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
