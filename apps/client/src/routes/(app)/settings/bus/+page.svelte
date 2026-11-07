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
	import { t } from '$lib/i18n/index.svelte';
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
		dashboard.settings.bus.secret ? t('bus.secretDashboard') : t('bus.secretFallback')
	);

	/*
	 * "Not connected" hides three situations with three different fixes, so the panel reports
	 * the poll's own state as well as the pool's: still probing, live, live but idle, retrying
	 * with a count, or given up. Only the in-flight states pulse. CHARTE §14.
	 */
	const status = $derived.by(() => {
		if (failures >= GIVE_UP) {
			return { tone: 'danger', label: t('bus.unreachable', { count: failures }), pulse: false } as const;
		}
		if (failures > 0) {
			return { tone: 'warning', label: t('bus.reconnecting', { count: failures, max: GIVE_UP }), pulse: true } as const;
		}
		if (!stats) return { tone: 'info', label: t('bus.connecting'), pulse: true } as const;
		if (stats.connections === 0) {
			return { tone: 'neutral', label: t('bus.listening'), pulse: false } as const;
		}
		return { tone: 'success', label: t('bus.connected', { count: stats.connections }), pulse: false } as const;
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
			lastError = error instanceof Error ? error.message : t('bus.statsFailed');
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
		title={t('bus.registration')}
		description={t('bus.registrationText')}
	>
		<SettingsRow
			label={t('bus.secret')}
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
				onchange={() => dashboard.autoSave(0)}
			/>
		</SettingsRow>

		<SettingsRow
			label={t('bus.logEvents')}
			description={t('bus.logEventsText')}
		>
			<!-- Switch only renders the label it is handed, so the accessible name has to come
			     through aria-label; the visible one is the row's. -->
			<Switch
				aria-label={t('bus.logEvents')}
				bind:checked={dashboard.settings.bus.logActivity}
				onchange={() => dashboard.autoSave(0)}
			/>
		</SettingsRow>
	</SettingsSection>

	<SettingsSection title={t('bus.connection')} description={t('bus.connectionText')}>
		{#snippet actions()}
			<Button variant="secondary" size="sm" icon={icons.refresh} onclick={retry}>{t('activity.refresh')}</Button>
		{/snippet}

		<SettingsRow label={t('bus.state')}>
			<StatusDot tone={status.tone} label={status.label} pulse={status.pulse} />
		</SettingsRow>

		<SettingsRow label={t('bus.epoch')} description={t('bus.epochText')}>
			<span class="truncate font-fc-mono text-fc-sm text-fc-fg" title={stats?.epoch}>
				{stats ? stats.epoch : '—'}
			</span>
		</SettingsRow>

		<SettingsRow label={t('bus.liveConnections')}>
			<span class="text-fc-sm font-medium text-fc-fg tabular-nums">{stats?.connections ?? '—'}</span>
		</SettingsRow>

		<SettingsRow label={t('bus.lastPoll')}>
			<span class="text-fc-sm text-fc-fg">{lastSeen ? formatRelative(lastSeen) : '—'}</span>
		</SettingsRow>

		{#if lastError}
			<SettingsRow label={t('bus.lastError')}>
				<span class="font-fc-mono text-fc-sm text-fc-danger">{lastError}</span>
			</SettingsRow>
		{/if}
	</SettingsSection>

	<SettingsSection
		title={t('bus.apps')}
		description={t('bus.appsText')}
		bare={!stats || stats.apps.length === 0}
	>
		{#if !stats || stats.apps.length === 0}
			<EmptyState
				icon={icons.plug}
				title={t('bus.noApp')}
				description={t('bus.noAppText')}
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
								<span class="text-fc-xs text-fc-fg-muted">{t('bus.emitsOnly')}</span>
							{:else if app.channels.length === 0}
								<span class="text-fc-xs text-fc-fg-muted">{t('bus.noChannel')}</span>
							{/if}
						</div>
					</div>
				</SettingsRow>
			{/each}
		{/if}
	</SettingsSection>
</div>
