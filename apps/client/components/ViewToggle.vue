<script setup lang="ts">
import { Icon } from "@iconify/vue";

const props = defineProps<{
  modelValue: "list" | "card";
}>();

const emit = defineEmits<{
  "update:modelValue": [value: "list" | "card"];
}>();
</script>

<template>
  <div class="view-toggle" role="radiogroup" aria-label="View mode">
    <button
      type="button"
      role="radio"
      :aria-checked="modelValue === 'list'"
      class="view-toggle-btn"
      :class="{ active: modelValue === 'list' }"
      title="List view"
      @click="emit('update:modelValue', 'list')"
    >
      <Icon icon="solar:list-bold" :width="16" :height="16" aria-hidden="true" />
    </button>
    <button
      type="button"
      role="radio"
      :aria-checked="modelValue === 'card'"
      class="view-toggle-btn"
      :class="{ active: modelValue === 'card' }"
      title="Card view"
      @click="emit('update:modelValue', 'card')"
    >
      <Icon icon="solar:widget-bold" :width="16" :height="16" aria-hidden="true" />
    </button>
  </div>
</template>

<style scoped>
.view-toggle {
  display: inline-flex;
  gap: 2px;
  padding: 2px;
  border: 1px solid var(--nook-border);
  border-radius: var(--radius-lg);
  background: var(--nook-surface-strong);
}

.view-toggle-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 1.75rem;
  height: 1.625rem;
  padding: 0;
  border: none;
  border-radius: calc(var(--radius-lg) - 2px);
  background: transparent;
  color: var(--nook-muted);
  cursor: pointer;
  transition: background 150ms, color 150ms, box-shadow 150ms;
}

.view-toggle-btn:hover {
  background: color-mix(in oklab, var(--foreground) 6%, var(--nook-surface-strong));
  color: var(--nook-text);
}

.view-toggle-btn:focus-visible {
  outline: 2px solid var(--ring);
  outline-offset: 1px;
}

.view-toggle-btn.active {
  background: var(--nook-bg);
  color: var(--nook-text);
  box-shadow: 0 1px 2px oklch(0 0 0 / 0.08), inset 0 0 0 1px oklch(0 0 0 / 0.05);
}
</style>
