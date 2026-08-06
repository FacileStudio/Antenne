<script lang="ts">
	import { page } from '$app/state';
	import { toast } from 'svelte-sonner';
	import EventTimeline from '$lib/components/EventTimeline.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import StatCard from '$lib/components/StatCard.svelte';
	import TagPill from '$lib/components/TagPill.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import { api } from '$lib/backend';
	import { dashboard } from '$lib/dashboard.svelte';
	import { providerIcon } from '$lib/icons';
	import { tagToneMap } from '$lib/tags';
	import { providerLabel, type EventEntry, type EventStats } from '$lib/types';

	const providerId = $derived(page.params.id ?? '');
	const provider = $derived(
		dashboard.settings.providers.items.find((candidate) => candidate.id === providerId)
	);
	const tones = $derived(tagToneMap(dashboard.availableTags));

	let events = $state<EventEntry[]>([]);
	let stats = $state<EventStats | null>(null);
	let loading = $state(false);

	async function refresh() {
		if (!providerId) return;
		loading = true;
		try {
			const payload = await api.events({ providerId, limit: 100 });
			events = payload.events;
			stats = payload.stats;
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to load the event log');
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void providerId;
		void refresh();
	});
</script>

<div class="grid gap-5">
	<a
		href="/providers"
		class="inline-flex w-fit items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
	>
		<Icon icon="solar:arrow-left-linear" size={15} />
		Back to providers
	</a>

	{#if provider}
		<div class="flex flex-wrap items-center gap-3">
			<span class="flex size-10 items-center justify-center rounded-lg border border-border text-muted-foreground">
				<Icon icon={providerIcon(provider.type)} size={22} />
			</span>
			<h2 class="font-heading text-xl font-semibold tracking-tight">{provider.name}</h2>
			<Badge variant="outline">{providerLabel(provider.type)}</Badge>
			{#each provider.tags as tag (tag)}
				<TagPill {tag} {tones} />
			{/each}
		</div>
	{:else}
		<p class="text-sm text-muted-foreground">This provider no longer exists.</p>
	{/if}

	<div class="grid gap-3 sm:grid-cols-3">
		<StatCard label="Events" value={stats?.totalEvents ?? 0} />
		<StatCard label="Delivered" value={stats?.deliverySuccessCount ?? 0} tone="success" />
		<StatCard label="Failed" value={stats?.deliveryErrorCount ?? 0} tone="destructive" />
	</div>

	<div class="flex items-center justify-between gap-3">
		<h3 class="font-heading text-base font-semibold">Event log</h3>
		<Button variant="ghost" size="icon" onclick={refresh} disabled={loading} title="Refresh">
			<Icon icon="solar:refresh-linear" size={15} class={loading ? 'animate-spin' : ''} />
		</Button>
	</div>

	<EventTimeline {events} />
</div>
