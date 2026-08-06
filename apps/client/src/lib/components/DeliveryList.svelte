<script lang="ts">
	import { toast } from 'svelte-sonner';
	import DeliveryEditorFields from './DeliveryEditorFields.svelte';
	import Icon from './Icon.svelte';
	import ViewToggle from './ViewToggle.svelte';
	import Badge from './ui/badge.svelte';
	import Button from './ui/button.svelte';
	import Select from './ui/select.svelte';
	import Sheet from './ui/sheet.svelte';
	import Switch from './ui/switch.svelte';
	import { api } from '$lib/backend';
	import { dashboard } from '$lib/dashboard.svelte';
	import { deliveryIcon } from '$lib/icons';
	import { tagToneMap } from '$lib/tags';
	import {
		createTarget,
		deliveryTargetTypes,
		targetLabel,
		type DeliveryTarget,
		type DeliveryTargetType
	} from '$lib/types';
	import { cn } from '$lib/utils';

	const targets = $derived(dashboard.settings.delivery.items);
	const providers = $derived(dashboard.settings.providers.items);
	const tones = $derived(tagToneMap(dashboard.availableTags));

	let nextType = $state<DeliveryTargetType>('matrix');
	let view = $state<'list' | 'card'>('card');

	let editing = $state<DeliveryTarget | null>(null);
	let editingIndex = $state(-1);
	let sheetOpen = $state(false);
	let testing = $state(false);

	const typeOptions = deliveryTargetTypes.map((type) => ({ value: type, label: targetLabel(type) }));

	/** A target with no provider and no tag selected can never receive anything. */
	function isSilent(target: DeliveryTarget): boolean {
		return target.providerIds.length === 0 && target.providerTags.length === 0;
	}

	function open(index: number) {
		editingIndex = index;
		editing = structuredClone($state.snapshot(targets[index])) as DeliveryTarget;
		sheetOpen = true;
	}

	function add() {
		dashboard.setTargets([createTarget(nextType), ...targets]);
		open(0);
	}

	async function commit() {
		if (!editing || editingIndex < 0) return;
		const draft = structuredClone($state.snapshot(editing)) as DeliveryTarget;
		dashboard.setTargets(targets.map((current, index) => (index === editingIndex ? draft : current)));
		await dashboard.save();
		sheetOpen = false;
	}

	async function toggleEnabled(index: number, enabled: boolean) {
		dashboard.setTargets(
			targets.map((target, position) => (position === index ? { ...target, enabled } : target))
		);
		await dashboard.save({ silent: true });
	}

	async function remove(index: number) {
		dashboard.setTargets(targets.filter((_, position) => position !== index));
		if (editingIndex === index) sheetOpen = false;
		await dashboard.save();
	}

	async function sendTest() {
		if (!editing) return;
		testing = true;
		try {
			await api.testTarget(editing.id);
			toast.success(`Test delivered to ${editing.name}.`);
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'The test delivery failed');
		} finally {
			testing = false;
		}
	}
</script>

<div class="grid gap-4">
	<div class="flex flex-wrap items-end gap-3">
		<div class="grid gap-1.5">
			<span class="text-xs text-muted-foreground">New target type</span>
			<Select bind:value={nextType as string} options={typeOptions} class="w-56" />
		</div>

		<Button onclick={add}>
			<Icon icon="solar:add-circle-linear" size={15} />
			Add target
		</Button>

		<div class="ml-auto">
			{#if targets.length > 0}
				<ViewToggle bind:value={view} />
			{/if}
		</div>
	</div>

	{#if targets.length === 0}
		<p class="rounded-xl border border-dashed border-border px-4 py-8 text-center text-sm text-muted-foreground">
			No delivery target configured yet. Nothing Nook sees will be delivered anywhere.
		</p>
	{:else if view === 'card'}
		<div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
			{#each targets as target, index (target.id)}
				<article
					class={cn(
						'flex flex-col gap-3 rounded-xl border border-border bg-card p-4 transition-colors hover:border-foreground/20',
						!target.enabled && 'opacity-55'
					)}
				>
					<div class="flex items-start justify-between">
						<span
							class="flex size-10 items-center justify-center rounded-lg border border-border text-muted-foreground"
						>
							<Icon icon={deliveryIcon(target.type)} size={22} />
						</span>
						<Switch
							size="sm"
							checked={target.enabled}
							aria-label={`Enable ${target.name}`}
							onCheckedChange={(checked: boolean) => toggleEnabled(index, checked)}
						/>
					</div>

					<div class="grid min-w-0 gap-0.5">
						<span class="truncate text-sm font-semibold">{target.name}</span>
						<span class="text-xs text-muted-foreground">{targetLabel(target.type)}</span>
					</div>

					{#if isSilent(target)}
						<Badge variant="outline" title="Attach a provider or a tag, or this target receives nothing.">
							No source attached
						</Badge>
					{:else}
						<span class="text-xs text-muted-foreground">
							{target.providerIds.length}
							{target.providerIds.length === 1 ? 'provider' : 'providers'}
							{#if target.providerTags.length > 0}
								· {target.providerTags.length}
								{target.providerTags.length === 1 ? 'tag' : 'tags'}
							{/if}
						</span>
					{/if}

					<div class="mt-auto flex gap-1 border-t border-border pt-2.5">
						<Button variant="ghost" size="sm" class="flex-1" onclick={() => open(index)} title="Edit">
							<Icon icon="solar:pen-linear" size={14} />
						</Button>
						<Button variant="ghost" size="sm" class="flex-1" href={`/targets/${target.id}`} title="Activity">
							<Icon icon="solar:chart-2-linear" size={14} />
						</Button>
						<Button variant="destructive" size="sm" class="flex-1" onclick={() => remove(index)} title="Delete">
							<Icon icon="solar:trash-bin-trash-linear" size={14} />
						</Button>
					</div>
				</article>
			{/each}
		</div>
	{:else}
		<div class="grid gap-2">
			{#each targets as target, index (target.id)}
				<article
					class={cn(
						'flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border bg-card px-4 py-3 transition-colors hover:border-foreground/20',
						!target.enabled && 'opacity-55'
					)}
				>
					<div class="flex min-w-0 flex-wrap items-center gap-2.5">
						<Switch
							size="sm"
							checked={target.enabled}
							aria-label={`Enable ${target.name}`}
							onCheckedChange={(checked: boolean) => toggleEnabled(index, checked)}
						/>
						<Icon icon={deliveryIcon(target.type)} size={19} class="text-muted-foreground" />
						<strong class="truncate text-sm font-semibold">{target.name}</strong>
						<Badge variant="outline">{targetLabel(target.type)}</Badge>
						{#if isSilent(target)}
							<Badge variant="outline">No source attached</Badge>
						{/if}
					</div>

					<div class="flex items-center gap-1">
						<Button variant="ghost" size="sm" onclick={() => open(index)}>
							<Icon icon="solar:pen-linear" size={14} />
							Edit
						</Button>
						<Button variant="ghost" size="icon" href={`/targets/${target.id}`} title="Activity">
							<Icon icon="solar:chart-2-linear" size={14} />
						</Button>
						<Button variant="destructive" size="icon" onclick={() => remove(index)} title="Delete">
							<Icon icon="solar:trash-bin-trash-linear" size={14} />
						</Button>
					</div>
				</article>
			{/each}
		</div>
	{/if}
</div>

<Sheet
	bind:open={sheetOpen}
	title={editing ? `Edit ${editing.name}` : 'Edit target'}
	description={editing ? targetLabel(editing.type) : ''}
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
		<Button variant="outline" onclick={sendTest} disabled={testing}>
			<Icon icon="solar:plain-linear" size={15} />
			{testing ? 'Sending…' : 'Send a test'}
		</Button>
		<Button onclick={commit} disabled={dashboard.saving}>
			<Icon icon="solar:diskette-linear" size={15} />
			{dashboard.saving ? 'Saving…' : 'Save'}
		</Button>
	{/snippet}
</Sheet>
