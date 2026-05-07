<script setup lang="ts">
import { computed } from "vue";
import { Eye, EyeOff } from "lucide-vue-next";

import RoomIdEditor from "./RoomIdEditor.vue";
import SmtpAddressEditor from "./SmtpAddressEditor.vue";
import { getProviderLabel, SYSTEM_PROVIDER_ID, type DeliveryTarget, type ProviderSettings } from "../types";

const props = defineProps<{
  editingTarget: DeliveryTarget;
  providers: ProviderSettings[];
  revealedSecrets: Record<string, boolean>;
}>();

const emit = defineEmits<{
  patch: [patch: Partial<DeliveryTarget>];
  "toggle-provider": [providerId: string];
  "toggle-reveal": [id: string];
}>();

const providerOptions = computed(() => [
  {
    id: SYSTEM_PROVIDER_ID,
    name: "Nook system",
    typeLabel: "System",
  },
  ...props.providers.map((provider) => ({
    id: provider.id,
    name: provider.name || getProviderLabel(provider.type),
    typeLabel: getProviderLabel(provider.type),
  })),
]);
</script>

<template>
  <label class="toggle">
    <input :checked="editingTarget.enabled" type="checkbox" @change="emit('patch', { enabled: ($event.target as HTMLInputElement).checked })">
    <span>Enabled</span>
  </label>

  <label class="field">
    <span>Name</span>
    <input :value="editingTarget.name" type="text" @input="emit('patch', { name: ($event.target as HTMLInputElement).value })">
  </label>

  <div class="field">
    <span>Providers</span>
    <p class="field-hint">Select which providers send to this target. Empty means this target receives nothing.</p>
    <div class="provider-picker">
      <label
        v-for="provider in providerOptions"
        :key="provider.id"
        class="provider-option"
        :class="{ selected: (editingTarget.providerIds ?? []).includes(provider.id) }"
      >
        <input
          type="checkbox"
          :checked="(editingTarget.providerIds ?? []).includes(provider.id)"
          @change="emit('toggle-provider', provider.id)"
        >
        <span class="provider-option__body">
          <span class="provider-option__name">{{ provider.name }}</span>
          <span class="provider-option__type">{{ provider.typeLabel }}</span>
        </span>
      </label>
      <p v-if="providers.length === 0" class="field-hint">No providers configured yet. System events can still be targeted.</p>
    </div>
  </div>

  <template v-if="editingTarget.type === 'matrix'">
    <label class="field required">
      <span>Homeserver</span>
      <input :value="editingTarget.homeserver" type="url" placeholder="https://matrix.example.com" @input="emit('patch', { homeserver: ($event.target as HTMLInputElement).value })">
    </label>

    <label class="field">
      <span>Access token</span>
      <p class="field-hint">Optional if password login is configured. Used directly until Matrix rejects it.</p>
      <div class="secret-field">
        <input
          :value="editingTarget.accessToken"
          :type="revealedSecrets[editingTarget.id] ? 'text' : 'password'"
          placeholder="syt_xxx"
          @input="emit('patch', { accessToken: ($event.target as HTMLInputElement).value })"
        >
        <button class="ghost action-button reveal-button" type="button" @click="emit('toggle-reveal', editingTarget.id)">
          <component :is="revealedSecrets[editingTarget.id] ? EyeOff : Eye" :size="16" aria-hidden="true" />
          <span>{{ revealedSecrets[editingTarget.id] ? "Hide" : "Show" }}</span>
        </button>
      </div>
    </label>

    <label class="field">
      <span>User ID</span>
      <p class="field-hint">Use with password for auto re-login.</p>
      <input :value="editingTarget.userId" type="text" placeholder="@nook:matrix.example.com" @input="emit('patch', { userId: ($event.target as HTMLInputElement).value })">
    </label>

    <label class="field">
      <span>Password</span>
      <div class="secret-field">
        <input
          :value="editingTarget.password ?? ''"
          :type="revealedSecrets[`${editingTarget.id}-password`] ? 'text' : 'password'"
          placeholder="hunter2 but less cursed"
          @input="emit('patch', { password: ($event.target as HTMLInputElement).value })"
        >
        <button class="ghost action-button reveal-button" type="button" @click="emit('toggle-reveal', `${editingTarget.id}-password`)">
          <component :is="revealedSecrets[`${editingTarget.id}-password`] ? EyeOff : Eye" :size="16" aria-hidden="true" />
          <span>{{ revealedSecrets[`${editingTarget.id}-password`] ? "Hide" : "Show" }}</span>
        </button>
      </div>
    </label>

    <label class="field">
      <span>Device ID</span>
      <p class="field-hint">Optional. Keep stable if you use it.</p>
      <input :value="editingTarget.deviceId ?? ''" type="text" placeholder="NOOKABC123" @input="emit('patch', { deviceId: ($event.target as HTMLInputElement).value })">
    </label>

    <label class="field">
      <span>Refresh token</span>
      <p class="field-hint">Optional. Used when access token expires.</p>
      <div class="secret-field">
        <input
          :value="editingTarget.refreshToken ?? ''"
          :type="revealedSecrets[`${editingTarget.id}-refresh`] ? 'text' : 'password'"
          placeholder="syr_xxx"
          @input="emit('patch', { refreshToken: ($event.target as HTMLInputElement).value })"
        >
        <button class="ghost action-button reveal-button" type="button" @click="emit('toggle-reveal', `${editingTarget.id}-refresh`)">
          <component :is="revealedSecrets[`${editingTarget.id}-refresh`] ? EyeOff : Eye" :size="16" aria-hidden="true" />
          <span>{{ revealedSecrets[`${editingTarget.id}-refresh`] ? "Hide" : "Show" }}</span>
        </button>
      </div>
    </label>

    <label class="field required">
      <span>Room IDs</span>
      <RoomIdEditor :model-value="editingTarget.roomIds" @update:model-value="emit('patch', { roomIds: $event })" />
    </label>
  </template>

  <template v-else-if="editingTarget.type === 'discord'">
    <label class="field required">
      <span>Webhook URL</span>
      <input :value="editingTarget.webhookUrl" type="url" placeholder="https://discord.com/api/webhooks/..." @input="emit('patch', { webhookUrl: ($event.target as HTMLInputElement).value })">
    </label>
  </template>

  <template v-else-if="editingTarget.type === 'smtp'">
    <label class="field required">
      <span>Host</span>
      <input :value="editingTarget.host" type="text" placeholder="smtp.example.com" @input="emit('patch', { host: ($event.target as HTMLInputElement).value })">
    </label>

    <label class="field required">
      <span>Port</span>
      <input :value="editingTarget.port" type="number" placeholder="587" @input="emit('patch', { port: Number(($event.target as HTMLInputElement).value) })">
    </label>

    <label class="toggle">
      <input :checked="editingTarget.secure" type="checkbox" @change="emit('patch', { secure: ($event.target as HTMLInputElement).checked })">
      <span>Use TLS</span>
    </label>

    <label class="field">
      <span>Username</span>
      <input :value="editingTarget.username" type="text" placeholder="user@example.com" @input="emit('patch', { username: ($event.target as HTMLInputElement).value })">
    </label>

    <label class="field">
      <span>Password</span>
      <div class="secret-field">
        <input
          :value="editingTarget.password ?? ''"
          :type="revealedSecrets[editingTarget.id] ? 'text' : 'password'"
          @input="emit('patch', { password: ($event.target as HTMLInputElement).value })"
        >
        <button class="ghost action-button reveal-button" type="button" @click="emit('toggle-reveal', editingTarget.id)">
          <component :is="revealedSecrets[editingTarget.id] ? EyeOff : Eye" :size="16" aria-hidden="true" />
          <span>{{ revealedSecrets[editingTarget.id] ? "Hide" : "Show" }}</span>
        </button>
      </div>
    </label>

    <label class="field required">
      <span>From address</span>
      <input :value="editingTarget.fromAddress" type="email" placeholder="nook@example.com" @input="emit('patch', { fromAddress: ($event.target as HTMLInputElement).value })">
    </label>

    <div class="field required">
      <span>To addresses</span>
      <SmtpAddressEditor :model-value="editingTarget.toAddresses" @update:model-value="emit('patch', { toAddresses: $event })" />
    </div>

    <label class="field">
      <span>Subject prefix</span>
      <input :value="editingTarget.subjectPrefix" type="text" placeholder="[Nook]" @input="emit('patch', { subjectPrefix: ($event.target as HTMLInputElement).value })">
    </label>
  </template>

  <template v-else-if="editingTarget.type === 'ntfy'">
    <label class="field required">
      <span>Server URL</span>
      <input :value="editingTarget.serverUrl" type="url" placeholder="https://ntfy.sh" @input="emit('patch', { serverUrl: ($event.target as HTMLInputElement).value })">
    </label>

    <label class="field required">
      <span>Topic</span>
      <input :value="editingTarget.topic" type="text" placeholder="my-alerts" @input="emit('patch', { topic: ($event.target as HTMLInputElement).value })">
    </label>

    <label class="field">
      <span>Access token</span>
      <div class="secret-field">
        <input
          :value="editingTarget.accessToken ?? ''"
          :type="revealedSecrets[editingTarget.id] ? 'text' : 'password'"
          placeholder="tk_xxx"
          @input="emit('patch', { accessToken: ($event.target as HTMLInputElement).value })"
        >
        <button class="ghost action-button reveal-button" type="button" @click="emit('toggle-reveal', editingTarget.id)">
          <component :is="revealedSecrets[editingTarget.id] ? EyeOff : Eye" :size="16" aria-hidden="true" />
          <span>{{ revealedSecrets[editingTarget.id] ? "Hide" : "Show" }}</span>
        </button>
      </div>
    </label>

    <label class="field">
      <span>Priority</span>
      <input :value="editingTarget.priority ?? 3" type="number" min="1" max="5" placeholder="3" @input="emit('patch', { priority: Number(($event.target as HTMLInputElement).value) })">
    </label>

    <label class="field">
      <span>Tags</span>
      <input :value="editingTarget.tags.join(', ')" type="text" placeholder="rotating_light,warning" @input="emit('patch', { tags: ($event.target as HTMLInputElement).value.split(',').map((tag) => tag.trim()).filter(Boolean) })">
    </label>

    <label class="field">
      <span>Click URL</span>
      <input :value="editingTarget.clickUrl ?? ''" type="url" placeholder="https://status.example.com" @input="emit('patch', { clickUrl: ($event.target as HTMLInputElement).value })">
    </label>
  </template>

  <template v-else>
    <label class="field required">
      <span>Webhook URL</span>
      <input :value="editingTarget.url" type="url" placeholder="https://example.com/hooks/nook" @input="emit('patch', { url: ($event.target as HTMLInputElement).value })">
    </label>

    <label class="field">
      <span>Secret header</span>
      <input :value="editingTarget.secretHeader ?? ''" type="text" placeholder="x-nook-secret" @input="emit('patch', { secretHeader: ($event.target as HTMLInputElement).value })">
    </label>

    <label class="field">
      <span>Secret value</span>
      <div class="secret-field">
        <input
          :value="editingTarget.secretValue ?? ''"
          :type="revealedSecrets[editingTarget.id] ? 'text' : 'password'"
          @input="emit('patch', { secretValue: ($event.target as HTMLInputElement).value })"
        >
        <button class="ghost action-button reveal-button" type="button" @click="emit('toggle-reveal', editingTarget.id)">
          <component :is="revealedSecrets[editingTarget.id] ? EyeOff : Eye" :size="16" aria-hidden="true" />
          <span>{{ revealedSecrets[editingTarget.id] ? "Hide" : "Show" }}</span>
        </button>
      </div>
    </label>
  </template>
</template>

<style scoped>
.field-hint {
  margin: 0;
  font-size: 0.82rem;
  color: var(--nook-muted);
}

.provider-picker {
  display: grid;
  gap: 0.55rem;
}

.provider-option {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 0.8rem;
  align-items: center;
  padding: 0.625rem 0.75rem;
  border-radius: var(--radius-lg);
  border: 1px solid var(--nook-border);
  background: var(--nook-surface);
  cursor: pointer;
  transition: background-color 150ms ease, border-color 150ms ease, transform 150ms ease;
}

.provider-option:hover {
  background: var(--nook-surface-strong);
  border-color: oklch(0.8 0 0);
}

.provider-option.selected {
  border-color: rgb(28 219 129 / 0.42);
  background: rgb(28 219 129 / 0.08);
  box-shadow: inset 0 0 0 1px rgb(28 219 129 / 0.12);
}

.provider-option input[type="checkbox"] {
  width: 1rem;
  height: 1rem;
  margin: 0;
  accent-color: #1cdb81;
}

.provider-option__body {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  min-width: 0;
}

.provider-option__name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--nook-text);
}

.provider-option__type {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 1.25rem;
  padding: 0.1rem 0.5rem;
  border-radius: var(--radius-pill);
  border: 1px solid var(--nook-border);
  background: oklch(0 0 0 / 4%);
  color: var(--nook-muted);
  font-size: 0.73rem;
  font-weight: 600;
  line-height: 1;
  white-space: nowrap;
  flex: none;
}

.provider-option.selected .provider-option__type {
  border-color: rgb(28 219 129 / 0.28);
  background: rgb(28 219 129 / 0.12);
  color: #7af0b6;
}

@media (max-width: 720px) {
  .provider-option__body {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
