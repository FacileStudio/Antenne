<script setup lang="ts">
import { ref, onMounted } from "vue";
import { Icon } from "@iconify/vue";
import { useSpaces } from "../composables/useSpaces";

const { spaces, currentSpace, loading, loadSpaces, switchSpace } = useSpaces();

const open = ref(false);

onMounted(() => {
  loadSpaces();
});

function toggle() {
  open.value = !open.value;
}

function select(id: string) {
  switchSpace(id);
  open.value = false;
}

function onClickOutside(event: MouseEvent) {
  const target = event.target as HTMLElement;
  if (!target.closest(".space-switcher")) {
    open.value = false;
  }
}

onMounted(() => {
  document.addEventListener("click", onClickOutside);
});
</script>

<template>
  <div class="space-switcher">
    <button
      type="button"
      class="space-trigger"
      :disabled="loading"
      @click.stop="toggle"
    >
      <Icon icon="solar:buildings-2-linear" class="shrink-0" :width="16" :height="16" aria-hidden="true" />
      <span class="space-name">{{ currentSpace?.name ?? "Espace" }}</span>
      <Icon
        icon="solar:alt-arrow-down-linear"
        class="shrink-0 chevron"
        :class="{ 'chevron-open': open }"
        :width="12"
        :height="12"
        aria-hidden="true"
      />
    </button>

    <div v-if="open" class="space-dropdown">
      <div class="space-dropdown-header">Espaces</div>
      <button
        v-for="space in spaces"
        :key="space.id"
        type="button"
        class="space-option"
        :class="{ 'space-option-active': space.id === currentSpace?.id }"
        @click="select(space.id)"
      >
        <Icon icon="solar:buildings-2-linear" class="shrink-0" :width="14" :height="14" aria-hidden="true" />
        <span>{{ space.name }}</span>
      </button>

      <div v-if="spaces.length === 0 && !loading" class="space-empty">
        No spaces yet
      </div>

      <div class="space-dropdown-footer">
        <NuxtLink to="/spaces" class="space-manage-link" @click="open = false">
          <Icon icon="solar:settings-linear" :width="14" :height="14" aria-hidden="true" />
          Manage spaces
        </NuxtLink>
      </div>
    </div>
  </div>
</template>

<style scoped>
.space-switcher {
  position: relative;
}

.space-trigger {
  display: flex;
  width: 100%;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  border-radius: var(--radius-lg);
  border: 1px solid var(--sidebar-border);
  background: transparent;
  color: var(--sidebar-foreground);
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: background 150ms, border-color 150ms;
  height: auto;
}

.space-trigger:hover {
  background: var(--sidebar-accent);
  border-color: color-mix(in oklab, var(--sidebar-border) 70%, var(--foreground));
}

.space-name {
  flex: 1;
  text-align: left;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chevron {
  transition: transform 150ms;
}

.chevron-open {
  transform: rotate(180deg);
}

.space-dropdown {
  position: absolute;
  bottom: calc(100% + 0.5rem);
  left: 0;
  right: 0;
  z-index: 50;
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  box-shadow: 0 8px 24px oklch(0 0 0 / 12%);
  overflow: hidden;
}

.space-dropdown-header {
  padding: 0.625rem 0.875rem 0.375rem;
  font-size: 0.72rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--muted-foreground);
}

.space-option {
  display: flex;
  width: 100%;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.875rem;
  border: none;
  border-radius: 0;
  background: transparent;
  color: var(--foreground);
  font-size: 0.84rem;
  cursor: pointer;
  transition: background 120ms;
  height: auto;
}

.space-option:hover {
  background: var(--muted);
}

.space-option-active {
  background: var(--accent);
  font-weight: 500;
}

.space-empty {
  padding: 0.75rem 0.875rem;
  font-size: 0.84rem;
  color: var(--muted-foreground);
  text-align: center;
}

.space-dropdown-footer {
  border-top: 1px solid var(--border);
  padding: 0.375rem;
}

.space-manage-link {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.625rem;
  border-radius: var(--radius-md);
  font-size: 0.84rem;
  color: var(--muted-foreground);
  text-decoration: none;
  transition: background 120ms, color 120ms;
}

.space-manage-link:hover {
  background: var(--muted);
  color: var(--foreground);
}
</style>
