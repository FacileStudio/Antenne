<script lang="ts">
	import { Card, LineChart, chartColor } from '@facile/muse';
	import { hourlyBuckets } from '$lib/activity';
	import { t } from '$lib/i18n/index.svelte';
	import type { EventEntry } from '$lib/types';

	let { events }: { events: EventEntry[] } = $props();

	const buckets = $derived(hourlyBuckets(events));
	const labels = $derived(buckets.map((bucket) => bucket.label));

	/*
	 * Two overlapping area series rather than a stack: "Delivered" is the hour's clean
	 * count and "With errors" is drawn at its own height, so the two curves read as
	 * independent trends the way the reference dashboard's Billable/Internal pair does.
	 *
	 * The failure series is `chartColor(1)`, not `fc-danger` — the danger token is reserved
	 * for status and never carries a series.
	 */
	const series = $derived([
		{
			name: t('chart.delivered'),
			data: buckets.map((bucket) => bucket.total - bucket.errors),
			color: chartColor(0)
		},
		{ name: t('chart.withErrors'), data: buckets.map((bucket) => bucket.errors), color: chartColor(1) }
	]);

	const peak = $derived(
		buckets.reduce((best, bucket) => (bucket.total > best.total ? bucket : best), buckets[0])
	);
	const recent = $derived(buckets.slice(-6).reduce((sum, bucket) => sum + bucket.total, 0));
</script>

<Card class="flex h-full flex-col gap-4">
	<header class="flex flex-wrap items-start justify-between gap-4">
		<div class="flex flex-col gap-1">
			<p class="text-fc-xs text-fc-fg-muted">{t('chart.last24')}</p>
			<h3 class="text-fc-lg font-semibold text-fc-fg">{t('activity.title')}</h3>
		</div>
		<div class="flex flex-col gap-1 text-right text-fc-xs text-fc-fg-muted tabular-nums">
			<span>{t('chart.recent', { count: recent })}</span>
			<span>{t('chart.peak', { count: peak?.total ?? 0, hour: peak?.label ?? '—' })}</span>
		</div>
	</header>

	<LineChart
		{series}
		{labels}
		area
		smooth
		height={200}
		yTicks={3}
		emptyLabel={t('chart.empty')}
		class="flex-1"
	/>
</Card>
