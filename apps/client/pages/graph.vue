<script setup lang="ts">
import FlowGraph from "../components/FlowGraph.vue";
import { useDashboardState } from "../composables/useDashboardState";

const { settings } = useDashboardState();
</script>

<template>
  <ClientOnly>
    <FlowGraph :providers="settings.providers.items" :targets="settings.delivery.items" />
  </ClientOnly>
</template>
