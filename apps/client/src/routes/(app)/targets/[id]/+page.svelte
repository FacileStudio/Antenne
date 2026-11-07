<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { Alert, Badge, Button, ConfirmModal, EmptyState, Spinner, StatCard, Switch, icons, toast } from '@facile/muse';
	import EventTimeline from '$lib/components/EventTimeline.svelte';
	import DeliveryEditorDrawer from '$lib/components/DeliveryEditorDrawer.svelte';
	import { api } from '$lib/backend';
	import { dashboard } from '$lib/dashboard.svelte';
	import { t } from '$lib/i18n/index.svelte';
	import { deliveryIcon } from '$lib/icons';
	import { hydrateDeliveryTarget, targetLabel, type EventEntry, type EventStats, type DeliveryTarget } from '$lib/types';
	import { tagToneMap } from '$lib/tags';

	const targetId = $derived(page.params.id ?? '');
	const target = $derived(
		dashboard.settings.delivery.items.find((candidate) => candidate.id === targetId)
	);

	let events = $state<EventEntry[]>([]);
	let stats = $state<EventStats | null>(null);
	let loading = $state(false);
	let drawerOpen = $state(false);
	let editing = $state<DeliveryTarget | null>(null);
	let confirmOpen = $state(false);
	const tones = $derived(tagToneMap(dashboard.availableTags));

	async function refresh() {
		if (!targetId) return;
		loading = true;
		try {
			const payload = await api.events({ targetId, limit: 100 });
			events = payload.events;
			stats = payload.stats;
		} catch (error) {
			toast.danger(error instanceof Error ? error.message : t('targets.loadFailed'));
		} finally {
			loading = false;
		}
	}

	function openEdit() {
		const t = dashboard.settings.delivery.items.find((c) => c.id === targetId);
		if (t) {
			editing = hydrateDeliveryTarget($state.snapshot(t));
			drawerOpen = true;
		}
	}

	function closeEdit() {
		editing = null;
		drawerOpen = false;
	}

	async function saveEdit() {
		if (!editing) return;
		const current = $state.snapshot(editing);
		try {
			dashboard.setTargets(dashboard.settings.delivery.items.map((t) => (t.id === current.id ? current : t)));
			await dashboard.save();
			closeEdit();
		} catch (error) {
			toast.danger(error instanceof Error ? error.message : t('providers.saveFailed'));
		}
	}

	async function toggleEnabled(enabled: boolean) {
		if (!target) return;
		dashboard.setTargets(
			dashboard.settings.delivery.items.map((t) =>
				t.id === target.id ? { ...t, enabled } : t
			)
		);
		await dashboard.save({ silent: true });
	}

	function askDelete() {
		confirmOpen = true;
	}

	async function removeTarget() {
		if (!target) return;
		confirmOpen = false;
		dashboard.setTargets(
			dashboard.settings.delivery.items.filter((t) => t.id !== target.id)
		);
		await dashboard.save();
		await goto('/delivery');
	}

	$effect(() => {
		void targetId;
		void refresh();
	});
</script>

<div class="flex flex-col gap-10">
	<div class="flex flex-col gap-4">
		<div class="flex flex-wrap items-center justify-between gap-4">
			<Button variant="ghost" size="sm" icon={icons.chevronLeft} href="/delivery" class="-ml-3.5 w-fit">
				{t('targets.back')}
			</Button>
			{#if target}
				<div class="flex items-center gap-3">
					<Switch
						label={t('common.enabled')}
						bind:checked={() => target.enabled, (enabled) => toggleEnabled(enabled)}
					/>
					<Button variant="secondary" size="sm" icon={icons.edit} onclick={openEdit}>
						{t('targets.editTarget')}
					</Button>
					<Button
						variant="ghost-danger"
						size="sm"
						icon={icons.remove}
						onclick={askDelete}
						aria-label={t('common.deleteName', { name: target.name })}
					>
						{t('common.delete')}
					</Button>
				</div>
			{/if}
		</div>

		{#if target}
			<div class="flex flex-col gap-2">
				<div class="flex flex-wrap items-center gap-3">
					<span
						class="flex size-10 shrink-0 items-center justify-center rounded-fc-md bg-fc-surface text-fc-fg-muted"
					>
						<iconify-icon icon={deliveryIcon(target.type)} width="20" height="20" class="block"
						></iconify-icon>
					</span>
					<h1 class="text-fc-2xl font-semibold text-fc-fg">{target.name}</h1>
					<Badge>{targetLabel(target.type)}</Badge>
				</div>
				<p class="text-fc-sm text-fc-fg-muted">{t('targets.history')}</p>
			</div>
		{:else}
			<Alert tone="warning" title={t('targets.goneTitle')}>
				{t('targets.goneText')}
			</Alert>
		{/if}
	</div>

	<div class="grid gap-4 sm:grid-cols-3">
		<StatCard label={t('providers.events')} value={stats?.totalEvents ?? 0} />
		<StatCard label={t('providers.delivered')} value={stats?.deliverySuccessCount ?? 0} />
		<StatCard label={t('providers.failed')} value={stats?.deliveryErrorCount ?? 0} />
	</div>

	<section class="flex flex-col gap-4">
		<div class="flex flex-wrap items-start justify-between gap-3">
			<div class="flex flex-col gap-1">
				<h2 class="text-fc-lg font-semibold text-fc-fg">{t('targets.log')}</h2>
				<p class="text-fc-sm text-fc-fg-muted">{t('targets.logText')}</p>
			</div>
			<Button variant="ghost" size="sm" icon={icons.refresh} onclick={refresh} disabled={loading}>
				{loading ? t('activity.refreshing') : t('activity.refresh')}
			</Button>
		</div>

		{#if loading && events.length === 0}
			<div class="flex justify-center py-12">
				<Spinner />
			</div>
		{:else if events.length === 0}
			<EmptyState
				icon={icons.history}
				title={t('targets.emptyTitle')}
				description={t('targets.emptyText')}
			/>
		{:else}
			<EventTimeline {events} />
		{/if}
	</section>
</div>

<DeliveryEditorDrawer
	bind:open={drawerOpen}
	bind:target={editing}
	providers={dashboard.settings.providers.items}
	availableTags={dashboard.availableTags}
	tones={tones}
	saving={dashboard.saving}
	onSave={saveEdit}
/>

<ConfirmModal
	bind:open={confirmOpen}
	tone="danger"
	title={t('delivery.deleteTitle')}
	description={target ? t('targets.deleteText', { name: target.name }) : ''}
	confirmLabel={t('common.delete')}
	onConfirm={removeTarget}
	onCancel={() => (confirmOpen = false)}
/>
