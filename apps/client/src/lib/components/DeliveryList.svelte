<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Button,
		ConfirmModal,
		EmptyState,
		Field,
		Select,
		icons,
		toast
	} from '@facile/muse';
	import DeliveryEditorDrawer from './DeliveryEditorDrawer.svelte';
	import DeliveryOutageAlert from './DeliveryOutageAlert.svelte';
	import DeliveryTargetCards from './DeliveryTargetCards.svelte';
	import DeliveryTargetTable from './DeliveryTargetTable.svelte';
	import ViewToggle from './ViewToggle.svelte';
	import { ApiError, api, isRateLimited } from '$lib/backend';
	import { dashboard } from '$lib/dashboard.svelte';
	import { t } from '$lib/i18n/index.svelte';
	import { tagToneMap } from '$lib/tags';
	import {
		createTarget,
		deliveryTargetTypes,
		hydrateDeliveryTarget,
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
			if (result.ok) toast.success(t('delivery.answered', { name: target.name }));
			else toast.danger(
					t('delivery.declined', { name: target.name, detail: result.detail || t('delivery.noReason') })
				);
		} catch (error) {
			if (isRateLimited(error)) {
				const wait =
					error instanceof ApiError && error.retryAfterSeconds > 0 ? error.retryAfterSeconds : 60;
				cooldownUntil = Date.now() + wait * 1000;
				clock = Date.now();
				toast.warning(
					t('delivery.tooMany', { name: target.name, seconds: wait })
				);
			} else {
				toast.danger(error instanceof Error ? error.message : t('delivery.checkFailed'));
			}
		} finally {
			checkingId = null;
		}
	}

	function open(index: number) {
		editingIndex = index;
		editing = hydrateDeliveryTarget($state.snapshot(targets[index]));
		drawerOpen = true;
	}

	function add() {
		dashboard.setTargets([createTarget(nextType), ...targets]);
		open(0);
	}

	async function commit() {
		if (!editing || editingIndex < 0) return;
		const draft = $state.snapshot(editing);
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
</script>

<div class="flex flex-col gap-4">
	{#if nothingDeliverable}
		<DeliveryOutageAlert {standingAlerts} />
	{/if}

	<div class="flex flex-wrap items-end gap-4">
		<Field label={t('delivery.newType')} class="w-full sm:w-56">
			<Select bind:value={nextType as string}>
				{#each deliveryTargetTypes as type (type)}
					<option value={type}>{targetLabel(type)}</option>
				{/each}
			</Select>
		</Field>

		<Button size="lg" icon={icons.plus} onclick={add}>{t('delivery.add')}</Button>

		{#if targets.length > 0}
			<div class="ml-auto">
				<ViewToggle bind:value={view} />
			</div>
		{/if}
	</div>

	{#if targets.length === 0}
		<EmptyState
			icon={icons.mail}
			title={t('delivery.emptyTitle')}
			description={t('delivery.emptyText')}
		>
			<Button icon={icons.plus} onclick={add}>{t('delivery.add')}</Button>
		</EmptyState>
	{:else if view === 'card'}
		<DeliveryTargetCards
			{targets}
			{health}
			{checkingId}
			{cooldown}
			onToggle={toggleEnabled}
			onOpen={open}
			onCheck={checkNow}
			onRemove={askRemove}
		/>
	{:else}
		<DeliveryTargetTable
			{targets}
			{health}
			{checkingId}
			{cooldown}
			onToggle={toggleEnabled}
			onOpen={open}
			onCheck={checkNow}
			onRemove={askRemove}
		/>
	{/if}
</div>

<DeliveryEditorDrawer
	bind:open={drawerOpen}
	bind:target={editing}
	{providers}
	availableTags={dashboard.availableTags}
	{tones}
	saving={dashboard.saving}
	onSave={commit}
/>

<ConfirmModal
	bind:open={confirmOpen}
	tone="danger"
	title={t('delivery.deleteTitle')}
	description={pendingRemoval
		? t('delivery.deleteText', { name: pendingRemoval.name })
		: undefined}
	confirmLabel={t('common.delete')}
	onConfirm={confirmRemove}
	onCancel={() => (pendingRemoval = null)}
/>
