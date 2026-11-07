<script lang="ts">
	import { Checkbox, Field, Input, SecretField, Switch, cn } from '@facile/muse';
	import StringListInput from './StringListInput.svelte';
	import TagInput from './TagInput.svelte';
	import { t } from '$lib/i18n/index.svelte';
	import {
		providerLabel,
		SYSTEM_PROVIDER_ID,
		type DeliveryTarget,
		type Provider
	} from '$lib/types';

	let {
		target = $bindable(),
		providers = [],
		availableTags = [],
		tones = undefined
	}: {
		target: DeliveryTarget;
		providers?: Provider[];
		availableTags?: string[];
		tones?: Record<string, number>;
	} = $props();

	const options = $derived([
		{ id: SYSTEM_PROVIDER_ID, name: t('target.system'), typeLabel: t('target.systemType') },
		...providers.map((provider) => ({
			id: provider.id,
			name: provider.name || providerLabel(provider.type),
			typeLabel: providerLabel(provider.type)
		}))
	]);

	function toggleProvider(id: string) {
		target.providerIds = target.providerIds.includes(id)
			? target.providerIds.filter((current) => current !== id)
			: [...target.providerIds, id];
	}
</script>

<div class="flex flex-col gap-4">
	<Switch label={t('common.enabled')} bind:checked={target.enabled} />

	<Field label={t('provider.name')}>
		<Input bind:value={target.name} />
	</Field>

	<!-- A checkbox group is a fieldset, not a Field: Field emits a <label for> that only a
	     single control can answer, and here there are as many controls as there are providers. -->
	<fieldset class="flex flex-col gap-1.5">
		<legend class="mb-1.5 text-fc-sm text-fc-fg">{t('target.providers')}</legend>

		<div class="flex flex-col gap-2">
			{#each options as option (option.id)}
				{@const selected = target.providerIds.includes(option.id)}
				<div
					class={cn(
						'flex min-h-11 items-center justify-between gap-3 rounded-fc-md px-3 py-2 transition-colors',
						selected ? 'bg-fc-accent text-fc-accent-fg' : 'bg-fc-surface'
					)}
				>
					<Checkbox
						label={option.name}
						checked={selected}
						onchange={() => toggleProvider(option.id)}
						class={cn('min-w-0 flex-1', selected && 'text-fc-accent-fg')}
					/>
					<span
						class={cn(
							'shrink-0 text-fc-xs',
							selected ? 'text-fc-accent-fg/70' : 'text-fc-fg-muted'
						)}
					>
						{option.typeLabel}
					</span>
				</div>
			{/each}
		</div>

		<span class="text-fc-xs text-fc-fg-muted">
			{t('target.routingHelp')}
		</span>
	</fieldset>

	<Field
		label={t('target.providerTags')}
		helper={t('target.providerTagsHelp')}
	>
		<TagInput bind:value={target.providerTags} suggestions={availableTags} {tones} />
	</Field>

	{#if target.type === 'matrix'}
		<Field label={t('target.homeserver')} helper={t('target.homeserverHelp')}>
			<Input
				type="url"
				bind:value={target.homeserver}
				placeholder="https://matrix.org"
				class="font-fc-mono"
			/>
		</Field>

		<SecretField
			editable
			copyable={false}
			label={t('target.accessToken')}
			helper={t('target.accessTokenHelp')}
			placeholder="syt_xxx"
			bind:value={target.accessToken}
		/>

		<Field label={t('target.userId')} helper={t('target.userIdHelp')}>
			<Input
				bind:value={target.userId}
				placeholder="@antenne:matrix.example.com"
				class="font-fc-mono"
			/>
		</Field>

		<SecretField editable copyable={false} label={t('target.password')} bind:value={target.password as string} />

		<Field
			label={t('target.deviceId')}
			helper={t('target.deviceIdHelp')}
		>
			<Input
				bind:value={target.deviceId as string}
				placeholder="ANTENNEABC123"
				class="font-fc-mono"
			/>
		</Field>

		<SecretField
			editable
			copyable={false}
			label={t('target.refreshToken')}
			helper={t('target.refreshTokenHelp')}
			placeholder="syr_xxx"
			bind:value={target.refreshToken as string}
		/>

		<Field label={t('target.roomIds')} helper={t('target.roomIdsHelp')}>
			<StringListInput
				bind:value={target.roomIds}
				placeholder="!room:matrix.example.com"
				addLabel={t('target.addRoom')}
			/>
		</Field>
	{:else if target.type === 'discord'}
		<SecretField
			editable
			copyable={false}
			label={t('target.webhookUrl')}
			helper={t('target.webhookUrlHelp')}
			placeholder="https://discord.com/api/webhooks/…"
			bind:value={target.webhookUrl}
		/>
	{:else if target.type === 'smtp'}
		<Field label={t('target.host')}>
			<Input bind:value={target.host} placeholder="smtp.example.com" required class="font-fc-mono" />
		</Field>

		<Field label={t('target.port')}>
			<Input type="number" min="1" step="1" bind:value={target.port} required />
		</Field>

		<div class="flex flex-col gap-1">
			<Switch label={t('target.implicitTls')} bind:checked={target.secure} />
			<p class="text-fc-xs text-fc-fg-muted">
				{t('target.implicitTlsHelp')}
			</p>
		</div>

		<Field label={t('target.username')}>
			<Input bind:value={target.username} autocomplete="off" />
		</Field>

		<SecretField editable copyable={false} label={t('target.password')} bind:value={target.password as string} />

		<Field label={t('target.fromAddress')}>
			<Input
				type="email"
				bind:value={target.fromAddress}
				placeholder="antenne@example.com"
				required
				class="font-fc-mono"
			/>
		</Field>

		<Field label={t('target.toAddresses')} helper={t('target.toAddressesHelp')}>
			<StringListInput
				bind:value={target.toAddresses}
				type="email"
				placeholder="ops@example.com"
				addLabel={t('target.addRecipient')}
			/>
		</Field>

		<Field label={t('target.subjectPrefix')}>
			<Input bind:value={target.subjectPrefix} placeholder="[Antenne]" />
		</Field>
	{:else if target.type === 'ntfy'}
		<Field label={t('target.serverUrl')}>
			<Input
				type="url"
				bind:value={target.serverUrl}
				placeholder="https://ntfy.sh"
				required
				class="font-fc-mono"
			/>
		</Field>

		<Field label={t('target.topic')}>
			<Input bind:value={target.topic} placeholder="my-alerts" required class="font-fc-mono" />
		</Field>

		<SecretField
			editable
			copyable={false}
			label={t('target.accessToken')}
			bind:value={target.accessToken as string}
		/>

		<Field label={t('target.priority')} helper={t('target.priorityHelp')}>
			<Input type="number" min="1" max="5" step="1" bind:value={target.priority as number} />
		</Field>

		<Field label={t('target.notificationTags')} helper={t('target.notificationTagsHelp')}>
			<TagInput bind:value={target.tags} placeholder="rotating_light" />
		</Field>

		<Field label={t('target.clickUrl')}>
			<Input type="url" bind:value={target.clickUrl as string} class="font-fc-mono" />
		</Field>
	{:else if target.type === 'perception'}
		<Field label={t('target.url')}>
			<Input type="url" bind:value={target.url} required class="font-fc-mono" />
		</Field>

		<Field label={t('target.workspace')}>
			<Input bind:value={target.workspace} required class="font-fc-mono" />
		</Field>

		<SecretField
			editable
			copyable={false}
			label={t('target.signingSecret')}
			helper={t('target.signingSecretRequired')}
			bind:value={target.secretValue as string}
		/>

		<div class="flex flex-col gap-1">
			<Switch label={t('target.rawPayload')} bind:checked={target.includeRawPayload} />
			<p class="text-fc-xs text-fc-fg-muted">
				{t('target.rawPayloadHelp')}
			</p>
		</div>
	{:else}
		<Field label={t('target.url')}>
			<Input
				type="url"
				bind:value={target.url}
				placeholder="https://example.com/hooks/antenne"
				required
				class="font-fc-mono"
			/>
		</Field>

		<SecretField
			editable
			copyable={false}
			label={t('target.signingSecret')}
			helper={t('target.signingSecretHelp')}
			bind:value={target.secretValue as string}
		/>

		<div class="flex flex-col gap-1">
			<Switch label={t('target.rawPayload')} bind:checked={target.includeRawPayload} />
			<p class="text-fc-xs text-fc-fg-muted">
				{t('target.rawPayloadHelp')}
			</p>
		</div>
	{/if}
</div>
