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

	const val = $derived(value ?? []);

	const unused = $derived(
		suggestions.filter((tag) => !val.includes(tag) && tag.toLowerCase().includes(draft.toLowerCase()))
	);

	function add(tag: string) {
		const trimmed = tag.trim();
		if (!trimmed || val.includes(trimmed)) {
			draft = '';
			return;
		}
		value = [...val, trimmed];
		draft = '';
	}

	function remove(tag: string) {
		value = val.filter((candidate) => candidate !== tag);
	}

	function onKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter' || event.key === ',') {
			event.preventDefault();
			add(draft);
			return;
		}
		if (event.key === 'Backspace' && draft === '' && val.length > 0) {
			remove(val[val.length - 1]);
		}
	}
</script>

<div class="flex flex-col gap-2">
	{#if val.length > 0}
		<div class="flex flex-wrap gap-2">
			{#each val as tag (tag)}
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
