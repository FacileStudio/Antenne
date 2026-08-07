<script lang="ts">
	import { Input, icons } from '@facile/muse';
	import TagPill from './TagPill.svelte';

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

<div class="flex flex-col gap-2">
	{#if value.length > 0}
		<div class="flex flex-wrap gap-2">
			{#each value as tag (tag)}
				<TagPill {tag} {tones} class="h-8 gap-1 pr-1.5">
					{tag}
					<button
						type="button"
						class="rounded-fc-pill p-1 opacity-60 transition-opacity hover:opacity-100 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
						onclick={() => remove(tag)}
						aria-label={`Remove the ${tag} tag`}
					>
						<iconify-icon icon={icons.close} width="12" height="12" class="block"></iconify-icon>
					</button>
				</TagPill>
			{/each}
		</div>
	{/if}

	<Input bind:value={draft} {placeholder} onkeydown={onKeydown} onblur={() => add(draft)} />

	{#if unused.length > 0}
		<div class="flex flex-wrap gap-2">
			{#each unused.slice(0, 12) as tag (tag)}
				<TagPill {tag} {tones} interactive onclick={() => add(tag)} />
			{/each}
		</div>
	{/if}
</div>
