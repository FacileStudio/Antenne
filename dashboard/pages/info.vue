<script setup lang="ts">
import { computed } from "vue";

import SectionCard from "../components/SectionCard.vue";
import StatusGrid from "../components/StatusGrid.vue";
import { useDashboardState } from "../composables/useDashboardState";

const { runtime } = useDashboardState();

const infoItems = computed(() => [
  {
    label: "Server port",
    value: runtime.value?.serverPort ? String(runtime.value.serverPort) : "Unknown",
  },
  {
    label: "Settings file",
    value: runtime.value?.settingsPath ?? "Unknown",
  },
  {
    label: "Source",
    value: runtime.value?.usingPersistedSettings ? "Dashboard override" : "Env/YAML defaults",
  },
  {
    label: "Reloaded",
    value: runtime.value?.lastReloadedAt ?? "Unknown",
  },
]);
</script>

<template>
  <SectionCard eyebrow="Info" title="Runtime">
    <StatusGrid :items="infoItems" />
  </SectionCard>
</template>
