<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
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
		Spinner,
		Textarea,
		icons,
		toast
	} from '@facile/muse';
	import { api } from '$lib/backend';
	import { spaces } from '$lib/spaces.svelte';
	import type { Space, SpaceMember, SpaceRole } from '$lib/types';
	import { formatDateTime } from '$lib/utils';

	const spaceId = $derived(page.params.id ?? '');

	let space = $state<Space | null>(null);
	let members = $state<SpaceMember[]>([]);
	let loading = $state(true);

	let name = $state('');
	let description = $state('');
	let savingSpace = $state(false);

	let newMember = $state('');
	/* Typed as a plain string because muse's `Select` binds `string`; narrowed at the call. */
	let newRole = $state<string>('member');
	let addingMember = $state(false);

	let confirmDeleteSpace = $state(false);
	let confirmRemoveMember = $state(false);
	let removalTarget = $state<SpaceMember | null>(null);

	const roleTones = {
		owner: 'owner',
		admin: 'admin',
		member: 'neutral'
	} as const;

	async function load() {
		if (!spaceId) return;
		loading = true;
		try {
			const payload = await api.space(spaceId);
			space = payload.space;
			members = payload.members;
			name = payload.space.name;
			description = payload.space.description;
		} catch (error) {
			toast.danger(error instanceof Error ? error.message : 'Failed to load the space');
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void spaceId;
		void load();
	});

	async function saveSpace() {
		savingSpace = true;
		try {
			const updated = await spaces.update(spaceId, {
				name: name.trim(),
				description: description.trim()
			});
			space = updated;
			toast.success('Space updated.');
		} catch (error) {
			toast.danger(error instanceof Error ? error.message : 'Failed to update the space');
		} finally {
			savingSpace = false;
		}
	}

	async function removeSpace() {
		try {
			await spaces.remove(spaceId);
			toast.success('Space deleted.');
			await goto('/spaces', { replaceState: true });
		} catch (error) {
			toast.danger(error instanceof Error ? error.message : 'Failed to delete the space');
		}
	}

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

<div class="flex flex-col gap-10">
	<Button variant="ghost" size="sm" icon={icons.chevronLeft} href="/spaces" class="w-fit">
		Back to spaces
	</Button>

	{#if loading}
		<div class="flex justify-center py-12">
			<Spinner />
		</div>
	{:else if !space}
		<EmptyState
			icon={icons.warning}
			title="This space no longer exists"
			description="It was deleted, or the id in the URL never matched one."
		>
			<Button href="/spaces">Back to spaces</Button>
		</EmptyState>
	{:else}
		<div class="flex flex-col gap-2">
			<h1 class="text-fc-2xl font-semibold text-fc-fg">{space.name}</h1>
			<p class="text-fc-sm text-fc-fg-muted">
				{members.length}
				{members.length === 1 ? 'member' : 'members'} · created
				{formatDateTime(space.createdAt)}
			</p>
		</div>

		<section class="flex flex-col gap-4">
			<div class="flex flex-col gap-1">
				<h2 class="text-fc-lg font-semibold text-fc-fg">Details</h2>
				<p class="text-fc-sm text-fc-fg-muted">The name members see in the switcher.</p>
			</div>

			<Card class="flex max-w-2xl flex-col gap-4">
				<Field label="Name">
					<Input bind:value={name} />
				</Field>

				<Field label="Description">
					<Textarea bind:value={description} rows={3} />
				</Field>

				<div>
					<Button icon={icons.check} onclick={saveSpace} disabled={savingSpace}>
						{savingSpace ? 'Saving…' : 'Save'}
					</Button>
				</div>
			</Card>
		</section>

		<section class="flex flex-col gap-4">
			<div class="flex flex-col gap-1">
				<h2 class="text-fc-lg font-semibold text-fc-fg">Members</h2>
				<p class="text-fc-sm text-fc-fg-muted">Who belongs to this space, and at what level.</p>
			</div>

			<Card class="flex flex-col gap-4">
				<div class="flex flex-col gap-4 sm:flex-row sm:items-end">
					<Field label="Identifier" class="min-w-0 flex-1">
						<Input bind:value={newMember} placeholder="someone@facile.studio" />
					</Field>
					<Field label="Role" class="sm:w-40">
						<Select bind:value={newRole}>
							<option value="owner">Owner</option>
							<option value="admin">Admin</option>
							<option value="member">Member</option>
						</Select>
					</Field>
					<Button size="lg" icon={icons.plus} onclick={addMember} disabled={addingMember}>
						Add
					</Button>
				</div>
			</Card>

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
								<!--
									The role is read off the change event rather than bound: `changeRole` refetches
									the row and rewrites `members`, and a binding would race its own listener —
									the refetched value lands back in the select mid-dispatch and the change is lost.
								-->
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
							</div>
						</div>
					{/each}
				</Card>
			{/if}
		</section>

		<section class="flex flex-col gap-4">
			<div class="flex flex-col gap-1">
				<h2 class="text-fc-lg font-semibold text-fc-fg">Danger zone</h2>
				<p class="text-fc-sm text-fc-fg-muted">Deleting a space cannot be undone.</p>
			</div>

			<Card class="flex flex-wrap items-center justify-between gap-4">
				<p class="text-fc-sm text-fc-fg-muted">
					Removes the space, every membership in it, and any scoping that pointed at it.
				</p>
				<Button variant="danger" icon={icons.remove} onclick={() => (confirmDeleteSpace = true)}>
					Delete space
				</Button>
			</Card>
		</section>
	{/if}
</div>

<ConfirmModal
	bind:open={confirmDeleteSpace}
	title="Delete this space?"
	description={`Every membership in ${space?.name ?? 'this space'} is dropped and anything scoped to it loses that scope. This cannot be undone.`}
	confirmLabel="Delete space"
	tone="danger"
	onConfirm={removeSpace}
/>

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
