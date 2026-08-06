<script lang="ts">
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';
	import ActivityChart from '$lib/components/ActivityChart.svelte';
	import EventTimeline from '$lib/components/EventTimeline.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import StatCard from '$lib/components/StatCard.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Input from '$lib/components/ui/input.svelte';
	import Select from '$lib/components/ui/select.svelte';
	import { api } from '$lib/backend';
	import { dashboard } from '$lib/dashboard.svelte';
	import type { EventEntry, EventStats } from '$lib/types';
	import { formatDateTime } from '$lib/utils';

	const PAGE_SIZE = 50;

	let events = $state<EventEntry[]>([]);
	let stats = $state<EventStats | null>(null);
	let total = $state(0);
	let offset = $state(0);
	let loading = $state(false);
	let loadingMore = $state(false);
	let search = $state('');
	let source = $state('');

	let searchTimer: ReturnType<typeof setTimeout> | null = null;

	const hasMore = $derived(offset < total);

	const successRate = $derived.by(() => {
		if (!stats) return 100;
		const attempts = stats.deliverySuccessCount + stats.deliveryErrorCount;
		return attempts === 0 ? 100 : Math.round((stats.deliverySuccessCount / attempts) * 100);
	});

	const recentCount = $derived.by(() => {
		const sixHoursAgo = Date.now() - 6 * 60 * 60 * 1000;
		return events.filter((entry) => new Date(entry.timestamp).getTime() > sixHoursAgo).length;
	});

	const sourceOptions = [
		{ value: '', label: 'All sources' },
		{ value: 'webhook', label: 'Webhook' },
		{ value: 'website', label: 'Website' },
		{ value: 'imap', label: 'Inbox' },
		{ value: 'rss', label: 'RSS' },
		{ value: 'system', label: 'System' },
		{ value: 'test', label: 'Test' }
	];

	async function fetchEvents({ append = false } = {}) {
		if (append) loadingMore = true;
		else loading = true;

		try {
			const payload = await api.events({
				limit: PAGE_SIZE,
				offset: append ? offset : 0,
				q: search || undefined,
				source: source || undefined
			});

			events = append ? [...events, ...payload.events] : payload.events;
			offset = append ? offset + payload.events.length : payload.events.length;
			total = payload.total;
			stats = payload.stats;
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to load the activity log');
		} finally {
			loading = false;
			loadingMore = false;
		}
	}

	function onSearchInput() {
		if (searchTimer) clearTimeout(searchTimer);
		searchTimer = setTimeout(() => void fetchEvents(), 300);
	}

	async function replay(eventId: string) {
		const targets = dashboard.settings.delivery.items;
		const target = targets.find((candidate) => candidate.enabled) ?? targets[0];
		if (!target) {
			toast.error('No delivery target configured.');
			return;
		}

		try {
			await api.replay(eventId, target.id);
			toast.success(`Replayed to ${target.name}.`);
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'The replay failed');
		}
	}

	onMount(() => {
		void fetchEvents();

		// The live feed is server-sent rather than polled: the traffic is one-way
		// and an EventSource reconnects on its own after a deploy.
		const stream = new EventSource('/api/events/stream');
		stream.onmessage = (message) => {
			try {
				const entry = JSON.parse(message.data) as EventEntry;
				// A filtered view must not gain rows that do not match it.
				if (search) return;
				if (source && entry.source !== source) return;
				events = [entry, ...events];
				total += 1;
			} catch {
				// A malformed frame is not worth breaking the page over.
			}
		};

		return () => {
			stream.close();
			if (searchTimer) clearTimeout(searchTimer);
		};
	});
</script>

<section class="grid gap-5">
	<PageHeader eyebrow="Overview" title="Stats">
		{#snippet actions()}
			<span
				class="inline-flex items-center gap-1.5 rounded-full border border-border bg-muted px-2.5 py-1 text-xs text-muted-foreground"
			>
				<Icon icon="solar:clock-circle-linear" size={13} />
				{stats?.lastEventAt ? formatDateTime(stats.lastEventAt) : 'No events yet'}
			</span>
			<Button variant="outline" onclick={() => fetchEvents()} disabled={loading}>
				<Icon icon="solar:refresh-linear" size={14} class={loading ? 'animate-spin' : ''} />
				{loading ? 'Refreshing…' : 'Refresh'}
			</Button>
		{/snippet}
	</PageHeader>

	<div class="grid grid-cols-2 gap-3 md:grid-cols-3 lg:grid-cols-5">
		<StatCard label="Tracked" value={stats?.totalEvents ?? events.length} />
		<StatCard label="Delivered" value={stats?.deliverySuccessCount ?? 0} tone="success" />
		<StatCard label="Failed" value={stats?.deliveryErrorCount ?? 0} tone="destructive" />
		<StatCard label="Hit rate" value={`${successRate}%`} />
		<StatCard label="Recent (6h)" value={recentCount} live />
	</div>

	<ActivityChart {events} />
</section>

<section class="grid gap-5">
	<PageHeader eyebrow="Recent feed" title="Activity stream" />

	<div class="flex flex-wrap gap-2">
		<Input
			bind:value={search}
			placeholder="Search events…"
			oninput={onSearchInput}
			class="min-w-56 flex-1"
		/>
		<!-- The value is read off the event rather than bound, so the refetch
		     cannot race the binding's own change listener. -->
		<Select
			value={source}
			options={sourceOptions}
			class="w-44 flex-none"
			onchange={(event) => {
				source = (event.currentTarget as HTMLSelectElement).value;
				fetchEvents();
			}}
		/>
	</div>

	<EventTimeline {events} onreplay={replay} />

	{#if hasMore}
		<Button
			variant="outline"
			class="h-10 w-full border-dashed"
			disabled={loadingMore}
			onclick={() => fetchEvents({ append: true })}
		>
			{loadingMore ? 'Loading…' : 'Load more'}
		</Button>
	{/if}
</section>
