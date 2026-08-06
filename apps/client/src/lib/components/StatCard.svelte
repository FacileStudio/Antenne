<script lang="ts">
	import { cn } from '$lib/utils';

	let {
		label,
		value,
		tone = 'default',
		live = false
	}: {
		label: string;
		value: string | number;
		tone?: 'default' | 'success' | 'destructive';
		live?: boolean;
	} = $props();
</script>

<div class="grid gap-1 rounded-xl border border-border bg-card px-4 py-3">
	<span class="text-xs text-muted-foreground">{label}</span>
	<span
		class={cn(
			'flex items-center gap-2 text-2xl leading-tight font-semibold tabular',
			tone === 'success' && 'text-success',
			tone === 'destructive' && 'text-destructive'
		)}
	>
		{#if live}
			<span class="relative flex size-2">
				<span class="absolute inline-flex size-full animate-ping rounded-full bg-success opacity-75"
				></span>
				<span class="relative inline-flex size-2 rounded-full bg-success"></span>
			</span>
		{/if}
		{value}
	</span>
</div>
