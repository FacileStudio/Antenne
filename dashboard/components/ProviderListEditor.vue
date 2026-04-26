<script setup lang="ts">
import { ArrowDown, ArrowUp, Copy, Eye, EyeOff, Plus, Trash2 } from "lucide-vue-next";
import { reactive, ref } from "vue";

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

const revealedSecrets = reactive<Record<string, boolean>>({});
const nextProviderType = ref<ProviderType>("website");

function updateProviders(nextProviders: ProviderSettings[]) {
  emit("update:modelValue", nextProviders);
}

function addProvider() {
  updateProviders([
    ...props.modelValue,
    createDefaultProvider(nextProviderType.value),
  ]);
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
}

function toggleReveal(id: string) {
  revealedSecrets[id] = !revealedSecrets[id];
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

      <button class="primary action-button add-provider-button" type="button" @click="addProvider">
        <Plus :size="16" aria-hidden="true" />
        <span>Add provider</span>
      </button>
    </div>

    <div v-if="modelValue.length === 0" class="empty">No providers configured.</div>

    <div v-else class="providers">
      <article
        v-for="(provider, index) in modelValue"
        :key="provider.id"
        class="provider-card"
      >
        <div class="provider-card-header">
          <div class="provider-heading">
            <strong>{{ provider.name || `${getProviderLabel(provider.type)} ${index + 1}` }}</strong>
            <span class="provider-chip">{{ getProviderLabel(provider.type) }}</span>
          </div>

          <div class="provider-actions">
            <button class="ghost action-button" type="button" :disabled="index === 0" @click="moveProvider(index, -1)">
              <ArrowUp :size="16" aria-hidden="true" />
              <span>Up</span>
            </button>
            <button class="ghost action-button" type="button" :disabled="index === modelValue.length - 1" @click="moveProvider(index, 1)">
              <ArrowDown :size="16" aria-hidden="true" />
              <span>Down</span>
            </button>
            <button class="ghost action-button" type="button" @click="duplicateProvider(index)">
              <Copy :size="16" aria-hidden="true" />
              <span>Duplicate</span>
            </button>
            <button class="danger action-button" type="button" @click="removeProvider(index)">
              <Trash2 :size="16" aria-hidden="true" />
              <span>Remove</span>
            </button>
          </div>
        </div>

        <div class="provider-fields">
          <label class="field">
            <span>Name</span>
            <input :value="provider.name" type="text" @input="patchProvider(index, { name: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field">
            <span>Type</span>
            <input :value="getProviderLabel(provider.type)" type="text" readonly>
          </label>

          <label class="toggle provider-toggle">
            <input :checked="provider.enabled" type="checkbox" @change="patchProvider(index, { enabled: ($event.target as HTMLInputElement).checked })">
            <span>Enabled</span>
          </label>
        </div>

        <div v-if="provider.type === 'website'" class="provider-fields detail-grid">
          <label class="field wide">
            <span>URL</span>
            <input :value="provider.url" type="url" @input="patchProvider(index, { url: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field">
            <span>Interval</span>
            <input :value="provider.interval" type="number" min="1" step="1" @input="patchProvider(index, { interval: Number(($event.target as HTMLInputElement).value) || 1 })">
          </label>
        </div>

        <div v-else-if="provider.type === 'imap'" class="provider-fields detail-grid imap-grid">
          <label class="field">
            <span>Host</span>
            <input :value="provider.host" type="text" @input="patchProvider(index, { host: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field">
            <span>Port</span>
            <input :value="provider.port" type="number" min="1" step="1" @input="patchProvider(index, { port: Number(($event.target as HTMLInputElement).value) || 993 })">
          </label>

          <label class="toggle provider-toggle">
            <input :checked="provider.secure" type="checkbox" @change="patchProvider(index, { secure: ($event.target as HTMLInputElement).checked })">
            <span>TLS</span>
          </label>

          <label class="field">
            <span>Username</span>
            <input :value="provider.username" type="text" @input="patchProvider(index, { username: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field wide">
            <span>Password</span>
            <div class="secret-field">
              <input
                :value="provider.password ?? ''"
                :type="revealedSecrets[provider.id] ? 'text' : 'password'"
                @input="patchProvider(index, { password: ($event.target as HTMLInputElement).value })"
              >
              <button class="ghost action-button reveal-button" type="button" @click="toggleReveal(provider.id)">
                <component :is="revealedSecrets[provider.id] ? EyeOff : Eye" :size="16" aria-hidden="true" />
                <span>{{ revealedSecrets[provider.id] ? "Hide" : "Show" }}</span>
              </button>
            </div>
          </label>

          <label class="field">
            <span>Mailbox</span>
            <input :value="provider.mailbox" type="text" @input="patchProvider(index, { mailbox: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field">
            <span>Poll interval</span>
            <input :value="provider.pollInterval" type="number" min="1" step="1" @input="patchProvider(index, { pollInterval: Number(($event.target as HTMLInputElement).value) || 60 })">
          </label>
        </div>

        <div v-else class="provider-fields detail-grid webhook-grid">
          <label class="field wide">
            <span>Path</span>
            <input :value="provider.path" type="text" @input="patchProvider(index, { path: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field">
            <span>Secret header</span>
            <input :value="provider.secretHeader ?? ''" type="text" @input="patchProvider(index, { secretHeader: ($event.target as HTMLInputElement).value })">
          </label>

          <label class="field wide">
            <span>Secret value</span>
            <div class="secret-field">
              <input
                :value="provider.secretValue ?? ''"
                :type="revealedSecrets[provider.id] ? 'text' : 'password'"
                @input="patchProvider(index, { secretValue: ($event.target as HTMLInputElement).value })"
              >
              <button class="ghost action-button reveal-button" type="button" @click="toggleReveal(provider.id)">
                <component :is="revealedSecrets[provider.id] ? EyeOff : Eye" :size="16" aria-hidden="true" />
                <span>{{ revealedSecrets[provider.id] ? "Hide" : "Show" }}</span>
              </button>
            </div>
          </label>
        </div>
      </article>
    </div>
  </div>
</template>

<style scoped>
.provider-list,
.providers {
  display: grid;
  gap: var(--space-4);
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

.provider-card {
  display: grid;
  gap: var(--space-4);
  padding: var(--space-4);
  border-radius: 1.25rem;
  border: 1px solid var(--nook-border);
  background: var(--nook-surface-strong);
}

.provider-card-header,
.provider-fields {
  display: flex;
  justify-content: space-between;
  gap: var(--space-4);
  align-items: flex-start;
  flex-wrap: wrap;
}

.provider-heading {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.provider-chip {
  display: inline-flex;
  align-items: center;
  min-height: 1.75rem;
  padding: 0 0.7rem;
  border-radius: 999px;
  border: 1px solid var(--nook-border);
  color: var(--nook-muted);
  font-size: 0.85rem;
}

.provider-actions {
  display: flex;
  gap: var(--space-2);
  flex-wrap: wrap;
  justify-content: flex-end;
}

.detail-grid {
  display: grid;
  gap: var(--space-3);
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.imap-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.webhook-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.field.wide {
  grid-column: span 2;
}

.provider-toggle {
  min-height: 3.5rem;
  padding-top: 1.75rem;
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

.add-provider-button {
  min-height: 3.5rem;
}

.empty {
  color: var(--nook-muted);
  padding: var(--space-4);
  border-radius: 1.25rem;
  border: 1px dashed var(--nook-border);
}

@media (max-width: 960px) {
  .provider-card-header,
  .provider-fields {
    flex-direction: column;
    align-items: stretch;
  }

  .provider-actions {
    justify-content: flex-start;
  }

  .detail-grid,
  .imap-grid,
  .webhook-grid {
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
