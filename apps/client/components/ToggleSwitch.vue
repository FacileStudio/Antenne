<script setup lang="ts">
import { useSlots } from "vue";

const props = defineProps<{
  modelValue: boolean;
  disabled?: boolean;
  size?: "sm" | "md";
  label?: string;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
}>();

const slots = useSlots();

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
    <span class="toggle-switch-track">
      <span class="toggle-switch-thumb" />
    </span>
    <span v-if="label || slots.default" class="toggle-switch-label">
      <slot>{{ label }}</slot>
    </span>
  </button>
</template>

<style scoped>
.toggle-switch {
  --toggle-pad: 0.1875rem;
  --toggle-thumb: calc(var(--toggle-height) - var(--toggle-pad) * 2);
  --toggle-travel: calc(var(--toggle-width) - var(--toggle-height));
  display: inline-flex;
  align-items: center;
  justify-content: flex-start;
  gap: 0.625rem;
  height: auto;
  padding: 0;
  border: none;
  border-radius: 0;
  background: transparent;
  color: inherit;
  text-align: left;
  white-space: normal;
  cursor: pointer;
  -webkit-tap-highlight-color: transparent;
}

.toggle-switch:hover {
  background: transparent;
  border-color: transparent;
}

.toggle-switch--md {
  --toggle-width: 2.5rem;
  --toggle-height: 1.5rem;
}

.toggle-switch--sm {
  --toggle-width: 1.875rem;
  --toggle-height: 1.125rem;
}

.toggle-switch-track {
  position: relative;
  display: block;
  flex: none;
  width: var(--toggle-width);
  height: var(--toggle-height);
  border-radius: 999px;
  background: color-mix(in oklab, var(--foreground) 14%, var(--nook-surface-strong));
  box-shadow:
    inset 0 1px 2px oklch(0 0 0 / 0.1),
    inset 0 0 0 1px oklch(0 0 0 / 0.05);
  transition:
    background-color 220ms cubic-bezier(0.4, 0, 0.2, 1),
    box-shadow 220ms cubic-bezier(0.4, 0, 0.2, 1);
}

.toggle-switch--on .toggle-switch-track {
  background: var(--success);
  box-shadow:
    inset 0 1px 0 oklch(1 0 0 / 0.28),
    inset 0 0 0 1px color-mix(in oklab, var(--success) 70%, black);
}

.toggle-switch:not(.toggle-switch--disabled):hover .toggle-switch-track {
  box-shadow:
    inset 0 1px 2px oklch(0 0 0 / 0.1),
    inset 0 0 0 1px oklch(0 0 0 / 0.05),
    0 0 0 0.2rem color-mix(in srgb, var(--foreground) 9%, transparent);
}

.toggle-switch--on:not(.toggle-switch--disabled):hover .toggle-switch-track {
  box-shadow:
    inset 0 1px 0 oklch(1 0 0 / 0.28),
    inset 0 0 0 1px color-mix(in oklab, var(--success) 70%, black),
    0 0 0 0.2rem color-mix(in srgb, var(--success) 22%, transparent);
}

.toggle-switch:focus-visible {
  outline: none;
}

.toggle-switch:focus-visible .toggle-switch-track {
  box-shadow:
    inset 0 0 0 1px oklch(0 0 0 / 0.05),
    0 0 0 2px var(--background),
    0 0 0 4px var(--ring);
}

.toggle-switch-thumb {
  position: absolute;
  top: var(--toggle-pad);
  left: var(--toggle-pad);
  display: block;
  width: var(--toggle-thumb);
  height: var(--toggle-thumb);
  border-radius: 999px;
  background: oklch(1 0 0);
  box-shadow:
    0 1px 2px oklch(0 0 0 / 0.24),
    0 0 0 0.5px oklch(0 0 0 / 0.06);
  transform-origin: left center;
  transition:
    transform 260ms cubic-bezier(0.34, 1.4, 0.5, 1),
    box-shadow 220ms ease;
}

.toggle-switch--on .toggle-switch-thumb {
  transform: translateX(var(--toggle-travel));
  transform-origin: right center;
}

.toggle-switch:not(.toggle-switch--disabled):active .toggle-switch-thumb {
  transform: scaleX(1.14);
  box-shadow:
    0 1px 3px oklch(0 0 0 / 0.28),
    0 0 0 0.5px oklch(0 0 0 / 0.08);
}

.toggle-switch--on:not(.toggle-switch--disabled):active .toggle-switch-thumb {
  transform: translateX(var(--toggle-travel)) scaleX(1.14);
}

.toggle-switch-label {
  color: var(--nook-muted);
  font-size: 0.875rem;
  line-height: 1.3;
  transition: color 150ms ease;
}

.toggle-switch:not(.toggle-switch--disabled):hover .toggle-switch-label {
  color: var(--nook-text);
}

.toggle-switch--disabled {
  opacity: 0.45;
  cursor: not-allowed;
  pointer-events: auto;
}

@media (prefers-reduced-motion: reduce) {
  .toggle-switch-track,
  .toggle-switch-thumb,
  .toggle-switch-label {
    transition-duration: 1ms;
  }
}
</style>
