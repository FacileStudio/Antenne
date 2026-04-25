<script setup lang="ts">
import SectionCard from "../components/SectionCard.vue";
import SiteListEditor from "../components/SiteListEditor.vue";
import { useDashboardState } from "../composables/useDashboardState";

const { settings, error } = useDashboardState();

function setError(message: string) {
  error.value = message;
}

function clearError() {
  error.value = "";
}
</script>

<template>
  <SectionCard eyebrow="Sites" title="Monitoring">
    <SiteListEditor
      v-model="settings.sites"
      @error="setError"
      @clear-error="clearError"
    />
  </SectionCard>
</template>
