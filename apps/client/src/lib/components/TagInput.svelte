<script lang="ts">
	import Icon from './Icon.svelte';
	import TagPill from './TagPill.svelte';
	import Input from './ui/input.svelte';

	let {
		value = $bindable<string[]>([]),
		suggestions = [],
		tones = undefined,
		placeholder = 'Type a tag and press Enter'
	}: {
		value?: string[];
		suggestions?: string[];
		tones?: Record<string, number>;
		placeholder?: string;
	} = $props();

	let draft = $state('');

	const unused = $derived(
		suggestions.filter((tag) => !value.includes(tag) && tag.toLowerCase().includes(draft.toLowerCase()))
	);

	function add(tag: string) {
		const trimmed = tag.trim();
		if (!trimmed || value.includes(trimmed)) {
			draft = '';
			return;
		}
		value = [...value, trimmed];
		draft = '';
	}

	function remove(tag: string) {
		value = value.filter((candidate) => candidate !== tag);
	}

	function onKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter' || event.key === ',') {
			event.preventDefault();
			add(draft);
			return;
		}
		// Backspace on an empty field removes the last tag, which is what every
		// other tag input does and what the fingers expect.
		if (event.key === 'Backspace' && draft === '' && value.length > 0) {
			remove(value[value.length - 1]);
		}
	}
</script>

<div class="grid gap-2">
	{#if value.length > 0}
		<div class="flex flex-wrap gap-1.5">
			{#each value as tag (tag)}
				<TagPill {tag} {tones} class="gap-1 pr-1">
					{tag}
					<button
						type="button"
						class="rounded-full p-0.5 opacity-60 transition-opacity hover:opacity-100"
						onclick={() => remove(tag)}
						aria-label={`Remove the ${tag} tag`}
					>
						<Icon icon="solar:close-circle-linear" size={11} />
					</button>
				</TagPill>
			{/each}
		</div>
	{/if}

	<Input bind:value={draft} {placeholder} onkeydown={onKeydown} onblur={() => add(draft)} />

	{#if unused.length > 0}
		<div class="flex flex-wrap gap-1.5">
			{#each unused.slice(0, 12) as tag (tag)}
				<TagPill {tag} {tones} interactive onclick={() => add(tag)} />
			{/each}
		</div>
	{/if}
</div>
