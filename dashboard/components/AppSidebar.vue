<script setup lang="ts">
const route = useRoute();

defineProps<{
  sections: ReadonlyArray<{ href: string; label: string }>;
  infoItems: Array<{ label: string; value: string }>;
}>();
</script>

<template>
  <aside class="sidebar">
    <div class="sidebar-brand">
      <div class="sidebar-mark" aria-hidden="true">
        <div class="sidebar-mark-inner" />
      </div>
      <div>
        <p class="sidebar-kicker">Nook</p>
        <h1 class="sidebar-title">Settings</h1>
      </div>
    </div>

    <nav class="sidebar-nav" aria-label="Settings sections">
      <NuxtLink
        v-for="section in sections"
        :key="section.href"
        class="sidebar-link"
        :class="{ active: route.path === section.href }"
        :to="section.href"
      >
        <span>{{ section.label }}</span>
      </NuxtLink>
    </nav>

    <div class="sidebar-info">
      <div
        v-for="item in infoItems"
        :key="item.label"
        class="sidebar-info-item"
      >
        <span>{{ item.label }}</span>
        <strong>{{ item.value }}</strong>
      </div>
    </div>

    <div class="sidebar-footer">
      <slot name="footer" />
    </div>
  </aside>
</template>

<style scoped>
.sidebar {
  position: sticky;
  top: 1.5rem;
  display: grid;
  gap: var(--space-8);
  padding: var(--space-7);
  border-radius: 1.75rem;
  border: 1px solid var(--nook-border);
  background: var(--nook-surface);
  box-shadow: inset 0 0 0 1px rgb(255 255 255 / 0.02);
}

.sidebar-brand {
  display: flex;
  align-items: center;
  gap: var(--space-4);
}

.sidebar-mark {
  width: 2.75rem;
  height: 2.75rem;
  display: grid;
  place-items: center;
  border-radius: 1rem;
  background: var(--nook-text);
}

.sidebar-mark-inner {
  width: 0.8rem;
  height: 0.8rem;
  background: var(--nook-bg);
  transform: rotate(45deg);
}

.sidebar-kicker {
  margin: 0 0 0.2rem;
  font-size: 0.75rem;
  color: var(--nook-muted);
}

.sidebar-title {
  margin: 0;
  font-size: 1.1rem;
}

.sidebar-nav {
  display: grid;
  gap: 0.45rem;
}

.sidebar-link {
  display: flex;
  align-items: center;
  min-height: 3rem;
  padding: 0.85rem 1rem;
  border-radius: 1rem;
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

.sidebar-info {
  display: grid;
  gap: var(--space-4);
  padding-top: var(--space-5);
  border-top: 1px solid var(--nook-border);
}

.sidebar-footer {
  padding-top: var(--space-5);
  border-top: 1px solid var(--nook-border);
}

.sidebar-info-item {
  display: grid;
  gap: 0.35rem;
}

.sidebar-info-item span {
  color: var(--nook-muted);
  font-size: 0.8rem;
}

@media (max-width: 960px) {
  .sidebar {
    position: static;
  }
}
</style>
