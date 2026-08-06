<script lang="ts">
	import type { Snippet } from 'svelte';
	import { tagTone } from '$lib/tags';
	import { cn } from '$lib/utils';

	let {
		tag,
		tones = undefined,
		interactive = false,
		selected = false,
		class: className = '',
		onclick = undefined,
		children
	}: {
		tag: string;
		tones?: Record<string, number>;
		interactive?: boolean;
		selected?: boolean;
		class?: string;
		onclick?: () => void;
		children?: Snippet;
	} = $props();

	const tone = $derived(tagTone(tag, tones));
</script>

{#if interactive}
	<button
		type="button"
		data-tag-tone={tone}
		{onclick}
		aria-pressed={selected}
		class={cn(
			'inline-flex h-6 items-center rounded-full px-2 text-[0.7rem] leading-none font-medium transition-all',
			'hover:opacity-80',
			selected && 'ring-2 ring-foreground/40 ring-offset-1 ring-offset-background',
			className
		)}
	>
		{@render children?.()}{#if !children}{tag}{/if}
	</button>
{:else}
	<span
		data-tag-tone={tone}
		class={cn(
			'inline-flex h-5 items-center rounded-full px-2 text-[0.68rem] leading-none font-medium',
			className
		)}
	>
		{@render children?.()}{#if !children}{tag}{/if}
	</span>
{/if}
