<script lang="ts">
	import Switch from './ui/switch.svelte';

	let {
		checked = $bindable(false),
		label,
		hint = ''
	}: { checked?: boolean; label: string; hint?: string } = $props();
</script>

<label
	class="flex cursor-pointer items-center justify-between gap-4 rounded-lg border border-border bg-muted/30 px-4 py-3 transition-colors hover:border-foreground/20 hover:bg-muted/60"
>
	<span class="grid gap-0.5">
		<span class="text-sm font-medium">{label}</span>
		{#if hint}
			<span class="text-xs text-muted-foreground">{hint}</span>
		{/if}
	</span>
	<Switch bind:checked aria-label={label} />
</label>
