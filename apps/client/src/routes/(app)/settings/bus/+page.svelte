<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Badge,
		Button,
		EmptyState,
		SecretField,
		SettingsRow,
		SettingsSection,
		StatusDot,
		Switch,
		icons
	} from '@facile/muse';
	import { api } from '$lib/backend';
	import { dashboard } from '$lib/dashboard.svelte';
	import { appIcon } from '$lib/icons';
	import type { BusStats } from '$lib/types';
	import { formatRelative } from '$lib/utils';

	/** Polls before the panel stops trying on its own; the operator can restart it by hand. */
	const GIVE_UP = 5;

	let stats = $state<BusStats | null>(null);
	let failures = $state(0);
	let lastError = $state<string | null>(null);
	let lastSeen = $state<string | null>(null);

	const secretSource = $derived(
		dashboard.settings.bus.secret ? 'Set from the dashboard' : 'Falling back to ANTENNE_SECRET or "pool"'
	);

	/*
	 * "Not connected" hides three situations with three different fixes, so the panel reports
	 * the poll's own state as well as the pool's: still probing, live, live but idle, retrying
	 * with a count, or given up. Only the in-flight states pulse. CHARTE §14.
	 */
	const status = $derived.by(() => {
		if (failures >= GIVE_UP) {
			return { tone: 'danger', label: `Unreachable — gave up after ${failures} attempts`, pulse: false } as const;
		}
		if (failures > 0) {
			return { tone: 'warning', label: `Reconnecting — attempt ${failures} of ${GIVE_UP}`, pulse: true } as const;
		}
		if (!stats) return { tone: 'info', label: 'Connecting…', pulse: true } as const;
		if (stats.connections === 0) {
			return { tone: 'neutral', label: 'Listening — no app connected', pulse: false } as const;
		}
		const plural = stats.connections === 1 ? '' : 's';
		return { tone: 'success', label: `Connected — ${stats.connections} live socket${plural}`, pulse: false } as const;
	});

	async function loadStats() {
		try {
			stats = await api.busStats();
			failures = 0;
			lastError = null;
			lastSeen = new Date().toISOString();
		} catch (error) {
			// The panel is informational and polls on a ten-second timer; a failure belongs in
			// the status line, never in a toast that reappears forever.
			failures += 1;
			lastError = error instanceof Error ? error.message : 'The stats endpoint did not answer.';
		}
	}

	function retry() {
		failures = 0;
		lastError = null;
		void loadStats();
	}

	onMount(() => {
		void loadStats();
		const timer = setInterval(() => {
			if (failures < GIVE_UP) void loadStats();
		}, 10_000);
		return () => clearInterval(timer);
	});
</script>

<div class="flex flex-col gap-10">
	<SettingsSection
		title="Registration"
		description="The shared secret a pool client presents at /pool/register."
	>
		<SettingsRow
			label="Registration secret"
			description={secretSource}
			for="bus-secret"
			stacked
		>
			<SecretField
				bind:value={dashboard.settings.bus.secret}
				for="bus-secret"
				editable
				placeholder="pool"
				class="w-full"
			/>
		</SettingsRow>

		<SettingsRow
			label="Log pool events to Activity"
			description="Connections, disconnections and relayed events appear in the activity feed."
		>
			<!-- Switch only renders the label it is handed, so the accessible name has to come
			     through aria-label; the visible one is the row's. -->
			<Switch
				aria-label="Log pool events to Activity"
				bind:checked={dashboard.settings.bus.logActivity}
			/>
		</SettingsRow>
	</SettingsSection>

	<SettingsSection title="Connection" description="What the pool reports right now.">
		{#snippet actions()}
			<Button variant="outline" size="sm" icon={icons.refresh} onclick={retry}>Refresh</Button>
		{/snippet}

		<SettingsRow label="State">
			<StatusDot tone={status.tone} label={status.label} pulse={status.pulse} />
		</SettingsRow>

		<SettingsRow label="Epoch" description="Restarts mint a new one; clients resync when it changes.">
			<span class="truncate font-fc-mono text-fc-sm text-fc-fg" title={stats?.epoch}>
				{stats ? stats.epoch : '—'}
			</span>
		</SettingsRow>

		<SettingsRow label="Live connections">
			<span class="text-fc-sm font-medium text-fc-fg tabular-nums">{stats?.connections ?? '—'}</span>
		</SettingsRow>

		<SettingsRow label="Last successful poll">
			<span class="text-fc-sm text-fc-fg">{lastSeen ? formatRelative(lastSeen) : '—'}</span>
		</SettingsRow>

		{#if lastError}
			<SettingsRow label="Last error">
				<span class="font-fc-mono text-fc-sm text-fc-danger">{lastError}</span>
			</SettingsRow>
		{/if}
	</SettingsSection>

	<SettingsSection
		title="Connected apps"
		description="Suite apps holding a socket, with the channels each subscribes to. An app that only produces events shows as emits only."
		bare={!stats || stats.apps.length === 0}
	>
		{#if !stats || stats.apps.length === 0}
			<EmptyState
				icon={icons.plug}
				title="No app is connected"
				description="An app appears here once it registers with the secret above and opens /pool/ws."
			/>
		{:else}
			{#each stats.apps as app (app.app_id)}
				<!-- `stacked` gives the row its full width so the identity can sit left of the
				     channels instead of being squeezed into the control column. -->
				<SettingsRow stacked>
					<div class="flex w-full flex-wrap items-center justify-between gap-3">
						<div class="flex min-w-0 items-center gap-2.5">
							<StatusDot tone="success" />
							<iconify-icon icon={appIcon(app.app)} width="18" height="18" class="block"
							></iconify-icon>
							<div class="flex min-w-0 flex-col">
								<span class="truncate text-fc-sm font-semibold text-fc-fg">{app.app}</span>
								<span class="truncate font-fc-mono text-fc-xs text-fc-fg-muted">{app.app_id}</span>
							</div>
						</div>
						<!-- Subscriptions first, then what the app produces. An
						     emit-only client subscribes to nothing on purpose, and
						     showing only subscriptions made that healthy state look
						     like a fault. -->
						<div class="flex flex-wrap items-center gap-1.5">
							{#each app.channels as channel (channel)}
								<Badge class="font-fc-mono">{channel}</Badge>
							{/each}
							{#if app.channels.length === 0 && app.emits.length > 0}
								<span class="text-fc-xs text-fc-fg-muted">emits only</span>
							{:else if app.channels.length === 0}
								<span class="text-fc-xs text-fc-fg-muted">No channel</span>
							{/if}
						</div>
					</div>
				</SettingsRow>
			{/each}
		{/if}
	</SettingsSection>
</div>
