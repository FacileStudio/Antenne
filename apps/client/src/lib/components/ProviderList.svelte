<script lang="ts">
	import {
		Button,
		ConfirmModal,
		Drawer,
		EmptyState,
		Field,
		Select,
		icons,
		toast
	} from '@facile/muse';
	import ProviderCards from './ProviderCards.svelte';
	import ProviderEditorFields from './ProviderEditorFields.svelte';
	import ProviderRows from './ProviderRows.svelte';
	import TagPill from './TagPill.svelte';
	import ViewToggle from './ViewToggle.svelte';
	import { api } from '$lib/backend';
	import { dashboard } from '$lib/dashboard.svelte';
	import { tagToneMap } from '$lib/tags';
	import { createProvider, providerLabel, providerTypes, type Provider, type ProviderType } from '$lib/types';

	const providers = $derived(dashboard.settings.providers.items);
	const tones = $derived(tagToneMap(dashboard.availableTags));

	let nextType = $state<string>('website');
	let view = $state<'list' | 'card'>('card');
	let selectedTags = $state<string[]>([]);

	let editing = $state<Provider | null>(null);
	let editingIndex = $state(-1);
	let drawerOpen = $state(false);
	let testing = $state(false);

	let pendingRemoval = $state(-1);
	let confirmOpen = $state(false);

	const visible = $derived(
		providers
			.map((provider, index) => ({ provider, index }))
			.filter(
				({ provider }) =>
					selectedTags.length === 0 || selectedTags.every((tag) => provider.tags.includes(tag))
			)
	);

	const doomed = $derived(pendingRemoval >= 0 ? providers[pendingRemoval] : null);

	const NULLABLE_TEXT = ['secretValue', 'secretHeader', 'password'] as const;

	function hydrate(provider: Provider): Provider {
		const draft = provider as unknown as Record<string, unknown>;
		for (const key of NULLABLE_TEXT) {
			if (draft[key] === null) draft[key] = '';
		}
		if (!draft.tags) draft.tags = [];
		return provider;
	}

	function open(index: number) {
		editingIndex = index;
		editing = hydrate(structuredClone($state.snapshot(providers[index])) as Provider);
		drawerOpen = true;
	}

	function add() {
		const provider = createProvider(nextType as ProviderType);
		dashboard.setProviders([provider, ...providers]);
		open(0);
	}

	async function commit() {
		if (!editing || editingIndex < 0) return;
		const draft = structuredClone($state.snapshot(editing)) as Provider;
		dashboard.setProviders(providers.map((current, index) => (index === editingIndex ? draft : current)));
		await dashboard.save();
		drawerOpen = false;
	}

	async function toggleEnabled(index: number, enabled: boolean) {
		dashboard.setProviders(
			providers.map((provider, position) =>
				position === index ? { ...provider, enabled } : provider
			)
		);
		await dashboard.save({ silent: true });
	}

	function askRemoval(index: number) {
		pendingRemoval = index;
		confirmOpen = true;
	}

	async function remove() {
		const index = pendingRemoval;
		if (index < 0) return;
		dashboard.setProviders(providers.filter((_, position) => position !== index));
		if (editingIndex === index) drawerOpen = false;
		pendingRemoval = -1;
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
			toast.danger(error instanceof Error ? error.message : 'The connection test failed');
		} finally {
			testing = false;
		}
	}
</script>

<div class="flex flex-col gap-4">
	<div class="flex flex-wrap items-end gap-4">
		<Field label="New provider type" class="w-56">
			<Select bind:value={nextType}>
				{#each providerTypes as type (type)}
					<option value={type}>{providerLabel(type)}</option>
				{/each}
			</Select>
		</Field>

		<Button size="lg" icon={icons.plus} onclick={add}>Add provider</Button>

		{#if providers.length > 0}
			<div class="ml-auto">
				<ViewToggle bind:value={view} />
			</div>
		{/if}
	</div>

	{#if dashboard.availableTags.length > 0}
		<div class="flex flex-wrap items-center gap-2">
			<span class="text-fc-sm text-fc-fg-muted">Filter by tag</span>
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
		<EmptyState
			icon={icons.plug}
			title="No provider configured yet"
			description="A provider is a source Antenne watches or listens to — a website, an inbox, a feed or an incoming webhook."
		>
			<Button icon={icons.plus} onclick={add}>Add provider</Button>
		</EmptyState>
	{:else if visible.length === 0}
		<EmptyState
			icon={icons.filter}
			title="No provider matches the current tag filter"
			description="Every selected tag has to be on the provider for it to show up here."
		>
			<Button variant="outline" onclick={() => (selectedTags = [])}>Clear the filter</Button>
		</EmptyState>
	{:else if view === 'card'}
		<ProviderCards
			entries={visible}
			{tones}
			onToggle={toggleEnabled}
			onOpen={open}
			onRemove={askRemoval}
		/>
	{:else}
		<ProviderRows
			entries={visible}
			{tones}
			onToggle={toggleEnabled}
			onOpen={open}
			onRemove={askRemoval}
		/>
	{/if}
</div>

<Drawer
	bind:open={drawerOpen}
	showClose
	title={editing ? `Edit ${editing.name}` : 'Edit provider'}
	description={editing ? providerLabel(editing.type) : undefined}
>
	{#if editing}
		<ProviderEditorFields bind:provider={editing} availableTags={dashboard.availableTags} {tones} />
	{/if}

	{#snippet footer()}
		{#if editing?.type === 'imap'}
			<Button variant="outline" icon={icons.plug} onclick={testConnection} disabled={testing}>
				{testing ? 'Testing…' : 'Test connection'}
			</Button>
		{/if}
		<Button icon={icons.check} onclick={commit} disabled={dashboard.saving}>
			{dashboard.saving ? 'Saving…' : 'Save'}
		</Button>
	{/snippet}
</Drawer>

<ConfirmModal
	bind:open={confirmOpen}
	tone="danger"
	title="Delete this provider?"
	description={doomed
		? `${doomed.name} stops being watched and every delivery target routing on it — by id or by tag — loses that source. Events already in the log are kept.`
		: ''}
	confirmLabel="Delete"
	onConfirm={remove}
	onCancel={() => (pendingRemoval = -1)}
/>
