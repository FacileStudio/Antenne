<script setup lang="ts">
import { Copy, Eye, EyeOff, Pencil, Plus, Trash2 } from "lucide-vue-next";
import { reactive, ref } from "vue";

import RoomIdEditor from "./RoomIdEditor.vue";
import SideDrawer from "./SideDrawer.vue";
import { useDashboardState } from "../composables/useDashboardState";
import {
  createDefaultDeliveryTarget,
  deliveryTargetTypes,
  getDeliveryTargetLabel,
  type DeliveryTarget,
  type DeliveryTargetType,
} from "../types";

const props = defineProps<{
  modelValue: DeliveryTarget[];
}>();

const emit = defineEmits<{
  "update:modelValue": [value: DeliveryTarget[]];
}>();

const { notifyError, notifySuccess, testDeliveryTarget } = useDashboardState();
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
        <Plus :size="16" aria-hidden="true" />
        <span>Add target</span>
      </button>
    </div>

    <div v-if="modelValue.length === 0" class="empty">No delivery targets configured.</div>

    <div v-else class="items">
      <article
        v-for="(target, index) in modelValue"
        :key="target.id"
        class="item-card"
        :class="{ dimmed: !target.enabled }"
      >
        <div class="item-info">
          <strong>{{ target.name || `${getDeliveryTargetLabel(target.type)} ${index + 1}` }}</strong>
          <span class="item-chip">{{ getDeliveryTargetLabel(target.type) }}</span>
          <span v-if="!target.enabled" class="item-chip item-chip--off">Off</span>
        </div>

        <div class="item-actions">
          <button class="ghost action-button" type="button" @click="openEditor(index)">
            <Pencil :size="16" aria-hidden="true" />
            <span>Edit</span>
          </button>
          <button class="ghost action-button" type="button" @click="duplicateTarget(index)">
            <Copy :size="16" aria-hidden="true" />
          </button>
          <button class="danger action-button" type="button" @click="removeTarget(index)">
            <Trash2 :size="16" aria-hidden="true" />
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
        <label class="toggle">
          <input :checked="editingTarget.enabled" type="checkbox" @change="patchTarget(editingIndex, { enabled: ($event.target as HTMLInputElement).checked })">
          <span>Enabled</span>
        </label>

        <label class="field">
          <span>Name</span>
          <input :value="editingTarget.name" type="text" @input="patchTarget(editingIndex, { name: ($event.target as HTMLInputElement).value })">
        </label>

        <template v-if="editingTarget.type === 'matrix'">
          <label class="field">
            <span>Homeserver</span>
            <input :value="editingTarget.homeserver" type="url" placeholder="https://matrix.example.com" @input="patchTarget(editingIndex, { homeserver: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field">
            <span>Access token</span>
            <div class="secret-field">
              <input
                :value="editingTarget.accessToken"
                :type="revealedSecrets[editingTarget.id] ? 'text' : 'password'"
                placeholder="syt_xxx"
                @input="patchTarget(editingIndex, { accessToken: ($event.target as HTMLInputElement).value })"
              >
              <button class="ghost action-button reveal-button" type="button" @click="toggleReveal(editingTarget.id)">
                <component :is="revealedSecrets[editingTarget.id] ? EyeOff : Eye" :size="16" aria-hidden="true" />
                <span>{{ revealedSecrets[editingTarget.id] ? "Hide" : "Show" }}</span>
              </button>
            </div>
          </label>

          <label class="field">
            <span>Room IDs</span>
            <RoomIdEditor :model-value="editingTarget.roomIds" @update:model-value="patchTarget(editingIndex, { roomIds: $event })" />
          </label>
        </template>

        <template v-else-if="editingTarget.type === 'discord'">
          <label class="field">
            <span>Webhook URL</span>
            <input :value="editingTarget.webhookUrl" type="url" placeholder="https://discord.com/api/webhooks/..." @input="patchTarget(editingIndex, { webhookUrl: ($event.target as HTMLInputElement).value })">
          </label>
        </template>

        <template v-else>
          <label class="field">
            <span>Webhook URL</span>
            <input :value="editingTarget.url" type="url" placeholder="https://example.com/hooks/nook" @input="patchTarget(editingIndex, { url: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field">
            <span>Secret header</span>
            <input :value="editingTarget.secretHeader ?? ''" type="text" placeholder="x-nook-secret" @input="patchTarget(editingIndex, { secretHeader: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field">
            <span>Secret value</span>
            <div class="secret-field">
              <input
                :value="editingTarget.secretValue ?? ''"
                :type="revealedSecrets[editingTarget.id] ? 'text' : 'password'"
                @input="patchTarget(editingIndex, { secretValue: ($event.target as HTMLInputElement).value })"
              >
              <button class="ghost action-button reveal-button" type="button" @click="toggleReveal(editingTarget.id)">
                <component :is="revealedSecrets[editingTarget.id] ? EyeOff : Eye" :size="16" aria-hidden="true" />
                <span>{{ revealedSecrets[editingTarget.id] ? "Hide" : "Show" }}</span>
              </button>
            </div>
          </label>
        </template>
      </template>

      <template v-if="editingTarget !== null" #footer>
        <button
          class="secondary action-button"
          type="button"
          :disabled="testingTargets[editingTarget.id]"
          @click="runTargetTest(editingTarget)"
        >
          <span>{{ testingTargets[editingTarget.id] ? "Testing..." : "Test" }}</span>
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
  border-radius: 1.25rem;
  border: 1px solid var(--nook-border);
  background: var(--nook-surface-strong);
  flex-wrap: wrap;
}

.item-card.dimmed {
  opacity: 0.5;
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
