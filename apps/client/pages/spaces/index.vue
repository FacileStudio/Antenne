<script setup lang="ts">
import { ref, onMounted } from "vue";
import { Icon } from "@iconify/vue";

import SectionCard from "../../components/SectionCard.vue";
import { useSpaces } from "../../composables/useSpaces";
import { useDashboardState } from "../../composables/useDashboardState";
import type { Space } from "../../types";

const { spaces, loading, loadSpaces, createSpace, deleteSpace } = useSpaces();
const { notifySuccess, notifyError } = useDashboardState();

const showCreate = ref(false);
const newName = ref("");
const newDescription = ref("");
const creating = ref(false);
const deleting = ref<string | null>(null);

onMounted(() => {
  loadSpaces();
});

async function handleCreate() {
  if (!newName.value.trim()) return;
  creating.value = true;
  try {
    await createSpace(newName.value.trim(), newDescription.value.trim());
    notifySuccess("Space created.");
    newName.value = "";
    newDescription.value = "";
    showCreate.value = false;
  } catch (e) {
    notifyError(e instanceof Error ? e.message : "Failed to create space");
  } finally {
    creating.value = false;
  }
}

async function handleDelete(space: Space) {
  if (!confirm(`Delete space "${space.name}"? This cannot be undone.`)) return;
  deleting.value = space.id;
  try {
    await deleteSpace(space.id);
    notifySuccess("Space deleted.");
  } catch (e) {
    notifyError(e instanceof Error ? e.message : "Failed to delete space");
  } finally {
    deleting.value = null;
  }
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString("fr-FR", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}
</script>

<template>
  <div class="space-y-8 px-4 py-4 md:px-8 md:py-5">
    <SectionCard eyebrow="Espaces" title="Spaces">
      <template #actions>
        <button class="action-button add-button" @click="showCreate = !showCreate">
          <Icon :icon="showCreate ? 'solar:close-circle-linear' : 'solar:add-circle-linear'" :width="16" :height="16" aria-hidden="true" />
          {{ showCreate ? "Cancel" : "New space" }}
        </button>
      </template>

      <div v-if="showCreate" class="create-form">
        <div class="form-fields">
          <label class="field required">
            <span>Name</span>
            <input v-model="newName" type="text" placeholder="My workspace" maxlength="64">
          </label>
          <label class="field">
            <span>Description</span>
            <input v-model="newDescription" type="text" placeholder="Optional description" maxlength="256">
          </label>
        </div>
        <button class="primary create-button" :disabled="!newName.trim() || creating" @click="handleCreate">
          <Icon icon="solar:add-circle-linear" :width="14" :height="14" aria-hidden="true" />
          {{ creating ? "Creating..." : "Create space" }}
        </button>
      </div>

      <div v-if="loading" class="space-list-empty">Loading spaces...</div>

      <div v-else-if="spaces.length === 0" class="space-list-empty">
        No spaces yet. Create one to get started.
      </div>

      <div v-else class="space-list">
        <NuxtLink
          v-for="space in spaces"
          :key="space.id"
          :to="`/spaces/${space.id}`"
          class="space-card"
        >
          <div class="space-card-info">
            <div class="space-card-header">
              <Icon icon="solar:buildings-2-linear" :width="18" :height="18" class="space-card-icon" aria-hidden="true" />
              <strong>{{ space.name }}</strong>
            </div>
            <p v-if="space.description" class="space-card-desc">{{ space.description }}</p>
            <span class="space-card-meta">Created {{ formatDate(space.createdAt) }}</span>
          </div>
          <button
            class="ghost space-delete-button"
            :disabled="deleting === space.id"
            @click.prevent.stop="handleDelete(space)"
          >
            <Icon icon="solar:trash-bin-minimalistic-linear" :width="16" :height="16" aria-hidden="true" />
          </button>
        </NuxtLink>
      </div>
    </SectionCard>
  </div>
</template>

<style scoped>
.create-form {
  display: grid;
  gap: var(--space-3);
  padding: 1rem;
  border: 1px solid var(--nook-border);
  border-radius: var(--radius-xl);
  background: var(--nook-surface);
}

.form-fields {
  display: grid;
  gap: var(--space-3);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  max-width: 36rem;
}

.create-button {
  justify-self: start;
}

.space-list-empty {
  padding: var(--space-4);
  text-align: center;
  color: var(--nook-muted);
  font-size: 0.875rem;
}

.space-list {
  display: grid;
  gap: 0.75rem;
}

.space-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: 1rem 1.25rem;
  border: 1px solid var(--nook-border);
  border-radius: var(--radius-xl);
  background: var(--nook-surface-strong);
  text-decoration: none;
  color: inherit;
  transition: background 150ms, border-color 150ms;
}

.space-card:hover {
  background: var(--muted);
  border-color: color-mix(in oklab, var(--border) 70%, var(--foreground));
}

.space-card-info {
  display: grid;
  gap: 0.25rem;
  min-width: 0;
}

.space-card-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.space-card-icon {
  color: var(--nook-muted);
  flex-shrink: 0;
}

.space-card-desc {
  margin: 0;
  font-size: 0.84rem;
  color: var(--nook-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.space-card-meta {
  font-size: 0.75rem;
  color: var(--nook-muted);
}

.space-delete-button {
  flex-shrink: 0;
  color: var(--nook-muted);
}

.space-delete-button:hover {
  color: var(--destructive);
}

@media (max-width: 720px) {
  .form-fields {
    grid-template-columns: 1fr;
  }
}
</style>
