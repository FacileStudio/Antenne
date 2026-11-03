export type SpaceRole = 'owner' | 'admin' | 'member';

export interface Space {
	id: string;
	name: string;
	description: string;
	/** The reading caller's own role in this space, not a property of the space. */
	role: SpaceRole;
	/** Registre's id for this space; null until the space is linked. */
	registreSpaceId: string | null;
	/** Where the space is managed in Registre; absent without single sign-on. */
	manageUrl?: string;
	createdAt: string;
	updatedAt: string;
}

export interface SpaceMember {
	spaceId: string;
	userId: string;
	avatarUrl?: string;
	role: SpaceRole;
	joinedAt: string;
}
