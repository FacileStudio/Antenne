<script lang="ts">
	import {
		Avatar,
		Badge,
		Button,
		Card,
		ConfirmModal,
		EmptyState,
		Field,
		Input,
		Select,
		icons,
		toast
	} from '@facile/muse';
	import { api } from '$lib/backend';
	import type { SpaceMember, SpaceRole } from '$lib/types';
	import { formatDateTime } from '$lib/utils';

	let {
		spaceId,
		role,
		members = $bindable()
	}: {
		spaceId: string;
		role: SpaceRole;
		members: SpaceMember[];
	} = $props();

	const isOwner = $derived(role === 'owner');
	const mayStaff = $derived(role === 'owner' || role === 'admin');
	const ownerCount = $derived(members.filter((member) => member.role === 'owner').length);

	function mayChangeRole(member: SpaceMember) {
		return isOwner && !(member.role === 'owner' && ownerCount <= 1);
	}

	function mayRemove(member: SpaceMember) {
		if (!mayStaff) return false;
		return member.role === 'owner' ? isOwner && ownerCount > 1 : true;
	}

	let newMember = $state('');
	/* Typed as a plain string because muse's `Select` binds `string`; narrowed at the call. */
	let newRole = $state<string>('member');
	let addingMember = $state(false);

	let confirmRemoveMember = $state(false);
	let removalTarget = $state<SpaceMember | null>(null);

	const roleTones = {
		owner: 'owner',
		admin: 'admin',
		member: 'neutral'
	} as const;

	async function addMember() {
		if (!newMember.trim()) {
			toast.danger('A member needs an identifier.');
			return;
		}

		addingMember = true;
		try {
			const payload = await api.addMember(spaceId, newMember.trim(), newRole as SpaceRole);
			// The API returns the existing membership unchanged when the person is
			// already in, so replace rather than append.
			members = [
				...members.filter((member) => member.userId !== payload.member.userId),
				payload.member
			];
			newMember = '';
			toast.success('Member added.');
		} catch (error) {
			toast.danger(error instanceof Error ? error.message : 'Failed to add the member');
		} finally {
			addingMember = false;
		}
	}

	async function changeRole(userId: string, role: SpaceRole) {
		try {
			const payload = await api.updateMember(spaceId, userId, role);
			members = members.map((member) => (member.userId === userId ? payload.member : member));
		} catch (error) {
			toast.danger(error instanceof Error ? error.message : 'Failed to change the role');
		}
	}

	async function removeMember(userId: string) {
		try {
			await api.removeMember(spaceId, userId);
			members = members.filter((member) => member.userId !== userId);
		} catch (error) {
			toast.danger(error instanceof Error ? error.message : 'Failed to remove the member');
		}
	}
</script>

<section class="flex flex-col gap-4">
	<div class="flex flex-col gap-1">
		<h2 class="text-fc-lg font-semibold text-fc-fg">Members</h2>
		<p class="text-fc-sm text-fc-fg-muted">Who belongs to this space, and at what level.</p>
	</div>

	{#if mayStaff}
		<Card class="flex flex-col gap-4">
			<div class="flex flex-col gap-4 sm:flex-row sm:items-end">
				<Field label="Identifier" class="min-w-0 flex-1">
					<Input bind:value={newMember} placeholder="someone@facile.studio" />
				</Field>
				<Field label="Role" class="sm:w-40">
					<Select bind:value={newRole}>
						{#if isOwner}
							<option value="owner">Owner</option>
						{/if}
						<option value="admin">Admin</option>
						<option value="member">Member</option>
					</Select>
				</Field>
				<Button size="lg" icon={icons.plus} onclick={addMember} disabled={addingMember}>
					Add
				</Button>
			</div>
		</Card>
	{/if}

	{#if members.length === 0}
		<EmptyState
			icon={icons.userCircle}
			title="No member yet"
			description="Add someone by the identifier they sign in with."
		/>
	{:else}
		<Card class="flex flex-col divide-y divide-fc-border p-0">
			{#each members as member (member.userId)}
				<div class="flex flex-wrap items-center justify-between gap-4 p-5">
					<div class="flex min-w-0 items-center gap-3">
						<Avatar name={member.userId} size="sm" />
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

					<div class="flex shrink-0 items-center gap-2">
						{#if mayChangeRole(member)}
							<Select
								value={member.role}
								class="w-36"
								onchange={(event) =>
									changeRole(member.userId, event.currentTarget.value as SpaceRole)}
							>
								<option value="owner">Owner</option>
								<option value="admin">Admin</option>
								<option value="member">Member</option>
							</Select>
						{/if}
						{#if mayRemove(member)}
							<Button
								variant="ghost-danger"
								size="lg"
								icon={icons.remove}
								class="w-11 px-0"
								aria-label={`Remove ${member.userId}`}
								onclick={() => {
									removalTarget = member;
									confirmRemoveMember = true;
								}}
							/>
						{/if}
					</div>
				</div>
			{/each}
		</Card>
	{/if}
</section>

<ConfirmModal
	bind:open={confirmRemoveMember}
	title="Remove this member?"
	description={`${removalTarget?.userId ?? 'This person'} loses access to the space. Their ${removalTarget?.role ?? 'member'} role is not kept, so re-adding them starts from scratch.`}
	confirmLabel="Remove"
	tone="danger"
	onConfirm={async () => {
		if (removalTarget) await removeMember(removalTarget.userId);
		removalTarget = null;
	}}
	onCancel={() => (removalTarget = null)}
/>
