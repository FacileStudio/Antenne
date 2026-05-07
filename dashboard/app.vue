<script setup lang="ts">
import { computed, onMounted } from "vue";
import { Icon } from "@iconify/vue";

import AppSidebar from "./components/AppSidebar.vue";
import ToastStack from "./components/ToastStack.vue";
import { useDashboardState } from "./composables/useDashboardState";

const {
  settings,
  runtime,
  loading,
  saving,
  testing,
  toasts,
  dismissToast,
  loadSettings,
  saveSettings,
  sendTestAlert,
  logout,
} = useDashboardState();

const sections = [
  { href: "/delivery", label: "Delivery", icon: "solar:delivery-bold" },
  { href: "/providers", label: "Providers", icon: "solar:link-circle-bold" },
  { href: "/activity", label: "Activity", icon: "solar:pulse-bold" },
  { href: "/settings", label: "Settings", icon: "solar:settings-bold" },
] as const;
const route = useRoute();

const currentTitle = computed(() => {
  if (route.path.startsWith("/providers/")) return "Provider";
  if (route.path.startsWith("/targets/")) return "Delivery Target";
  if (route.path === "/info") return "Settings";
  return sections.find((section) => section.href === route.path)?.label ?? "Settings";
});

onMounted(() => {
  void loadSettings();
});
</script>

<template>
  <main class="shell">
    <section class="layout">
      <AppSidebar :sections="sections">
        <template #footer>
          <button class="danger action-button sidebar-logout" type="button" @click="logout">
              <Icon icon="solar:logout-bold" width="16" aria-hidden="true" />
              <span>Logout</span>
          </button>
        </template>
      </AppSidebar>

      <div class="content-shell">
        <header class="topbar">
          <div class="topbar-copy">
            <h1>{{ currentTitle }}</h1>
          </div>

          <div class="topbar-actions">
            <button class="secondary action-button" type="button" :disabled="testing || loading" @click="sendTestAlert">
              <Icon icon="solar:plain-bold" width="16" aria-hidden="true" />
              <span>{{ testing ? "Sending..." : "Send test alert" }}</span>
            </button>
            <button class="primary action-button" type="button" :disabled="saving || loading" @click="saveSettings">
              <Icon icon="solar:diskette-bold" width="16" aria-hidden="true" />
              <span>{{ saving ? "Saving..." : "Save settings" }}</span>
            </button>
          </div>
        </header>

        <section v-if="loading" class="panel loading-card">
          Loading dashboard...
        </section>

        <div v-else class="content">
          <NuxtPage />
        </div>
      </div>
    </section>

    <ToastStack :toasts="toasts" @dismiss="dismissToast" />
  </main>
</template>

<style scoped>
.shell {
  min-height: 100vh;
}

.layout {
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr);
  min-height: 100vh;
}

.content-shell {
  display: grid;
  align-content: start;
  gap: var(--space-5);
  min-width: 0;
  padding: var(--space-6) var(--space-6) var(--space-7);
}

.topbar {
  display: flex;
  justify-content: space-between;
  gap: var(--space-4);
  align-items: flex-end;
}

.topbar-actions {
  display: flex;
  gap: var(--space-3);
  flex-wrap: wrap;
  justify-content: flex-end;
}

.topbar-copy h1 {
  margin: 0;
}

.content {
  display: grid;
  gap: var(--space-5);
  min-width: 0;
}

.action-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
}

.sidebar-logout {
  width: 100%;
}

.sidebar-logout.danger {
  border-color: oklch(40% 0.15 25 / 30%);
  color: oklch(75% 0.12 20);
}

.sidebar-logout.danger:hover {
  background: oklch(20% 0.08 25 / 45%);
  border-color: oklch(40% 0.15 25 / 45%);
}

.loading-card {
  color: var(--nook-text);
}

@media (max-width: 960px) {
  .layout {
    grid-template-columns: 1fr;
  }

  .content-shell {
    padding: var(--space-4) var(--space-4) var(--space-6);
  }

  .topbar {
    flex-direction: column;
    align-items: stretch;
  }

  .topbar-actions {
    justify-content: flex-start;
  }
}
</style>
