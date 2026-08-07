<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@facile/muse';
	import { tagTone } from '$lib/tags';

	/**
	 * Tags keep the twelve-hue `[data-tag-tone]` scale rather than a muse `Badge`: the hues are
	 * categorical identity, and muse only offers six semantic tones — which would make two
	 * unrelated tags look like the same category.
	 */
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

	const base = 'inline-flex items-center rounded-fc-pill px-2.5 text-fc-xs font-medium leading-none';
</script>

{#if interactive}
	<button
		type="button"
		data-tag-tone={tone}
		{onclick}
		aria-pressed={selected}
		class={cn(
			base,
			'h-8 cursor-pointer transition-opacity hover:opacity-80 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring',
			// The tint carries the tag's identity, so a selected pill cannot invert without
			// losing it — the ring is the only marker left that does not touch the hue.
			selected && 'outline-2 outline-offset-2 outline-fc-fg',
			className
		)}
	>
		{@render children?.()}{#if !children}{tag}{/if}
	</button>
{:else}
	<span data-tag-tone={tone} class={cn(base, 'h-6', className)}>
		{@render children?.()}{#if !children}{tag}{/if}
	</span>
{/if}
