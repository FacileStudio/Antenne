<script lang="ts">
	import { page } from '$app/state';
	import { Alert, Badge, Button, EmptyState, Spinner, StatCard, icons, toast } from '@facile/muse';
	import EventTimeline from '$lib/components/EventTimeline.svelte';
	import DeliveryEditorDrawer from '$lib/components/DeliveryEditorDrawer.svelte';
	import { api } from '$lib/backend';
	import { dashboard } from '$lib/dashboard.svelte';
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
	const tones = $derived(tagToneMap(dashboard.availableTags));

	async function refresh() {
		if (!targetId) return;
		loading = true;
		try {
			const payload = await api.events({ targetId, limit: 100 });
			events = payload.events;
			stats = payload.stats;
		} catch (error) {
			toast.danger(error instanceof Error ? error.message : 'Failed to load the delivery log');
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
			toast.danger(error instanceof Error ? error.message : 'Failed to save changes');
		}
	}

	$effect(() => {
		void targetId;
		void refresh();
	});
</script>

<div class="flex flex-col gap-10">
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between">
			<Button variant="ghost" size="sm" icon={icons.chevronLeft} href="/delivery" class="-ml-3.5 w-fit">
				Back to delivery
			</Button>
			{#if target}
				<Button variant="outline" size="sm" icon={icons.edit} onclick={openEdit}>
					Edit target
				</Button>
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
				<p class="text-fc-sm text-fc-fg-muted">Everything Antenne has tried to deliver here.</p>
			</div>
		{:else}
			<Alert tone="warning" title="This delivery target no longer exists">
				It was deleted, or the link points at an id this Antenne never had. The log below is empty
				for the same reason.
			</Alert>
		{/if}
	</div>

	<div class="grid gap-4 sm:grid-cols-3">
		<StatCard label="Events" value={stats?.totalEvents ?? 0} />
		<StatCard label="Delivered" value={stats?.deliverySuccessCount ?? 0} />
		<StatCard label="Failed" value={stats?.deliveryErrorCount ?? 0} />
	</div>

	<section class="flex flex-col gap-4">
		<div class="flex flex-wrap items-start justify-between gap-3">
			<div class="flex flex-col gap-1">
				<h2 class="text-fc-lg font-semibold text-fc-fg">Delivery log</h2>
				<p class="text-fc-sm text-fc-fg-muted">The last 100 events routed to this target.</p>
			</div>
			<Button variant="ghost" size="sm" icon={icons.refresh} onclick={refresh} disabled={loading}>
				{loading ? 'Refreshing…' : 'Refresh'}
			</Button>
		</div>

		{#if loading && events.length === 0}
			<div class="flex justify-center py-12">
				<Spinner />
			</div>
		{:else if events.length === 0}
			<EmptyState
				icon={icons.history}
				title="Nothing delivered yet"
				description="Events routed to this target show up here, successful or not."
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
