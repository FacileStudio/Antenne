<script lang="ts">
	import Icon from './Icon.svelte';
	import Button from './ui/button.svelte';
	import { theme } from '$lib/theme.svelte';
</script>

<Button
	variant="ghost"
	size="icon"
	onclick={() => theme.toggle()}
	aria-label={theme.current === 'dark' ? 'Switch to the light theme' : 'Switch to the dark theme'}
	title={theme.current === 'dark' ? 'Light theme' : 'Dark theme'}
>
	<Icon icon={theme.current === 'dark' ? 'solar:sun-linear' : 'solar:moon-linear'} size={16} />
</Button>
