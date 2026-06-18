<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { Icon } from "@iconify/vue";

import SectionCard from "../../components/SectionCard.vue";
import { useSpaces } from "../../composables/useSpaces";
import { useDashboardState } from "../../composables/useDashboardState";
import type { SpaceRole, SpaceMember } from "../../types";

const route = useRoute();
const router = useRouter();

const {
  spaces,
  loadSpaces,
  updateSpace,
  deleteSpace,
  loadMembers,
  members,
  addMember,
  updateMember,
  removeMember,
} = useSpaces();
const { notifySuccess, notifyError } = useDashboardState();

const spaceId = computed(() => route.params.id as string);
const space = computed(() => spaces.value.find((s) => s.id === spaceId.value) ?? null);

const editName = ref("");
const editDescription = ref("");
const saving = ref(false);

const newUserId = ref("");
const newRole = ref<SpaceRole>("member");
const addingMember = ref(false);

const roles: { value: SpaceRole; label: string }[] = [
  { value: "owner", label: "Owner" },
  { value: "admin", label: "Admin" },
  { value: "member", label: "Member" },
];

onMounted(async () => {
  await loadSpaces();
  if (space.value) {
    editName.value = space.value.name;
    editDescription.value = space.value.description;
    await loadMembers(spaceId.value);
  }
});

async function handleSave() {
  if (!editName.value.trim()) return;
  saving.value = true;
  try {
    await updateSpace(spaceId.value, {
      name: editName.value.trim(),
      description: editDescription.value.trim(),
    });
    notifySuccess("Space updated.");
  } catch (e) {
    notifyError(e instanceof Error ? e.message : "Failed to update space");
  } finally {
    saving.value = false;
  }
}

async function handleDelete() {
  if (!space.value) return;
  if (!confirm(`Delete space "${space.value.name}"? This cannot be undone.`)) return;
  try {
    await deleteSpace(spaceId.value);
    notifySuccess("Space deleted.");
    router.push("/spaces");
  } catch (e) {
    notifyError(e instanceof Error ? e.message : "Failed to delete space");
  }
}

async function handleAddMember() {
  if (!newUserId.value.trim()) return;
  addingMember.value = true;
  try {
    await addMember(spaceId.value, newUserId.value.trim(), newRole.value);
    notifySuccess("Member added.");
    newUserId.value = "";
    newRole.value = "member";
  } catch (e) {
    notifyError(e instanceof Error ? e.message : "Failed to add member");
  } finally {
    addingMember.value = false;
  }
}

async function handleRoleChange(member: SpaceMember, role: SpaceRole) {
  try {
    await updateMember(spaceId.value, member.userId, role);
    notifySuccess("Role updated.");
  } catch (e) {
    notifyError(e instanceof Error ? e.message : "Failed to update role");
  }
}

async function handleRemoveMember(member: SpaceMember) {
  if (!confirm(`Remove "${member.userId}" from this space?`)) return;
  try {
    await removeMember(spaceId.value, member.userId);
    notifySuccess("Member removed.");
  } catch (e) {
    notifyError(e instanceof Error ? e.message : "Failed to remove member");
  }
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString("fr-FR", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}

function roleLabel(role: SpaceRole) {
  return roles.find((r) => r.value === role)?.label ?? role;
}
</script>

<template>
  <div class="space-y-8 px-4 py-4 md:px-8 md:py-5">
    <div v-if="!space" class="space-list-empty">
      Space not found.
      <NuxtLink to="/spaces" class="back-link">Back to spaces</NuxtLink>
    </div>

    <template v-else>
      <NuxtLink to="/spaces" class="back-link">
        <Icon icon="solar:arrow-left-linear" :width="14" :height="14" aria-hidden="true" />
        All spaces
      </NuxtLink>

      <SectionCard eyebrow="Espace" :title="space.name">
        <template #actions>
          <button class="ghost danger" @click="handleDelete">
            <Icon icon="solar:trash-bin-minimalistic-linear" :width="14" :height="14" aria-hidden="true" />
            Delete
          </button>
        </template>

        <div class="form-fields">
          <label class="field required">
            <span>Name</span>
            <input v-model="editName" type="text" placeholder="Space name" maxlength="64">
          </label>
          <label class="field">
            <span>Description</span>
            <input v-model="editDescription" type="text" placeholder="Optional description" maxlength="256">
          </label>
        </div>
        <button class="primary save-button" :disabled="!editName.trim() || saving" @click="handleSave">
          <Icon icon="solar:diskette-linear" :width="14" :height="14" aria-hidden="true" />
          {{ saving ? "Saving..." : "Save" }}
        </button>
      </SectionCard>

      <SectionCard eyebrow="Espace" title="Members">
        <div class="add-member-form">
          <label class="field">
            <span>User ID</span>
            <input v-model="newUserId" type="text" placeholder="user@example.com">
          </label>
          <label class="field">
            <span>Role</span>
            <select v-model="newRole">
              <option v-for="r in roles" :key="r.value" :value="r.value">{{ r.label }}</option>
            </select>
          </label>
          <button class="primary add-member-button" :disabled="!newUserId.trim() || addingMember" @click="handleAddMember">
            <Icon icon="solar:user-plus-linear" :width="14" :height="14" aria-hidden="true" />
            {{ addingMember ? "Adding..." : "Add" }}
          </button>
        </div>

        <div v-if="members.length === 0" class="members-empty">
          No members in this space.
        </div>

        <div v-else class="members-list">
          <div v-for="member in members" :key="member.userId" class="member-row">
            <div class="member-info">
              <Icon icon="solar:user-linear" :width="16" :height="16" class="member-icon" aria-hidden="true" />
              <span class="member-id">{{ member.userId }}</span>
              <span class="member-role-badge" :data-role="member.role">{{ roleLabel(member.role) }}</span>
            </div>
            <div class="member-actions">
              <select
                :value="member.role"
                class="role-select"
                @change="handleRoleChange(member, ($event.target as HTMLSelectElement).value as SpaceRole)"
              >
                <option v-for="r in roles" :key="r.value" :value="r.value">{{ r.label }}</option>
              </select>
              <button class="ghost member-remove" @click="handleRemoveMember(member)">
                <Icon icon="solar:close-circle-linear" :width="14" :height="14" aria-hidden="true" />
              </button>
            </div>
          </div>
        </div>
      </SectionCard>

      <SectionCard eyebrow="Espace" title="Info">
        <div class="status-list">
          <div class="status-row">
            <span class="status-label">ID</span>
            <span class="status-value">{{ space.id }}</span>
          </div>
          <div class="status-row">
            <span class="status-label">Created</span>
            <span class="status-value">{{ formatDate(space.createdAt) }}</span>
          </div>
          <div class="status-row">
            <span class="status-label">Updated</span>
            <span class="status-value">{{ formatDate(space.updatedAt) }}</span>
          </div>
          <div class="status-row">
            <span class="status-label">Members</span>
            <span class="status-value">{{ members.length }}</span>
          </div>
        </div>
      </SectionCard>
    </template>
  </div>
</template>

<style scoped>
.back-link {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.84rem;
  color: var(--nook-muted);
  text-decoration: none;
  transition: color 150ms;
}

.back-link:hover {
  color: var(--foreground);
}

.form-fields {
  display: grid;
  gap: var(--space-3);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  max-width: 36rem;
}

.save-button {
  justify-self: start;
}

.add-member-form {
  display: grid;
  gap: var(--space-3);
  grid-template-columns: 1fr auto auto;
  align-items: end;
  max-width: 36rem;
}

.add-member-button {
  height: 2rem;
}

.members-empty {
  padding: var(--space-3);
  text-align: center;
  color: var(--nook-muted);
  font-size: 0.875rem;
}

.members-list {
  display: grid;
  gap: 0;
  border: 1px solid var(--nook-border);
  border-radius: var(--radius-xl);
  overflow: hidden;
}

.member-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: 0.75rem 1rem;
}

.member-row + .member-row {
  border-top: 1px solid var(--nook-border);
}

.member-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
}

.member-icon {
  color: var(--nook-muted);
  flex-shrink: 0;
}

.member-id {
  font-size: 0.875rem;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.member-role-badge {
  display: inline-flex;
  align-items: center;
  padding: 0.125rem 0.5rem;
  border-radius: var(--radius-pill);
  font-size: 0.72rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  flex-shrink: 0;
}

.member-role-badge[data-role="owner"] {
  background: oklch(0.96 0.035 220);
  color: oklch(0.42 0.11 236);
}

.member-role-badge[data-role="admin"] {
  background: oklch(0.965 0.05 150);
  color: oklch(0.44 0.11 155);
}

.member-role-badge[data-role="member"] {
  background: var(--nook-surface-strong);
  color: var(--nook-muted);
}

.member-actions {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  flex-shrink: 0;
}

.role-select {
  width: auto;
  min-width: 7rem;
  font-size: 0.84rem;
}

.member-remove {
  color: var(--nook-muted);
  padding: 0.25rem;
}

.member-remove:hover {
  color: var(--destructive);
}

.space-list-empty {
  padding: var(--space-4);
  text-align: center;
  color: var(--nook-muted);
  font-size: 0.875rem;
}

.status-list {
  display: grid;
  gap: 0;
  border: 1px solid var(--nook-border);
  border-radius: var(--radius-xl);
  overflow: hidden;
}

.status-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
  padding: 0.75rem 1rem;
  min-width: 0;
}

.status-row + .status-row {
  border-top: 1px solid var(--nook-border);
}

.status-label {
  color: var(--nook-muted);
  font-size: 0.84rem;
  flex-shrink: 0;
}

.status-value {
  font-size: 0.875rem;
  font-weight: 500;
  text-align: right;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

@media (max-width: 720px) {
  .form-fields {
    grid-template-columns: 1fr;
  }

  .add-member-form {
    grid-template-columns: 1fr;
  }
}
</style>
