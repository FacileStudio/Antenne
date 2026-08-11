export type SpaceRole = 'owner' | 'admin' | 'member';

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
