<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		eyebrow = '',
		title,
		actions
	}: { eyebrow?: string; title: string; actions?: Snippet } = $props();
</script>

<header class="flex flex-wrap items-end justify-between gap-4">
	<div>
		{#if eyebrow}
			<p class="mb-1 text-xs text-muted-foreground">{eyebrow}</p>
		{/if}
		<h2 class="font-heading text-xl font-semibold tracking-tight">{title}</h2>
	</div>
	{#if actions}
		<div class="flex flex-wrap items-center gap-2">{@render actions()}</div>
	{/if}
</header>
