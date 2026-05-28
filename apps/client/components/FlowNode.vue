<script setup lang="ts">
import { Icon } from "@iconify/vue";

defineProps<{
  icon: string;
  name: string;
  label: string;
  enabled: boolean;
  selected: boolean;
  dimmed: boolean;
}>();
</script>

<template>
  <div
    class="flow-node"
    :class="{
      'flow-node--selected': selected,
      'flow-node--dimmed': dimmed,
      'flow-node--disabled': !enabled,
    }"
  >
    <Icon :icon="icon" :width="20" :height="20" class="flow-node-icon" aria-hidden="true" />
    <div class="flow-node-body">
      <span class="flow-node-name">{{ name }}</span>
      <span class="flow-node-label">{{ label }}</span>
    </div>
    <span class="flow-node-dot" :class="enabled ? 'flow-node-dot--on' : 'flow-node-dot--off'" />
  </div>
</template>

<style scoped>
.flow-node {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  width: 220px;
  height: 60px;
  padding: 0 1rem;
  border: 1px solid var(--nook-border);
  border-radius: var(--radius-xl);
  background: var(--card);
  cursor: pointer;
  transition: border-color 150ms, opacity 150ms, box-shadow 150ms;
  user-select: none;
}

.flow-node:hover {
  border-color: color-mix(in oklab, var(--nook-border) 60%, var(--foreground));
}

.flow-node--selected {
  border-color: var(--foreground);
  box-shadow: 0 0 0 1px var(--foreground);
}

.flow-node--dimmed {
  opacity: 0.25;
}

.flow-node--disabled .flow-node-icon,
.flow-node--disabled .flow-node-name {
  opacity: 0.4;
}

.flow-node-icon {
  flex-shrink: 0;
  color: var(--nook-muted);
}

.flow-node-body {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  min-width: 0;
  flex: 1;
}

.flow-node-name {
  font-size: 0.875rem;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.flow-node-label {
  font-size: 0.72rem;
  color: var(--nook-muted);
}

.flow-node-dot {
  width: 0.5rem;
  height: 0.5rem;
  border-radius: 50%;
  flex-shrink: 0;
}

.flow-node-dot--on {
  background: var(--success);
}

.flow-node-dot--off {
  background: var(--nook-border);
}
</style>
