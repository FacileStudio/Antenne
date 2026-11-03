<script lang="ts">
	import { page } from '$app/state';
	import { Button, EmptyState, Spinner, icons, toast } from '@facile/muse';
	import SpaceMembers from './SpaceMembers.svelte';
	import { api } from '$lib/backend';
	import type { Space, SpaceMember } from '$lib/types';
	import { formatDateTime } from '$lib/utils';

	const spaceId = $derived(page.params.id ?? '');

	let space = $state<Space | null>(null);
	let members = $state<SpaceMember[]>([]);
	let loading = $state(true);

	async function load() {
		if (!spaceId) return;
		loading = true;
		try {
			const payload = await api.space(spaceId);
			space = payload.space;
			members = payload.members;
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
			icon={icons.space}
			title="This space no longer exists"
			description="It was removed, or the id in the URL never matched one."
		>
			<Button href="/spaces">Back to spaces</Button>
		</EmptyState>
	{:else}
		<div class="flex flex-wrap items-start justify-between gap-4">
			<div class="flex flex-col gap-2">
				<h1 class="text-fc-2xl font-semibold text-fc-fg">{space.name}</h1>
				<p class="text-fc-sm text-fc-fg-muted">
					{members.length}
					{members.length === 1 ? 'member' : 'members'} · created
					{formatDateTime(space.createdAt)}
				</p>
				{#if space.description}
					<p class="max-w-2xl text-fc-sm text-fc-fg-muted">{space.description}</p>
				{/if}
			</div>

			{#if space.manageUrl}
				<Button href={space.manageUrl} icon={icons.space} iconRight={icons.arrow} variant="outline">
					Manage in Registre
				</Button>
			{/if}
		</div>

		<SpaceMembers {members} />
	{/if}
</div>
