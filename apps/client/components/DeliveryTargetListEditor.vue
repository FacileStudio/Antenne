<script setup lang="ts">
import { Icon } from "@iconify/vue";
import { reactive, ref } from "vue";

import DeliveryTargetEditorFields from "./DeliveryTargetEditorFields.vue";
import SideDrawer from "./SideDrawer.vue";
import ToggleSwitch from "./ToggleSwitch.vue";
import ViewToggle from "./ViewToggle.vue";
import { useDashboardState } from "../composables/useDashboardState";
import { getDeliveryIcon } from "../utils/icons";
import {
  createDefaultDeliveryTarget,
  deliveryTargetTypes,
  getDeliveryTargetLabel,
  type DeliveryTarget,
  type DeliveryTargetType,
  type ProviderSettings,
  SYSTEM_PROVIDER_ID,
} from "../types";

const props = defineProps<{
  modelValue: DeliveryTarget[];
  providers: ProviderSettings[];
}>();

const emit = defineEmits<{
  "update:modelValue": [value: DeliveryTarget[]];
}>();

const { notifyError, notifySuccess, testDeliveryTarget, saveSettings, saving } = useDashboardState();
const nextTargetType = ref<DeliveryTargetType>("matrix");
const revealedSecrets = reactive<Record<string, boolean>>({});
const testingTargets = reactive<Record<string, boolean>>({});
const editingIndex = ref<number | null>(null);
const viewMode = ref<"list" | "card">("card");

const editingTarget = computed(() =>
  editingIndex.value !== null ? props.modelValue[editingIndex.value] ?? null : null
);

function openEditor(index: number) {
  editingIndex.value = index;
}

function closeEditor() {
  editingIndex.value = null;
}

function updateTargets(nextTargets: DeliveryTarget[]) {
  emit("update:modelValue", nextTargets);
}

function addTarget() {
  const providerIds = [SYSTEM_PROVIDER_ID, ...props.providers.map((provider) => provider.id)];
  const nextTargets = [
    { ...createDefaultDeliveryTarget(nextTargetType.value), providerIds },
    ...props.modelValue,
  ];
  updateTargets(nextTargets);
  editingIndex.value = 0;
}

function patchTarget(index: number, patch: Partial<DeliveryTarget>) {
  const target = props.modelValue[index];
  if (!target) {
    return;
  }

  updateTargets(
    props.modelValue.map((currentTarget, currentIndex) =>
      currentIndex === index ? { ...target, ...patch } as DeliveryTarget : currentTarget
    ),
  );
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
    const nextTargets = [...props.modelValue];
    const [moved] = nextTargets.splice(dragIndex.value, 1);
    nextTargets.splice(dragOverIndex.value, 0, moved);
    updateTargets(nextTargets);

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

function removeTarget(index: number) {
  if (editingIndex.value === index) {
    closeEditor();
  } else if (editingIndex.value !== null && editingIndex.value > index) {
    editingIndex.value--;
  }
  updateTargets(props.modelValue.filter((_, currentIndex) => currentIndex !== index));
}

function toggleEnabled(index: number) {
  const target = props.modelValue[index];
  if (!target) return;
  patchTarget(index, { enabled: !target.enabled });
  saveSettings();
}

function toggleReveal(id: string) {
  revealedSecrets[id] = !revealedSecrets[id];
}

function toggleProvider(index: number, providerId: string) {
  const target = props.modelValue[index];
  if (!target) return;
  const current = target.providerIds ?? [];
  const next = current.includes(providerId)
    ? current.filter((id) => id !== providerId)
    : [...current, providerId];
  patchTarget(index, { providerIds: next });
}

async function runTargetTest(target: DeliveryTarget) {
  testingTargets[target.id] = true;
  const label = target.name || getDeliveryTargetLabel(target.type);

  try {
    await testDeliveryTarget(target.id);
    notifySuccess(`Test sent to ${label}.`);
  } catch (error) {
    notifyError(error instanceof Error ? error.message : "Failed to test delivery target");
  } finally {
    testingTargets[target.id] = false;
  }
}
</script>

<template>
  <div class="delivery-list">
    <div class="toolbar">
      <label class="field toolbar-field">
        <span>New target type</span>
        <select v-model="nextTargetType">
          <option v-for="type in deliveryTargetTypes" :key="type" :value="type">
            {{ getDeliveryTargetLabel(type) }}
          </option>
        </select>
      </label>

      <button class="primary action-button add-button" type="button" @click="addTarget">
        <Icon icon="mdi:plus" class="button-icon" :width="16" :height="16" aria-hidden="true" />
        <span>Add target</span>
      </button>

      <ViewToggle v-if="modelValue.length > 0" v-model="viewMode" />
    </div>

    <div v-if="modelValue.length === 0" class="empty">No delivery targets configured.</div>

    <div v-else-if="viewMode === 'list'" class="items">
      <article
        v-for="(target, index) in modelValue"
        :key="target.id"
        class="item-card"
        :class="{ dimmed: !target.enabled, 'drag-over': dragOverIndex === index && dragIndex !== index }"
        draggable="true"
        @dragstart="onDragStart(index)"
        @dragover="onDragOver($event, index)"
        @dragend="onDragEnd"
      >
        <div class="item-info">
          <span class="drag-handle" title="Drag to reorder"><Icon icon="solar:hamburger-menu-linear" :width="18" :height="18" aria-hidden="true" /></span>
          <ToggleSwitch :model-value="target.enabled" size="sm" @update:model-value="toggleEnabled(index)" />
          <Icon :icon="getDeliveryIcon(target.type)" :width="20" :height="20" class="item-icon" aria-hidden="true" />
          <strong>{{ target.name || `${getDeliveryTargetLabel(target.type)} ${index + 1}` }}</strong>
          <span class="item-chip">{{ getDeliveryTargetLabel(target.type) }}</span>
        </div>

        <div class="item-actions">
          <button class="ghost action-button" type="button" @click="openEditor(index)">
            <Icon icon="solar:pen-linear" :width="16" :height="16" aria-hidden="true" />
            <span>Edit</span>
          </button>
          <NuxtLink class="ghost action-button" :to="`/targets/${target.id}`">
            <Icon icon="solar:chart-2-linear" :width="16" :height="16" aria-hidden="true" />
          </NuxtLink>
          <button class="danger action-button" type="button" @click="removeTarget(index)">
            <Icon icon="solar:trash-bin-trash-linear" :width="16" :height="16" aria-hidden="true" />
          </button>
        </div>
      </article>
    </div>

    <div v-else class="card-grid">
      <article
        v-for="(target, index) in modelValue"
        :key="target.id"
        class="target-card"
        :class="{ dimmed: !target.enabled }"
      >
        <div class="target-card-header">
          <div class="target-card-icon-wrap">
            <Icon :icon="getDeliveryIcon(target.type)" :width="24" :height="24" aria-hidden="true" />
          </div>
          <ToggleSwitch :model-value="target.enabled" size="sm" @update:model-value="toggleEnabled(index)" />
        </div>

        <div class="target-card-body">
          <span class="target-card-name">{{ target.name || `${getDeliveryTargetLabel(target.type)} ${index + 1}` }}</span>
          <span class="target-card-type">{{ getDeliveryTargetLabel(target.type) }}</span>
        </div>

        <div class="target-card-meta">
          <span class="target-card-stat">
            <Icon icon="solar:link-bold-duotone" :width="13" :height="13" aria-hidden="true" />
            {{ target.providerIds.length }} sources
          </span>
        </div>

        <div class="target-card-actions">
          <button class="ghost action-button" type="button" @click="openEditor(index)" title="Edit">
            <Icon icon="solar:pen-linear" :width="15" :height="15" aria-hidden="true" />
          </button>
          <NuxtLink class="ghost action-button" :to="`/targets/${target.id}`" title="Activity">
            <Icon icon="solar:chart-2-linear" :width="15" :height="15" aria-hidden="true" />
          </NuxtLink>
          <button class="danger action-button" type="button" @click="removeTarget(index)" title="Delete">
            <Icon icon="solar:trash-bin-trash-linear" :width="15" :height="15" aria-hidden="true" />
          </button>
        </div>
      </article>
    </div>

    <SideDrawer
      :open="editingTarget !== null"
      :title="`Edit ${editingTarget ? (editingTarget.name || getDeliveryTargetLabel(editingTarget.type)) : 'target'}`"
      @close="closeEditor"
    >
      <template v-if="editingTarget !== null && editingIndex !== null">
        <DeliveryTargetEditorFields
          :editing-target="editingTarget"
          :providers="providers"
          :revealed-secrets="revealedSecrets"
          @patch="patchTarget(editingIndex, $event)"
          @toggle-provider="toggleProvider(editingIndex, $event)"
          @toggle-reveal="toggleReveal($event)"
        />
      </template>

      <template v-if="editingTarget !== null" #footer>
        <button
          class="secondary action-button"
          type="button"
          :disabled="testingTargets[editingTarget.id]"
          @click="runTargetTest(editingTarget)"
        >
          <Icon icon="solar:plain-linear" :width="16" :height="16" aria-hidden="true" />
          <span>{{ testingTargets[editingTarget.id] ? "Testing..." : "Test saved" }}</span>
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
.delivery-list,
.items {
  display: grid;
  gap: var(--space-3);
}

.toolbar {
  display: flex;
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

.target-card {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1rem;
  border-radius: var(--radius-xl);
  border: 1px solid var(--nook-border);
  background: var(--nook-surface-strong);
  transition: border-color 150ms, box-shadow 150ms, opacity 150ms;
}

.target-card:hover {
  border-color: oklch(0.8 0 0);
  box-shadow: 0 2px 8px oklch(0 0 0 / 0.04);
}

.target-card.dimmed {
  opacity: 0.5;
}

.target-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.target-card-icon-wrap {
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

.target-card-body {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  min-width: 0;
}

.target-card-name {
  font-size: 0.88rem;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.target-card-type {
  font-size: 0.72rem;
  color: var(--nook-muted);
}

.target-card-meta {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.target-card-stat {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  font-size: 0.72rem;
  color: var(--nook-muted);
}

.target-card-actions {
  display: flex;
  gap: 0.25rem;
  border-top: 1px solid var(--nook-border);
  padding-top: 0.625rem;
  margin-top: auto;
}

.target-card-actions .action-button {
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

.field-hint {
  margin: 0;
  font-size: 0.82rem;
  color: var(--nook-muted);
}

.provider-picker {
  display: grid;
  gap: 0.35rem;
}

.provider-option {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.5rem 0.75rem;
  border-radius: var(--radius-lg);
  border: 1px solid var(--nook-border);
  background: var(--nook-surface);
  cursor: pointer;
  transition: background 150ms ease, border-color 150ms ease;
}

.provider-option:hover {
  background: var(--nook-surface-strong);
}

.provider-option.selected {
  border-color: rgb(28 219 129 / 0.4);
  background: rgb(28 219 129 / 0.06);
}

.provider-option input[type="checkbox"] {
  width: auto;
  margin: 0;
  accent-color: #1CDB81;
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
