import { createJsonResponse } from "./server-responses";
import {
  listSpaces,
  getSpace,
  createSpace,
  updateSpace,
  deleteSpace,
  getSpaceMembers,
  addSpaceMember,
  updateMemberRole,
  removeSpaceMember,
  type SpaceRole,
} from "./spaces";

const VALID_ROLES: SpaceRole[] = ["owner", "admin", "member"];

async function parseJson(req: Request) {
  try {
    return await req.json();
  } catch {
    throw new Error("Invalid JSON body.");
  }
}

export async function handleSpacesApiRequest(req: Request, url: URL): Promise<Response | null> {
  if (url.pathname === "/api/spaces" && req.method === "GET") {
    return createJsonResponse({ spaces: listSpaces() });
  }

  if (url.pathname === "/api/spaces" && req.method === "POST") {
    const body = await parseJson(req) as { name?: string; description?: string; ownerUserId?: string };
    if (!body.name || typeof body.name !== "string" || !body.name.trim()) {
      return createJsonResponse({ error: "name is required" }, { status: 400 });
    }
    const space = createSpace(
      body.name.trim(),
      typeof body.description === "string" ? body.description.trim() : "",
      typeof body.ownerUserId === "string" ? body.ownerUserId : undefined,
    );
    console.log(`[spaces] created space "${space.name}" (${space.id})`);
    return createJsonResponse({ space }, { status: 201 });
  }

  const spaceMatch = url.pathname.match(/^\/api\/spaces\/([^/]+)$/);
  if (spaceMatch) {
    const spaceId = spaceMatch[1]!;

    if (req.method === "GET") {
      const space = getSpace(spaceId);
      if (!space) return createJsonResponse({ error: "space not found" }, { status: 404 });
      const members = getSpaceMembers(spaceId);
      return createJsonResponse({ space, members });
    }

    if (req.method === "PUT") {
      const body = await parseJson(req) as { name?: string; description?: string };
      const space = updateSpace(spaceId, {
        name: typeof body.name === "string" ? body.name.trim() : undefined,
        description: typeof body.description === "string" ? body.description.trim() : undefined,
      });
      if (!space) return createJsonResponse({ error: "space not found" }, { status: 404 });
      console.log(`[spaces] updated space "${space.name}" (${space.id})`);
      return createJsonResponse({ space });
    }

    if (req.method === "DELETE") {
      const deleted = deleteSpace(spaceId);
      if (!deleted) return createJsonResponse({ error: "space not found" }, { status: 404 });
      console.log(`[spaces] deleted space ${spaceId}`);
      return createJsonResponse({ ok: true });
    }
  }

  const membersMatch = url.pathname.match(/^\/api\/spaces\/([^/]+)\/members$/);
  if (membersMatch) {
    const spaceId = membersMatch[1]!;

    if (req.method === "GET") {
      if (!getSpace(spaceId)) return createJsonResponse({ error: "space not found" }, { status: 404 });
      return createJsonResponse({ members: getSpaceMembers(spaceId) });
    }

    if (req.method === "POST") {
      const body = await parseJson(req) as { userId?: string; role?: string };
      if (!body.userId || typeof body.userId !== "string") {
        return createJsonResponse({ error: "userId is required" }, { status: 400 });
      }
      const role = (typeof body.role === "string" && VALID_ROLES.includes(body.role as SpaceRole))
        ? body.role as SpaceRole
        : "member";
      const member = addSpaceMember(spaceId, body.userId, role);
      if (!member) return createJsonResponse({ error: "space not found" }, { status: 404 });
      console.log(`[spaces] added member "${body.userId}" to space ${spaceId} as ${role}`);
      return createJsonResponse({ member }, { status: 201 });
    }
  }

  const memberMatch = url.pathname.match(/^\/api\/spaces\/([^/]+)\/members\/([^/]+)$/);
  if (memberMatch) {
    const spaceId = memberMatch[1]!;
    const userId = decodeURIComponent(memberMatch[2]!);

    if (req.method === "PUT") {
      const body = await parseJson(req) as { role?: string };
      if (!body.role || !VALID_ROLES.includes(body.role as SpaceRole)) {
        return createJsonResponse({ error: "valid role is required (owner, admin, member)" }, { status: 400 });
      }
      const member = updateMemberRole(spaceId, userId, body.role as SpaceRole);
      if (!member) return createJsonResponse({ error: "member not found" }, { status: 404 });
      console.log(`[spaces] updated role for "${userId}" in space ${spaceId} to ${body.role}`);
      return createJsonResponse({ member });
    }

    if (req.method === "DELETE") {
      const removed = removeSpaceMember(spaceId, userId);
      if (!removed) return createJsonResponse({ error: "member not found" }, { status: 404 });
      console.log(`[spaces] removed member "${userId}" from space ${spaceId}`);
      return createJsonResponse({ ok: true });
    }
  }

  return null;
}
