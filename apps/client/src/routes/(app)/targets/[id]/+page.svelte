<script lang="ts">
	import { page } from '$app/state';
	import { toast } from 'svelte-sonner';
	import EventTimeline from '$lib/components/EventTimeline.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import StatCard from '$lib/components/StatCard.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import { api } from '$lib/backend';
	import { dashboard } from '$lib/dashboard.svelte';
	import { deliveryIcon } from '$lib/icons';
	import { targetLabel, type EventEntry, type EventStats } from '$lib/types';

	const targetId = $derived(page.params.id ?? '');
	const target = $derived(
		dashboard.settings.delivery.items.find((candidate) => candidate.id === targetId)
	);

	let events = $state<EventEntry[]>([]);
	let stats = $state<EventStats | null>(null);
	let loading = $state(false);

	async function refresh() {
		if (!targetId) return;
		loading = true;
		try {
			const payload = await api.events({ targetId, limit: 100 });
			events = payload.events;
			stats = payload.stats;
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to load the delivery log');
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void targetId;
		void refresh();
	});
</script>

<div class="grid gap-5">
	<a
		href="/delivery"
		class="inline-flex w-fit items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
	>
		<Icon icon="solar:arrow-left-linear" size={15} />
		Back to delivery
	</a>

	{#if target}
		<div class="flex flex-wrap items-center gap-3">
			<span class="flex size-10 items-center justify-center rounded-lg border border-border text-muted-foreground">
				<Icon icon={deliveryIcon(target.type)} size={22} />
			</span>
			<h2 class="font-heading text-xl font-semibold tracking-tight">{target.name}</h2>
			<Badge variant="outline">{targetLabel(target.type)}</Badge>
		</div>
	{:else}
		<p class="text-sm text-muted-foreground">This delivery target no longer exists.</p>
	{/if}

	<div class="grid gap-3 sm:grid-cols-3">
		<StatCard label="Events" value={stats?.totalEvents ?? 0} />
		<StatCard label="Delivered" value={stats?.deliverySuccessCount ?? 0} tone="success" />
		<StatCard label="Failed" value={stats?.deliveryErrorCount ?? 0} tone="destructive" />
	</div>

	<div class="flex items-center justify-between gap-3">
		<h3 class="font-heading text-base font-semibold">Delivery log</h3>
		<Button variant="ghost" size="icon" onclick={refresh} disabled={loading} title="Refresh">
			<Icon icon="solar:refresh-linear" size={15} class={loading ? 'animate-spin' : ''} />
		</Button>
	</div>

	<EventTimeline {events} />
</div>
