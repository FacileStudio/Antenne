<script lang="ts">
	import { Button, Card, cn, icons } from '@facile/muse';
	import ProviderEnableSwitch from './ProviderEnableSwitch.svelte';
	import TagPill from './TagPill.svelte';
	import { providerIcon } from '$lib/icons';
	import { providerLabel, type Provider } from '$lib/types';

	let {
		entries,
		tones,
		onToggle,
		onOpen,
		onRemove
	}: {
		entries: { provider: Provider; index: number }[];
		tones: Record<string, number>;
		onToggle: (index: number, enabled: boolean) => void;
		onOpen: (index: number) => void;
		onRemove: (index: number) => void;
	} = $props();
</script>

<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
	{#each entries as { provider, index } (provider.id)}
		<a href={`/providers/${provider.id}`} class="block">
			<Card class={cn('flex flex-col gap-4', !provider.enabled && 'opacity-60')}>
				<div class="flex items-start justify-between gap-4">
					<span
						class="flex size-10 shrink-0 items-center justify-center rounded-fc-md bg-fc-surface text-fc-fg-muted"
					>
						<iconify-icon icon={providerIcon(provider.type)} width="22" height="22" class="block"
						></iconify-icon>
					</span>
					<ProviderEnableSwitch {provider} {index} {onToggle} />
				</div>

				<div class="flex min-w-0 flex-col gap-1">
					<span class="truncate text-fc-sm font-semibold text-fc-fg">{provider.name}</span>
					<span class="text-fc-xs text-fc-fg-muted">{providerLabel(provider.type)}</span>
				</div>

				{#if provider.tags.length > 0}
					<div class="flex flex-wrap gap-2">
						{#each provider.tags as tag (tag)}
							<TagPill {tag} {tones} />
						{/each}
					</div>
				{/if}

				<div class="mt-auto flex items-center gap-1 border-t border-fc-border pt-4">
					<Button
						variant="ghost"
						icon={icons.edit}
						class="flex-1"
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
						icon={icons.remove}
						onclick={(e) => {
							e.preventDefault();
							e.stopPropagation();
							onRemove(index);
						}}
						aria-label={`Delete ${provider.name}`}
					/>
				</div>
			</Card>
		</a>
	{/each}
</div>
