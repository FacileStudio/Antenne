<script setup lang="ts">
import SectionCard from "../../components/SectionCard.vue";
import ProviderListEditor from "../../components/ProviderListEditor.vue";
import { useDashboardState } from "../../composables/useDashboardState";

const { settings } = useDashboardState();
</script>

<template>
  <SectionCard eyebrow="Providers" title="Sources">
    <ProviderListEditor v-model="settings.providers.items" />
  </SectionCard>
</template>
