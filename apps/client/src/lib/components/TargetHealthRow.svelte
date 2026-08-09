<script lang="ts">
	import { Badge, Button, StatusDot, icons } from '@facile/muse';
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

	const SOURCES: Record<Exclude<HealthSource, ''>, { label: string; title: string }> = {
		delivery: {
			label: 'real delivery',
			title: 'Measured by an actual send. The most authoritative signal there is.'
		},
		probe: {
			label: 'probe',
			title: 'Measured by a read-only check, not by delivering anything.'
		},
		synthetic: {
			label: 'synthetic send',
			title: 'Measured by a real message sent to the destination you nominated.'
		}
	};

	function formatDuration(seconds: number): string {
		if (seconds < 60) return `${Math.max(seconds, 1)}s`;
		const minutes = Math.floor(seconds / 60);
		if (minutes < 60) return `${minutes}m`;
		const hours = Math.floor(minutes / 60);
		if (hours < 24) return `${hours}h ${minutes % 60}m`;
		return `${Math.floor(hours / 24)}d ${hours % 24}h`;
	}

	function describe(current: TargetHealth): string {
		switch (current.state) {
			case 'ok':
				return 'Reachable';
			case 'pending':
				return 'Failing — not confirmed yet';
			case 'down':
				return current.downtimeSeconds > 0
					? `Down for ${formatDuration(current.downtimeSeconds)}`
					: 'Down';
			case 'recovering':
				return current.downtimeSeconds > 0
					? `Recovering — answering again after ${formatDuration(current.downtimeSeconds)}`
					: 'Recovering — answering again';
			case 'flapping':
				return 'Unstable — breaking and recovering';
			case 'stale':
				return 'Stale — no recent observation';
			case 'probe_error':
				return "Probe failed — Antenne's side, not the target's";
			default:
				return 'Never measured';
		}
	}

	const state = $derived<HealthState>(health?.state ?? 'unknown');
	const source = $derived(health?.source ? SOURCES[health.source] : null);
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
			label={health ? describe(health) : 'Loading…'}
			pulse={PULSING.includes(state)}
		/>

		{#if source}
			<Badge title={source.title}>{source.label}</Badge>
		{/if}

		<Button
			variant="outline"
			size="sm"
			icon={icons.refresh}
			onclick={onCheck}
			disabled={checking || cooldown > 0}
			title={cooldown > 0
				? "Antenne's own limit on checks, not a refusal from this target."
				: 'Probe this target right now'}
			class="ml-auto"
		>
			{#if checking}
				Checking…
			{:else if cooldown > 0}
				Retry in {cooldown}s
			{:else}
				Check now
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
				<span>Checked {formatRelative(health.lastObservedAt)}</span>
			{/if}
			{#if health.lastSuccessAt && health.state !== 'ok'}
				<span>Last worked {formatRelative(health.lastSuccessAt)}</span>
			{/if}
			{#if health.alertCount > 1}
				<span>{health.alertCount} notices sent</span>
			{/if}
		</div>
	{/if}

	{#if suspended}
		<!-- Probing a credential the far side has rejected outright is harmful:
		     Discord blocks an IP after enough 401s, and repeated SMTP auth
		     failures trip fail2ban and smart lockout. Say why it stopped. -->
		<p class="text-fc-xs text-fc-fg-muted">
			Probing paused until {formatDateTime(suspended)} — the credential was rejected outright, so
			retrying it can only get Antenne blocked.
		</p>
	{/if}
</div>
