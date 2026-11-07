<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Badge,
		Button,
		Input,
		Select,
		Sparkline,
		Spinner,
		StatCard,
		chartColor,
		icons,
		toast
	} from '@facile/muse';
	import ActivityChart from '$lib/components/ActivityChart.svelte';
	import EventTimeline from '$lib/components/EventTimeline.svelte';
	import { api } from '$lib/backend';
	import { dashboard } from '$lib/dashboard.svelte';
	import { formatNumber } from '$lib/i18n/format';
	import { t } from '$lib/i18n/index.svelte';
	import { hourlyBuckets } from '$lib/activity';
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

	// One shared set of hourly buckets feeds every stat card sparkline, so each trend
	// reads off the same 24h window as the chart below it.
	const buckets = $derived(hourlyBuckets(events));
	const trackedTrend = $derived(buckets.map((bucket) => bucket.total));
	const deliveredTrend = $derived(buckets.map((bucket) => bucket.total - bucket.errors));
	const failedTrend = $derived(buckets.map((bucket) => bucket.errors));
	const hitRateTrend = $derived(
		buckets.map((bucket) => (bucket.total === 0 ? 100 : Math.round(((bucket.total - bucket.errors) / bucket.total) * 100)))
	);
	const recentTrend = $derived(buckets.slice(-6).map((bucket) => bucket.total));

	const sourceOptions = $derived([
		{ value: '', label: t('activity.allSources') },
		{ value: 'webhook', label: t('source.webhook') },
		{ value: 'website', label: t('source.website') },
		{ value: 'imap', label: t('source.imap') },
		{ value: 'rss', label: t('source.rss') },
		{ value: 'system', label: t('source.system') },
		{ value: 'test', label: t('source.test') }
	]);

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
			toast.danger(error instanceof Error ? error.message : t('activity.loadFailed'));
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
			toast.danger(t('activity.noTarget'));
			return;
		}

		try {
			await api.replay(eventId, target.id);
			toast.success(t('activity.replayed', { name: target.name }));
		} catch (error) {
			toast.danger(error instanceof Error ? error.message : t('activity.replayFailed'));
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

<div class="flex flex-col gap-10">
	<div class="flex flex-wrap items-start justify-between gap-4">
		<div class="flex flex-col gap-2">
			<h1 class="text-fc-2xl font-semibold text-fc-fg">{t('activity.title')}</h1>
			<p class="text-fc-sm text-fc-fg-muted">
				{t('activity.subtitle')}
			</p>
		</div>

		<div class="flex flex-wrap items-center gap-2">
			<Badge>
				<iconify-icon icon={icons.clock} width="13" height="13" class="block"></iconify-icon>
				{stats?.lastEventAt ? formatDateTime(stats.lastEventAt) : t('activity.noEventsYet')}
			</Badge>
			<Button
				variant="secondary"
				icon={icons.refresh}
				onclick={() => fetchEvents()}
				disabled={loading}
			>
				{loading ? t('activity.refreshing') : t('activity.refresh')}
			</Button>
		</div>
	</div>

	<section class="flex flex-col gap-4">
		<div class="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-5">
			<StatCard label={t('activity.tracked')} value={stats?.totalEvents ?? events.length}>
				<Sparkline data={trackedTrend} color={chartColor(0)} class="mt-2" />
			</StatCard>
			<StatCard label={t('providers.delivered')} value={stats?.deliverySuccessCount ?? 0}>
				<Sparkline data={deliveredTrend} color={chartColor(2)} class="mt-2" />
			</StatCard>
			<StatCard label={t('providers.failed')} value={stats?.deliveryErrorCount ?? 0}>
				<Sparkline data={failedTrend} color={chartColor(1)} class="mt-2" />
			</StatCard>
			<StatCard label={t('activity.hitRate')} value={formatNumber(successRate / 100, { style: 'percent' })}>
				<Sparkline data={hitRateTrend} color={chartColor(2)} class="mt-2" />
			</StatCard>
			<StatCard label={t('activity.recent')} value={recentCount}>
				<Sparkline data={recentTrend} color={chartColor(0)} class="mt-2" />
			</StatCard>
		</div>

		<ActivityChart {events} />
	</section>

	<section class="flex flex-col gap-4">
		<div class="flex flex-col gap-2">
			<h2 class="text-fc-lg font-semibold text-fc-fg">{t('activity.stream')}</h2>
			<p class="text-fc-sm text-fc-fg-muted">{t('activity.live')}</p>
		</div>

		<div class="flex flex-wrap gap-4">
			<Input
				bind:value={search}
				placeholder={t('activity.search')}
				oninput={onSearchInput}
				class="min-w-56 flex-1"
			/>
			<!-- The value is read off the event rather than bound, so the refetch
			     cannot race the binding's own change listener. -->
			<Select
				value={source}
				class="w-44 flex-none"
				aria-label={t('activity.filterSource')}
				onchange={(event) => {
					source = event.currentTarget.value;
					fetchEvents();
				}}
			>
				{#each sourceOptions as option (option.value)}
					<option value={option.value}>{option.label}</option>
				{/each}
			</Select>
		</div>

		{#if loading && events.length === 0}
			<div class="flex justify-center py-16">
				<Spinner />
			</div>
		{:else}
			<EventTimeline {events} onreplay={replay} />
		{/if}

		{#if hasMore}
			<Button
				variant="secondary"
				class="w-full"
				disabled={loadingMore}
				onclick={() => fetchEvents({ append: true })}
			>
				{loadingMore ? t('common.loading') : t('activity.loadMore')}
			</Button>
		{/if}
	</section>
</div>
