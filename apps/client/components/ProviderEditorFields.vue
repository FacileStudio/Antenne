<script setup lang="ts">
import { Eye, EyeOff } from "lucide-vue-next";

import TagPillInput from "./TagPillInput.vue";
import type { ProviderSettings } from "../types";

const props = defineProps<{
  editingProvider: ProviderSettings;
  revealedSecrets: Record<string, boolean>;
  availableTags?: string[];
  tagToneMap?: Record<string, number>;
}>();

const emit = defineEmits<{
  patch: [patch: Partial<ProviderSettings>];
  "toggle-reveal": [id: string];
}>();
</script>

<template>
  <label class="toggle">
    <input :checked="editingProvider.enabled" type="checkbox" @change="emit('patch', { enabled: ($event.target as HTMLInputElement).checked })">
    <span>Enabled</span>
  </label>

  <label class="field">
    <span>Name</span>
    <input :value="editingProvider.name" type="text" @input="emit('patch', { name: ($event.target as HTMLInputElement).value })">
  </label>

  <label class="field">
    <span>Tags</span>
    <TagPillInput
      :model-value="props.editingProvider.tags"
      :suggestions="props.availableTags"
      :tone-map="props.tagToneMap"
      :placeholder="'Type a tag and press Enter'"
      @update:model-value="emit('patch', { tags: $event })"
    />
  </label>

  <template v-if="editingProvider.type === 'website'">
    <label class="field required">
      <span>URL</span>
      <input :value="editingProvider.url" type="url" @input="emit('patch', { url: ($event.target as HTMLInputElement).value })">
    </label>

    <label class="field">
      <span>Interval</span>
      <input :value="editingProvider.interval" type="number" min="1" step="1" @input="emit('patch', { interval: Number(($event.target as HTMLInputElement).value) || 1 })">
    </label>
  </template>

  <template v-else-if="editingProvider.type === 'rss'">
    <label class="field required">
      <span>Feed URL</span>
      <input :value="editingProvider.feedUrl" type="url" @input="emit('patch', { feedUrl: ($event.target as HTMLInputElement).value })">
    </label>

    <label class="field">
      <span>Poll interval</span>
      <input :value="editingProvider.pollInterval" type="number" min="10" step="1" @input="emit('patch', { pollInterval: Number(($event.target as HTMLInputElement).value) || 300 })">
    </label>
  </template>

  <template v-else-if="editingProvider.type === 'imap'">
    <label class="field required">
      <span>Host</span>
      <input :value="editingProvider.host" type="text" @input="emit('patch', { host: ($event.target as HTMLInputElement).value })">
    </label>

    <label class="field required">
      <span>Port</span>
      <input :value="editingProvider.port" type="number" min="1" step="1" @input="emit('patch', { port: Number(($event.target as HTMLInputElement).value) || 993 })">
    </label>

    <label class="toggle">
      <input :checked="editingProvider.secure" type="checkbox" @change="emit('patch', { secure: ($event.target as HTMLInputElement).checked })">
      <span>TLS</span>
    </label>

    <label class="field required">
      <span>Username</span>
      <input :value="editingProvider.username" type="text" @input="emit('patch', { username: ($event.target as HTMLInputElement).value })">
    </label>

    <label class="field required">
      <span>Password</span>
      <div class="secret-field">
        <input
          :value="editingProvider.password ?? ''"
          :type="revealedSecrets[editingProvider.id] ? 'text' : 'password'"
          @input="emit('patch', { password: ($event.target as HTMLInputElement).value })"
        >
        <button class="ghost action-button reveal-button" type="button" @click="emit('toggle-reveal', editingProvider.id)">
          <component :is="revealedSecrets[editingProvider.id] ? EyeOff : Eye" :size="16" aria-hidden="true" />
          <span>{{ revealedSecrets[editingProvider.id] ? "Hide" : "Show" }}</span>
        </button>
      </div>
    </label>

    <label class="field">
      <span>Poll interval</span>
      <input :value="editingProvider.pollInterval" type="number" min="1" step="1" @input="emit('patch', { pollInterval: Number(($event.target as HTMLInputElement).value) || 60 })">
    </label>

    <label class="toggle">
      <input :checked="editingProvider.includeEmailBody" type="checkbox" @change="emit('patch', { includeEmailBody: ($event.target as HTMLInputElement).checked })">
      <span>Include email body</span>
    </label>
  </template>

  <template v-else>
    <label class="field required">
      <span>Path</span>
      <input :value="editingProvider.path" type="text" @input="emit('patch', { path: ($event.target as HTMLInputElement).value })">
    </label>

    <label class="field">
      <span>Secret header</span>
      <input :value="editingProvider.secretHeader ?? ''" type="text" @input="emit('patch', { secretHeader: ($event.target as HTMLInputElement).value })">
    </label>

    <label class="field">
      <span>Secret value</span>
      <div class="secret-field">
        <input
          :value="editingProvider.secretValue ?? ''"
          :type="revealedSecrets[editingProvider.id] ? 'text' : 'password'"
          @input="emit('patch', { secretValue: ($event.target as HTMLInputElement).value })"
        >
        <button class="ghost action-button reveal-button" type="button" @click="emit('toggle-reveal', editingProvider.id)">
          <component :is="revealedSecrets[editingProvider.id] ? EyeOff : Eye" :size="16" aria-hidden="true" />
          <span>{{ revealedSecrets[editingProvider.id] ? "Hide" : "Show" }}</span>
        </button>
      </div>
    </label>
  </template>
</template>
