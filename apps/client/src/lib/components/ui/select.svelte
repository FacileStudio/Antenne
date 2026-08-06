<script lang="ts">
	import type { HTMLSelectAttributes } from 'svelte/elements';
	import { cn } from '$lib/utils';

	/**
	 * A native select rather than a bits-ui listbox. Every use here is a short
	 * list of fixed options in a form, where the platform control is smaller,
	 * keyboard- and screen-reader-correct for free, and behaves properly on
	 * mobile. The styled listbox is worth it for a combobox; this is not one.
	 */
	let {
		value = $bindable<string>(),
		class: className = '',
		options,
		...rest
	}: {
		value?: string;
		class?: string;
		options: { value: string; label: string }[];
	} & HTMLSelectAttributes = $props();
</script>

<select
	bind:value
	class={cn(
		'flex h-9 w-full appearance-none rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs transition-colors',
		'bg-[length:1rem] bg-[position:right_0.6rem_center] bg-no-repeat pr-8',
		"bg-[url('data:image/svg+xml;utf8,<svg xmlns=%22http://www.w3.org/2000/svg%22 viewBox=%220 0 24 24%22 fill=%22none%22 stroke=%22%23888%22 stroke-width=%222%22 stroke-linecap=%22round%22><path d=%22M6 9l6 6 6-6%22/></svg>')]",
		'focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/25 focus-visible:outline-none',
		'disabled:cursor-not-allowed disabled:opacity-50',
		className
	)}
	{...rest}
>
	{#each options as option (option.value)}
		<option value={option.value}>{option.label}</option>
	{/each}
</select>
