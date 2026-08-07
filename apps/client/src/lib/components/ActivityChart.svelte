<script lang="ts">
	import { BarChart, Card, chartColor } from '@facile/muse';
	import type { EventEntry } from '$lib/types';

	let { events }: { events: EventEntry[] } = $props();

	type Bucket = { key: string; label: string; total: number; errors: number };

	/**
	 * 24 hourly buckets ending at the current hour. Bucketing by local hour and
	 * not by "last N minutes" is what makes the x labels readable clock times
	 * rather than arbitrary offsets.
	 */
	const buckets = $derived.by<Bucket[]>(() => {
		const now = new Date();
		now.setMinutes(0, 0, 0);

		const items: Bucket[] = Array.from({ length: 24 }, (_, index) => {
			const at = new Date(now);
			at.setHours(now.getHours() - (23 - index));
			return {
				key: at.toISOString().slice(0, 13),
				label: at.toLocaleTimeString([], { hour: '2-digit' }),
				total: 0,
				errors: 0
			};
		});

		const indexByHour = new Map(items.map((bucket, index) => [bucket.key, index]));

		for (const event of events) {
			const at = new Date(event.timestamp);
			if (Number.isNaN(at.getTime())) continue;
			at.setMinutes(0, 0, 0);

			const index = indexByHour.get(at.toISOString().slice(0, 13));
			if (index === undefined) continue;

			items[index].total += 1;
			if (event.deliveries.some((delivery) => delivery.status === 'error')) {
				items[index].errors += 1;
			}
		}

		return items;
	});

	const labels = $derived(buckets.map((bucket) => bucket.label));

	/*
	 * Stacked, with the clean count carved out of the total rather than the failures drawn
	 * on top of it: the bar height stays the hour's event count and the two segments still
	 * sum to it, so the tooltip cannot claim more events than happened.
	 *
	 * The failure segment is `chartColor(1)`, not `fc-danger` — the danger token is reserved
	 * for status and never carries a series.
	 */
	const series = $derived([
		{
			name: 'Delivered',
			data: buckets.map((bucket) => bucket.total - bucket.errors),
			color: chartColor(0)
		},
		{ name: 'With errors', data: buckets.map((bucket) => bucket.errors), color: chartColor(1) }
	]);

	const peak = $derived(
		buckets.reduce((best, bucket) => (bucket.total > best.total ? bucket : best), buckets[0])
	);
	const recent = $derived(buckets.slice(-6).reduce((sum, bucket) => sum + bucket.total, 0));
</script>

<Card class="flex h-full flex-col gap-4">
	<header class="flex flex-wrap items-start justify-between gap-4">
		<div class="flex flex-col gap-1">
			<p class="text-fc-xs text-fc-fg-muted">Last 24 hours</p>
			<h3 class="text-fc-lg font-semibold text-fc-fg">Activity</h3>
		</div>
		<div class="flex flex-col gap-1 text-right text-fc-xs text-fc-fg-muted tabular-nums">
			<span>{recent} {recent === 1 ? 'event' : 'events'} in 6h</span>
			<span>Peak {peak?.total ?? 0} at {peak?.label ?? '—'}</span>
		</div>
	</header>

	<BarChart
		{series}
		{labels}
		stacked
		height={200}
		yTicks={3}
		emptyLabel="Nothing in the last 24 hours"
		class="flex-1"
	/>
</Card>
