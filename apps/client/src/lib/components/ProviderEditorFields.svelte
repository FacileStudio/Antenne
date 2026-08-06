<script lang="ts">
	import SecretInput from './SecretInput.svelte';
	import TagInput from './TagInput.svelte';
	import ToggleRow from './ToggleRow.svelte';
	import Field from './ui/field.svelte';
	import Input from './ui/input.svelte';
	import type { Provider } from '$lib/types';

	/**
	 * Binds straight into a draft the parent owns. The sheet edits a copy and
	 * the parent commits it on save, so an abandoned edit changes nothing.
	 */
	let {
		provider = $bindable(),
		availableTags = [],
		tones = undefined
	}: {
		provider: Provider;
		availableTags?: string[];
		tones?: Record<string, number>;
	} = $props();
</script>

<ToggleRow
	label="Enabled"
	bind:checked={provider.enabled}
	hint="A disabled provider stops polling and stops answering on its path."
/>

<Field label="Name">
	<Input bind:value={provider.name} />
</Field>

<Field label="Tags" hint="A delivery target can subscribe by tag instead of naming this provider.">
	<TagInput bind:value={provider.tags} suggestions={availableTags} {tones} />
</Field>

{#if provider.type === 'website'}
	<Field label="URL" required>
		<Input type="url" bind:value={provider.url} placeholder="https://example.com" />
	</Field>

	<Field label="Check interval" hint="In seconds. Two consecutive failures raise an alert.">
		<Input type="number" min="1" step="1" bind:value={provider.interval} />
	</Field>
{:else if provider.type === 'rss'}
	<Field label="Feed URL" required>
		<Input type="url" bind:value={provider.feedUrl} placeholder="https://example.com/feed.xml" />
	</Field>

	<Field
		label="Poll interval"
		hint="In seconds. The first poll only records what is already published, so a restart does not replay the feed."
	>
		<Input type="number" min="10" step="1" bind:value={provider.pollInterval} />
	</Field>

	<ToggleRow label="Include the full content" bind:checked={provider.includeContent} />
{:else if provider.type === 'imap'}
	<Field label="Host" required>
		<Input bind:value={provider.host} placeholder="imap.example.com" />
	</Field>

	<Field label="Port" required>
		<Input type="number" min="1" step="1" bind:value={provider.port} />
	</Field>

	<ToggleRow
		label="Implicit TLS"
		bind:checked={provider.secure}
		hint="Off still upgrades with STARTTLS when the server offers it."
	/>

	<Field label="Username" required>
		<Input bind:value={provider.username} autocomplete="off" />
	</Field>

	<Field label="Password" required>
		<SecretInput bind:value={provider.password as string} />
	</Field>

	<Field label="Poll interval" hint="In seconds.">
		<Input type="number" min="1" step="1" bind:value={provider.pollInterval} />
	</Field>

	<ToggleRow label="Include the email body" bind:checked={provider.includeEmailBody} />
{:else}
	<Field
		label="Path"
		required
		hint="The URL this provider receives on. It is registered in the sending system, so changing it means changing it there too."
	>
		<Input bind:value={provider.path} />
	</Field>

	<Field label="Secret header">
		<Input bind:value={provider.secretHeader as string} />
	</Field>

	<Field label="Secret value" hint="Empty accepts unauthenticated calls on this path.">
		<SecretInput bind:value={provider.secretValue as string} />
	</Field>
{/if}
