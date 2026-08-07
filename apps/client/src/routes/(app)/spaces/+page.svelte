<script lang="ts">
	import { Button, Card, Drawer, EmptyState, Field, Input, Spinner, Textarea, icons, toast } from '@facile/muse';
	import { spaces } from '$lib/spaces.svelte';
	import { formatDateTime } from '$lib/utils';

	let drawerOpen = $state(false);
	let name = $state('');
	let description = $state('');
	let submitting = $state(false);

	$effect(() => {
		void spaces.load();
	});

	function openCreate() {
		name = '';
		description = '';
		drawerOpen = true;
	}

	async function create() {
		if (!name.trim()) {
			toast.danger('A space needs a name.');
			return;
		}

		submitting = true;
		try {
			const space = await spaces.create(name.trim(), description.trim());
			toast.success(`Created ${space.name}.`);
			drawerOpen = false;
		} catch (error) {
			toast.danger(error instanceof Error ? error.message : 'Failed to create the space');
		} finally {
			submitting = false;
		}
	}
</script>

<div class="flex flex-col gap-10">
	<section class="flex flex-col gap-4">
		<div class="flex flex-wrap items-start justify-between gap-4">
			<div class="flex flex-col gap-2">
				<h1 class="text-fc-2xl font-semibold text-fc-fg">Spaces</h1>
				<p class="text-fc-sm text-fc-fg-muted">
					Spaces group members. Antenne's providers and delivery targets stay instance-wide.
				</p>
			</div>

			<Button icon={icons.plus} onclick={openCreate}>New space</Button>
		</div>

		{#if spaces.loading && !spaces.loaded}
			<div class="flex justify-center py-12">
				<Spinner />
			</div>
		{:else if spaces.spaces.length === 0}
			<EmptyState
				icon={icons.usersGroup}
				title="No space yet"
				description="Spaces group members; Antenne's providers and targets are instance-wide."
			>
				<Button icon={icons.plus} onclick={openCreate}>New space</Button>
			</EmptyState>
		{:else}
			<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
				{#each spaces.spaces as space (space.id)}
					<Card href={`/spaces/${space.id}`} class="flex flex-col gap-4">
						<div class="flex items-center gap-3">
							<!-- The tile inverts on hover instead of tinting, so it stays legible once the
							     card itself steps up to fc-surface. -->
							<span
								class="flex size-9 shrink-0 items-center justify-center rounded-fc-md bg-fc-surface text-fc-fg-muted transition-colors group-hover:bg-fc-accent group-hover:text-fc-accent-fg"
							>
								<iconify-icon icon={icons.usersGroup} width="18" height="18" class="block"
								></iconify-icon>
							</span>
							<span class="truncate text-fc-sm font-semibold text-fc-fg">{space.name}</span>
						</div>

						<div class="flex flex-col gap-1">
							{#if space.description}
								<p class="line-clamp-2 text-fc-sm text-fc-fg-muted">{space.description}</p>
							{/if}
							<p class="text-fc-xs text-fc-fg-muted">Created {formatDateTime(space.createdAt)}</p>
						</div>
					</Card>
				{/each}
			</div>
		{/if}
	</section>
</div>

<Drawer bind:open={drawerOpen} title="New space" description="Spaces group members." showClose>
	<div class="flex flex-col gap-4">
		<Field label="Name">
			<Input bind:value={name} placeholder="Studio" />
		</Field>

		<Field label="Description" helper="Optional — what this space is for.">
			<Textarea bind:value={description} rows={3} placeholder="What this space is for" />
		</Field>
	</div>

	{#snippet footer()}
		<div class="flex flex-col gap-2 sm:flex-row sm:justify-end">
			<Button variant="ghost" size="lg" onclick={() => (drawerOpen = false)}>Cancel</Button>
			<Button size="lg" onclick={create} disabled={submitting}>
				{submitting ? 'Creating…' : 'Create space'}
			</Button>
		</div>
	{/snippet}
</Drawer>
