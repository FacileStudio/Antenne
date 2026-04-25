<script setup lang="ts">
import { computed, onMounted } from "vue";
import { LogOut, Save, Send } from "lucide-vue-next";

import AppSidebar from "./components/AppSidebar.vue";
import { useDashboardState } from "./composables/useDashboardState";

const {
  settings,
  runtime,
  loading,
  saving,
  testing,
  error,
  success,
  loadSettings,
  saveSettings,
  sendTestAlert,
  logout,
} = useDashboardState();

const sections = [
  { href: "/delivery", label: "Delivery" },
  { href: "/webhooks", label: "Webhooks" },
  { href: "/sites", label: "Sites" },
  { href: "/info", label: "Info" },
] as const;

const sidebarInfoItems = computed(() => [
  {
    label: "Delivery",
    value: [
      settings.delivery.matrix.enabled && settings.delivery.matrix.roomIds.length > 0 ? "Matrix" : null,
      settings.delivery.discord.enabled && settings.delivery.discord.webhookUrl ? "Discord" : null,
      settings.delivery.webhook.enabled && settings.delivery.webhook.url ? "HTTP" : null,
    ].filter(Boolean).join(", ") || "Not configured",
  },
  {
    label: "Sites",
    value: `${settings.sites.length}`,
  },
  {
    label: "Status",
    value: runtime.value?.usingPersistedSettings ? "Saved settings" : "Defaults",
  },
]);

onMounted(() => {
  void loadSettings();
});
</script>

<template>
  <main class="shell">
    <header class="topbar">
      <div class="topbar-copy">
        <p class="eyebrow">Nook</p>
        <h1>Settings</h1>
      </div>

      <div class="topbar-actions">
        <button class="secondary action-button" type="button" :disabled="testing || loading" @click="sendTestAlert">
          <Send :size="16" aria-hidden="true" />
          <span>{{ testing ? "Sending..." : "Send test alert" }}</span>
        </button>
        <button class="primary action-button" type="button" :disabled="saving || loading" @click="saveSettings">
          <Save :size="16" aria-hidden="true" />
          <span>{{ saving ? "Saving..." : "Save settings" }}</span>
        </button>
      </div>
    </header>

    <p v-if="error" class="banner error">{{ error }}</p>
    <p v-if="success" class="banner success">{{ success }}</p>

    <section v-if="loading" class="panel loading-card">
      Loading dashboard...
    </section>

    <template v-else>
      <section class="layout">
        <AppSidebar :sections="sections" :info-items="sidebarInfoItems">
          <template #footer>
            <button class="ghost action-button sidebar-logout" type="button" @click="logout">
              <LogOut :size="16" aria-hidden="true" />
              <span>Logout</span>
            </button>
          </template>
        </AppSidebar>

        <div class="content">
          <NuxtPage />
        </div>
      </section>
    </template>
  </main>
</template>

<style scoped>
.shell {
  width: min(1200px, calc(100vw - 2rem));
  margin: 0 auto;
  padding: var(--space-6) 0 var(--space-7);
}

.topbar {
  display: flex;
  justify-content: space-between;
  gap: var(--space-4);
  align-items: flex-end;
  margin-bottom: var(--space-5);
}

.topbar-copy .eyebrow {
  margin: 0 0 0.35rem;
  font-size: 0.75rem;
  color: var(--nook-muted);
}

.topbar-copy h1 {
  margin: 0;
}

.topbar-actions {
  display: flex;
  gap: var(--space-3);
  flex-wrap: wrap;
  justify-content: flex-end;
}

.layout {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  gap: var(--space-5);
  align-items: start;
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

.banner {
  margin: 0 0 var(--space-4);
  padding: var(--space-3) var(--space-4);
  border-radius: 1rem;
  border: 1px solid var(--nook-border);
}

.banner.error,
.banner.success,
.loading-card {
  color: var(--nook-text);
}

@media (max-width: 960px) {
  .layout {
    grid-template-columns: 1fr;
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
