<script setup lang="ts">
import { Handle, Position } from "@vue-flow/core";
import { Icon } from "@iconify/vue";
import ToggleSwitch from "./ToggleSwitch.vue";

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
    <Icon :icon="props.data.icon" :width="20" :height="20" class="flow-card-icon" aria-hidden="true" />
    <div class="flow-card-body">
      <span class="flow-card-name">{{ props.data.name }}</span>
      <span class="flow-card-label">{{ props.data.label }}</span>
    </div>
    <ToggleSwitch
      :model-value="props.data.enabled"
      size="sm"
      class="nodrag"
      @update:model-value="props.data.onToggle(props.data.nodeId)"
    />
    <Handle type="source" :position="Position.Right" class="flow-handle flow-handle--source" />
  </div>
</template>

<style scoped>
.flow-card {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  width: 240px;
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
