<script lang="ts">
	import { Badge, Button, StatusDot, icons } from '@facile/muse';
	import { formatUnit } from '$lib/i18n/format';
	import { t } from '$lib/i18n/index.svelte';
	import { formatDateTime, formatRelative } from '$lib/utils';
	import type { HealthSource, HealthState, TargetHealth } from '$lib/types';

	let {
		health,
		checking = false,
		cooldown = 0,
		onCheck
	}: {
		health?: TargetHealth;
		checking?: boolean;
		/** Seconds left on Antenne's own rate limit. Shared across targets. */
		cooldown?: number;
		onCheck: () => void;
	} = $props();

	/*
	 * Eight states onto six tones, so three decisions matter more than the rest:
	 *
	 *  - `unknown` is grey, never green. A generic webhook has no safe read-only
	 *    probe, so it never leaves this state on its own, and a green dot nobody
	 *    can justify is exactly the lie this feature exists to stop.
	 *  - `probe_error` is blue, not red. It means Antenne's own check could not
	 *    run; red would bill the target for our bug and send an operator to
	 *    rotate a credential that was never the problem.
	 *  - `recovering` is amber, never red. The target is answering; only the
	 *    dwell time is still running. Red next to a success the operator can
	 *    reproduce by pressing Check now is how a status dot loses its meaning.
	 */
	const TONES: Record<HealthState, 'neutral' | 'info' | 'success' | 'warning' | 'danger'> = {
		unknown: 'neutral',
		ok: 'success',
		pending: 'warning',
		down: 'danger',
		recovering: 'warning',
		flapping: 'warning',
		stale: 'warning',
		probe_error: 'info'
	};

	/* Only in-flight states pulse: something is still being decided. A steady
	   outage is not in flight, and a permanently pulsing dot stops meaning
	   anything. */
	const PULSING: HealthState[] = ['pending', 'recovering', 'flapping'];

	const sources = $derived<Record<Exclude<HealthSource, ''>, { label: string; title: string }>>({
		delivery: { label: t('health.sourceDelivery'), title: t('health.sourceDeliveryHelp') },
		probe: { label: t('health.sourceProbe'), title: t('health.sourceProbeHelp') },
		synthetic: { label: t('health.sourceSynthetic'), title: t('health.sourceSyntheticHelp') }
	});

	function formatDuration(seconds: number): string {
		if (seconds < 60) return formatUnit(Math.max(seconds, 1), 'second');
		const minutes = Math.floor(seconds / 60);
		if (minutes < 60) return formatUnit(minutes, 'minute');
		const hours = Math.floor(minutes / 60);
		if (hours < 24) return `${formatUnit(hours, 'hour')} ${formatUnit(minutes % 60, 'minute')}`;
		return `${formatUnit(Math.floor(hours / 24), 'day')} ${formatUnit(hours % 24, 'hour')}`;
	}

	function describe(current: TargetHealth): string {
		switch (current.state) {
			case 'ok':
				return t('health.ok');
			case 'pending':
				return t('health.pending');
			case 'down':
				return current.downtimeSeconds > 0
					? t('health.downFor', { duration: formatDuration(current.downtimeSeconds) })
					: t('health.down');
			case 'recovering':
				return current.downtimeSeconds > 0
					? t('health.recoveringAfter', { duration: formatDuration(current.downtimeSeconds) })
					: t('health.recovering');
			case 'flapping':
				return t('health.flapping');
			case 'stale':
				return t('health.stale');
			case 'probe_error':
				return t('health.probeError');
			default:
				return t('health.unknown');
		}
	}

	const state = $derived<HealthState>(health?.state ?? 'unknown');
	const source = $derived(health?.source ? sources[health.source] : null);
	const suspended = $derived(
		health?.probeSuspendedUntil && new Date(health.probeSuspendedUntil) > new Date()
			? health.probeSuspendedUntil
			: null
	);
</script>

<div class="flex flex-col gap-2">
	<div class="flex flex-wrap items-center gap-x-3 gap-y-2">
		<StatusDot
			tone={TONES[state]}
			label={health ? describe(health) : t('health.loading')}
			pulse={PULSING.includes(state)}
		/>

		{#if source}
			<Badge title={source.title}>{source.label}</Badge>
		{/if}

		<Button
			variant="secondary"
			size="sm"
			icon={icons.refresh}
			onclick={onCheck}
			disabled={checking || cooldown > 0}
			title={cooldown > 0
				? t('health.cooldownHelp')
				: t('health.checkHelp')}
			class="ml-auto"
		>
			{#if checking}
				{t('health.checking')}
			{:else if cooldown > 0}
				{t('health.retryIn', { seconds: cooldown })}
			{:else}
				{t('health.checkNow')}
			{/if}
		</Button>
	</div>

	{#if health?.detail}
		<!-- Verbatim, monospaced and select-all: this string is what an operator
		     pastes into a search or forwards to their provider. -->
		<p
			class="rounded-fc-md bg-fc-surface p-2 font-fc-mono text-fc-xs break-words text-fc-fg-muted select-all"
		>
			{health.detail}
		</p>
	{/if}

	{#if health}
		<div class="flex flex-wrap gap-x-4 text-fc-xs text-fc-fg-muted">
			{#if health.lastObservedAt}
				<span>{t('health.checked', { when: formatRelative(health.lastObservedAt) })}</span>
			{/if}
			{#if health.lastSuccessAt && health.state !== 'ok'}
				<span>{t('health.lastWorked', { when: formatRelative(health.lastSuccessAt) })}</span>
			{/if}
			{#if health.alertCount > 1}
				<span>{t('health.noticesSent', { count: health.alertCount })}</span>
			{/if}
		</div>
	{/if}

	{#if suspended}
		<!-- Probing a credential the far side has rejected outright is harmful:
		     Discord blocks an IP after enough 401s, and repeated SMTP auth
		     failures trip fail2ban and smart lockout. Say why it stopped. -->
		<p class="text-fc-xs text-fc-fg-muted">
			{t('health.paused', { when: formatDateTime(suspended) })}
		</p>
	{/if}
</div>
