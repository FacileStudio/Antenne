<script lang="ts">
	import { Alert } from '@facile/muse';
	import { t } from '$lib/i18n/index.svelte';
	import { formatRelative } from '$lib/utils';
	import type { StandingAlert } from '$lib/types';

	let { standingAlerts }: { standingAlerts: StandingAlert[] } = $props();
</script>

<Alert tone="danger" title={t('outage.title')}>
	<p>
		{t('outage.text')}
	</p>
	{#if standingAlerts.length > 0}
		<p class="mt-2">
			{t('outage.notices', { count: standingAlerts.length })}
		</p>
		<ul class="mt-1 list-disc pl-5">
			{#each standingAlerts as alert (alert.targetId)}
				<li>{t('outage.since', { title: alert.title, when: formatRelative(alert.since) })}</li>
			{/each}
		</ul>
	{/if}
</Alert>
