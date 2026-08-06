<script lang="ts">
	import Icon from './Icon.svelte';
	import { cn } from '$lib/utils';

	let { value = $bindable<'list' | 'card'>('card') }: { value?: 'list' | 'card' } = $props();

	const modes = [
		{ key: 'list' as const, icon: 'solar:list-bold', label: 'List view' },
		{ key: 'card' as const, icon: 'solar:widget-bold', label: 'Card view' }
	];
</script>

<div class="inline-flex gap-0.5 rounded-lg border border-border bg-muted/50 p-0.5" role="radiogroup" aria-label="View mode">
	{#each modes as mode (mode.key)}
		<button
			type="button"
			role="radio"
			aria-checked={value === mode.key}
			title={mode.label}
			onclick={() => (value = mode.key)}
			class={cn(
				'inline-flex size-7 items-center justify-center rounded-md transition-colors',
				value === mode.key
					? 'bg-background text-foreground shadow-xs'
					: 'text-muted-foreground hover:text-foreground'
			)}
		>
			<Icon icon={mode.icon} size={15} />
		</button>
	{/each}
</div>
