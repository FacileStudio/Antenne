<script setup lang="ts">
import { computed, onMounted } from "vue";
import { Icon } from "@iconify/vue";

import AppSidebar from "./components/AppSidebar.vue";
import ToastStack from "./components/ToastStack.vue";
import { useDashboardState } from "./composables/useDashboardState";

const {
  loading,
  saving,
  testing,
  toasts,
  passwordRequired,
  dismissToast,
  loadSettings,
  saveSettings,
  sendTestAlert,
  logout,
} = useDashboardState();

const sections = [
  { href: "/delivery", label: "Delivery", icon: "solar:delivery-linear", activePaths: ["/delivery", "/targets"] },
  { href: "/providers", label: "Providers", icon: "solar:link-linear" },
  { href: "/graph", label: "Flow", icon: "solar:share-linear" },
  { href: "/activity", label: "Activity", icon: "solar:chart-2-linear" },
  { href: "/spaces", label: "Spaces", icon: "solar:buildings-2-linear" },
  { href: "/settings", label: "Settings", icon: "solar:settings-linear", activePaths: ["/settings", "/info"] },
];

const route = useRoute();

const currentTitle = computed(() => {
  if (route.path.startsWith("/providers/")) return "Provider";
  if (route.path.startsWith("/targets/")) return "Delivery Target";
  if (route.path.startsWith("/spaces/") && route.params.id) return "Space";
  if (route.path === "/info") return "Settings";
  return sections.find((s) => route.path === s.href || route.path.startsWith(s.href + "/"))?.label ?? "Settings";
});

onMounted(() => {
  void loadSettings();
});
</script>

<template>
  <div class="flex min-h-screen bg-background">
    <AppSidebar :sections="sections">
      <template #footer>
        <button
          v-if="passwordRequired"
          type="button"
          class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
          @click="logout"
        >
          <Icon icon="solar:logout-2-linear" class="shrink-0" :width="16" :height="16" aria-hidden="true" />
          Logout
        </button>
      </template>
    </AppSidebar>

    <div class="flex min-w-0 flex-1 flex-col gap-8 p-10 max-lg:p-6">
      <header class="flex items-end justify-between gap-6 max-sm:flex-col max-sm:items-stretch">
        <h1 class="m-0 font-heading text-3xl font-semibold">{{ currentTitle }}</h1>

        <div class="flex flex-wrap items-center justify-end gap-3 max-sm:justify-start">
          <button
            type="button"
            class="inline-flex h-8 items-center gap-2 rounded-lg border border-border bg-transparent px-3 text-sm font-medium text-foreground transition-all hover:bg-muted disabled:pointer-events-none disabled:opacity-50"
            :disabled="testing || loading"
            @click="sendTestAlert"
          >
            <Icon icon="solar:plain-linear" class="shrink-0" :width="14" :height="14" aria-hidden="true" />
            {{ testing ? "Sending..." : "Send test alert" }}
          </button>

          <button
            type="button"
            class="inline-flex h-8 items-center gap-2 rounded-lg border border-foreground bg-foreground px-3 text-sm font-medium text-background transition-all hover:bg-foreground/85 disabled:pointer-events-none disabled:opacity-50"
            :disabled="saving || loading"
            @click="saveSettings"
          >
            <Icon icon="solar:diskette-linear" class="shrink-0" :width="14" :height="14" aria-hidden="true" />
            {{ saving ? "Saving..." : "Save settings" }}
          </button>
        </div>
      </header>

      <div v-if="loading" class="panel text-foreground">
        Loading dashboard...
      </div>

      <div v-else class="flex flex-col gap-8 min-w-0">
        <NuxtPage />
      </div>
    </div>

    <ToastStack :toasts="toasts" @dismiss="dismissToast" />
  </div>
</template>
