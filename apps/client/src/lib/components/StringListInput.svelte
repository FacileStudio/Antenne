<script lang="ts">
	import Icon from './Icon.svelte';
	import Button from './ui/button.svelte';
	import Input from './ui/input.svelte';

	/** An ordered list of free-text values: Matrix room ids, SMTP recipients. */
	let {
		value = $bindable<string[]>([]),
		placeholder = '',
		addLabel = 'Add',
		type = 'text'
	}: {
		value?: string[];
		placeholder?: string;
		addLabel?: string;
		type?: 'text' | 'email';
	} = $props();

	function update(index: number, next: string) {
		value = value.map((current, position) => (position === index ? next : current));
	}

	function add() {
		value = [...value, ''];
	}

	function remove(index: number) {
		value = value.filter((_, position) => position !== index);
	}
</script>

<div class="grid gap-2">
	{#each value as entry, index (index)}
		<div class="flex gap-2">
			<Input
				{type}
				{placeholder}
				value={entry}
				oninput={(event) => update(index, (event.currentTarget as HTMLInputElement).value)}
			/>
			<Button
				variant="ghost"
				size="icon"
				onclick={() => remove(index)}
				aria-label="Remove this entry"
				class="hover:text-destructive"
			>
				<Icon icon="solar:trash-bin-trash-linear" size={15} />
			</Button>
		</div>
	{/each}

	<Button variant="outline" size="sm" onclick={add} class="w-fit">
		<Icon icon="solar:add-circle-linear" size={14} />
		{addLabel}
	</Button>
</div>
