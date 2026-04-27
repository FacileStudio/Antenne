<script setup lang="ts">
import SectionCard from "../components/SectionCard.vue";
import DeliveryTargetListEditor from "../components/DeliveryTargetListEditor.vue";
import { useDashboardState } from "../composables/useDashboardState";

const { settings } = useDashboardState();
</script>

<template>
  <SectionCard eyebrow="Delivery" title="Targets">
    <DeliveryTargetListEditor v-model="settings.delivery.items" :providers="settings.providers.items" />
  </SectionCard>
</template>
