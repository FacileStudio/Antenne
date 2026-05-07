<script setup lang="ts">
import { Icon } from "@iconify/vue";
import { reactive, ref } from "vue";

import DeliveryTargetEditorFields from "./DeliveryTargetEditorFields.vue";
import SideDrawer from "./SideDrawer.vue";
import { useDashboardState } from "../composables/useDashboardState";
import {
  createDefaultDeliveryTarget,
  deliveryTargetTypes,
  getDeliveryTargetLabel,
  type DeliveryTarget,
  type DeliveryTargetType,
  type ProviderSettings,
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
  const nextTargets = [
    createDefaultDeliveryTarget(nextTargetType.value),
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

function duplicateTarget(index: number) {
  const target = props.modelValue[index];
  if (!target) {
    return;
  }

  const duplicate = {
    ...target,
    id: globalThis.crypto.randomUUID(),
    name: target.name ? `${target.name} copy` : getDeliveryTargetLabel(target.type),
  } as DeliveryTarget;
  const nextTargets = [...props.modelValue];
  nextTargets.splice(index + 1, 0, duplicate);
  updateTargets(nextTargets);

  if (editingIndex.value !== null && editingIndex.value > index) {
    editingIndex.value++;
  }
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
        <Icon icon="solar:add-circle-bold" width="16" aria-hidden="true" />
        <span>Add target</span>
      </button>
    </div>

    <div v-if="modelValue.length === 0" class="empty">No delivery targets configured.</div>

    <div v-else class="items">
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
          <span class="drag-handle" title="Drag to reorder"><Icon icon="solar:sort-vertical-bold" width="18" aria-hidden="true" /></span>
          <strong>{{ target.name || `${getDeliveryTargetLabel(target.type)} ${index + 1}` }}</strong>
          <span class="item-chip">{{ getDeliveryTargetLabel(target.type) }}</span>
          <span v-if="!target.enabled" class="item-chip item-chip--off">Off</span>
        </div>

        <div class="item-actions">
          <button class="ghost action-button" type="button" @click="openEditor(index)">
            <Icon icon="solar:pen-2-bold" width="16" aria-hidden="true" />
            <span>Edit</span>
          </button>
          <NuxtLink class="ghost action-button" :to="`/targets/${target.id}`">
            <Icon icon="solar:chart-bold" width="16" aria-hidden="true" />
          </NuxtLink>
          <button class="ghost action-button" type="button" @click="duplicateTarget(index)">
            <Icon icon="solar:copy-bold" width="16" aria-hidden="true" />
          </button>
          <button class="danger action-button" type="button" @click="removeTarget(index)">
            <Icon icon="solar:trash-bin-2-bold" width="16" aria-hidden="true" />
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
          <Icon icon="solar:plain-bold" width="16" aria-hidden="true" />
          <span>{{ testingTargets[editingTarget.id] ? "Testing..." : "Test saved" }}</span>
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
}

.item-card.dimmed {
  opacity: 0.5;
}

.item-card.drag-over {
  border-color: oklch(1 0 0 / 30%);
  background: oklch(1 0 0 / 8%);
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
  border-color: oklch(40% 0.15 25 / 30%);
  color: oklch(75% 0.12 20);
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
  border-color: oklch(1 0 0 / 14%);
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

  .secret-field {
    grid-template-columns: 1fr;
  }
}
</style>
