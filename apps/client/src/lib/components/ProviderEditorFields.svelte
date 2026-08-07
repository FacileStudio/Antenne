<script lang="ts">
	import { Field, Input, SecretField, Switch } from '@facile/muse';
	import TagInput from './TagInput.svelte';
	import type { Provider } from '$lib/types';

	/**
	 * Binds straight into a draft the parent owns. The drawer edits a copy and
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

<div class="flex flex-col gap-4">
	<div class="flex flex-col gap-1">
		<Switch label="Enabled" bind:checked={provider.enabled} />
		<p class="text-fc-xs text-fc-fg-muted">
			A disabled provider stops polling and stops answering on its path.
		</p>
	</div>

	<Field label="Name">
		<Input bind:value={provider.name} />
	</Field>

	<Field
		label="Tags"
		helper="A delivery target can subscribe by tag instead of naming this provider."
	>
		<TagInput bind:value={provider.tags} suggestions={availableTags} {tones} />
	</Field>

	{#if provider.type === 'website'}
		<Field label="URL">
			<Input type="url" required bind:value={provider.url} placeholder="https://example.com" />
		</Field>

		<Field label="Check interval" helper="In seconds. Two consecutive failures raise an alert.">
			<Input type="number" min="1" step="1" bind:value={provider.interval} />
		</Field>
	{:else if provider.type === 'rss'}
		<Field label="Feed URL">
			<Input
				type="url"
				required
				bind:value={provider.feedUrl}
				placeholder="https://example.com/feed.xml"
			/>
		</Field>

		<Field
			label="Poll interval"
			helper="In seconds. The first poll only records what is already published, so a restart does not replay the feed."
		>
			<Input type="number" min="10" step="1" bind:value={provider.pollInterval} />
		</Field>

		<Switch label="Include the full content" bind:checked={provider.includeContent} />
	{:else if provider.type === 'imap'}
		<Field label="Host">
			<Input required bind:value={provider.host} placeholder="imap.example.com" />
		</Field>

		<Field label="Port">
			<Input type="number" required min="1" step="1" bind:value={provider.port} />
		</Field>

		<div class="flex flex-col gap-1">
			<Switch label="Implicit TLS" bind:checked={provider.secure} />
			<p class="text-fc-xs text-fc-fg-muted">
				Off still upgrades with STARTTLS when the server offers it.
			</p>
		</div>

		<Field label="Username">
			<Input required bind:value={provider.username} autocomplete="off" />
		</Field>

		<!-- `SecretField` round-trips the redacted placeholder untouched, which is what tells the
		     API to keep the stored password. -->
		<SecretField editable label="Password" bind:value={provider.password as string} />

		<Field label="Poll interval" helper="In seconds.">
			<Input type="number" min="1" step="1" bind:value={provider.pollInterval} />
		</Field>

		<Switch label="Include the email body" bind:checked={provider.includeEmailBody} />
	{:else}
		<Field
			label="Path"
			helper="The URL this provider receives on. It is registered in the sending system, so changing it means changing it there too."
		>
			<Input required bind:value={provider.path} class="font-fc-mono" />
		</Field>

		<Field label="Secret header">
			<Input bind:value={provider.secretHeader as string} class="font-fc-mono" />
		</Field>

		<SecretField
			editable
			label="Secret value"
			helper="Empty accepts unauthenticated calls on this path."
			bind:value={provider.secretValue as string}
		/>
	{/if}
</div>
