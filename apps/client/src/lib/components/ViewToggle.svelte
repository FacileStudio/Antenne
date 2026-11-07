<script lang="ts">
	import { OptionCards } from '@facile/muse';
	import { t } from '$lib/i18n/index.svelte';

	/**
	 * Kept as a component rather than inlining `OptionCards`: the delivery list uses the same
	 * toggle, and the two lists have to agree on the labels and on the mode names.
	 */
	let { value = $bindable<'list' | 'card'>('card') }: { value?: 'list' | 'card' } = $props();

	const options = $derived([
		{ value: 'list', label: t('common.list'), icon: 'solar:list-linear' },
		{ value: 'card', label: t('common.cards'), icon: 'solar:widget-linear' }
	]);
</script>

<OptionCards
	{options}
	{value}
	label={t('common.viewMode')}
	onSelect={(next) => (value = next as 'list' | 'card')}
/>
