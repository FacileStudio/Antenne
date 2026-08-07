<script lang="ts">
	import { OptionCards } from '@facile/muse';

	/**
	 * Kept as a component rather than inlining `OptionCards`: the delivery list uses the same
	 * toggle, and the two lists have to agree on the labels and on the mode names.
	 */
	let { value = $bindable<'list' | 'card'>('card') }: { value?: 'list' | 'card' } = $props();

	const options = [
		{ value: 'list', label: 'List', icon: 'solar:list-linear' },
		{ value: 'card', label: 'Cards', icon: 'solar:widget-linear' }
	];
</script>

<OptionCards
	{options}
	{value}
	label="View mode"
	onSelect={(next) => (value = next as 'list' | 'card')}
/>
