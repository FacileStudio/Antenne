<script lang="ts">
	import { Field, Input, SecretField, Switch } from '@facile/muse';
	import TagInput from './TagInput.svelte';
	import { t } from '$lib/i18n/index.svelte';
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
		<Switch label={t('common.enabled')} bind:checked={provider.enabled} />
		<p class="text-fc-xs text-fc-fg-muted">
			{t('provider.enabledHelp')}
		</p>
	</div>

	<Field label={t('provider.name')}>
		<Input bind:value={provider.name} />
	</Field>

	<Field
		label={t('provider.tags')}
		helper={t('provider.tagsHelp')}
	>
		<TagInput bind:value={provider.tags} suggestions={availableTags} {tones} />
	</Field>

	{#if provider.type === 'website'}
		<Field label={t('provider.url')}>
			<Input type="url" required bind:value={provider.url} placeholder="https://example.com" />
		</Field>

		<Field label={t('provider.checkInterval')} helper={t('provider.checkIntervalHelp')}>
			<Input type="number" min="1" step="1" bind:value={provider.interval} />
		</Field>
	{:else if provider.type === 'rss'}
		<Field label={t('provider.feedUrl')}>
			<Input
				type="url"
				required
				bind:value={provider.feedUrl}
				placeholder="https://example.com/feed.xml"
			/>
		</Field>

		<Field
			label={t('provider.pollInterval')}
			helper={t('provider.pollIntervalRss')}
		>
			<Input type="number" min="10" step="1" bind:value={provider.pollInterval} />
		</Field>

		<Switch label={t('provider.fullContent')} bind:checked={provider.includeContent} />
	{:else if provider.type === 'imap'}
		<Field label={t('provider.host')}>
			<Input required bind:value={provider.host} placeholder="imap.example.com" />
		</Field>

		<Field label={t('provider.port')}>
			<Input type="number" required min="1" step="1" bind:value={provider.port} />
		</Field>

		<div class="flex flex-col gap-1">
			<Switch label={t('provider.implicitTls')} bind:checked={provider.secure} />
			<p class="text-fc-xs text-fc-fg-muted">
				{t('provider.implicitTlsHelp')}
			</p>
		</div>

		<Field label={t('provider.username')}>
			<Input required bind:value={provider.username} autocomplete="off" />
		</Field>

		<!-- `SecretField` round-trips the redacted placeholder untouched, which is what tells the
		     API to keep the stored password. -->
		<SecretField editable label={t('provider.password')} bind:value={provider.password as string} />

		<Field label={t('provider.pollInterval')} helper={t('provider.pollIntervalImap')}>
			<Input type="number" min="1" step="1" bind:value={provider.pollInterval} />
		</Field>

		<Switch label={t('provider.emailBody')} bind:checked={provider.includeEmailBody} />
	{:else}
		<Field
			label={t('provider.path')}
			helper={t('provider.pathHelp')}
		>
			<Input required bind:value={provider.path} class="font-fc-mono" />
		</Field>

		<Field label={t('provider.secretHeader')}>
			<Input bind:value={provider.secretHeader as string} class="font-fc-mono" />
		</Field>

		<SecretField
			editable
			label={t('provider.secretValue')}
			helper={t('provider.secretValueHelp')}
			bind:value={provider.secretValue as string}
		/>
	{/if}
</div>
