<script setup lang="ts">
import { Copy, Eye, EyeOff, Plus, Trash2 } from "lucide-vue-next";
import { reactive, ref } from "vue";

import RoomIdEditor from "./RoomIdEditor.vue";
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

function updateTargets(nextTargets: DeliveryTarget[]) {
  emit("update:modelValue", nextTargets);
}

function addTarget() {
  updateTargets([
    createDefaultDeliveryTarget(nextTargetType.value),
    ...props.modelValue,
  ]);
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
}

function removeTarget(index: number) {
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

      <button class="primary action-button add-target-button" type="button" @click="addTarget">
        <Plus :size="16" aria-hidden="true" />
        <span>Add target</span>
      </button>
    </div>

    <div v-if="modelValue.length === 0" class="empty">No delivery targets configured.</div>

    <div v-else class="targets">
      <article v-for="(target, index) in modelValue" :key="target.id" class="target-card">
        <div class="target-card-header">
          <div class="target-heading">
            <strong>{{ target.name || `${getDeliveryTargetLabel(target.type)} ${index + 1}` }}</strong>
            <span class="target-chip">{{ getDeliveryTargetLabel(target.type) }}</span>
          </div>

          <div class="target-actions">
            <button class="ghost action-button" type="button" :disabled="testingTargets[target.id]" @click="runTargetTest(target)">
              <span>{{ testingTargets[target.id] ? "Testing..." : "Test" }}</span>
            </button>
            <button class="ghost action-button" type="button" @click="duplicateTarget(index)">
              <Copy :size="16" aria-hidden="true" />
              <span>Duplicate</span>
            </button>
            <button class="danger action-button" type="button" @click="removeTarget(index)">
              <Trash2 :size="16" aria-hidden="true" />
              <span>Remove</span>
            </button>
          </div>
        </div>

        <label class="toggle">
          <input :checked="target.enabled" type="checkbox" @change="patchTarget(index, { enabled: ($event.target as HTMLInputElement).checked })">
          <span>Enabled</span>
        </label>

        <div class="target-fields">
          <label class="field">
            <span>Name</span>
            <input :value="target.name" type="text" @input="patchTarget(index, { name: ($event.target as HTMLInputElement).value })">
          </label>
        </div>

        <div v-if="target.type === 'matrix'" class="target-fields detail-grid">
          <label class="field">
            <span>Homeserver</span>
            <input :value="target.homeserver" type="url" placeholder="https://matrix.example.com" @input="patchTarget(index, { homeserver: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field">
            <span>Access token</span>
            <div class="secret-field">
              <input
                :value="target.accessToken"
                :type="revealedSecrets[target.id] ? 'text' : 'password'"
                placeholder="syt_xxx"
                @input="patchTarget(index, { accessToken: ($event.target as HTMLInputElement).value })"
              >
              <button class="ghost action-button reveal-button" type="button" @click="toggleReveal(target.id)">
                <component :is="revealedSecrets[target.id] ? EyeOff : Eye" :size="16" aria-hidden="true" />
                <span>{{ revealedSecrets[target.id] ? "Hide" : "Show" }}</span>
              </button>
            </div>
          </label>

          <label class="field wide">
            <span>Room IDs</span>
            <RoomIdEditor :model-value="target.roomIds" @update:model-value="patchTarget(index, { roomIds: $event })" />
          </label>
        </div>

        <div v-else-if="target.type === 'discord'" class="target-fields detail-grid">
          <label class="field wide">
            <span>Webhook URL</span>
            <input :value="target.webhookUrl" type="url" placeholder="https://discord.com/api/webhooks/..." @input="patchTarget(index, { webhookUrl: ($event.target as HTMLInputElement).value })">
          </label>
        </div>

        <div v-else class="target-fields detail-grid">
          <label class="field wide">
            <span>Webhook URL</span>
            <input :value="target.url" type="url" placeholder="https://example.com/hooks/nook" @input="patchTarget(index, { url: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field">
            <span>Secret header</span>
            <input :value="target.secretHeader ?? ''" type="text" placeholder="x-nook-secret" @input="patchTarget(index, { secretHeader: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field wide">
            <span>Secret value</span>
            <div class="secret-field">
              <input
                :value="target.secretValue ?? ''"
                :type="revealedSecrets[target.id] ? 'text' : 'password'"
                @input="patchTarget(index, { secretValue: ($event.target as HTMLInputElement).value })"
              >
              <button class="ghost action-button reveal-button" type="button" @click="toggleReveal(target.id)">
                <component :is="revealedSecrets[target.id] ? EyeOff : Eye" :size="16" aria-hidden="true" />
                <span>{{ revealedSecrets[target.id] ? "Hide" : "Show" }}</span>
              </button>
            </div>
          </label>
        </div>
      </article>
    </div>
  </div>
</template>

<style scoped>
.delivery-list,
.targets {
  display: grid;
  gap: var(--space-4);
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

.target-card {
  display: grid;
  gap: var(--space-4);
  padding: var(--space-4);
  border-radius: 1.25rem;
  border: 1px solid var(--nook-border);
  background: var(--nook-surface-strong);
}

.target-card-header,
.target-fields {
  display: flex;
  justify-content: space-between;
  gap: var(--space-4);
  align-items: flex-start;
  flex-wrap: wrap;
}

.target-heading {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.target-chip {
  display: inline-flex;
  align-items: center;
  min-height: 1.75rem;
  padding: 0 0.7rem;
  border-radius: 999px;
  border: 1px solid var(--nook-border);
  color: var(--nook-muted);
  font-size: 0.85rem;
}

.target-actions {
  display: flex;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.detail-grid {
  display: grid;
  gap: var(--space-3);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.field.wide {
  grid-column: span 2;
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

.add-target-button {
  min-height: 3.5rem;
}

.empty {
  color: var(--nook-muted);
  padding: var(--space-4);
  border-radius: 1.25rem;
  border: 1px dashed var(--nook-border);
}

@media (max-width: 960px) {
  .target-card-header,
  .target-fields {
    flex-direction: column;
    align-items: stretch;
  }

  .detail-grid {
    grid-template-columns: 1fr;
  }

  .field.wide {
    grid-column: auto;
  }

  .secret-field {
    grid-template-columns: 1fr;
  }
}
</style>
