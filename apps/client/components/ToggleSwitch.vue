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

.toggle-switch--md {
  width: 2rem;
  height: 1.125rem;
}

.toggle-switch--sm {
  width: 1.625rem;
  height: 0.875rem;
}

.toggle-switch--on {
  background: oklch(0.65 0.2 155);
}

.toggle-switch--off {
  background: var(--nook-border);
}

.toggle-switch--disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.toggle-switch-thumb {
  display: block;
  border-radius: 9999px;
  background: white;
  box-shadow: 0 1px 2px oklch(0 0 0 / 0.12);
  transition: transform 150ms ease;
}

.toggle-switch--md .toggle-switch-thumb {
  width: 0.875rem;
  height: 0.875rem;
}

.toggle-switch--sm .toggle-switch-thumb {
  width: 0.625rem;
  height: 0.625rem;
}

.toggle-switch--on.toggle-switch--md .toggle-switch-thumb {
  transform: translateX(0.875rem);
}

.toggle-switch--on.toggle-switch--sm .toggle-switch-thumb {
  transform: translateX(0.75rem);
}
</style>
