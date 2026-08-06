<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '$lib/utils';

	let {
		class: className = '',
		children
	}: { class?: string; children?: Snippet } = $props();
</script>

<div class={cn('rounded-xl border border-border bg-card text-card-foreground', className)}>
	{@render children?.()}
</div>
