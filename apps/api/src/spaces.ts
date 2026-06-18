import { mkdir } from "node:fs/promises";

export type SpaceRole = "owner" | "admin" | "member";

export interface Space {
  id: string;
  name: string;
  description: string;
  createdAt: string;
  updatedAt: string;
}

export interface SpaceMember {
  spaceId: string;
  userId: string;
  role: SpaceRole;
  joinedAt: string;
}

export interface SpaceStore {
  spaces: Space[];
  members: SpaceMember[];
}

let store: SpaceStore = { spaces: [], members: [] };
let persistPath: string | null = null;
let flushTimer: Timer | null = null;

function createId(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

function scheduleFlush() {
  if (!persistPath) return;
  if (flushTimer) return;
  flushTimer = setTimeout(() => {
    flushTimer = null;
    void flushToDisk();
  }, 500);
}

async function flushToDisk() {
  if (!persistPath) return;
  try {
    const directory = persistPath.includes("/") ? persistPath.slice(0, persistPath.lastIndexOf("/")) : ".";
    await mkdir(directory, { recursive: true });
    await Bun.write(persistPath, JSON.stringify(store, null, 2));
  } catch (error) {
    console.error(`[spaces] failed to persist: ${error instanceof Error ? error.message : String(error)}`);
  }
}

export async function initSpaces(path: string) {
  persistPath = path;
  try {
    const file = Bun.file(path);
    if (await file.exists()) {
      const raw = await file.json() as Partial<SpaceStore>;
      store = {
        spaces: Array.isArray(raw.spaces) ? raw.spaces : [],
        members: Array.isArray(raw.members) ? raw.members : [],
      };
      console.log(`[spaces] loaded ${store.spaces.length} spaces from ${path}`);
      return;
    }
  } catch (error) {
    console.error(`[spaces] failed to load: ${error instanceof Error ? error.message : String(error)}`);
  }
  store = { spaces: [], members: [] };
}

export function listSpaces(): Space[] {
  return [...store.spaces];
}

export function getSpace(id: string): Space | null {
  return store.spaces.find((s) => s.id === id) ?? null;
}

export function createSpace(name: string, description: string, ownerUserId?: string): Space {
  const now = new Date().toISOString();
  const space: Space = {
    id: createId(),
    name,
    description,
    createdAt: now,
    updatedAt: now,
  };
  store.spaces.push(space);

  if (ownerUserId) {
    store.members.push({
      spaceId: space.id,
      userId: ownerUserId,
      role: "owner",
      joinedAt: now,
    });
  }

  scheduleFlush();
  return space;
}

export function updateSpace(id: string, updates: { name?: string; description?: string }): Space | null {
  const space = store.spaces.find((s) => s.id === id);
  if (!space) return null;

  if (updates.name !== undefined) space.name = updates.name;
  if (updates.description !== undefined) space.description = updates.description;
  space.updatedAt = new Date().toISOString();

  scheduleFlush();
  return { ...space };
}

export function deleteSpace(id: string): boolean {
  const index = store.spaces.findIndex((s) => s.id === id);
  if (index === -1) return false;

  store.spaces.splice(index, 1);
  store.members = store.members.filter((m) => m.spaceId !== id);
  scheduleFlush();
  return true;
}

export function getSpaceMembers(spaceId: string): SpaceMember[] {
  return store.members.filter((m) => m.spaceId === spaceId);
}

export function addSpaceMember(spaceId: string, userId: string, role: SpaceRole): SpaceMember | null {
  if (!store.spaces.find((s) => s.id === spaceId)) return null;

  const existing = store.members.find((m) => m.spaceId === spaceId && m.userId === userId);
  if (existing) return existing;

  const member: SpaceMember = {
    spaceId,
    userId,
    role,
    joinedAt: new Date().toISOString(),
  };
  store.members.push(member);
  scheduleFlush();
  return member;
}

export function updateMemberRole(spaceId: string, userId: string, role: SpaceRole): SpaceMember | null {
  const member = store.members.find((m) => m.spaceId === spaceId && m.userId === userId);
  if (!member) return null;

  member.role = role;
  scheduleFlush();
  return { ...member };
}

export function removeSpaceMember(spaceId: string, userId: string): boolean {
  const index = store.members.findIndex((m) => m.spaceId === spaceId && m.userId === userId);
  if (index === -1) return false;

  store.members.splice(index, 1);
  scheduleFlush();
  return true;
}

export function getUserSpaces(userId: string): Array<Space & { role: SpaceRole }> {
  const memberships = store.members.filter((m) => m.userId === userId);
  return memberships
    .map((m) => {
      const space = store.spaces.find((s) => s.id === m.spaceId);
      if (!space) return null;
      return { ...space, role: m.role };
    })
    .filter((s): s is Space & { role: SpaceRole } => s !== null);
}

export function getMemberRole(spaceId: string, userId: string): SpaceRole | null {
  const member = store.members.find((m) => m.spaceId === spaceId && m.userId === userId);
  return member?.role ?? null;
}

export function isSpaceAdmin(spaceId: string, userId: string): boolean {
  const role = getMemberRole(spaceId, userId);
  return role === "owner" || role === "admin";
}
