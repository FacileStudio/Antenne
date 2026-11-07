<script lang="ts">
	import { Button, Input, icons } from '@facile/muse';
	import { t } from '$lib/i18n/index.svelte';

	/** An ordered list of free-text values: Matrix room ids, SMTP recipients. */
	let {
		value = $bindable<string[]>([]),
		placeholder = '',
		addLabel = undefined,
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

<div class="flex flex-col gap-2">
	{#each value as entry, index (index)}
		<div class="flex items-center gap-2">
			<Input
				{type}
				{placeholder}
				value={entry}
				oninput={(event) => update(index, (event.currentTarget as HTMLInputElement).value)}
			/>
			<!-- No confirmation here: this is one row of an unsaved draft field, not a stored
			     entity, and the shell's Save is still the only thing that persists it. -->
			<Button
				variant="ghost-danger"
				icon={icons.remove}
				onclick={() => remove(index)}
				aria-label={t('list.remove')}
			/>
		</div>
	{/each}

	<Button variant="secondary" size="sm" icon={icons.plus} onclick={add} class="w-fit">
		{addLabel ?? t('list.add')}
	</Button>
</div>
