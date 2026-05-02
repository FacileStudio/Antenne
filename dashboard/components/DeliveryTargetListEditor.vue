<script setup lang="ts">
import { Icon } from "@iconify/vue";
import { reactive, ref } from "vue";

import RoomIdEditor from "./RoomIdEditor.vue";
import SideDrawer from "./SideDrawer.vue";
import SmtpAddressEditor from "./SmtpAddressEditor.vue";
import { useDashboardState } from "../composables/useDashboardState";
import {
  createDefaultDeliveryTarget,
  deliveryTargetTypes,
  getDeliveryTargetLabel,
  getProviderLabel,
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
          <span class="drag-handle"><Icon icon="solar:menu-dots-bold" width="16" aria-hidden="true" /></span>
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
        <label class="toggle">
          <input :checked="editingTarget.enabled" type="checkbox" @change="patchTarget(editingIndex, { enabled: ($event.target as HTMLInputElement).checked })">
          <span>Enabled</span>
        </label>

        <label class="field">
          <span>Name</span>
          <input :value="editingTarget.name" type="text" @input="patchTarget(editingIndex, { name: ($event.target as HTMLInputElement).value })">
        </label>

        <div class="field">
          <span>Providers</span>
          <p class="field-hint">Select which providers send to this target. Leave empty for all.</p>
          <div class="provider-picker">
            <label
              v-for="provider in providers"
              :key="provider.id"
              class="provider-option"
              :class="{ selected: (editingTarget.providerIds ?? []).includes(provider.id) }"
            >
              <input
                type="checkbox"
                :checked="(editingTarget.providerIds ?? []).includes(provider.id)"
                @change="toggleProvider(editingIndex, provider.id)"
              >
              <span>{{ provider.name || getProviderLabel(provider.type) }}</span>
              <span class="item-chip">{{ getProviderLabel(provider.type) }}</span>
            </label>
            <p v-if="providers.length === 0" class="field-hint">No providers configured yet.</p>
          </div>
        </div>

        <template v-if="editingTarget.type === 'matrix'">
          <label class="field required">
            <span>Homeserver</span>
            <input :value="editingTarget.homeserver" type="url" placeholder="https://matrix.example.com" @input="patchTarget(editingIndex, { homeserver: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field">
            <span>Access token</span>
            <p class="field-hint">Optional if password login is configured. Used directly until Matrix rejects it.</p>
            <div class="secret-field">
              <input
                :value="editingTarget.accessToken"
                :type="revealedSecrets[editingTarget.id] ? 'text' : 'password'"
                placeholder="syt_xxx"
                @input="patchTarget(editingIndex, { accessToken: ($event.target as HTMLInputElement).value })"
              >
              <button class="ghost action-button reveal-button" type="button" @click="toggleReveal(editingTarget.id)">
                <Icon :icon="revealedSecrets[editingTarget.id] ? 'solar:eye-closed-bold' : 'solar:eye-bold'" width="16" aria-hidden="true" />
                <span>{{ revealedSecrets[editingTarget.id] ? "Hide" : "Show" }}</span>
              </button>
            </div>
          </label>

          <label class="field">
            <span>User ID</span>
            <p class="field-hint">Use with password for auto re-login.</p>
            <input :value="editingTarget.userId" type="text" placeholder="@nook:matrix.example.com" @input="patchTarget(editingIndex, { userId: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field">
            <span>Password</span>
            <div class="secret-field">
              <input
                :value="editingTarget.password ?? ''"
                :type="revealedSecrets[`${editingTarget.id}-password`] ? 'text' : 'password'"
                placeholder="hunter2 but less cursed"
                @input="patchTarget(editingIndex, { password: ($event.target as HTMLInputElement).value })"
              >
              <button class="ghost action-button reveal-button" type="button" @click="toggleReveal(`${editingTarget.id}-password`)">
                <Icon :icon="revealedSecrets[`${editingTarget.id}-password`] ? 'solar:eye-closed-bold' : 'solar:eye-bold'" width="16" aria-hidden="true" />
                <span>{{ revealedSecrets[`${editingTarget.id}-password`] ? "Hide" : "Show" }}</span>
              </button>
            </div>
          </label>

          <label class="field">
            <span>Device ID</span>
            <p class="field-hint">Optional. Keep stable if you use it.</p>
            <input :value="editingTarget.deviceId ?? ''" type="text" placeholder="NOOKABC123" @input="patchTarget(editingIndex, { deviceId: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field">
            <span>Refresh token</span>
            <p class="field-hint">Optional. Used when access token expires.</p>
            <div class="secret-field">
              <input
                :value="editingTarget.refreshToken ?? ''"
                :type="revealedSecrets[`${editingTarget.id}-refresh`] ? 'text' : 'password'"
                placeholder="syr_xxx"
                @input="patchTarget(editingIndex, { refreshToken: ($event.target as HTMLInputElement).value })"
              >
              <button class="ghost action-button reveal-button" type="button" @click="toggleReveal(`${editingTarget.id}-refresh`)">
                <Icon :icon="revealedSecrets[`${editingTarget.id}-refresh`] ? 'solar:eye-closed-bold' : 'solar:eye-bold'" width="16" aria-hidden="true" />
                <span>{{ revealedSecrets[`${editingTarget.id}-refresh`] ? "Hide" : "Show" }}</span>
              </button>
            </div>
          </label>

          <label class="field required">
            <span>Room IDs</span>
            <RoomIdEditor :model-value="editingTarget.roomIds" @update:model-value="patchTarget(editingIndex, { roomIds: $event })" />
          </label>
        </template>

        <template v-else-if="editingTarget.type === 'discord'">
          <label class="field required">
            <span>Webhook URL</span>
            <input :value="editingTarget.webhookUrl" type="url" placeholder="https://discord.com/api/webhooks/..." @input="patchTarget(editingIndex, { webhookUrl: ($event.target as HTMLInputElement).value })">
          </label>
        </template>

        <template v-else-if="editingTarget.type === 'smtp'">
          <label class="field required">
            <span>Host</span>
            <input :value="editingTarget.host" type="text" placeholder="smtp.example.com" @input="patchTarget(editingIndex, { host: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field required">
            <span>Port</span>
            <input :value="editingTarget.port" type="number" placeholder="587" @input="patchTarget(editingIndex, { port: Number(($event.target as HTMLInputElement).value) })">
          </label>

          <label class="toggle">
            <input :checked="editingTarget.secure" type="checkbox" @change="patchTarget(editingIndex, { secure: ($event.target as HTMLInputElement).checked })">
            <span>Use TLS</span>
          </label>

          <label class="field">
            <span>Username</span>
            <input :value="editingTarget.username" type="text" placeholder="user@example.com" @input="patchTarget(editingIndex, { username: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field">
            <span>Password</span>
            <div class="secret-field">
              <input
                :value="editingTarget.password ?? ''"
                :type="revealedSecrets[editingTarget.id] ? 'text' : 'password'"
                @input="patchTarget(editingIndex, { password: ($event.target as HTMLInputElement).value })"
              >
              <button class="ghost action-button reveal-button" type="button" @click="toggleReveal(editingTarget.id)">
                <Icon :icon="revealedSecrets[editingTarget.id] ? 'solar:eye-closed-bold' : 'solar:eye-bold'" width="16" aria-hidden="true" />
                <span>{{ revealedSecrets[editingTarget.id] ? "Hide" : "Show" }}</span>
              </button>
            </div>
          </label>

          <label class="field required">
            <span>From address</span>
            <input :value="editingTarget.fromAddress" type="email" placeholder="nook@example.com" @input="patchTarget(editingIndex, { fromAddress: ($event.target as HTMLInputElement).value })">
          </label>

          <div class="field required">
            <span>To addresses</span>
            <SmtpAddressEditor :model-value="editingTarget.toAddresses" @update:model-value="patchTarget(editingIndex, { toAddresses: $event })" />
          </div>

          <label class="field">
            <span>Subject prefix</span>
            <input :value="editingTarget.subjectPrefix" type="text" placeholder="[Nook]" @input="patchTarget(editingIndex, { subjectPrefix: ($event.target as HTMLInputElement).value })">
          </label>
        </template>

        <template v-else-if="editingTarget.type === 'ntfy'">
          <label class="field required">
            <span>Server URL</span>
            <input :value="editingTarget.serverUrl" type="url" placeholder="https://ntfy.sh" @input="patchTarget(editingIndex, { serverUrl: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field required">
            <span>Topic</span>
            <input :value="editingTarget.topic" type="text" placeholder="my-alerts" @input="patchTarget(editingIndex, { topic: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field">
            <span>Access token</span>
            <div class="secret-field">
              <input
                :value="editingTarget.accessToken ?? ''"
                :type="revealedSecrets[editingTarget.id] ? 'text' : 'password'"
                placeholder="tk_xxx"
                @input="patchTarget(editingIndex, { accessToken: ($event.target as HTMLInputElement).value })"
              >
              <button class="ghost action-button reveal-button" type="button" @click="toggleReveal(editingTarget.id)">
                <Icon :icon="revealedSecrets[editingTarget.id] ? 'solar:eye-closed-bold' : 'solar:eye-bold'" width="16" aria-hidden="true" />
                <span>{{ revealedSecrets[editingTarget.id] ? "Hide" : "Show" }}</span>
              </button>
            </div>
          </label>

          <label class="field">
            <span>Priority</span>
            <input :value="editingTarget.priority ?? 3" type="number" min="1" max="5" placeholder="3" @input="patchTarget(editingIndex, { priority: Number(($event.target as HTMLInputElement).value) })">
          </label>

          <label class="field">
            <span>Tags</span>
            <input :value="editingTarget.tags.join(', ')" type="text" placeholder="rotating_light,warning" @input="patchTarget(editingIndex, { tags: ($event.target as HTMLInputElement).value.split(',').map((tag) => tag.trim()).filter(Boolean) })">
          </label>

          <label class="field">
            <span>Click URL</span>
            <input :value="editingTarget.clickUrl ?? ''" type="url" placeholder="https://status.example.com" @input="patchTarget(editingIndex, { clickUrl: ($event.target as HTMLInputElement).value })">
          </label>
        </template>

        <template v-else>
          <label class="field required">
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
                <Icon :icon="revealedSecrets[editingTarget.id] ? 'solar:eye-closed-bold' : 'solar:eye-bold'" width="16" aria-hidden="true" />
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
          <Icon icon="solar:plain-bold" width="16" aria-hidden="true" />
          <span>{{ testingTargets[editingTarget.id] ? "Testing..." : "Test" }}</span>
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
  padding: 0.6rem 0.85rem;
  border-radius: 0.85rem;
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
