<script lang="ts">
	import type { HTMLLabelAttributes } from 'svelte/elements';
	import type { Snippet } from 'svelte';
	import { cn } from '$lib/utils';

	let {
		class: className = '',
		children,
		...rest
	}: { class?: string; children?: Snippet } & HTMLLabelAttributes = $props();
</script>

<label class={cn('text-sm font-medium leading-none', className)} {...rest}>
	{@render children?.()}
</label>
