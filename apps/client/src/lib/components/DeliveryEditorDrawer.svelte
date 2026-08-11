<script lang="ts">
	import { Button, Drawer, icons, toast } from '@facile/muse';
	import DeliveryEditorFields from './DeliveryEditorFields.svelte';
	import { api } from '$lib/backend';
	import { targetLabel, type DeliveryTarget, type Provider } from '$lib/types';

	let {
		open = $bindable(),
		target = $bindable(),
		providers,
		availableTags,
		tones,
		saving,
		onSave
	}: {
		open: boolean;
		target: DeliveryTarget | null;
		providers: Provider[];
		availableTags: string[];
		tones: Record<string, number>;
		saving: boolean;
		onSave: () => void;
	} = $props();

	let testing = $state(false);

	async function sendTest() {
		if (!target) return;
		testing = true;
		try {
			await api.testTarget(target.id);
			toast.success(`Test delivered to ${target.name}.`);
		} catch (error) {
			toast.danger(error instanceof Error ? error.message : 'The test delivery failed');
		} finally {
			testing = false;
		}
	}
</script>

<!-- The editor works on a detached clone: nothing the drawer touches reaches the settings
     document until the parent writes it back and saves. -->
<Drawer
	bind:open
	showClose
	title={target ? `Edit ${target.name}` : 'Edit target'}
	description={target ? targetLabel(target.type) : undefined}
>
	{#if target}
		<DeliveryEditorFields bind:target {providers} {availableTags} {tones} />
	{/if}

	{#snippet footer()}
		<div class="flex flex-wrap justify-end gap-2">
			<Button variant="outline" icon="solar:plain-linear" onclick={sendTest} disabled={testing}>
				{testing ? 'Sending…' : 'Send a test'}
			</Button>
			<Button icon={icons.check} onclick={onSave} disabled={saving}>
				{saving ? 'Saving…' : 'Save'}
			</Button>
		</div>
	{/snippet}
</Drawer>
