<script lang="ts">
	import { Button, Card, Switch, cn, icons } from '@facile/muse';
	import TagPill from './TagPill.svelte';
	import { t } from '$lib/i18n/index.svelte';
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
		onRemove?: (index: number) => void;
	} = $props();
</script>

<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
	{#each entries as { provider, index } (provider.id)}
		<Card
			href={`/providers/${provider.id}`}
			class={cn('flex flex-col gap-4', !provider.enabled && 'opacity-55', 'min-h-[200px]')}
		>
			<div class="flex items-start justify-between gap-3">
				<span
					class="flex size-10 shrink-0 items-center justify-center rounded-fc-md bg-fc-surface text-fc-fg-muted"
				>
					<iconify-icon icon={providerIcon(provider.type)} width="20" height="20" class="block"
					></iconify-icon>
				</span>
				<Switch
					label={t('common.enabled')}
					bind:checked={() => provider.enabled, (enabled) => onToggle(index, enabled)}
				/>
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

			<div class="mt-auto flex items-center">
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
					{t('common.edit')}
				</Button>
			</div>
		</Card>
	{/each}
</div>
