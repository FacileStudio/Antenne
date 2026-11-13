<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { Badge, Button, ConfirmModal, Drawer, EmptyState, Spinner, StatCard, Switch, Tooltip, icons, toast } from '@facile/muse';
	import EventTimeline from '$lib/components/EventTimeline.svelte';
	import ProviderEditorFields from '$lib/components/ProviderEditorFields.svelte';
	import TagPill from '$lib/components/TagPill.svelte';
	import { api } from '$lib/backend';
	import { dashboard } from '$lib/dashboard.svelte';
	import { t } from '$lib/i18n/index.svelte';
	import { providerIcon } from '$lib/icons';
	import { tagToneMap } from '$lib/tags';
	import { hydrateProvider, providerLabel, type EventEntry, type EventStats, type Provider } from '$lib/types';

	const providerId = $derived(page.params.id ?? '');
	const provider = $derived(
		dashboard.settings.providers.items.find((candidate) => candidate.id === providerId)
	);
	const tones = $derived(tagToneMap(dashboard.availableTags));

	let events = $state<EventEntry[]>([]);
	let stats = $state<EventStats | null>(null);
	let loading = $state(false);
	let editing = $state<Provider | null>(null);
	let drawerOpen = $state(false);
	let confirmOpen = $state(false);

	async function refresh() {
		if (!providerId) return;
		loading = true;
		try {
			const payload = await api.events({ providerId, limit: 100 });
			events = payload.events;
			stats = payload.stats;
		} catch (error) {
			toast.danger(error instanceof Error ? error.message : t('providers.loadFailed'));
		} finally {
			loading = false;
		}
	}

	function openEdit() {
		const p = dashboard.settings.providers.items.find((c) => c.id === providerId);
		if (p) {
			editing = hydrateProvider($state.snapshot(p));
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
			dashboard.setProviders(dashboard.settings.providers.items.map((p) => (p.id === current.id ? current : p)));
			await dashboard.save();
			closeEdit();
		} catch (error) {
			toast.danger(error instanceof Error ? error.message : t('providers.saveFailed'));
		}
	}

	async function toggleEnabled(enabled: boolean) {
		if (!provider) return;
		dashboard.setProviders(
			dashboard.settings.providers.items.map((p) =>
				p.id === provider.id ? { ...p, enabled } : p
			)
		);
		await dashboard.save({ silent: true });
	}

	function askDelete() {
		confirmOpen = true;
	}

	async function removeProvider() {
		if (!provider) return;
		confirmOpen = false;
		dashboard.setProviders(
			dashboard.settings.providers.items.filter((p) => p.id !== provider.id)
		);
		await dashboard.save();
		await goto('/providers');
	}

	$effect(() => {
		void providerId;
		void refresh();
	});
</script>

<div class="flex flex-col gap-10">
	<div class="flex flex-col gap-4">
		<div class="flex flex-wrap items-center justify-between gap-4">
			<Button variant="ghost" size="sm" icon={icons.chevronLeft} href="/providers" class="-ml-3.5 w-fit">
				{t('providers.back')}
			</Button>
			{#if provider}
				<div class="flex items-center gap-3">
					<Switch
						label={t('common.enabled')}
						bind:checked={() => provider.enabled, (enabled) => toggleEnabled(enabled)}
					/>
					<Button variant="secondary" size="sm" icon={icons.edit} onclick={openEdit}>
						{t('providers.editProvider')}
					</Button>
					<Button
						variant="ghost-danger"
						size="sm"
						icon={icons.remove}
						onclick={askDelete}
						aria-label={t('common.deleteName', { name: provider.name })}
					>
						{t('common.delete')}
					</Button>
				</div>
			{/if}
		</div>

		{#if provider}
			<div class="flex flex-wrap items-center gap-3">
				<span
					class="flex size-10 shrink-0 items-center justify-center rounded-fc-md bg-fc-surface text-fc-fg-muted"
				>
					<iconify-icon icon={providerIcon(provider.type)} width="22" height="22" class="block"
					></iconify-icon>
				</span>
				<h1 class="text-fc-2xl font-semibold text-fc-fg">{provider.name}</h1>
				<Badge>{providerLabel(provider.type)}</Badge>
				{#each provider.tags as tag (tag)}
					<TagPill {tag} {tones} />
				{/each}
			</div>
		{/if}
	</div>

	{#if provider}
		<div class="grid gap-4 sm:grid-cols-3">
			<StatCard label={t('providers.events')} value={stats?.totalEvents ?? 0} />
			<StatCard label={t('providers.delivered')} value={stats?.deliverySuccessCount ?? 0} />
			<StatCard label={t('providers.failed')} value={stats?.deliveryErrorCount ?? 0} />
		</div>

		<div class="flex flex-col gap-4">
			<div class="flex items-start justify-between gap-4">
				<div class="flex flex-col gap-1">
					<h2 class="text-fc-lg font-semibold text-fc-fg">{t('providers.eventLog')}</h2>
					<p class="text-fc-sm text-fc-fg-muted">{t('providers.eventLogText')}</p>
				</div>
				<Tooltip text={t('providers.refresh')} describe={false} class="contents">
					<Button
						variant="ghost"
						icon={icons.refresh}
						onclick={refresh}
						disabled={loading}
						aria-label={t('providers.refresh')}
					/>
				</Tooltip>
			</div>

			{#if loading && events.length === 0}
				<div class="flex justify-center py-12">
					<Spinner />
				</div>
			{:else}
				<EventTimeline {events} />
			{/if}
		</div>
	{:else}
		<EmptyState
			icon={icons.warning}
			title={t('providers.goneTitle')}
			description={t('providers.goneText')}
		>
			<Button href="/providers">{t('providers.back')}</Button>
		</EmptyState>
	{/if}
</div>

<Drawer
	bind:open={drawerOpen}
	showClose
	title={editing ? t('providers.editTitle', { name: editing.name }) : t('providers.editFallback')}
	description={editing ? providerLabel(editing.type) : undefined}
>
	{#if editing}
		<ProviderEditorFields bind:provider={editing} availableTags={dashboard.availableTags} {tones} />
	{/if}

	{#snippet footer()}
		<Button icon={icons.check} onclick={saveEdit} disabled={dashboard.saving}>
			{dashboard.saving ? t('common.saving') : t('common.save')}
		</Button>
	{/snippet}
</Drawer>

<ConfirmModal
	bind:open={confirmOpen}
	tone="danger"
	title={t('providers.deleteTitle')}
	description={provider
		? t('providers.deleteText', { name: provider.name })
		: ''}
	confirmLabel={t('common.delete')}
	onConfirm={removeProvider}
	onCancel={() => (confirmOpen = false)}
/>
