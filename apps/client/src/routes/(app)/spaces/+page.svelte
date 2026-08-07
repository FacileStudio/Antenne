<script lang="ts">
	import { toast } from 'svelte-sonner';
	import Icon from '$lib/components/Icon.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Field from '$lib/components/ui/field.svelte';
	import Input from '$lib/components/ui/input.svelte';
	import Sheet from '$lib/components/ui/sheet.svelte';
	import { spaces } from '$lib/spaces.svelte';
	import { formatDateTime } from '$lib/utils';

	let sheetOpen = $state(false);
	let name = $state('');
	let description = $state('');
	let submitting = $state(false);

	$effect(() => {
		void spaces.load();
	});

	function openCreate() {
		name = '';
		description = '';
		sheetOpen = true;
	}

	async function create() {
		if (!name.trim()) {
			toast.error('A space needs a name.');
			return;
		}

		submitting = true;
		try {
			const space = await spaces.create(name.trim(), description.trim());
			toast.success(`Created ${space.name}.`);
			sheetOpen = false;
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to create the space');
		} finally {
			submitting = false;
		}
	}
</script>

<section class="grid gap-5">
	<PageHeader eyebrow="Spaces" title="Teams">
		{#snippet actions()}
			<Button onclick={openCreate}>
				<Icon icon="solar:add-circle-linear" size={15} />
				New space
			</Button>
		{/snippet}
	</PageHeader>

	{#if spaces.spaces.length === 0}
		<p class="rounded-xl border border-dashed border-border px-4 py-8 text-center text-sm text-muted-foreground">
			No space yet. Spaces group members; Antenne's providers and targets are instance-wide.
		</p>
	{:else}
		<div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
			{#each spaces.spaces as space (space.id)}
				<a
					href={`/spaces/${space.id}`}
					class="grid gap-2 rounded-xl border border-border bg-card p-4 transition-colors hover:border-foreground/20"
				>
					<div class="flex items-center gap-2.5">
						<span class="flex size-9 items-center justify-center rounded-lg border border-border text-muted-foreground">
							<Icon icon="solar:buildings-2-linear" size={18} />
						</span>
						<strong class="truncate text-sm font-semibold">{space.name}</strong>
					</div>

					{#if space.description}
						<p class="line-clamp-2 text-sm text-muted-foreground">{space.description}</p>
					{/if}

					<span class="text-xs text-muted-foreground">
						Created {formatDateTime(space.createdAt)}
					</span>
				</a>
			{/each}
		</div>
	{/if}
</section>

<Sheet bind:open={sheetOpen} title="New space" description="Spaces group members.">
	<Field label="Name" required>
		<Input bind:value={name} placeholder="Studio" />
	</Field>

	<Field label="Description">
		<Input bind:value={description} placeholder="What this space is for" />
	</Field>

	{#snippet footer()}
		<Button onclick={create} disabled={submitting}>
			{submitting ? 'Creating…' : 'Create space'}
		</Button>
	{/snippet}
</Sheet>
