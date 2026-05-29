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
  border: 1px solid var(--nook-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.view-toggle-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2rem;
  height: 2rem;
  border: none;
  background: transparent;
  color: var(--nook-muted);
  cursor: pointer;
  transition: background 150ms, color 150ms;
}

.view-toggle-btn:hover {
  background: var(--nook-surface-strong);
  color: var(--nook-text);
}

.view-toggle-btn.active {
  background: var(--nook-text);
  color: var(--nook-bg);
}

.view-toggle-btn + .view-toggle-btn {
  border-left: 1px solid var(--nook-border);
}
</style>
