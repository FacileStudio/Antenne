import type { EventEntry } from './types';

export type HourBucket = { key: string; label: string; total: number; errors: number };

/**
 * 24 hourly buckets ending at the current hour. Bucketing by local hour and not by
 * "last N minutes" is what makes the x labels readable clock times rather than
 * arbitrary offsets. Shared by the activity chart and the stat card sparklines so
 * both read the same trend.
 */
export function hourlyBuckets(events: EventEntry[], hours = 24): HourBucket[] {
	const now = new Date();
	now.setMinutes(0, 0, 0);

	const items: HourBucket[] = Array.from({ length: hours }, (_, index) => {
		const at = new Date(now);
		at.setHours(now.getHours() - (hours - 1 - index));
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
}
