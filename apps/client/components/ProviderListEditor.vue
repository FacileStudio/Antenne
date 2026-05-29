<script setup lang="ts">
import { Icon } from "@iconify/vue";
import { reactive, ref } from "vue";

import ProviderEditorFields from "./ProviderEditorFields.vue";
import SideDrawer from "./SideDrawer.vue";
import ToggleSwitch from "./ToggleSwitch.vue";
import ViewToggle from "./ViewToggle.vue";
import { useDashboardState } from "../composables/useDashboardState";
import { createTagToneMap, getTagTone } from "../utils/provider-tags";
import { getProviderIcon } from "../utils/icons";
import {
  createDefaultProvider,
  getProviderLabel,
  providerTypes,
  type ProviderSettings,
  type ProviderType,
} from "../types";

const props = defineProps<{
  modelValue: ProviderSettings[];
}>();

const emit = defineEmits<{
  "update:modelValue": [value: ProviderSettings[]];
}>();

const { notifyError, notifySuccess, testProviderConnection, saveSettings, saving } = useDashboardState();
const revealedSecrets = reactive<Record<string, boolean>>({});
const nextProviderType = ref<ProviderType>("website");
const testingProviders = reactive<Record<string, boolean>>({});
const editingIndex = ref<number | null>(null);
const selectedTags = ref<string[]>([]);
const viewMode = ref<"list" | "card">("card");

const availableTags = computed(() =>
  [...new Set(props.modelValue.flatMap((provider) => provider.tags))]
    .sort((left, right) => left.localeCompare(right))
);
const tagToneMap = computed(() => createTagToneMap(availableTags.value));

const visibleProviders = computed(() =>
  props.modelValue
    .map((provider, index) => ({ provider, index }))
    .filter(({ provider }) =>
      selectedTags.value.length === 0 || selectedTags.value.every((tag) => provider.tags.includes(tag))
    )
);

const editingProvider = computed(() =>
  editingIndex.value !== null ? props.modelValue[editingIndex.value] ?? null : null
);

function openEditor(index: number) {
  editingIndex.value = index;
}

function closeEditor() {
  editingIndex.value = null;
}

function updateProviders(nextProviders: ProviderSettings[]) {
  emit("update:modelValue", nextProviders);
}

function addProvider() {
  const nextProviders = [
    createDefaultProvider(nextProviderType.value),
    ...props.modelValue,
  ];
  updateProviders(nextProviders);
  editingIndex.value = 0;
}

function patchProvider(index: number, patch: Partial<ProviderSettings>) {
  const provider = props.modelValue[index];
  if (!provider) {
    return;
  }

  updateProviders(
    props.modelValue.map((currentProvider, currentIndex) =>
      currentIndex === index ? { ...provider, ...patch } as ProviderSettings : currentProvider
    ),
  );
}

function removeProvider(index: number) {
  if (editingIndex.value === index) {
    closeEditor();
  } else if (editingIndex.value !== null && editingIndex.value > index) {
    editingIndex.value--;
  }
  updateProviders(props.modelValue.filter((_, currentIndex) => currentIndex !== index));
}

const dragIndex = ref<number | null>(null);
const dragOverIndex = ref<number | null>(null);

function onDragStart(index: number) {
  dragIndex.value = index;
}

function onDragOver(event: DragEvent, index: number) {
  event.preventDefault();
  dragOverIndex.value = index;
}

function onDragEnd() {
  if (dragIndex.value !== null && dragOverIndex.value !== null && dragIndex.value !== dragOverIndex.value) {
    const nextProviders = [...props.modelValue];
    const [moved] = nextProviders.splice(dragIndex.value, 1);
    nextProviders.splice(dragOverIndex.value, 0, moved);
    updateProviders(nextProviders);

    if (editingIndex.value === dragIndex.value) {
      editingIndex.value = dragOverIndex.value;
    } else if (editingIndex.value !== null) {
      const from = dragIndex.value;
      const to = dragOverIndex.value;
      if (editingIndex.value > from && editingIndex.value <= to) {
        editingIndex.value--;
      } else if (editingIndex.value < from && editingIndex.value >= to) {
        editingIndex.value++;
      }
    }
  }
  dragIndex.value = null;
  dragOverIndex.value = null;
}

function toggleEnabled(index: number) {
  const provider = props.modelValue[index];
  if (!provider) return;
  patchProvider(index, { enabled: !provider.enabled });
  saveSettings();
}

function toggleReveal(id: string) {
  revealedSecrets[id] = !revealedSecrets[id];
}

function toggleTagFilter(tag: string) {
  selectedTags.value = selectedTags.value.includes(tag)
    ? selectedTags.value.filter((currentTag) => currentTag !== tag)
    : [...selectedTags.value, tag];
}

function clearTagFilters() {
  selectedTags.value = [];
}

async function runProviderConnectionTest(provider: ProviderSettings) {
  testingProviders[provider.id] = true;
  const label = provider.name || getProviderLabel(provider.type);

  try {
    await testProviderConnection(provider.id);
    notifySuccess(`Connection OK for ${label}.`);
  } catch (error) {
    notifyError(error instanceof Error ? error.message : "Failed to test provider connection");
  } finally {
    testingProviders[provider.id] = false;
  }
}
</script>

<template>
  <div class="provider-list">
    <div class="toolbar">
      <label class="field toolbar-field">
        <span>New provider type</span>
        <select v-model="nextProviderType">
          <option v-for="type in providerTypes" :key="type" :value="type">
            {{ getProviderLabel(type) }}
          </option>
        </select>
      </label>

      <button class="primary action-button add-button" type="button" @click="addProvider">
        <Icon icon="mdi:plus" class="button-icon" :width="16" :height="16" aria-hidden="true" />
        <span>Add provider</span>
      </button>

      <ViewToggle v-if="modelValue.length > 0" v-model="viewMode" />
    </div>

    <div v-if="modelValue.length === 0" class="empty">No providers configured.</div>

    <div v-else-if="availableTags.length > 0" class="filter-bar">
      <span class="filter-label">Filter by tags</span>
      <button
        v-for="tag in availableTags"
        :key="tag"
        class="tag-pill tag-pill--interactive tag-filter"
        :class="{ 'tag-pill--selected': selectedTags.includes(tag) }"
        :data-tag-tone="getTagTone(tag, tagToneMap)"
        type="button"
        @click="toggleTagFilter(tag)"
      >
        {{ tag }}
      </button>
      <button
        v-if="selectedTags.length > 0"
        class="ghost action-button clear-filters"
        type="button"
        @click="clearTagFilters"
      >
        Clear
      </button>
    </div>

    <div v-if="modelValue.length > 0 && visibleProviders.length === 0" class="empty">No providers match current tag filter.</div>

    <div v-else-if="modelValue.length > 0 && viewMode === 'list'" class="items">
      <article
        v-for="({ provider, index }) in visibleProviders"
        :key="provider.id"
        class="item-card"
        :class="{ dimmed: !provider.enabled, 'drag-over': dragOverIndex === index && dragIndex !== index }"
        draggable="true"
        @dragstart="onDragStart(index)"
        @dragover="onDragOver($event, index)"
        @dragend="onDragEnd"
      >
        <div class="item-info">
          <span class="drag-handle" title="Drag to reorder"><Icon icon="solar:hamburger-menu-linear" :width="18" :height="18" aria-hidden="true" /></span>
          <ToggleSwitch :model-value="provider.enabled" size="sm" @update:model-value="toggleEnabled(index)" />
          <Icon :icon="getProviderIcon(provider.type)" :width="20" :height="20" class="item-icon" aria-hidden="true" />
          <strong>{{ provider.name || `${getProviderLabel(provider.type)} ${index + 1}` }}</strong>
          <span class="item-chip">{{ getProviderLabel(provider.type) }}</span>
          <span
            v-for="tag in provider.tags"
            :key="tag"
            class="tag-pill tag-pill--filled item-tag"
            :data-tag-tone="getTagTone(tag, tagToneMap)"
          >
            {{ tag }}
          </span>
        </div>

        <div class="item-actions">
          <button class="ghost action-button" type="button" @click="openEditor(index)">
            <Icon icon="solar:pen-linear" :width="16" :height="16" aria-hidden="true" />
            <span>Edit</span>
          </button>
          <NuxtLink class="ghost action-button" :to="`/providers/${provider.id}`">
            <Icon icon="solar:chart-2-linear" :width="16" :height="16" aria-hidden="true" />
          </NuxtLink>
          <button class="danger action-button" type="button" @click="removeProvider(index)">
            <Icon icon="solar:trash-bin-trash-linear" :width="16" :height="16" aria-hidden="true" />
          </button>
        </div>
      </article>
    </div>

    <div v-else-if="modelValue.length > 0 && viewMode === 'card'" class="card-grid">
      <article
        v-for="({ provider, index }) in visibleProviders"
        :key="provider.id"
        class="provider-card"
        :class="{ dimmed: !provider.enabled }"
      >
        <div class="provider-card-header">
          <div class="provider-card-icon-wrap">
            <Icon :icon="getProviderIcon(provider.type)" :width="24" :height="24" aria-hidden="true" />
          </div>
          <ToggleSwitch :model-value="provider.enabled" size="sm" @update:model-value="toggleEnabled(index)" />
        </div>

        <div class="provider-card-body">
          <span class="provider-card-name">{{ provider.name || `${getProviderLabel(provider.type)} ${index + 1}` }}</span>
          <span class="provider-card-type">{{ getProviderLabel(provider.type) }}</span>
        </div>

        <div v-if="provider.tags.length > 0" class="provider-card-tags">
          <span
            v-for="tag in provider.tags"
            :key="tag"
            class="tag-pill tag-pill--filled provider-card-tag"
            :data-tag-tone="getTagTone(tag, tagToneMap)"
          >
            {{ tag }}
          </span>
        </div>

        <div class="provider-card-actions">
          <button class="ghost action-button" type="button" @click="openEditor(index)" title="Edit">
            <Icon icon="solar:pen-linear" :width="15" :height="15" aria-hidden="true" />
          </button>
          <NuxtLink class="ghost action-button" :to="`/providers/${provider.id}`" title="Activity">
            <Icon icon="solar:chart-2-linear" :width="15" :height="15" aria-hidden="true" />
          </NuxtLink>
          <button class="danger action-button" type="button" @click="removeProvider(index)" title="Delete">
            <Icon icon="solar:trash-bin-trash-linear" :width="15" :height="15" aria-hidden="true" />
          </button>
        </div>
      </article>
    </div>

    <SideDrawer
      :open="editingProvider !== null"
      :title="`Edit ${editingProvider ? (editingProvider.name || getProviderLabel(editingProvider.type)) : 'provider'}`"
      @close="closeEditor"
    >
      <template v-if="editingProvider !== null && editingIndex !== null">
        <ProviderEditorFields
          :editing-provider="editingProvider"
          :revealed-secrets="revealedSecrets"
          :available-tags="availableTags"
          :tag-tone-map="tagToneMap"
          @patch="patchProvider(editingIndex, $event)"
          @toggle-reveal="toggleReveal($event)"
        />
      </template>

      <template v-if="editingProvider !== null" #footer>
        <button
          v-if="editingProvider.type === 'imap'"
          class="secondary action-button"
          type="button"
          :disabled="testingProviders[editingProvider.id]"
          @click="runProviderConnectionTest(editingProvider)"
        >
          <Icon icon="solar:plug-circle-linear" :width="16" :height="16" aria-hidden="true" />
          <span>{{ testingProviders[editingProvider.id] ? "Testing..." : "Test connection" }}</span>
        </button>
        <button class="primary action-button" type="button" :disabled="saving" @click="saveSettings">
          <Icon icon="solar:diskette-linear" :width="16" :height="16" aria-hidden="true" />
          <span>{{ saving ? "Saving..." : "Save" }}</span>
        </button>
      </template>
    </SideDrawer>
  </div>
</template>

<style scoped>
.provider-list,
.items {
  display: grid;
  gap: var(--space-3);
}

.toolbar {
  display: flex;
  justify-content: flex-start;
  gap: var(--space-3);
  align-items: end;
  flex-wrap: wrap;
}

.toolbar-field {
  min-width: 15rem;
}

.filter-bar {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  flex-wrap: wrap;
}

.filter-label {
  font-size: 0.82rem;
  color: var(--nook-muted);
}

.tag-filter {
  min-height: 2rem;
}

.clear-filters {
  padding: 0.32rem 0.6rem;
}

.item-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-xl);
  border: 1px solid var(--nook-border);
  background: var(--nook-surface-strong);
  flex-wrap: wrap;
  transition: border-color 150ms, opacity 150ms;
}

.item-card:hover {
  border-color: oklch(0.8 0 0);
}

.item-card.dimmed {
  opacity: 0.5;
}

.item-card.drag-over {
  border-color: oklch(0 0 0 / 30%);
  background: oklch(0 0 0 / 5%);
}

.drag-handle {
  cursor: grab;
  color: var(--nook-muted);
  display: inline-flex;
  align-items: center;
  flex: none;
}

.drag-handle:active {
  cursor: grabbing;
}

.item-info {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
  min-width: 0;
}

.item-icon {
  color: var(--nook-muted);
  flex-shrink: 0;
}

.item-chip {
  display: inline-flex;
  align-items: center;
  min-height: 1.25rem;
  padding: 0.1rem 0.5rem;
  border-radius: var(--radius-pill);
  border: 1px solid var(--nook-border);
  color: var(--nook-muted);
  font-size: 0.75rem;
}

.item-tag {
  min-height: 1.5rem;
}

.item-actions {
  display: flex;
  gap: var(--space-1);
  flex-wrap: wrap;
  justify-content: flex-end;
}

.card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: var(--space-3);
}

.provider-card {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1rem;
  border-radius: var(--radius-xl);
  border: 1px solid var(--nook-border);
  background: var(--nook-surface-strong);
  transition: border-color 150ms, box-shadow 150ms, opacity 150ms;
}

.provider-card:hover {
  border-color: oklch(0.8 0 0);
  box-shadow: 0 2px 8px oklch(0 0 0 / 0.04);
}

.provider-card.dimmed {
  opacity: 0.5;
}

.provider-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.provider-card-icon-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 2.25rem;
  height: 2.25rem;
  border-radius: var(--radius-lg);
  background: var(--nook-surface);
  border: 1px solid var(--nook-border);
  color: var(--nook-muted);
}

.provider-card-body {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  min-width: 0;
}

.provider-card-name {
  font-size: 0.88rem;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.provider-card-type {
  font-size: 0.72rem;
  color: var(--nook-muted);
}

.provider-card-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
}

.provider-card-tag {
  font-size: 0.68rem;
  min-height: 1.25rem;
  padding: 0.05rem 0.45rem;
}

.provider-card-actions {
  display: flex;
  gap: 0.25rem;
  border-top: 1px solid var(--nook-border);
  padding-top: 0.625rem;
  margin-top: auto;
}

.provider-card-actions .action-button {
  flex: 1;
  height: 1.75rem;
  border-radius: var(--radius-md);
}

.action-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
}

a.action-button {
  border: 1px solid var(--nook-border);
  border-radius: var(--radius-lg);
  height: 2rem;
  padding: 0 0.625rem;
  text-decoration: none;
  cursor: pointer;
  transition: all 200ms;
}

a.action-button:hover {
  background: var(--nook-surface-strong);
  border-color: oklch(0.8 0 0);
}

.add-button {
  height: 2rem;
}

.empty {
  color: var(--nook-muted);
  padding: var(--space-4);
  border-radius: var(--radius-xl);
  border: 1px dashed var(--nook-border);
}

@media (max-width: 960px) {
  .item-card {
    flex-direction: column;
    align-items: stretch;
  }

  .item-actions {
    justify-content: flex-start;
  }

  .card-grid {
    grid-template-columns: 1fr;
  }
}
</style>
