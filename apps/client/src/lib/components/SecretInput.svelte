<script lang="ts">
	import Icon from './Icon.svelte';
	import Button from './ui/button.svelte';
	import Input from './ui/input.svelte';

	/**
	 * A password field with a reveal toggle. The value it holds is usually the
	 * redaction marker rather than the real secret — the API only sends those —
	 * and sending it back unchanged is what keeps the stored value.
	 */
	let {
		value = $bindable<string>(''),
		placeholder = ''
	}: { value?: string; placeholder?: string } = $props();

	let revealed = $state(false);
</script>

<div class="flex gap-2">
	<Input
		bind:value
		type={revealed ? 'text' : 'password'}
		{placeholder}
		autocomplete="off"
		spellcheck={false}
	/>
	<Button
		variant="outline"
		size="icon"
		onclick={() => (revealed = !revealed)}
		aria-label={revealed ? 'Hide the value' : 'Show the value'}
		title={revealed ? 'Hide' : 'Show'}
	>
		<Icon icon={revealed ? 'solar:eye-closed-linear' : 'solar:eye-linear'} size={15} />
	</Button>
</div>
