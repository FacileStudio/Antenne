<script setup lang="ts">
const props = defineProps<{
  modelValue: boolean;
  disabled?: boolean;
  size?: "sm" | "md";
}>();

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
}>();

function toggle() {
  if (props.disabled) return;
  emit("update:modelValue", !props.modelValue);
}
</script>

<template>
  <button
    type="button"
    role="switch"
    :aria-checked="modelValue"
    :disabled="disabled"
    class="toggle-switch"
    :class="[
      modelValue ? 'toggle-switch--on' : 'toggle-switch--off',
      size === 'sm' ? 'toggle-switch--sm' : 'toggle-switch--md',
      { 'toggle-switch--disabled': disabled },
    ]"
    @click.stop="toggle"
  >
    <span class="toggle-switch-thumb" />
  </button>
</template>

<style scoped>
.toggle-switch {
  position: relative;
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
  border: none;
  border-radius: 9999px;
  padding: 2px;
  cursor: pointer;
  transition: background 150ms ease;
}

.toggle-switch:focus-visible {
  outline: 2px solid var(--ring);
  outline-offset: 2px;
}

.toggle-switch--md {
  width: 2.25rem;
  height: 1.25rem;
}

.toggle-switch--sm {
  width: 1.75rem;
  height: 1rem;
}

.toggle-switch--on {
  background: var(--success);
}

.toggle-switch--on:hover {
  background: color-mix(in oklab, var(--success) 88%, var(--foreground));
}

.toggle-switch--off {
  background: var(--nook-border);
  box-shadow: inset 0 0 0 1px oklch(0 0 0 / 0.04);
}

.toggle-switch--off:hover {
  background: color-mix(in oklab, var(--nook-muted) 20%, var(--nook-border));
}

.toggle-switch--disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.toggle-switch-thumb {
  display: block;
  border-radius: 9999px;
  background: white;
  box-shadow: 0 1px 2px oklch(0 0 0 / 0.15);
  transition: transform 150ms ease;
}

.toggle-switch--md .toggle-switch-thumb {
  width: 1rem;
  height: 1rem;
}

.toggle-switch--sm .toggle-switch-thumb {
  width: 0.75rem;
  height: 0.75rem;
}

.toggle-switch--on.toggle-switch--md .toggle-switch-thumb {
  transform: translateX(1rem);
}

.toggle-switch--on.toggle-switch--sm .toggle-switch-thumb {
  transform: translateX(0.75rem);
}
</style>
