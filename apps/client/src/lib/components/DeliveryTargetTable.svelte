<script lang="ts">
	import { Badge, Button, Switch, Table, cn, icons } from '@facile/muse';
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
							bind:checked={() => target.enabled, (enabled) => onToggle(index, enabled)}
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
						onCheck={() => onCheck(target)}
					/>
				</td>
				<td>
					<div class="flex items-center justify-end gap-1">
						<Button variant="ghost" size="sm" icon={icons.edit} onclick={() => onOpen(index)}>
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
							onclick={() => onRemove(target)}
							title="Delete"
							aria-label={`Delete ${target.name}`}
						/>
					</div>
				</td>
			</tr>
		{/each}
	</tbody>
</Table>
