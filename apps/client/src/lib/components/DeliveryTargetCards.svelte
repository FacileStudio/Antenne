<script lang="ts">
	import { Badge, Button, Card, Switch, cn, icons } from '@facile/muse';
	import TargetHealthRow from './TargetHealthRow.svelte';
	import { deliveryIcon } from '$lib/icons';
	import {
		isSilent,
		routingSummary,
		targetLabel,
		type DeliveryTarget,
		type TargetHealth
	} from '$lib/types';

	let {
		targets,
		health,
		checkingId,
		cooldown,
		onToggle,
		onOpen,
		onCheck,
		onRemove
	}: {
		targets: DeliveryTarget[];
		health: Record<string, TargetHealth>;
		checkingId: string | null;
		cooldown: number;
		onToggle: (index: number, enabled: boolean) => void;
		onOpen: (index: number) => void;
		onCheck: (target: DeliveryTarget) => void;
		onRemove: (target: DeliveryTarget) => void;
	} = $props();
</script>

<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
	{#each targets as target, index (target.id)}
		<Card
			href={`/targets/${target.id}`}
			class={cn('flex flex-col gap-4', !target.enabled && 'opacity-55')}
		>
				<div class="flex items-start justify-between gap-3">
					<span
						class="flex size-10 shrink-0 items-center justify-center rounded-fc-md bg-fc-surface text-fc-fg-muted"
					>
						<iconify-icon icon={deliveryIcon(target.type)} width="20" height="20" class="block"
						></iconify-icon>
					</span>
					<Switch
						label="Enabled"
						bind:checked={() => target.enabled, (enabled) => onToggle(index, enabled)}
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
					onCheck={() => onCheck(target)}
				/>

				<div class="mt-auto flex items-center gap-2">
					<Button
						variant="outline"
						size="sm"
						icon={icons.edit}
						onclick={(e) => {
							e.preventDefault();
							e.stopPropagation();
							onOpen(index);
						}}
					>
						Edit
					</Button>
					<Button
						variant="ghost-danger"
						size="sm"
						icon={icons.remove}
						onclick={(e) => {
							e.preventDefault();
							e.stopPropagation();
							onRemove(target);
						}}
						title="Delete"
						aria-label={`Delete ${target.name}`}
						class="ml-auto"
					/>
				</div>
			</Card>
	{/each}
</div>
