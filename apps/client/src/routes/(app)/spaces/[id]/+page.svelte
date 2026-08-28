<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import {
		Button,
		Card,
		ConfirmModal,
		EmptyState,
		Field,
		Input,
		Spinner,
		Textarea,
		icons,
		toast
	} from '@facile/muse';
	import SpaceMembers from './SpaceMembers.svelte';
	import { api } from '$lib/backend';
	import { spaces } from '$lib/spaces.svelte';
	import type { Space, SpaceMember } from '$lib/types';
	import { formatDateTime } from '$lib/utils';

	const spaceId = $derived(page.params.id ?? '');

	let space = $state<Space | null>(null);
	let members = $state<SpaceMember[]>([]);
	let loading = $state(true);

	let name = $state('');
	let description = $state('');
	let savingSpace = $state(false);

	let confirmDeleteSpace = $state(false);

	const role = $derived(space?.role ?? 'member');
	const mayEdit = $derived(role === 'owner' || role === 'admin');
	const mayDelete = $derived(role === 'owner');

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
			{#if !mayEdit && space.description}
				<p class="max-w-2xl text-fc-sm text-fc-fg-muted">{space.description}</p>
			{/if}
		</div>

		{#if mayEdit}
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
		{/if}

		<SpaceMembers {spaceId} {role} bind:members />

		{#if mayDelete}
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
