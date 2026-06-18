import { ref, computed } from "vue";
import type { Space, SpaceMember, SpaceRole } from "../types";

const STORAGE_KEY = "nook-current-space-id";

const spaces = ref<Space[]>([]);
const currentSpaceId = ref<string | null>(null);
const members = ref<SpaceMember[]>([]);
const loading = ref(false);
const spacesLoaded = ref(false);

function persistCurrentSpaceId(id: string | null) {
  if (import.meta.client) {
    if (id) {
      localStorage.setItem(STORAGE_KEY, id);
    } else {
      localStorage.removeItem(STORAGE_KEY);
    }
  }
}

function restoreCurrentSpaceId(): string | null {
  if (import.meta.client) {
    return localStorage.getItem(STORAGE_KEY);
  }
  return null;
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
  });

  if (res.status === 401) {
    if (import.meta.client) {
      window.location.href = "/dashboard";
    }
    throw new Error("Unauthorized");
  }

  if (!res.ok) {
    const payload = await res.json().catch(() => ({ error: `HTTP ${res.status}` })) as { error?: string };
    throw new Error(payload.error ?? `HTTP ${res.status}`);
  }

  return await res.json() as T;
}

export function useSpaces() {
  const currentSpace = computed(() =>
    spaces.value.find((s) => s.id === currentSpaceId.value) ?? null,
  );

  async function loadSpaces() {
    loading.value = true;
    try {
      const payload = await apiFetch<{ spaces: Space[] }>("/api/spaces");
      spaces.value = payload.spaces;

      const restored = restoreCurrentSpaceId();
      if (restored && spaces.value.find((s) => s.id === restored)) {
        currentSpaceId.value = restored;
      } else if (spaces.value.length > 0 && !currentSpaceId.value) {
        currentSpaceId.value = spaces.value[0]!.id;
        persistCurrentSpaceId(currentSpaceId.value);
      }

      spacesLoaded.value = true;
    } finally {
      loading.value = false;
    }
  }

  function switchSpace(id: string) {
    currentSpaceId.value = id;
    persistCurrentSpaceId(id);
  }

  async function createNewSpace(name: string, description: string): Promise<Space> {
    const payload = await apiFetch<{ space: Space }>("/api/spaces", {
      method: "POST",
      body: JSON.stringify({ name, description }),
    });
    spaces.value = [...spaces.value, payload.space];
    return payload.space;
  }

  async function updateExistingSpace(id: string, updates: { name?: string; description?: string }): Promise<Space> {
    const payload = await apiFetch<{ space: Space }>(`/api/spaces/${id}`, {
      method: "PUT",
      body: JSON.stringify(updates),
    });
    spaces.value = spaces.value.map((s) => (s.id === id ? payload.space : s));
    return payload.space;
  }

  async function deleteExistingSpace(id: string) {
    await apiFetch<{ ok: boolean }>(`/api/spaces/${id}`, { method: "DELETE" });
    spaces.value = spaces.value.filter((s) => s.id !== id);
    if (currentSpaceId.value === id) {
      currentSpaceId.value = spaces.value[0]?.id ?? null;
      persistCurrentSpaceId(currentSpaceId.value);
    }
  }

  async function loadMembers(spaceId: string) {
    const payload = await apiFetch<{ members: SpaceMember[] }>(`/api/spaces/${spaceId}/members`);
    members.value = payload.members;
    return payload.members;
  }

  async function addMember(spaceId: string, userId: string, role: SpaceRole) {
    const payload = await apiFetch<{ member: SpaceMember }>(`/api/spaces/${spaceId}/members`, {
      method: "POST",
      body: JSON.stringify({ userId, role }),
    });
    members.value = [...members.value, payload.member];
    return payload.member;
  }

  async function updateMember(spaceId: string, userId: string, role: SpaceRole) {
    const payload = await apiFetch<{ member: SpaceMember }>(`/api/spaces/${spaceId}/members/${encodeURIComponent(userId)}`, {
      method: "PUT",
      body: JSON.stringify({ role }),
    });
    members.value = members.value.map((m) =>
      m.spaceId === spaceId && m.userId === userId ? payload.member : m,
    );
    return payload.member;
  }

  async function removeMember(spaceId: string, userId: string) {
    await apiFetch<{ ok: boolean }>(`/api/spaces/${spaceId}/members/${encodeURIComponent(userId)}`, {
      method: "DELETE",
    });
    members.value = members.value.filter((m) => !(m.spaceId === spaceId && m.userId === userId));
  }

  return {
    spaces,
    currentSpaceId,
    currentSpace,
    members,
    loading,
    spacesLoaded,
    loadSpaces,
    switchSpace,
    createSpace: createNewSpace,
    updateSpace: updateExistingSpace,
    deleteSpace: deleteExistingSpace,
    loadMembers,
    addMember,
    updateMember,
    removeMember,
  };
}
