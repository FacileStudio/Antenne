<script lang="ts">
	import { Badge, Button, Card, cn, icons } from '@facile/muse';
	import ProviderEnableSwitch from './ProviderEnableSwitch.svelte';
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
		onRemove: (index: number) => void;
	} = $props();
</script>

<Card class="p-0">
	<ul class="flex flex-col gap-1">
		{#each entries as { provider, index } (provider.id)}
			<li
				class={cn(
					'flex flex-wrap items-center justify-between gap-4 px-5 py-4',
					!provider.enabled && 'opacity-60'
				)}
			>
				<div class="flex min-w-0 flex-wrap items-center gap-3">
					<ProviderEnableSwitch {provider} {index} {onToggle} />
					<iconify-icon
						icon={providerIcon(provider.type)}
						width="20"
						height="20"
						class="block shrink-0 text-fc-fg-muted"
					></iconify-icon>
					<span class="truncate text-fc-sm font-semibold text-fc-fg">{provider.name}</span>
					<Badge>{providerLabel(provider.type)}</Badge>
					{#each provider.tags as tag (tag)}
						<TagPill {tag} {tones} />
					{/each}
				</div>

				<div class="flex items-center gap-1">
					<Button variant="ghost" icon={icons.edit} onclick={() => onOpen(index)}>{t('common.edit')}</Button>
					<Button
						variant="ghost"
						icon={icons.dashboard}
						href={`/providers/${provider.id}`}
						aria-label={t('common.activityFor', { name: provider.name })}
					/>
					<Button
						variant="ghost-danger"
						icon={icons.remove}
						onclick={() => onRemove(index)}
						aria-label={t('common.deleteName', { name: provider.name })}
					/>
				</div>
			</li>
		{/each}
	</ul>
</Card>
