<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Toaster } from 'svelte-sonner';
	import { theme } from '$lib/theme.svelte';
	import '../app.css';

	let { children }: { children: Snippet } = $props();

	$effect(() => {
		theme.hydrate();
	});
</script>

{@render children()}

<Toaster
	position="top-center"
	richColors={false}
	toastOptions={{
		classes: {
			toast:
				'!rounded-xl !border !border-border !bg-popover !text-popover-foreground !shadow-lg !font-sans',
			description: '!text-muted-foreground',
			error: '!text-destructive',
			success: '!text-success'
		}
	}}
/>
