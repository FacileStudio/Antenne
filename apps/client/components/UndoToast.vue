<script setup lang="ts">
import { useUndo } from "../composables/useUndo";

const { current, undoLast, dismissUndo } = useUndo();
</script>

<template>
  <Transition name="undo-toast">
    <div v-if="current" class="undo-toast">
      <span class="undo-toast-label">{{ current.label }}</span>
      <button class="undo-toast-btn undo-toast-btn--undo" @click="undoLast()">Undo</button>
      <button class="undo-toast-btn undo-toast-btn--dismiss" @click="dismissUndo()">Dismiss</button>
    </div>
  </Transition>
</template>

<style scoped>
.undo-toast {
  position: fixed;
  bottom: 1.25rem;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.625rem 1rem;
  background: var(--foreground);
  color: var(--background);
  border-radius: var(--radius-xl);
  font-size: 0.84rem;
  z-index: 9999;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.25);
}

.undo-toast-label {
  font-weight: 500;
}

.undo-toast-btn {
  border: none;
  background: none;
  font-size: 0.78rem;
  font-weight: 600;
  cursor: pointer;
  padding: 0.25rem 0.5rem;
  border-radius: var(--radius-md);
  transition: opacity 150ms;
}

.undo-toast-btn--undo {
  color: var(--background);
  text-decoration: underline;
  text-underline-offset: 2px;
}

.undo-toast-btn--dismiss {
  color: var(--background);
  opacity: 0.5;
}

.undo-toast-btn--dismiss:hover {
  opacity: 0.8;
}

.undo-toast-enter-active,
.undo-toast-leave-active {
  transition: all 200ms ease;
}

.undo-toast-enter-from,
.undo-toast-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(12px);
}
</style>
