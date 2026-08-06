<script lang="ts">
	import type { Snippet } from 'svelte';
	import Label from './label.svelte';
	import { cn } from '$lib/utils';

	let {
		label,
		hint = '',
		required = false,
		class: className = '',
		children
	}: {
		label: string;
		hint?: string;
		required?: boolean;
		class?: string;
		children?: Snippet;
	} = $props();
</script>

<div class={cn('grid gap-1.5', className)}>
	<Label>
		{label}
		{#if required}
			<span class="text-muted-foreground/60" aria-hidden="true">*</span>
		{/if}
	</Label>
	{#if hint}
		<p class="text-xs leading-relaxed text-muted-foreground">{hint}</p>
	{/if}
	{@render children?.()}
</div>
