<script lang="ts">
	import { toast } from 'svelte-sonner';
	import Icon from './Icon.svelte';
	import ProviderEditorFields from './ProviderEditorFields.svelte';
	import TagPill from './TagPill.svelte';
	import ViewToggle from './ViewToggle.svelte';
	import Badge from './ui/badge.svelte';
	import Button from './ui/button.svelte';
	import Select from './ui/select.svelte';
	import Sheet from './ui/sheet.svelte';
	import Switch from './ui/switch.svelte';
	import { api } from '$lib/backend';
	import { dashboard } from '$lib/dashboard.svelte';
	import { providerIcon } from '$lib/icons';
	import { tagToneMap } from '$lib/tags';
	import { createProvider, providerLabel, providerTypes, type Provider, type ProviderType } from '$lib/types';
	import { cn } from '$lib/utils';

	const providers = $derived(dashboard.settings.providers.items);
	const tones = $derived(tagToneMap(dashboard.availableTags));

	let nextType = $state<ProviderType>('website');
	let view = $state<'list' | 'card'>('card');
	let selectedTags = $state<string[]>([]);

	let editing = $state<Provider | null>(null);
	let editingIndex = $state(-1);
	let sheetOpen = $state(false);
	let testing = $state(false);

	const visible = $derived(
		providers
			.map((provider, index) => ({ provider, index }))
			.filter(
				({ provider }) =>
					selectedTags.length === 0 || selectedTags.every((tag) => provider.tags.includes(tag))
			)
	);

	const typeOptions = providerTypes.map((type) => ({ value: type, label: providerLabel(type) }));

	function open(index: number) {
		editingIndex = index;
		// A structured clone, so closing without saving discards the edit.
		editing = structuredClone($state.snapshot(providers[index])) as Provider;
		sheetOpen = true;
	}

	function add() {
		const provider = createProvider(nextType);
		dashboard.setProviders([provider, ...providers]);
		open(0);
	}

	async function commit() {
		if (!editing || editingIndex < 0) return;
		const draft = structuredClone($state.snapshot(editing)) as Provider;
		dashboard.setProviders(providers.map((current, index) => (index === editingIndex ? draft : current)));
		await dashboard.save();
		sheetOpen = false;
	}

	async function toggleEnabled(index: number, enabled: boolean) {
		dashboard.setProviders(
			providers.map((provider, position) =>
				position === index ? { ...provider, enabled } : provider
			)
		);
		await dashboard.save({ silent: true });
	}

	async function remove(index: number) {
		dashboard.setProviders(providers.filter((_, position) => position !== index));
		if (editingIndex === index) sheetOpen = false;
		await dashboard.save();
	}

	function toggleTag(tag: string) {
		selectedTags = selectedTags.includes(tag)
			? selectedTags.filter((current) => current !== tag)
			: [...selectedTags, tag];
	}

	async function testConnection() {
		if (!editing) return;
		testing = true;
		try {
			await api.testProvider(editing.id);
			toast.success(`Connection OK for ${editing.name}.`);
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'The connection test failed');
		} finally {
			testing = false;
		}
	}
</script>

<div class="grid gap-4">
	<div class="flex flex-wrap items-end gap-3">
		<div class="grid gap-1.5">
			<span class="text-xs text-muted-foreground">New provider type</span>
			<Select bind:value={nextType as string} options={typeOptions} class="w-56" />
		</div>

		<Button onclick={add}>
			<Icon icon="solar:add-circle-linear" size={15} />
			Add provider
		</Button>

		<div class="ml-auto">
			{#if providers.length > 0}
				<ViewToggle bind:value={view} />
			{/if}
		</div>
	</div>

	{#if dashboard.availableTags.length > 0}
		<div class="flex flex-wrap items-center gap-2">
			<span class="text-xs text-muted-foreground">Filter by tag</span>
			{#each dashboard.availableTags as tag (tag)}
				<TagPill
					{tag}
					{tones}
					interactive
					selected={selectedTags.includes(tag)}
					onclick={() => toggleTag(tag)}
				/>
			{/each}
			{#if selectedTags.length > 0}
				<Button variant="ghost" size="sm" onclick={() => (selectedTags = [])}>Clear</Button>
			{/if}
		</div>
	{/if}

	{#if providers.length === 0}
		<p class="rounded-xl border border-dashed border-border px-4 py-8 text-center text-sm text-muted-foreground">
			No provider configured yet.
		</p>
	{:else if visible.length === 0}
		<p class="rounded-xl border border-dashed border-border px-4 py-8 text-center text-sm text-muted-foreground">
			No provider matches the current tag filter.
		</p>
	{:else if view === 'card'}
		<div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
			{#each visible as { provider, index } (provider.id)}
				<article
					class={cn(
						'flex flex-col gap-3 rounded-xl border border-border bg-card p-4 transition-colors hover:border-foreground/20',
						!provider.enabled && 'opacity-55'
					)}
				>
					<div class="flex items-start justify-between">
						<span
							class="flex size-10 items-center justify-center rounded-lg border border-border text-muted-foreground"
						>
							<Icon icon={providerIcon(provider.type)} size={22} />
						</span>
						<Switch
							size="sm"
							checked={provider.enabled}
							aria-label={`Enable ${provider.name}`}
							onCheckedChange={(checked: boolean) => toggleEnabled(index, checked)}
						/>
					</div>

					<div class="grid min-w-0 gap-0.5">
						<span class="truncate text-sm font-semibold">{provider.name}</span>
						<span class="text-xs text-muted-foreground">{providerLabel(provider.type)}</span>
					</div>

					{#if provider.tags.length > 0}
						<div class="flex flex-wrap gap-1.5">
							{#each provider.tags as tag (tag)}
								<TagPill {tag} {tones} />
							{/each}
						</div>
					{/if}

					<div class="mt-auto flex gap-1 border-t border-border pt-2.5">
						<Button variant="ghost" size="sm" class="flex-1" onclick={() => open(index)} title="Edit">
							<Icon icon="solar:pen-linear" size={14} />
						</Button>
						<Button variant="ghost" size="sm" class="flex-1" href={`/providers/${provider.id}`} title="Activity">
							<Icon icon="solar:chart-2-linear" size={14} />
						</Button>
						<Button
							variant="destructive"
							size="sm"
							class="flex-1"
							onclick={() => remove(index)}
							title="Delete"
						>
							<Icon icon="solar:trash-bin-trash-linear" size={14} />
						</Button>
					</div>
				</article>
			{/each}
		</div>
	{:else}
		<div class="grid gap-2">
			{#each visible as { provider, index } (provider.id)}
				<article
					class={cn(
						'flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border bg-card px-4 py-3 transition-colors hover:border-foreground/20',
						!provider.enabled && 'opacity-55'
					)}
				>
					<div class="flex min-w-0 flex-wrap items-center gap-2.5">
						<Switch
							size="sm"
							checked={provider.enabled}
							aria-label={`Enable ${provider.name}`}
							onCheckedChange={(checked: boolean) => toggleEnabled(index, checked)}
						/>
						<Icon icon={providerIcon(provider.type)} size={19} class="text-muted-foreground" />
						<strong class="truncate text-sm font-semibold">{provider.name}</strong>
						<Badge variant="outline">{providerLabel(provider.type)}</Badge>
						{#each provider.tags as tag (tag)}
							<TagPill {tag} {tones} />
						{/each}
					</div>

					<div class="flex items-center gap-1">
						<Button variant="ghost" size="sm" onclick={() => open(index)}>
							<Icon icon="solar:pen-linear" size={14} />
							Edit
						</Button>
						<Button variant="ghost" size="icon" href={`/providers/${provider.id}`} title="Activity">
							<Icon icon="solar:chart-2-linear" size={14} />
						</Button>
						<Button variant="destructive" size="icon" onclick={() => remove(index)} title="Delete">
							<Icon icon="solar:trash-bin-trash-linear" size={14} />
						</Button>
					</div>
				</article>
			{/each}
		</div>
	{/if}
</div>

<Sheet
	bind:open={sheetOpen}
	title={editing ? `Edit ${editing.name}` : 'Edit provider'}
	description={editing ? providerLabel(editing.type) : ''}
>
	{#if editing}
		<ProviderEditorFields
			bind:provider={editing}
			availableTags={dashboard.availableTags}
			{tones}
		/>
	{/if}

	{#snippet footer()}
		{#if editing?.type === 'imap'}
			<Button variant="outline" onclick={testConnection} disabled={testing}>
				<Icon icon="solar:plug-circle-linear" size={15} />
				{testing ? 'Testing…' : 'Test connection'}
			</Button>
		{/if}
		<Button onclick={commit} disabled={dashboard.saving}>
			<Icon icon="solar:diskette-linear" size={15} />
			{dashboard.saving ? 'Saving…' : 'Save'}
		</Button>
	{/snippet}
</Sheet>
