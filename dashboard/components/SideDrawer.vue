<script setup lang="ts">
import { Icon } from "@iconify/vue";
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
              <Icon icon="solar:close-circle-bold" width="18" aria-hidden="true" />
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
  background: rgb(0 0 0 / 0.62);
  backdrop-filter: blur(10px);
  display: flex;
  justify-content: flex-end;
}

.drawer-panel {
  width: min(32rem, 100vw);
  height: 100vh;
  background:
    linear-gradient(180deg, rgb(255 255 255 / 0.045), transparent 12rem),
    var(--nook-bg);
  border-left: 1px solid var(--nook-border);
  display: grid;
  grid-template-rows: auto 1fr auto;
  overflow: hidden;
  box-shadow: -24px 0 60px rgb(0 0 0 / 0.32);
}

.drawer-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: 1.35rem 1.5rem;
  border-bottom: 1px solid var(--nook-border);
  background: rgb(255 255 255 / 0.02);
}

.drawer-header strong {
  font-family: "Poppins", "Helvetica Neue", Helvetica, Arial, sans-serif;
  font-size: 1.05rem;
  line-height: 1.2;
  letter-spacing: -0.03em;
}

.drawer-close {
  padding: 0.5rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.drawer-body {
  padding: 1.4rem 1.5rem 1.6rem;
  overflow-y: auto;
  display: grid;
  align-content: start;
  gap: 1.15rem;
}

.drawer-footer {
  padding: 1rem 1.5rem 1.35rem;
  border-top: 1px solid var(--nook-border);
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.75rem;
  flex-wrap: wrap;
  background: rgb(255 255 255 / 0.02);
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

@media (max-width: 720px) {
  .drawer-panel {
    width: 100vw;
  }

  .drawer-header,
  .drawer-body,
  .drawer-footer {
    padding-inline: 1rem;
  }

  .drawer-body {
    padding-block: 1rem 1.2rem;
  }

  .drawer-footer {
    justify-content: stretch;
  }

  .drawer-footer > :deep(*) {
    flex: 1 1 100%;
  }
}
</style>
