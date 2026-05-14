<script setup lang="ts">
import { Icon } from "@iconify/vue";
import { ref } from "vue";

const props = defineProps<{
  modelValue: string[];
}>();

const emit = defineEmits<{
  "update:modelValue": [value: string[]];
}>();

const draft = ref("");

function addRoomId() {
  const roomId = draft.value.trim();
  if (!roomId) {
    return;
  }

  if (!props.modelValue.includes(roomId)) {
    emit("update:modelValue", [...props.modelValue, roomId]);
  }

  draft.value = "";
}

function removeRoomId(index: number) {
  emit("update:modelValue", props.modelValue.filter((_, currentIndex) => currentIndex !== index));
}
</script>

<template>
  <div class="room-editor">
    <div v-if="modelValue.length > 0" class="chip-list">
      <span
        v-for="(roomId, index) in modelValue"
        :key="roomId"
        class="chip"
      >
        <span>{{ roomId }}</span>
        <button class="chip-remove" type="button" aria-label="Remove room" @click="removeRoomId(index)">
          <Icon icon="solar:close-circle-linear" :width="14" :height="14" aria-hidden="true" />
        </button>
      </span>
    </div>

    <div class="room-input">
      <input
        v-model="draft"
        type="text"
        placeholder="!room:example.com"
        @keydown.enter.prevent="addRoomId"
      >
      <button class="secondary action-button" type="button" @click="addRoomId">
        <Icon icon="mdi:plus" :width="16" :height="16" aria-hidden="true" />
        <span>Add room</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.room-editor {
  display: grid;
  gap: var(--space-3);
}

.room-input {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: var(--space-3);
  align-items: center;
}

.chip-list {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.chip {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  padding: 0.2rem 0.4rem 0.2rem 0.75rem;
  border-radius: var(--radius-pill);
  border: 1px solid var(--nook-border);
  background: var(--nook-surface-strong);
}

.chip-remove,
.action-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
}

.chip-remove {
  width: 1.9rem;
  height: 1.9rem;
  padding: 0;
}

@media (max-width: 720px) {
  .room-input {
    grid-template-columns: 1fr;
  }
}
</style>
