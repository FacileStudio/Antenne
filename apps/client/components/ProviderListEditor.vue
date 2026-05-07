<script setup lang="ts">
import { Pencil, BarChart2, Copy, Trash2, PlusCircle, GripVertical, PlugZap, Save } from "lucide-vue-next";
import { reactive, ref } from "vue";

import ProviderEditorFields from "./ProviderEditorFields.vue";
import SideDrawer from "./SideDrawer.vue";
import { useDashboardState } from "../composables/useDashboardState";
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

const availableTags = computed(() =>
  [...new Set(props.modelValue.flatMap((provider) => provider.tags))]
    .sort((left, right) => left.localeCompare(right))
);

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

function duplicateProvider(index: number) {
  const provider = props.modelValue[index];
  if (!provider) {
    return;
  }

  const duplicate = {
    ...provider,
    id: globalThis.crypto.randomUUID(),
    name: provider.name ? `${provider.name} copy` : getProviderLabel(provider.type),
  } as ProviderSettings;
  const nextProviders = [...props.modelValue];
  nextProviders.splice(index + 1, 0, duplicate);
  updateProviders(nextProviders);

  if (editingIndex.value !== null && editingIndex.value > index) {
    editingIndex.value++;
  }
}

function toggleReveal(id: string) {
  revealedSecrets[id] = !revealedSecrets[id];
}

function toggleTagFilter(tag: string) {
  selectedTags.value = selectedTags.value.includes(tag)
    ? selectedTags.value.filter((currentTag) => currentTag !== tag)
    : [...selectedTags.value, tag];
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
        <PlusCircle :size="16" aria-hidden="true" />
        <span>Add provider</span>
      </button>
    </div>

    <div v-if="modelValue.length === 0" class="empty">No providers configured.</div>

    <div v-else-if="availableTags.length > 0" class="filter-bar">
      <span class="filter-label">Filter by tags</span>
      <button
        v-for="tag in availableTags"
        :key="tag"
        class="tag-filter"
        :class="{ active: selectedTags.includes(tag) }"
        type="button"
        @click="toggleTagFilter(tag)"
      >
        {{ tag }}
      </button>
      <button
        v-if="selectedTags.length > 0"
        class="ghost action-button clear-filters"
        type="button"
        @click="selectedTags = []"
      >
        Clear
      </button>
    </div>

    <div v-if="modelValue.length > 0 && visibleProviders.length === 0" class="empty">No providers match current tag filter.</div>

    <div v-else-if="modelValue.length > 0" class="items">
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
          <span class="drag-handle" title="Drag to reorder"><GripVertical :size="18" aria-hidden="true" /></span>
          <strong>{{ provider.name || `${getProviderLabel(provider.type)} ${index + 1}` }}</strong>
          <span class="item-chip">{{ getProviderLabel(provider.type) }}</span>
          <span v-if="!provider.enabled" class="item-chip item-chip--off">Off</span>
          <span v-for="tag in provider.tags" :key="tag" class="item-chip item-chip--tag">{{ tag }}</span>
        </div>

        <div class="item-actions">
          <button class="ghost action-button" type="button" @click="openEditor(index)">
            <Pencil :size="16" aria-hidden="true" />
            <span>Edit</span>
          </button>
          <NuxtLink class="ghost action-button" :to="`/providers/${provider.id}`">
            <BarChart2 :size="16" aria-hidden="true" />
          </NuxtLink>
          <button class="ghost action-button" type="button" @click="duplicateProvider(index)">
            <Copy :size="16" aria-hidden="true" />
          </button>
          <button class="danger action-button" type="button" @click="removeProvider(index)">
            <Trash2 :size="16" aria-hidden="true" />
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
          <PlugZap :size="16" aria-hidden="true" />
          <span>{{ testingProviders[editingProvider.id] ? "Testing..." : "Test connection" }}</span>
        </button>
        <button class="primary action-button" type="button" :disabled="saving" @click="saveSettings">
          <Save :size="16" aria-hidden="true" />
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
  border: 1px solid var(--nook-border);
  background: var(--nook-surface);
  color: var(--nook-muted);
  border-radius: var(--radius-pill);
  padding: 0.32rem 0.7rem;
  font: inherit;
  cursor: pointer;
  transition: border-color 150ms ease, color 150ms ease, background-color 150ms ease;
}

.tag-filter.active {
  color: var(--nook-text);
  border-color: rgb(28 219 129 / 0.45);
  background: rgb(28 219 129 / 0.08);
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

.item-chip--off {
  border-color: oklch(60% 0.15 25 / 60%);
  color: oklch(45% 0.18 25);
}

.item-chip--tag {
  border-color: rgb(159 221 255 / 0.22);
  background: rgb(159 221 255 / 0.08);
  color: rgb(159 221 255);
}

.item-actions {
  display: flex;
  gap: var(--space-1);
  flex-wrap: wrap;
  justify-content: flex-end;
}

.secret-field {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: var(--space-2);
  align-items: stretch;
}

.reveal-button {
  height: 2rem;
  padding-inline: 0.625rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.375rem;
  line-height: 1;
}

.reveal-button span {
  display: inline-flex;
  align-items: center;
}

.reveal-button :deep(svg) {
  flex: none;
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

  .secret-field {
    grid-template-columns: 1fr;
  }
}
</style>
