<script lang="ts">
	import { Alert, Button, SettingsRow, SettingsSection, Switch, icons } from '@facile/muse';
	import { dashboard } from '$lib/dashboard.svelte';
	import { t } from '$lib/i18n/index.svelte';
	import { SYSTEM_PROVIDER_ID, type SystemEvents } from '$lib/types';

	const events = $derived(dashboard.settings.system.events);

	/** `key` is the wire name — a Go json tag under `system.events`. Do not rename one. */
	const rows: { key: keyof SystemEvents; label: string }[] = $derived([
		{ key: 'startup', label: t('sysevents.startup') },
		{ key: 'providerEnabled', label: t('sysevents.providerEnabled') },
		{ key: 'providerDisabled', label: t('sysevents.providerDisabled') },
		{ key: 'providerAdded', label: t('sysevents.providerAdded') },
		{ key: 'providerRemoved', label: t('sysevents.providerRemoved') },
		{ key: 'deliveryFailure', label: t('sysevents.deliveryFailure') }
	]);
</script>

<div class="flex flex-col gap-10">
	<SettingsSection
		title={t('sysevents.title')}
		description={t('sysevents.text')}
	>
		{#snippet actions()}
			<Button
				variant="outline"
				size="sm"
				icon={icons.bolt}
				onclick={() => dashboard.sendTestAlert()}
				disabled={dashboard.testing || dashboard.loading}
			>
				{dashboard.testing ? t('delivery.sending') : t('sysevents.sendTest')}
			</Button>
		{/snippet}

		{#each rows as row (row.key)}
			<!-- The description is the wire key, so the row's own paragraph is set in mono
			     rather than dropping a second element beside it. -->
			<SettingsRow
				label={row.label}
				description={`system.events.${row.key}`}
				class="[&_p]:font-fc-mono"
			>
				<Switch
					aria-label={row.label}
					bind:checked={events[row.key]}
					onchange={() => dashboard.autoSave(0)}
				/>
			</SettingsRow>
		{/each}
	</SettingsSection>

	<Alert tone="info" title={t('sysevents.alertTitle')}>
		{t('sysevents.alertText', { id: SYSTEM_PROVIDER_ID })}
	</Alert>
</div>
