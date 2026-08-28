export type SpaceRole = 'owner' | 'admin' | 'member';

export interface Space {
	id: string;
	name: string;
	description: string;
	/** The reading caller's own role in this space, not a property of the space. */
	role: SpaceRole;
	createdAt: string;
	updatedAt: string;
}

export interface SpaceMember {
	spaceId: string;
	userId: string;
	role: SpaceRole;
	joinedAt: string;
}
