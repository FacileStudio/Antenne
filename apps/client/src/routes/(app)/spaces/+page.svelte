<script lang="ts">
	import { Button, Card, EmptyState, Spinner, icons } from '@facile/muse';
	import { spaces } from '$lib/spaces.svelte';
	import { formatDateTime } from '$lib/utils';

	$effect(() => {
		void spaces.load();
	});
</script>

{#snippet manage()}
	{#if spaces.manageUrl}
		<Button href={spaces.manageUrl} iconRight={icons.arrow} variant="outline">
			Manage in Registre
		</Button>
	{/if}
{/snippet}

<div class="flex flex-col gap-10">
	<section class="flex flex-col gap-4">
		<div class="flex flex-wrap items-start justify-between gap-4">
			<div class="flex flex-col gap-2">
				<h1 class="text-fc-2xl font-semibold text-fc-fg">Spaces</h1>
				<p class="text-fc-sm text-fc-fg-muted">
					Spaces and their members come from Registre, the same in every Facile app. Antenne's
					providers and delivery targets stay instance-wide.
				</p>
			</div>

			{@render manage()}
		</div>

		{#if spaces.loading && !spaces.loaded}
			<div class="flex justify-center py-12">
				<Spinner />
			</div>
		{:else if spaces.spaces.length === 0}
			<EmptyState
				icon={icons.usersGroup}
				title="No space yet"
				description="Create a space or join one in Registre and it shows up here."
			>
				{@render manage()}
			</EmptyState>
		{:else}
			<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
				{#each spaces.spaces as space (space.id)}
					<Card href={`/spaces/${space.id}`} class="flex flex-col gap-4">
						<div class="flex items-center gap-3">
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
