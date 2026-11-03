<script lang="ts">
	import { Avatar, Badge, Card, EmptyState, icons } from '@facile/muse';
	import type { SpaceMember } from '$lib/types';
	import { formatDateTime } from '$lib/utils';

	let { members }: { members: SpaceMember[] } = $props();

	const roleTones = {
		owner: 'owner',
		admin: 'admin',
		member: 'neutral'
	} as const;
</script>

<section class="flex flex-col gap-4">
	<div class="flex flex-col gap-1">
		<h2 class="text-fc-lg font-semibold text-fc-fg">Members</h2>
		<p class="text-fc-sm text-fc-fg-muted">
			Who belongs to this space, and at what level. Invites and roles are managed in Registre.
		</p>
	</div>

	{#if members.length === 0}
		<EmptyState
			icon={icons.userCircle}
			title="No member yet"
			description="Invite someone in Registre and they show up here."
		/>
	{:else}
		<Card class="flex flex-col divide-y divide-fc-border p-0">
			{#each members as member (member.userId)}
				<div class="flex min-w-0 items-center gap-3 p-5">
					<Avatar src={member.avatarUrl || undefined} name={member.userId} size="sm" />
					<div class="flex min-w-0 flex-col gap-1">
						<div class="flex min-w-0 items-center gap-2">
							<span class="truncate text-fc-sm font-medium text-fc-fg">{member.userId}</span>
							<Badge tone={roleTones[member.role]}>{member.role}</Badge>
						</div>
						<span class="text-fc-xs text-fc-fg-muted">
							Joined {formatDateTime(member.joinedAt)}
						</span>
					</div>
				</div>
			{/each}
		</Card>
	{/if}
</section>
