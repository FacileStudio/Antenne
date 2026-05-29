<script setup lang="ts">
import { Handle, Position } from "@vue-flow/core";
import { Icon } from "@iconify/vue";

const props = defineProps<{
  data: {
    icon: string;
    name: string;
    label: string;
    enabled: boolean;
    nodeId: string;
    onToggle: (id: string) => void;
  };
}>();
</script>

<template>
  <div class="flow-card" :class="{ 'flow-card--disabled': !props.data.enabled }">
    <Handle type="target" :position="Position.Left" class="flow-handle flow-handle--target" />
    <Icon :icon="props.data.icon" :width="20" :height="20" class="flow-card-icon" aria-hidden="true" />
    <div class="flow-card-body">
      <span class="flow-card-name">{{ props.data.name }}</span>
      <span class="flow-card-label">{{ props.data.label }}</span>
    </div>
    <button
      class="flow-card-toggle nodrag"
      :class="props.data.enabled ? 'flow-card-toggle--on' : 'flow-card-toggle--off'"
      :title="props.data.enabled ? 'Disable' : 'Enable'"
      @click.stop="props.data.onToggle(props.data.nodeId)"
    />
  </div>
</template>

<style scoped>
.flow-card {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  width: 220px;
  height: 56px;
  padding: 0 1rem;
  border: 1px solid var(--nook-border);
  border-radius: var(--radius-xl);
  background: var(--card);
  cursor: grab;
  transition: border-color 150ms, opacity 150ms;
  user-select: none;
}

.flow-card:hover {
  border-color: color-mix(in oklab, var(--nook-border) 60%, var(--foreground));
}

.flow-card--disabled .flow-card-icon,
.flow-card--disabled .flow-card-name {
  opacity: 0.4;
}

.flow-card-icon {
  flex-shrink: 0;
  color: var(--nook-muted);
}

.flow-card-body {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  min-width: 0;
  flex: 1;
}

.flow-card-name {
  font-size: 0.84rem;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.flow-card-label {
  font-size: 0.68rem;
  color: var(--nook-muted);
}

.flow-card-toggle {
  width: 0.5rem;
  height: 0.5rem;
  border-radius: 50%;
  flex-shrink: 0;
  border: none;
  padding: 0;
  cursor: pointer;
  transition: background 150ms;
}

.flow-card-toggle--on {
  background: var(--success);
}

.flow-card-toggle--off {
  background: var(--nook-border);
}

.flow-handle {
  width: 8px !important;
  height: 8px !important;
  background: var(--nook-border) !important;
  border: 1.5px solid var(--card) !important;
  border-radius: 50% !important;
  transition: background 150ms;
}

.flow-handle:hover {
  background: var(--foreground) !important;
}
</style>
