<script lang="ts">
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

	const max = $derived(Math.max(...buckets.map((bucket) => bucket.total), 1));
	const half = $derived(Math.ceil(max / 2));
	const peak = $derived(
		buckets.reduce((best, bucket) => (bucket.total > best.total ? bucket : best), buckets[0])
	);
	const recent = $derived(buckets.slice(-6).reduce((sum, bucket) => sum + bucket.total, 0));

	const axisLabels = $derived([0, 6, 12, 18, 23].map((index) => buckets[index]?.label ?? ''));

	function heightOf(count: number): string {
		return `${(count / max) * 100}%`;
	}
</script>

<section class="grid gap-4 rounded-xl border border-border bg-card p-5">
	<header class="flex flex-wrap items-start justify-between gap-3">
		<div>
			<p class="text-xs text-muted-foreground">Last 24 hours</p>
			<h3 class="font-heading text-base font-semibold">Activity</h3>
		</div>
		<div class="grid gap-0.5 text-right text-xs text-muted-foreground tabular">
			<span>{recent} {recent === 1 ? 'event' : 'events'} in 6h</span>
			<span>Peak {peak?.total ?? 0} at {peak?.label ?? '--'}</span>
		</div>
	</header>

	<div class="relative pt-4">
		<!-- Two recessive gridlines rather than a full grid: they give the eye a
		     scale without competing with the bars. -->
		<div class="pointer-events-none absolute inset-x-0 top-4 border-t border-dashed border-border">
			<span class="absolute right-0 -top-4 text-[0.65rem] text-muted-foreground tabular">{max}</span>
		</div>
		{#if half < max}
			<div
				class="pointer-events-none absolute inset-x-0 border-t border-dashed border-border/60"
				style="top: calc(1rem + (100% - 1rem) / 2)"
			>
				<span class="absolute right-0 -top-4 text-[0.65rem] text-muted-foreground tabular">
					{half}
				</span>
			</div>
		{/if}

		<div class="grid h-40 grid-cols-24 items-end gap-1 border-b border-border">
			{#each buckets as bucket (bucket.key)}
				<div class="group relative flex h-full items-end justify-center">
					{#if bucket.total > 0}
						<div
							class="relative w-full overflow-hidden rounded-t-[4px] bg-foreground/80 transition-colors group-hover:bg-foreground"
							style="height: {heightOf(bucket.total)}; min-height: 3px"
						>
							{#if bucket.errors > 0}
								<!-- The failure share is anchored to the baseline with a 2px
								     surface gap, so the two segments stay legible where the
								     bar is only a few pixels tall. -->
								<div
									class="absolute inset-x-0 bottom-0 border-t-2 border-card bg-destructive"
									style="height: {(bucket.errors / bucket.total) * 100}%; min-height: 3px"
								></div>
							{/if}
						</div>
					{/if}

					<div
						role="tooltip"
						class="pointer-events-none absolute bottom-[calc(100%+6px)] left-1/2 z-10 hidden -translate-x-1/2 flex-col gap-0.5 rounded-md border border-border bg-popover px-2 py-1.5 text-[0.7rem] whitespace-nowrap text-popover-foreground shadow-md group-hover:flex"
					>
						<span class="text-muted-foreground">{bucket.label}</span>
						<span class="tabular">{bucket.total} {bucket.total === 1 ? 'event' : 'events'}</span>
						{#if bucket.errors > 0}
							<span class="text-destructive tabular">{bucket.errors} with errors</span>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	</div>

	<div class="flex justify-between text-[0.7rem] text-muted-foreground tabular">
		{#each axisLabels as label, index (index)}
			<span>{label}</span>
		{/each}
	</div>

	<div class="flex items-center gap-4 text-xs text-muted-foreground">
		<span class="inline-flex items-center gap-1.5">
			<span class="size-2.5 rounded-[2px] bg-foreground/80"></span>
			Events
		</span>
		<span class="inline-flex items-center gap-1.5">
			<span class="size-2.5 rounded-[2px] bg-destructive"></span>
			With errors
		</span>
	</div>
</section>

<style>
	/* Tailwind has no 24-column preset and the bucket count is fixed by the
	   24-hour window, so the grid is declared once here. */
	.grid-cols-24 {
		grid-template-columns: repeat(24, minmax(0, 1fr));
	}
</style>
