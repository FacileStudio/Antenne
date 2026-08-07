<script lang="ts">
	import { Alert, SettingsRow, SettingsSection, Switch } from '@facile/muse';
	import { dashboard } from '$lib/dashboard.svelte';
	import { SYSTEM_PROVIDER_ID, type SystemEvents } from '$lib/types';

	const events = $derived(dashboard.settings.system.events);

	/** `key` is the wire name — a Go json tag under `system.events`. Do not rename one. */
	const rows: { key: keyof SystemEvents; label: string }[] = [
		{ key: 'startup', label: 'Antenne started' },
		{ key: 'providerEnabled', label: 'A provider was enabled' },
		{ key: 'providerDisabled', label: 'A provider was disabled' },
		{ key: 'providerAdded', label: 'A provider was added' },
		{ key: 'providerRemoved', label: 'A provider was removed' },
		{ key: 'deliveryFailure', label: 'A delivery target failed' }
	];
</script>

<div class="flex flex-col gap-10">
	<SettingsSection
		title="System events"
		description="The events Antenne emits about itself, on top of the ones its providers report."
	>
		{#each rows as row (row.key)}
			<!-- The description is the wire key, so the row's own paragraph is set in mono
			     rather than dropping a second element beside it. -->
			<SettingsRow
				label={row.label}
				description={`system.events.${row.key}`}
				class="[&_p]:font-fc-mono"
			>
				<Switch aria-label={row.label} bind:checked={events[row.key]} />
			</SettingsRow>
		{/each}
	</SettingsSection>

	<Alert tone="info" title="Enabling an event is not enough on its own">
		Delivery routing is opt-in with no fallthrough: a system event only reaches a target that
		names the reserved provider id
		<code class="font-fc-mono">{SYSTEM_PROVIDER_ID}</code> in its provider filter. A target
		filtered on tags alone never sees one.
	</Alert>
</div>
