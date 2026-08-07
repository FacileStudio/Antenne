<script lang="ts">
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';
	import Icon from '$lib/components/Icon.svelte';
	import AntenneLogo from '$lib/components/AntenneLogo.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import SecretInput from '$lib/components/SecretInput.svelte';
	import ToggleRow from '$lib/components/ToggleRow.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Field from '$lib/components/ui/field.svelte';
	import Input from '$lib/components/ui/input.svelte';
	import { api } from '$lib/backend';
	import { dashboard } from '$lib/dashboard.svelte';
	import { appIcon } from '$lib/icons';
	import type { PoolStats } from '$lib/types';
	import { cn, formatDateTime } from '$lib/utils';

	type Tab = 'general' | 'pool' | 'events' | 'advanced';

	let tab = $state<Tab>('general');
	let poolStats = $state<PoolStats | null>(null);
	let importInput = $state<HTMLInputElement | null>(null);

	const tabs: { key: Tab; label: string; icon: string | null }[] = [
		{ key: 'general', label: 'General', icon: 'solar:settings-linear' },
		{ key: 'pool', label: 'Pool', icon: null },
		{ key: 'events', label: 'Events', icon: 'solar:bell-linear' },
		{ key: 'advanced', label: 'Advanced', icon: 'solar:tuning-2-linear' }
	];

	const events = $derived(dashboard.settings.system.events);
	const runtime = $derived(dashboard.runtime);

	const poolSecretSource = $derived(
		dashboard.settings.bus.secret ? 'Dashboard config' : 'Environment variable or default'
	);

	async function loadPoolStats() {
		try {
			poolStats = await api.poolStats();
		} catch {
			// The pool panel is informational; a failure here must not toast on a
			// ten-second timer.
		}
	}

	async function handleImport(event: Event) {
		const file = (event.currentTarget as HTMLInputElement).files?.[0];
		if (!file) return;
		await dashboard.importConfig(file);
		if (importInput) importInput.value = '';
	}

	onMount(() => {
		void loadPoolStats();
		const timer = setInterval(loadPoolStats, 10_000);
		return () => clearInterval(timer);
	});
</script>

<div class="grid gap-6">
	<nav class="flex flex-wrap items-center gap-1.5 border-b border-border pb-4" aria-label="Settings sections">
		{#each tabs as entry (entry.key)}
			<button
				type="button"
				onclick={() => (tab = entry.key)}
				aria-current={tab === entry.key ? 'page' : undefined}
				class={cn(
					'inline-flex h-9 items-center gap-2 rounded-md px-4 text-sm font-medium transition-colors',
					tab === entry.key
						? 'bg-foreground text-background'
						: 'text-muted-foreground hover:bg-muted hover:text-foreground'
				)}
			>
				{#if entry.icon}
					<Icon icon={entry.icon} size={16} />
				{:else}
					<AntenneLogo size={16} />
				{/if}
				{entry.label}
			</button>
		{/each}
	</nav>

	{#if tab === 'general'}
		<section class="grid gap-5">
			<PageHeader eyebrow="Settings" title="System" />

			<div class="grid max-w-2xl gap-4 sm:grid-cols-2">
				<Field label="Request timeout" hint="In seconds. Applies to website checks.">
					<Input type="number" min="1" step="1" bind:value={dashboard.settings.timeout} />
				</Field>

				<Field
					label="Listener port"
					hint="Takes effect on the next restart. The published webhook URLs point at it, so the deployment has to agree."
				>
					<Input type="number" min="1" step="1" bind:value={dashboard.settings.providers.port} />
				</Field>
			</div>
		</section>
	{:else if tab === 'pool'}
		<section class="grid gap-5">
			<PageHeader eyebrow="Pool" title="Secret" />

			<div class="grid max-w-2xl gap-2">
				<Field label="Registration secret" hint={`Source: ${poolSecretSource}`}>
					<SecretInput bind:value={dashboard.settings.bus.secret} placeholder="bus" />
				</Field>
			</div>

			<PageHeader eyebrow="Pool" title="Activity log" />
			<div class="max-w-2xl">
				<ToggleRow
					label="Log pool events to the activity feed"
					hint="Connections, disconnections and relayed events appear in Activity."
					bind:checked={dashboard.settings.bus.logActivity}
				/>
			</div>

			<PageHeader eyebrow="Pool" title="Connected apps" />
			{#if !poolStats || poolStats.apps.length === 0}
				<p class="rounded-xl border border-dashed border-border px-4 py-8 text-center text-sm text-muted-foreground">
					No app is connected.
				</p>
			{:else}
				<div class="grid gap-2">
					{#each poolStats.apps as app (app.app_id)}
						<div class="flex items-center justify-between gap-3 rounded-xl border border-border bg-card px-4 py-3">
							<div class="flex min-w-0 items-center gap-2.5">
								<span class="size-2 shrink-0 rounded-full bg-success"></span>
								<Icon icon={appIcon(app.app)} size={18} class="text-muted-foreground" />
								<strong class="truncate text-sm font-semibold">{app.app}</strong>
							</div>
							<span class="shrink-0 text-xs text-muted-foreground">
								{app.channels.length}
								{app.channels.length === 1 ? 'channel' : 'channels'}
							</span>
						</div>
					{/each}
				</div>
			{/if}

			{#if poolStats}
				<dl class="grid max-w-2xl divide-y divide-border overflow-hidden rounded-xl border border-border">
					<div class="flex items-center justify-between gap-4 px-4 py-3">
						<dt class="text-sm text-muted-foreground">Epoch</dt>
						<dd class="truncate font-mono text-sm" title={poolStats.epoch}>
							{poolStats.epoch.slice(0, 8)}
						</dd>
					</div>
					<div class="flex items-center justify-between gap-4 px-4 py-3">
						<dt class="text-sm text-muted-foreground">Live connections</dt>
						<dd class="text-sm font-medium tabular">{poolStats.connections}</dd>
					</div>
				</dl>
			{/if}
		</section>
	{:else if tab === 'events'}
		<section class="grid gap-5">
			<PageHeader eyebrow="Settings" title="System events" />

			<div class="grid max-w-2xl gap-2">
				<ToggleRow label="Antenne started" bind:checked={events.startup} />
				<ToggleRow label="A provider was enabled" bind:checked={events.providerEnabled} />
				<ToggleRow label="A provider was disabled" bind:checked={events.providerDisabled} />
				<ToggleRow label="A provider was added" bind:checked={events.providerAdded} />
				<ToggleRow label="A provider was removed" bind:checked={events.providerRemoved} />
				<ToggleRow label="A delivery target failed" bind:checked={events.deliveryFailure} />
			</div>

			<p class="max-w-2xl text-sm text-muted-foreground">
				Enabling an event is not enough on its own: a delivery target only receives these once it
				lists <strong class="font-medium text-foreground">Antenne system</strong> in its provider filter.
			</p>
		</section>
	{:else}
		<section class="grid gap-5">
			<PageHeader eyebrow="Config" title="Export and import" />

			<div class="grid max-w-2xl gap-4 sm:grid-cols-2">
				<div class="grid gap-2.5">
					<p class="text-sm text-muted-foreground">
						Download the current config. Secrets come out redacted, so the file is safe to keep.
					</p>
					<Button variant="outline" class="w-fit" onclick={() => dashboard.exportConfig()}>
						<Icon icon="solar:download-minimalistic-linear" size={15} />
						Export config
					</Button>
				</div>

				<div class="grid gap-2.5">
					<p class="text-sm text-muted-foreground">
						Import a previously exported config. This overwrites every provider and target.
					</p>
					<input
						bind:this={importInput}
						type="file"
						accept=".json,application/json"
						disabled={dashboard.saving}
						onchange={handleImport}
						class="text-sm file:mr-3 file:rounded-md file:border file:border-border file:bg-transparent file:px-3 file:py-1.5 file:text-sm file:font-medium hover:file:bg-muted"
					/>
				</div>
			</div>

			<PageHeader eyebrow="Runtime" title="Status" />

			<dl class="grid max-w-2xl divide-y divide-border overflow-hidden rounded-xl border border-border">
				<div class="flex items-center justify-between gap-4 px-4 py-3">
					<dt class="text-sm text-muted-foreground">Listening on</dt>
					<dd class="text-sm font-medium tabular">{runtime?.serverPort ?? '—'}</dd>
				</div>
				<div class="flex items-center justify-between gap-4 px-4 py-3">
					<dt class="text-sm text-muted-foreground">Settings store</dt>
					<dd class="text-sm font-medium">{runtime?.settingsPath ?? '—'}</dd>
				</div>
				<div class="flex items-center justify-between gap-4 px-4 py-3">
					<dt class="text-sm text-muted-foreground">Source</dt>
					<dd class="text-sm font-medium">
						{runtime?.usingPersistedSettings ? 'Saved from the dashboard' : 'Defaults'}
					</dd>
				</div>
				<div class="flex items-center justify-between gap-4 px-4 py-3">
					<dt class="text-sm text-muted-foreground">Last reload</dt>
					<dd class="text-sm font-medium">{formatDateTime(runtime?.lastReloadedAt)}</dd>
				</div>
			</dl>
		</section>
	{/if}
</div>
