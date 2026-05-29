<script setup lang="ts">
import FlowGraph from "../components/FlowGraph.vue";
import UndoToast from "../components/UndoToast.vue";
import { useDashboardState } from "../composables/useDashboardState";

const { settings } = useDashboardState();
</script>

<template>
  <ClientOnly>
    <FlowGraph :providers="settings.providers.items" :targets="settings.delivery.items" />
    <UndoToast />
  </ClientOnly>
</template>
