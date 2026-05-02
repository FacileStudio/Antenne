<script setup lang="ts">
import { Icon } from "@iconify/vue";
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
        <Icon icon="solar:add-circle-bold" width="16" aria-hidden="true" />
        <span>Add provider</span>
      </button>
    </div>

    <div v-if="modelValue.length === 0" class="empty">No providers configured.</div>

    <div v-else class="items">
      <article
        v-for="(provider, index) in modelValue"
        :key="provider.id"
        class="item-card"
        :class="{ dimmed: !provider.enabled, 'drag-over': dragOverIndex === index && dragIndex !== index }"
        draggable="true"
        @dragstart="onDragStart(index)"
        @dragover="onDragOver($event, index)"
        @dragend="onDragEnd"
      >
        <div class="item-info">
          <span class="drag-handle" title="Drag to reorder"><Icon icon="solar:sort-vertical-bold" width="18" aria-hidden="true" /></span>
          <strong>{{ provider.name || `${getProviderLabel(provider.type)} ${index + 1}` }}</strong>
          <span class="item-chip">{{ getProviderLabel(provider.type) }}</span>
          <span v-if="!provider.enabled" class="item-chip item-chip--off">Off</span>
        </div>

        <div class="item-actions">
          <button class="ghost action-button" type="button" @click="openEditor(index)">
            <Icon icon="solar:pen-2-bold" width="16" aria-hidden="true" />
            <span>Edit</span>
          </button>
          <NuxtLink class="ghost action-button" :to="`/providers/${provider.id}`">
            <Icon icon="solar:chart-bold" width="16" aria-hidden="true" />
          </NuxtLink>
          <button class="ghost action-button" type="button" @click="duplicateProvider(index)">
            <Icon icon="solar:copy-bold" width="16" aria-hidden="true" />
          </button>
          <button class="danger action-button" type="button" @click="removeProvider(index)">
            <Icon icon="solar:trash-bin-2-bold" width="16" aria-hidden="true" />
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
          <Icon icon="solar:plug-circle-bold" width="16" aria-hidden="true" />
          <span>{{ testingProviders[editingProvider.id] ? "Testing..." : "Test connection" }}</span>
        </button>
        <button class="primary action-button" type="button" :disabled="saving" @click="saveSettings">
          <Icon icon="solar:diskette-bold" width="16" aria-hidden="true" />
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

.item-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-radius: 1.25rem;
  border: 1px solid var(--nook-border);
  background: var(--nook-surface-strong);
  flex-wrap: wrap;
}

.item-card.dimmed {
  opacity: 0.5;
}

.item-card.drag-over {
  border-color: rgb(255 255 255 / 0.3);
  background: rgb(255 255 255 / 0.08);
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
  min-height: 1.5rem;
  padding: 0 0.6rem;
  border-radius: 999px;
  border: 1px solid var(--nook-border);
  color: var(--nook-muted);
  font-size: 0.8rem;
}

.item-chip--off {
  border-color: rgb(186 55 55 / 0.3);
  color: rgb(255 131 131);
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
  min-height: 3.5rem;
  padding-inline: 0.95rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
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
  border-radius: 999px;
  padding: 0.85rem 1.15rem;
  text-decoration: none;
  cursor: pointer;
  transition: background-color 150ms ease, border-color 150ms ease;
}

a.action-button:hover {
  background: var(--nook-surface-strong);
  border-color: rgb(255 255 255 / 0.14);
}

.add-button {
  min-height: 3.5rem;
}

.empty {
  color: var(--nook-muted);
  padding: var(--space-4);
  border-radius: 1.25rem;
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
