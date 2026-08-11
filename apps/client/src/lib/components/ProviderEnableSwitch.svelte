<script lang="ts">
	import { Switch } from '@facile/muse';
	import type { Provider } from '$lib/types';

	/**
	 * `label` rather than `aria-label`: muse's Switch renders its own <label> element and the
	 * text span is the only accessible name it exposes, so the name is hidden visually instead
	 * of being dropped. The row toggle saves on the spot — the draft toggles inside the editor
	 * are the ones that wait for the shell's Save.
	 */
	let {
		provider,
		index,
		onToggle
	}: {
		provider: Provider;
		index: number;
		onToggle: (index: number, enabled: boolean) => void;
	} = $props();
</script>

<Switch
	label={`Enable ${provider.name}`}
	class="[&>span:last-child]:sr-only"
	bind:checked={() => provider.enabled, (enabled) => void onToggle(index, enabled)}
/>
