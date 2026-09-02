<script lang="ts">
	import { page } from '$app/state';
	import { Badge, Button, Drawer, EmptyState, Spinner, StatCard, icons, toast } from '@facile/muse';
	import EventTimeline from '$lib/components/EventTimeline.svelte';
	import ProviderEditorFields from '$lib/components/ProviderEditorFields.svelte';
	import TagPill from '$lib/components/TagPill.svelte';
	import { api } from '$lib/backend';
	import { dashboard } from '$lib/dashboard.svelte';
	import { providerIcon } from '$lib/icons';
	import { tagToneMap } from '$lib/tags';
	import { providerLabel, type EventEntry, type EventStats, type Provider } from '$lib/types';

	const providerId = $derived(page.params.id ?? '');
	const provider = $derived(
		dashboard.settings.providers.items.find((candidate) => candidate.id === providerId)
	);
	const tones = $derived(tagToneMap(dashboard.availableTags));

	let events = $state<EventEntry[]>([]);
	let stats = $state<EventStats | null>(null);
	let loading = $state(false);
	let editing = $state<Provider | null>(null);
	let drawerOpen = $state(false);

	async function refresh() {
		if (!providerId) return;
		loading = true;
		try {
			const payload = await api.events({ providerId, limit: 100 });
			events = payload.events;
			stats = payload.stats;
		} catch (error) {
			toast.danger(error instanceof Error ? error.message : 'Failed to load the event log');
		} finally {
			loading = false;
		}
	}

	function openEdit() {
		const p = dashboard.settings.providers.items.find((c) => c.id === providerId);
		if (p) {
			editing = structuredClone(p);
			drawerOpen = true;
		}
	}

	function closeEdit() {
		editing = null;
		drawerOpen = false;
	}

	async function saveEdit() {
		if (!editing) return;
		const current = editing;
		try {
			dashboard.setProviders(dashboard.settings.providers.items.map((p) => (p.id === current.id ? current : p)));
			await dashboard.save();
			closeEdit();
		} catch (error) {
			toast.danger(error instanceof Error ? error.message : 'Failed to save changes');
		}
	}

	$effect(() => {
		void providerId;
		void refresh();
	});
</script>

<div class="flex flex-col gap-10">
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between">
			<Button variant="ghost" size="sm" icon={icons.chevronLeft} href="/providers" class="-ml-3.5 w-fit">
				Back to providers
			</Button>
			{#if provider}
				<Button variant="outline" size="sm" icon={icons.edit} onclick={openEdit}>
					Edit provider
				</Button>
			{/if}
		</div>

		{#if provider}
			<div class="flex flex-wrap items-center gap-3">
				<span
					class="flex size-10 shrink-0 items-center justify-center rounded-fc-md bg-fc-surface text-fc-fg-muted"
				>
					<iconify-icon icon={providerIcon(provider.type)} width="22" height="22" class="block"
					></iconify-icon>
				</span>
				<h1 class="text-fc-2xl font-semibold text-fc-fg">{provider.name}</h1>
				<Badge>{providerLabel(provider.type)}</Badge>
				{#each provider.tags as tag (tag)}
					<TagPill {tag} {tones} />
				{/each}
			</div>
		{/if}
	</div>

	{#if provider}
		<div class="grid gap-4 sm:grid-cols-3">
			<StatCard label="Events" value={stats?.totalEvents ?? 0} />
			<StatCard label="Delivered" value={stats?.deliverySuccessCount ?? 0} />
			<StatCard label="Failed" value={stats?.deliveryErrorCount ?? 0} />
		</div>

		<div class="flex flex-col gap-4">
			<div class="flex items-start justify-between gap-4">
				<div class="flex flex-col gap-1">
					<h2 class="text-fc-lg font-semibold text-fc-fg">Event log</h2>
					<p class="text-fc-sm text-fc-fg-muted">The last 100 events this provider produced.</p>
				</div>
				<Button
					variant="ghost"
					icon={icons.refresh}
					onclick={refresh}
					disabled={loading}
					aria-label="Refresh the event log"
				/>
			</div>

			{#if loading && events.length === 0}
				<div class="flex justify-center py-12">
					<Spinner />
				</div>
			{:else}
				<EventTimeline {events} />
			{/if}
		</div>
	{:else}
		<EmptyState
			icon={icons.warning}
			title="This provider no longer exists"
			description="It was deleted, or the link points at an id this Antenne has never known."
		>
			<Button href="/providers">Back to providers</Button>
		</EmptyState>
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
		<Button icon={icons.check} onclick={saveEdit} disabled={dashboard.saving}>
			{dashboard.saving ? 'Saving…' : 'Save'}
		</Button>
	{/snippet}
</Drawer>
