<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Alert,
		Badge,
		Button,
		Card,
		ConfirmModal,
		Drawer,
		EmptyState,
		Field,
		Select,
		Switch,
		Table,
		cn,
		icons,
		toast
	} from '@facile/muse';
	import DeliveryEditorFields from './DeliveryEditorFields.svelte';
	import TargetHealthRow from './TargetHealthRow.svelte';
	import ViewToggle from './ViewToggle.svelte';
	import { ApiError, api, isRateLimited } from '$lib/backend';
	import { dashboard } from '$lib/dashboard.svelte';
	import { deliveryIcon } from '$lib/icons';
	import { tagToneMap } from '$lib/tags';
	import { formatRelative } from '$lib/utils';
	import {
		createTarget,
		deliveryTargetTypes,
		targetLabel,
		type DeliveryTarget,
		type DeliveryTargetType,
		type StandingAlert,
		type TargetHealth
	} from '$lib/types';

	const targets = $derived(dashboard.settings.delivery.items);
	const providers = $derived(dashboard.settings.providers.items);
	const tones = $derived(tagToneMap(dashboard.availableTags));

	let nextType = $state<DeliveryTargetType>('matrix');
	let view = $state<'list' | 'card'>('card');

	let editing = $state<DeliveryTarget | null>(null);
	let editingIndex = $state(-1);
	let drawerOpen = $state(false);
	let testing = $state(false);

	let pendingRemoval = $state<DeliveryTarget | null>(null);
	let confirmOpen = $state(false);

	let health = $state<Record<string, TargetHealth>>({});
	let healthLoaded = $state(false);
	let anyDeliverable = $state(true);
	let standingAlerts = $state<StandingAlert[]>([]);
	let checkingId = $state<string | null>(null);

	/*
	 * The check budget is per client, not per target, so one countdown covers the
	 * whole page. It is held as a deadline rather than a counter: a ticking
	 * counter would restart its own interval on every write, and this way one
	 * interval runs for the length of the block and stops itself.
	 */
	let cooldownUntil = $state(0);
	let clock = $state(Date.now());
	const cooldown = $derived(Math.max(0, Math.ceil((cooldownUntil - clock) / 1000)));

	$effect(() => {
		if (cooldownUntil === 0) return;
		const timer = setInterval(() => {
			clock = Date.now();
			if (clock >= cooldownUntil) cooldownUntil = 0;
		}, 500);
		return () => clearInterval(timer);
	});

	onMount(loadHealth);

	async function loadHealth() {
		try {
			const page = await api.deliveryHealth();
			health = Object.fromEntries(page.targets.map((row) => [row.targetId, row]));
			anyDeliverable = page.anyDeliverable;
			standingAlerts = page.standingAlerts ?? [];
			healthLoaded = true;
		} catch {
			// Health is an overlay on a list that already renders without it. A
			// failure here must not blank the page or bury the operator in toasts.
			healthLoaded = false;
		}
	}

	/**
	 * The banner for the case the whole feature exists for: nothing can be
	 * delivered, so Antenne cannot announce anything through itself and this page
	 * is the only channel left. Suppressed while there are no targets at all —
	 * the empty state already says that, and better.
	 */
	const nothingDeliverable = $derived(healthLoaded && targets.length > 0 && !anyDeliverable);

	async function checkNow(target: DeliveryTarget) {
		checkingId = target.id;
		try {
			const result = await api.checkDeliveryTarget(target.id);
			// Re-read the page rather than patch the one row: a recovery here can
			// clear the banner, and whether the *set* of targets can still deliver is
			// the server's answer to give, not this component's to re-derive.
			await loadHealth();

			// A refusal from the far side arrived as a 200: the check worked. Its
			// own words go through untouched, because a paraphrase is the thing an
			// operator cannot search for.
			if (result.ok) toast.success(`${target.name} answered.`);
			else toast.danger(`${target.name} declined: ${result.detail || 'no reason given'}`);
		} catch (error) {
			if (isRateLimited(error)) {
				const wait =
					error instanceof ApiError && error.retryAfterSeconds > 0 ? error.retryAfterSeconds : 60;
				cooldownUntil = Date.now() + wait * 1000;
				clock = Date.now();
				toast.warning(
					`Too many checks — Antenne's own limit, not a refusal from ${target.name}. Retry in ${wait}s.`
				);
			} else {
				toast.danger(error instanceof Error ? error.message : 'The check could not run');
			}
		} finally {
			checkingId = null;
		}
	}

	/*
	 * Every optional credential is `string | null` on the wire and the form controls want a
	 * string. The API maps an empty string back to null on the way in (settings.optionalString),
	 * so the round-trip is lossless and no field silently gains an empty-string value.
	 */
	const NULLABLE_TEXT = [
		'accessToken',
		'refreshToken',
		'password',
		'deviceId',
		'secretValue',
		'clickUrl'
	] as const;

	function hydrate(target: DeliveryTarget): DeliveryTarget {
		const draft = target as unknown as Record<string, unknown>;
		for (const key of NULLABLE_TEXT) {
			if (draft[key] === null) draft[key] = '';
		}
		return target;
	}

	/** A target with no provider and no tag selected can never receive anything. */
	function isSilent(target: DeliveryTarget): boolean {
		return target.providerIds.length === 0 && target.providerTags.length === 0;
	}

	function routingSummary(target: DeliveryTarget): string {
		const providerCount = `${target.providerIds.length} ${target.providerIds.length === 1 ? 'provider' : 'providers'}`;
		if (target.providerTags.length === 0) return providerCount;
		return `${providerCount} · ${target.providerTags.length} ${target.providerTags.length === 1 ? 'tag' : 'tags'}`;
	}

	function open(index: number) {
		editingIndex = index;
		editing = hydrate(structuredClone($state.snapshot(targets[index])) as DeliveryTarget);
		drawerOpen = true;
	}

	function add() {
		dashboard.setTargets([createTarget(nextType), ...targets]);
		open(0);
	}

	async function commit() {
		if (!editing || editingIndex < 0) return;
		const draft = structuredClone($state.snapshot(editing)) as DeliveryTarget;
		dashboard.setTargets(
			targets.map((current, index) => (index === editingIndex ? draft : current))
		);
		await dashboard.save();
		drawerOpen = false;
	}

	/** A row toggle is its own edit and persists on the spot; the editor waits for Save. */
	async function toggleEnabled(index: number, enabled: boolean) {
		dashboard.setTargets(
			targets.map((target, position) => (position === index ? { ...target, enabled } : target))
		);
		await dashboard.save({ silent: true });
	}

	function askRemove(target: DeliveryTarget) {
		pendingRemoval = target;
		confirmOpen = true;
	}

	async function confirmRemove() {
		const doomed = pendingRemoval;
		if (!doomed) return;
		if (editing?.id === doomed.id) drawerOpen = false;
		dashboard.setTargets(targets.filter((target) => target.id !== doomed.id));
		await dashboard.save();
		pendingRemoval = null;
	}

	async function sendTest() {
		if (!editing) return;
		testing = true;
		try {
			await api.testTarget(editing.id);
			toast.success(`Test delivered to ${editing.name}.`);
		} catch (error) {
			toast.danger(error instanceof Error ? error.message : 'The test delivery failed');
		} finally {
			testing = false;
		}
	}
</script>

<div class="flex flex-col gap-4">
	{#if nothingDeliverable}
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
	{/if}

	<div class="flex flex-wrap items-end gap-4">
		<Field label="New target type" class="w-full sm:w-56">
			<Select bind:value={nextType as string}>
				{#each deliveryTargetTypes as type (type)}
					<option value={type}>{targetLabel(type)}</option>
				{/each}
			</Select>
		</Field>

		<Button size="lg" icon={icons.plus} onclick={add}>Add target</Button>

		{#if targets.length > 0}
			<div class="ml-auto">
				<ViewToggle bind:value={view} />
			</div>
		{/if}
	</div>

	{#if targets.length === 0}
		<EmptyState
			icon={icons.mail}
			title="No delivery target yet"
			description="Antenne is watching, but nothing it sees leaves the box until a target exists."
		>
			<Button icon={icons.plus} onclick={add}>Add target</Button>
		</EmptyState>
	{:else if view === 'card'}
		<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
			{#each targets as target, index (target.id)}
				<Card class={cn('flex flex-col gap-4', !target.enabled && 'opacity-55')}>
					<div class="flex items-start justify-between gap-3">
						<span
							class="flex size-10 shrink-0 items-center justify-center rounded-fc-md bg-fc-surface text-fc-fg-muted"
						>
							<iconify-icon icon={deliveryIcon(target.type)} width="20" height="20" class="block"
							></iconify-icon>
						</span>
						<Switch
							label="Enabled"
							bind:checked={() => target.enabled, (enabled) => toggleEnabled(index, enabled)}
						/>
					</div>

					<div class="flex min-w-0 flex-col gap-1">
						<span class="truncate text-fc-sm font-semibold text-fc-fg">{target.name}</span>
						<span class="text-fc-xs text-fc-fg-muted">{targetLabel(target.type)}</span>
					</div>

					{#if isSilent(target)}
						<Badge tone="warning" title="Attach a provider or a tag, or this target receives nothing."
							>No source attached</Badge
						>
					{:else}
						<span class="text-fc-xs text-fc-fg-muted">{routingSummary(target)}</span>
					{/if}

					<TargetHealthRow
						health={health[target.id]}
						checking={checkingId === target.id}
						{cooldown}
						onCheck={() => checkNow(target)}
					/>

					<div class="mt-auto flex items-center gap-2">
						<Button variant="outline" size="sm" icon={icons.edit} onclick={() => open(index)}>
							Edit
						</Button>
						<Button
							variant="ghost"
							size="sm"
							icon={icons.dashboard}
							href={`/targets/${target.id}`}
							title="Activity"
							aria-label={`Activity for ${target.name}`}
						/>
						<Button
							variant="ghost-danger"
							size="sm"
							icon={icons.remove}
							onclick={() => askRemove(target)}
							title="Delete"
							aria-label={`Delete ${target.name}`}
							class="ml-auto"
						/>
					</div>
				</Card>
			{/each}
		</div>
	{:else}
		<Table>
			<thead>
				<tr>
					<th scope="col">Target</th>
					<th scope="col">Type</th>
					<th scope="col">Routing</th>
					<th scope="col">Health</th>
					<th scope="col"><span class="sr-only">Actions</span></th>
				</tr>
			</thead>
			<tbody>
				{#each targets as target, index (target.id)}
					<tr class={cn(!target.enabled && 'opacity-55')}>
						<td>
							<div class="flex min-w-0 items-center gap-3">
								<iconify-icon
									icon={deliveryIcon(target.type)}
									width="18"
									height="18"
									class="block shrink-0"
								></iconify-icon>
								<!-- The switch is labelled with the target's own name, so the control and the
								     row's identity are one thing rather than a column of "Enabled". -->
								<Switch
									label={target.name}
									bind:checked={() => target.enabled, (enabled) => toggleEnabled(index, enabled)}
									class="min-w-0 font-semibold"
								/>
							</div>
						</td>
						<td class="text-fc-fg-muted">{targetLabel(target.type)}</td>
						<td>
							{#if isSilent(target)}
								<Badge
									tone="warning"
									title="Attach a provider or a tag, or this target receives nothing."
									>No source attached</Badge
								>
							{:else}
								<span class="text-fc-fg-muted">{routingSummary(target)}</span>
							{/if}
						</td>
						<td class="min-w-64">
							<TargetHealthRow
								health={health[target.id]}
								checking={checkingId === target.id}
								{cooldown}
								onCheck={() => checkNow(target)}
							/>
						</td>
						<td>
							<div class="flex items-center justify-end gap-1">
								<Button variant="ghost" size="sm" icon={icons.edit} onclick={() => open(index)}>
									Edit
								</Button>
								<Button
									variant="ghost"
									size="sm"
									icon={icons.dashboard}
									href={`/targets/${target.id}`}
									title="Activity"
									aria-label={`Activity for ${target.name}`}
								/>
								<Button
									variant="ghost-danger"
									size="sm"
									icon={icons.remove}
									onclick={() => askRemove(target)}
									title="Delete"
									aria-label={`Delete ${target.name}`}
								/>
							</div>
						</td>
					</tr>
				{/each}
			</tbody>
		</Table>
	{/if}
</div>

<!-- The editor works on a detached clone: nothing the drawer touches reaches the settings
     document until commit() writes it back and saves. -->
<Drawer
	bind:open={drawerOpen}
	showClose
	title={editing ? `Edit ${editing.name}` : 'Edit target'}
	description={editing ? targetLabel(editing.type) : undefined}
>
	{#if editing}
		<DeliveryEditorFields
			bind:target={editing}
			{providers}
			availableTags={dashboard.availableTags}
			{tones}
		/>
	{/if}

	{#snippet footer()}
		<div class="flex flex-wrap justify-end gap-2">
			<Button variant="outline" icon="solar:plain-linear" onclick={sendTest} disabled={testing}>
				{testing ? 'Sending…' : 'Send a test'}
			</Button>
			<Button icon={icons.check} onclick={commit} disabled={dashboard.saving}>
				{dashboard.saving ? 'Saving…' : 'Save'}
			</Button>
		</div>
	{/snippet}
</Drawer>

<ConfirmModal
	bind:open={confirmOpen}
	tone="danger"
	title="Delete this delivery target?"
	description={pendingRemoval
		? `${pendingRemoval.name} stops receiving events the moment this is saved, and any provider routed only here goes nowhere. Deleting cannot be undone.`
		: undefined}
	confirmLabel="Delete"
	onConfirm={confirmRemove}
	onCancel={() => (pendingRemoval = null)}
/>
