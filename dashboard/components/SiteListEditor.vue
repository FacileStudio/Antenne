<script setup lang="ts">
import { ArrowDown, ArrowUp, Copy, Plus, Trash2 } from "lucide-vue-next";
import { reactive } from "vue";

import type { SiteConfig } from "../types";

const props = defineProps<{
  modelValue: SiteConfig[];
}>();

const emit = defineEmits<{
  "update:modelValue": [value: SiteConfig[]];
  error: [message: string];
  clearError: [];
}>();

const draft = reactive<SiteConfig>({
  name: "",
  url: "https://",
  interval: 60,
});

function updateSites(nextSites: SiteConfig[]) {
  emit("update:modelValue", nextSites);
}

function resetDraft() {
  draft.name = "";
  draft.url = "https://";
  draft.interval = 60;
}

function addSiteFromDraft() {
  const url = draft.url.trim();
  if (!url || url === "https://") {
    emit("error", "Site URL is required.");
    return;
  }

  updateSites([
    ...props.modelValue,
    {
      name: draft.name.trim(),
      url,
      interval: draft.interval,
    },
  ]);
  resetDraft();
  emit("clearError");
}

function addBlankSite() {
  updateSites([
    ...props.modelValue,
    {
      name: "",
      url: "https://",
      interval: 60,
    },
  ]);
}

function updateSite(index: number, patch: Partial<SiteConfig>) {
  updateSites(
    props.modelValue.map((site, currentIndex) => currentIndex === index ? { ...site, ...patch } : site),
  );
}

function removeSite(index: number) {
  updateSites(props.modelValue.filter((_, currentIndex) => currentIndex !== index));
}

function moveSite(index: number, direction: -1 | 1) {
  const targetIndex = index + direction;
  if (targetIndex < 0 || targetIndex >= props.modelValue.length) {
    return;
  }

  const nextSites = [...props.modelValue];
  const [site] = nextSites.splice(index, 1);
  nextSites.splice(targetIndex, 0, site);
  updateSites(nextSites);
}

function duplicateSite(index: number) {
  const site = props.modelValue[index];
  if (!site) {
    return;
  }

  const nextSites = [...props.modelValue];
  nextSites.splice(index + 1, 0, {
    ...site,
    name: site.name ? `${site.name} copy` : "",
  });
  updateSites(nextSites);
}
</script>

<template>
  <div class="site-list">
    <header class="site-list-header">
      <slot name="actions">
        <button class="secondary action-button" type="button" @click="addBlankSite">
          <Plus :size="16" aria-hidden="true" />
          <span>Add blank</span>
        </button>
      </slot>
    </header>

    <article class="site-draft">
      <label class="field">
        <span>Name</span>
        <input v-model="draft.name" type="text" placeholder="facile.studio">
      </label>

      <label class="field wide">
        <span>URL</span>
        <input v-model="draft.url" type="url" placeholder="https://example.com">
      </label>

      <label class="field">
        <span>Interval</span>
        <input v-model.number="draft.interval" type="number" min="1" step="1">
      </label>

      <button class="primary action-button" type="button" @click="addSiteFromDraft">
        <Plus :size="16" aria-hidden="true" />
        <span>Add site</span>
      </button>
    </article>

    <div v-if="modelValue.length === 0" class="empty">No sites configured.</div>

    <div v-else class="sites">
      <article
        v-for="(site, index) in modelValue"
        :key="`${site.url}-${index}`"
        class="site-card"
      >
        <div class="site-card-header">
          <div>
            <strong>{{ site.name || `Site ${index + 1}` }}</strong>
          </div>

          <div class="site-actions">
            <button class="ghost action-button" type="button" :disabled="index === 0" @click="moveSite(index, -1)">
              <ArrowUp :size="16" aria-hidden="true" />
              <span>Up</span>
            </button>
            <button class="ghost action-button" type="button" :disabled="index === modelValue.length - 1" @click="moveSite(index, 1)">
              <ArrowDown :size="16" aria-hidden="true" />
              <span>Down</span>
            </button>
            <button class="ghost action-button" type="button" @click="duplicateSite(index)">
              <Copy :size="16" aria-hidden="true" />
              <span>Duplicate</span>
            </button>
            <button class="danger action-button" type="button" @click="removeSite(index)">
              <Trash2 :size="16" aria-hidden="true" />
              <span>Remove</span>
            </button>
          </div>
        </div>

        <div class="site-fields">
          <label class="field">
            <span>Name</span>
            <input :value="site.name" type="text" placeholder="facile.studio" @input="updateSite(index, { name: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field wide">
            <span>URL</span>
            <input :value="site.url" type="url" placeholder="https://example.com" @input="updateSite(index, { url: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field">
            <span>Interval</span>
            <input :value="site.interval" type="number" min="1" step="1" @input="updateSite(index, { interval: Number(($event.target as HTMLInputElement).value) || 1 })">
          </label>
        </div>
      </article>
    </div>
  </div>
</template>

<style scoped>
.site-list,
.sites {
  display: grid;
  gap: var(--space-4);
}

.site-list-header {
  display: flex;
  justify-content: flex-end;
}

.site-draft,
.site-fields {
  display: grid;
  gap: var(--space-3);
  grid-template-columns: 1.1fr 2fr 0.7fr auto;
}

.site-draft,
.site-card {
  padding: var(--space-4);
  border-radius: 1.25rem;
  border: 1px solid var(--nook-border);
  background: var(--nook-surface-strong);
}

.site-card {
  display: grid;
  gap: var(--space-4);
}

.site-card-header {
  display: flex;
  justify-content: space-between;
  gap: var(--space-4);
  align-items: flex-start;
}

.site-actions {
  display: flex;
  gap: var(--space-2);
  flex-wrap: wrap;
  justify-content: flex-end;
}

.action-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
}

.empty {
  color: var(--nook-muted);
  padding: var(--space-4);
  border-radius: 1.25rem;
  border: 1px dashed var(--nook-border);
}

@media (max-width: 960px) {
  .site-draft,
  .site-fields {
    grid-template-columns: 1fr;
  }

  .site-card-header {
    flex-direction: column;
  }

  .site-actions,
  .site-list-header {
    justify-content: flex-start;
  }
}
</style>
