<script setup lang="ts">
import { X } from "lucide-vue-next";
import { computed, ref } from "vue";

import { getTagTone, sanitizeTags } from "../utils/provider-tags";

const props = defineProps<{
  modelValue: string[];
  suggestions?: string[];
  placeholder?: string;
  toneMap?: Record<string, number>;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: string[]];
}>();

const draft = ref("");

const normalizedValue = computed(() => sanitizeTags(props.modelValue));
const normalizedSuggestions = computed(() =>
  sanitizeTags(props.suggestions ?? []).filter((tag) => !normalizedValue.value.includes(tag))
);
const filteredSuggestions = computed(() => {
  const query = draft.value.trim().toLowerCase();

  if (!query) {
    return normalizedSuggestions.value.slice(0, 8);
  }

  return normalizedSuggestions.value
    .filter((tag) => tag.toLowerCase().includes(query))
    .slice(0, 8);
});

function updateTags(nextTags: string[]) {
  emit("update:modelValue", sanitizeTags(nextTags));
}

function commitDraft() {
  const nextTag = draft.value.trim().replace(/,$/, "");
  if (!nextTag) {
    draft.value = "";
    return;
  }

  updateTags([...normalizedValue.value, nextTag]);
  draft.value = "";
}

function removeTag(tag: string) {
  updateTags(normalizedValue.value.filter((currentTag) => currentTag !== tag));
}

function addSuggestion(tag: string) {
  updateTags([...normalizedValue.value, tag]);
  draft.value = "";
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === "Enter" || event.key === ",") {
    event.preventDefault();
    commitDraft();
    return;
  }

  if (event.key === "Backspace" && draft.value.length === 0) {
    const lastTag = normalizedValue.value[normalizedValue.value.length - 1];
    if (lastTag) {
      removeTag(lastTag);
    }
  }
}
</script>

<template>
  <div class="tag-input">
    <div class="tag-input__shell">
      <span
        v-for="tag in normalizedValue"
        :key="tag"
        class="tag-pill tag-pill--filled"
        :data-tag-tone="getTagTone(tag, props.toneMap)"
      >
        <span class="tag-pill__label">{{ tag }}</span>
        <button class="tag-pill__remove" type="button" :aria-label="`Remove ${tag}`" @click="removeTag(tag)">
          <X :size="12" aria-hidden="true" />
        </button>
      </span>

      <input
        v-model="draft"
        class="tag-input__field"
        type="text"
        :placeholder="normalizedValue.length === 0 ? (placeholder ?? 'Add a tag') : ''"
        @keydown="onKeydown"
        @blur="commitDraft"
      >
    </div>

    <div v-if="filteredSuggestions.length > 0" class="tag-input__suggestions">
      <button
        v-for="tag in filteredSuggestions"
        :key="tag"
        class="tag-pill tag-pill--interactive"
        :data-tag-tone="getTagTone(tag, props.toneMap)"
        type="button"
        @mousedown.prevent
        @click="addSuggestion(tag)"
      >
        {{ tag }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.tag-input {
  display: grid;
  gap: 0.75rem;
}

.tag-input__shell {
  display: flex;
  flex-wrap: wrap;
  gap: 0.55rem;
  min-height: 3rem;
  padding: 0.65rem;
  border: 1px solid var(--nook-border);
  border-radius: var(--radius-xl);
  background:
    linear-gradient(135deg, rgb(255 255 255 / 0.9), rgb(246 248 250 / 0.94)),
    var(--nook-surface);
  align-items: center;
  transition: border-color 180ms ease, box-shadow 180ms ease;
}

.tag-input__shell:focus-within {
  border-color: oklch(0.7 0.12 215);
  box-shadow: 0 0 0 0.2rem rgb(89 179 255 / 0.14);
}

.tag-input__field {
  flex: 1 1 10rem;
  min-width: 9rem;
  border: 0;
  background: transparent;
  color: var(--nook-text);
  min-height: 2.2rem;
  padding: 0 0.35rem;
  outline: none;
}

.tag-input__field::placeholder {
  color: color-mix(in srgb, var(--nook-muted) 78%, transparent);
}

.tag-input__suggestions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.tag-pill__remove {
  width: 1.1rem;
  min-width: 1.1rem;
  height: 1.1rem;
  padding: 0;
  border: 0;
  border-radius: 999px;
  background: rgb(255 255 255 / 0.55);
  color: currentColor;
}

.tag-pill__remove:hover {
  background: rgb(255 255 255 / 0.78);
}
</style>
