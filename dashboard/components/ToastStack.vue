<script setup lang="ts">
defineProps<{
  toasts: Array<{
    id: number;
    type: "success" | "error";
    message: string;
  }>;
}>();

const emit = defineEmits<{
  dismiss: [id: number];
}>();
</script>

<template>
  <Teleport to="body">
    <TransitionGroup name="toast" tag="div" class="toast-stack">
      <div
        v-for="toast in toasts"
        :key="toast.id"
        class="toast"
        :class="toast.type"
        role="status"
        aria-live="polite"
      >
        <p>{{ toast.message }}</p>
        <button class="toast-dismiss" type="button" aria-label="Dismiss notification" @click="emit('dismiss', toast.id)">
          Close
        </button>
      </div>
    </TransitionGroup>
  </Teleport>
</template>

<style scoped>
.toast-stack {
  position: fixed;
  top: 1.25rem;
  left: 50%;
  transform: translateX(-50%);
  z-index: 9999;
  display: grid;
  gap: 0.75rem;
  width: min(24rem, calc(100vw - 2rem));
}

.toast {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.75rem 1rem;
  border-radius: var(--radius-xl);
  border: 1px solid var(--nook-border);
  background: oklch(1 0 0);
  box-shadow: 0 18px 40px oklch(0 0 0 / 12%);
}

.toast.success {
  border-color: oklch(60% 0.15 145 / 40%);
  background: oklch(95% 0.04 145);
}

.toast.error {
  border-color: oklch(0 0 0 / 12%);
}

.toast p {
  margin: 0;
  line-height: 1.45;
  color: var(--nook-text);
}

.toast.success p {
  color: oklch(35% 0.12 145);
}

.toast-dismiss {
  flex: none;
  padding: 0.25rem 0.5rem;
  border-radius: var(--radius-lg);
  border: 1px solid transparent;
  background: transparent;
  color: var(--nook-muted);
}

.toast-enter-active,
.toast-leave-active {
  transition: opacity 180ms ease, transform 180ms ease;
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(-0.75rem);
}

.toast-move {
  transition: transform 180ms ease;
}

@media (max-width: 960px) {
  .toast-stack {
    left: 50%;
    transform: translateX(-50%);
    width: calc(100vw - 2rem);
  }
}
</style>
