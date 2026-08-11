<script lang="ts">
	import { Alert } from '@facile/muse';
	import { formatRelative } from '$lib/utils';
	import type { StandingAlert } from '$lib/types';

	let { standingAlerts }: { standingAlerts: StandingAlert[] } = $props();
</script>

<Alert tone="danger" title="Nothing can be delivered right now">
	<p>
		Every enabled target is down, so Antenne cannot raise an alert through any channel it has —
		including the one that would tell you about this. Until one target recovers, this page is
		the only place a failure will appear.
	</p>
	{#if standingAlerts.length > 0}
		<p class="mt-2">
			{standingAlerts.length}
			{standingAlerts.length === 1 ? 'notice' : 'notices'} could not be carried anywhere:
		</p>
		<ul class="mt-1 list-disc pl-5">
			{#each standingAlerts as alert (alert.targetId)}
				<li>{alert.title} — since {formatRelative(alert.since)}</li>
			{/each}
		</ul>
	{/if}
</Alert>
