<script setup lang="ts">
import { X } from "lucide-vue-next";
import { watch } from "vue";

const props = defineProps<{
  open: boolean;
  title?: string;
}>();

const emit = defineEmits<{
  close: [];
}>();

watch(() => props.open, (value) => {
  document.body.style.overflow = value ? "hidden" : "";
});

function onOverlayClick() {
  emit("close");
}
</script>

<template>
  <Teleport to="body">
    <Transition name="drawer">
      <div v-if="open" class="drawer-overlay" @click.self="onOverlayClick">
        <aside class="drawer-panel">
          <div class="drawer-header">
            <strong v-if="title">{{ title }}</strong>
            <button class="ghost drawer-close" type="button" @click="emit('close')">
              <X :size="18" aria-hidden="true" />
            </button>
          </div>

          <div class="drawer-body">
            <slot />
          </div>

          <div v-if="$slots.footer" class="drawer-footer">
            <slot name="footer" />
          </div>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.drawer-overlay {
  position: fixed;
  inset: 0;
  z-index: 900;
  background: rgb(0 0 0 / 0.5);
  display: flex;
  justify-content: flex-end;
}

.drawer-panel {
  width: min(28rem, 100vw);
  height: 100vh;
  background: var(--nook-bg);
  border-left: 1px solid var(--nook-border);
  display: grid;
  grid-template-rows: auto 1fr auto;
  overflow: hidden;
}

.drawer-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--nook-border);
}

.drawer-header strong {
  font-size: 1.1rem;
}

.drawer-close {
  padding: 0.5rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.drawer-body {
  padding: var(--space-5);
  overflow-y: auto;
  display: grid;
  align-content: start;
  gap: var(--space-4);
}

.drawer-footer {
  padding: var(--space-4) var(--space-5);
  border-top: 1px solid var(--nook-border);
  display: flex;
  gap: var(--space-3);
  flex-wrap: wrap;
}

.drawer-enter-active,
.drawer-leave-active {
  transition: opacity 200ms ease;
}

.drawer-enter-active .drawer-panel,
.drawer-leave-active .drawer-panel {
  transition: transform 200ms ease;
}

.drawer-enter-from,
.drawer-leave-to {
  opacity: 0;
}

.drawer-enter-from .drawer-panel,
.drawer-leave-to .drawer-panel {
  transform: translateX(100%);
}
</style>
