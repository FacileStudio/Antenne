<script lang="ts">
	import { ICON_DATA } from '$lib/icon-data';
	import { cn } from '$lib/utils';

	let {
		icon,
		size = 16,
		class: className = ''
	}: { icon: string; size?: number; class?: string } = $props();

	const data = $derived(ICON_DATA[icon]);
</script>

{#if data}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		viewBox={data.viewBox}
		width={size}
		height={size}
		class={cn('shrink-0', className)}
		aria-hidden="true"
		focusable="false"
	>
		<!-- The body comes from the generated icon set, never from user input. -->
		{@html data.body}
	</svg>
{/if}
