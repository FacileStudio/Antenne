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
  right: 1.25rem;
  bottom: 1.25rem;
  z-index: 50;
  display: grid;
  gap: 0.75rem;
  width: min(24rem, calc(100vw - 2rem));
}

.toast {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 1rem 1.1rem;
  border-radius: 1.1rem;
  border: 1px solid var(--nook-border);
  background: rgb(15 15 15 / 0.96);
  box-shadow: 0 18px 40px rgb(0 0 0 / 0.35);
}

.toast.success {
  border-color: rgb(58 140 84 / 0.6);
  background: rgb(10 29 15 / 0.96);
}

.toast.error {
  border-color: rgb(255 255 255 / 0.12);
}

.toast p {
  margin: 0;
  line-height: 1.45;
  color: var(--nook-text);
}

.toast.success p {
  color: rgb(185 255 199);
}

.toast-dismiss {
  flex: none;
  padding: 0.45rem 0.7rem;
  border-radius: 999px;
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
  transform: translateY(0.75rem);
}

.toast-move {
  transition: transform 180ms ease;
}

@media (max-width: 960px) {
  .toast-stack {
    right: 1rem;
    left: 1rem;
    width: auto;
  }
}
</style>
