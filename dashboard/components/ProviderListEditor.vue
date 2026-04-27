<script setup lang="ts">
import { ArrowDown, ArrowUp, BarChart3, Copy, Eye, EyeOff, Pencil, Plug, Plus, Save, Send, Trash2 } from "lucide-vue-next";
import { reactive, ref } from "vue";

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

function moveProvider(index: number, direction: -1 | 1) {
  const targetIndex = index + direction;
  if (targetIndex < 0 || targetIndex >= props.modelValue.length) {
    return;
  }

  const nextProviders = [...props.modelValue];
  const [provider] = nextProviders.splice(index, 1);
  nextProviders.splice(targetIndex, 0, provider);
  updateProviders(nextProviders);

  if (editingIndex.value === index) {
    editingIndex.value = targetIndex;
  } else if (editingIndex.value === targetIndex) {
    editingIndex.value = index;
  }
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
        <Plus :size="16" aria-hidden="true" />
        <span>Add provider</span>
      </button>
    </div>

    <div v-if="modelValue.length === 0" class="empty">No providers configured.</div>

    <div v-else class="items">
      <article
        v-for="(provider, index) in modelValue"
        :key="provider.id"
        class="item-card"
        :class="{ dimmed: !provider.enabled }"
      >
        <div class="item-info">
          <strong>{{ provider.name || `${getProviderLabel(provider.type)} ${index + 1}` }}</strong>
          <span class="item-chip">{{ getProviderLabel(provider.type) }}</span>
          <span v-if="!provider.enabled" class="item-chip item-chip--off">Off</span>
        </div>

        <div class="item-actions">
          <button class="ghost action-button" type="button" :disabled="index === 0" @click="moveProvider(index, -1)">
            <ArrowUp :size="16" aria-hidden="true" />
          </button>
          <button class="ghost action-button" type="button" :disabled="index === modelValue.length - 1" @click="moveProvider(index, 1)">
            <ArrowDown :size="16" aria-hidden="true" />
          </button>
          <button class="ghost action-button" type="button" @click="openEditor(index)">
            <Pencil :size="16" aria-hidden="true" />
            <span>Edit</span>
          </button>
          <NuxtLink class="ghost action-button" :to="`/providers/${provider.id}`">
            <BarChart3 :size="16" aria-hidden="true" />
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
        <label class="toggle">
          <input :checked="editingProvider.enabled" type="checkbox" @change="patchProvider(editingIndex, { enabled: ($event.target as HTMLInputElement).checked })">
          <span>Enabled</span>
        </label>

        <label class="field">
          <span>Name</span>
          <input :value="editingProvider.name" type="text" @input="patchProvider(editingIndex, { name: ($event.target as HTMLInputElement).value })">
        </label>

        <template v-if="editingProvider.type === 'website'">
          <label class="field required">
            <span>URL</span>
            <input :value="editingProvider.url" type="url" @input="patchProvider(editingIndex, { url: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field">
            <span>Interval</span>
            <input :value="editingProvider.interval" type="number" min="1" step="1" @input="patchProvider(editingIndex, { interval: Number(($event.target as HTMLInputElement).value) || 1 })">
          </label>
        </template>

        <template v-else-if="editingProvider.type === 'imap'">
          <label class="field required">
            <span>Host</span>
            <input :value="editingProvider.host" type="text" @input="patchProvider(editingIndex, { host: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field required">
            <span>Port</span>
            <input :value="editingProvider.port" type="number" min="1" step="1" @input="patchProvider(editingIndex, { port: Number(($event.target as HTMLInputElement).value) || 993 })">
          </label>

          <label class="toggle">
            <input :checked="editingProvider.secure" type="checkbox" @change="patchProvider(editingIndex, { secure: ($event.target as HTMLInputElement).checked })">
            <span>TLS</span>
          </label>

          <label class="field required">
            <span>Username</span>
            <input :value="editingProvider.username" type="text" @input="patchProvider(editingIndex, { username: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field required">
            <span>Password</span>
            <div class="secret-field">
              <input
                :value="editingProvider.password ?? ''"
                :type="revealedSecrets[editingProvider.id] ? 'text' : 'password'"
                @input="patchProvider(editingIndex, { password: ($event.target as HTMLInputElement).value })"
              >
              <button class="ghost action-button reveal-button" type="button" @click="toggleReveal(editingProvider.id)">
                <component :is="revealedSecrets[editingProvider.id] ? EyeOff : Eye" :size="16" aria-hidden="true" />
                <span>{{ revealedSecrets[editingProvider.id] ? "Hide" : "Show" }}</span>
              </button>
            </div>
          </label>

          <label class="field">
            <span>Poll interval</span>
            <input :value="editingProvider.pollInterval" type="number" min="1" step="1" @input="patchProvider(editingIndex, { pollInterval: Number(($event.target as HTMLInputElement).value) || 60 })">
          </label>
        </template>

        <template v-else>
          <label class="field required">
            <span>Path</span>
            <input :value="editingProvider.path" type="text" @input="patchProvider(editingIndex, { path: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field">
            <span>Secret header</span>
            <input :value="editingProvider.secretHeader ?? ''" type="text" @input="patchProvider(editingIndex, { secretHeader: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field">
            <span>Secret value</span>
            <div class="secret-field">
              <input
                :value="editingProvider.secretValue ?? ''"
                :type="revealedSecrets[editingProvider.id] ? 'text' : 'password'"
                @input="patchProvider(editingIndex, { secretValue: ($event.target as HTMLInputElement).value })"
              >
              <button class="ghost action-button reveal-button" type="button" @click="toggleReveal(editingProvider.id)">
                <component :is="revealedSecrets[editingProvider.id] ? EyeOff : Eye" :size="16" aria-hidden="true" />
                <span>{{ revealedSecrets[editingProvider.id] ? "Hide" : "Show" }}</span>
              </button>
            </div>
          </label>
        </template>
      </template>

      <template v-if="editingProvider !== null" #footer>
        <button
          v-if="editingProvider.type === 'imap'"
          class="secondary action-button"
          type="button"
          :disabled="testingProviders[editingProvider.id]"
          @click="runProviderConnectionTest(editingProvider)"
        >
          <Plug :size="16" aria-hidden="true" />
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
