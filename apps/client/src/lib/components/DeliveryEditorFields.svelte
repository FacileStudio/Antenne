<script lang="ts">
	import { Checkbox, Field, Input, SecretField, Switch, cn } from '@facile/muse';
	import StringListInput from './StringListInput.svelte';
	import TagInput from './TagInput.svelte';
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
		{ id: SYSTEM_PROVIDER_ID, name: 'Antenne system', typeLabel: 'System' },
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
	<Switch label="Enabled" bind:checked={target.enabled} />

	<Field label="Name">
		<Input bind:value={target.name} />
	</Field>

	<!-- A checkbox group is a fieldset, not a Field: Field emits a <label for> that only a
	     single control can answer, and here there are as many controls as there are providers. -->
	<fieldset class="flex flex-col gap-1.5">
		<legend class="mb-1.5 text-fc-sm text-fc-fg">Providers</legend>

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
			Routing is opt-in with no fallthrough: a target with nothing selected here and no tag below
			receives nothing. Antenne's own events need “Antenne system”.
		</span>
	</fieldset>

	<Field
		label="Provider tags"
		helper="A provider carrying any of these tags reaches this target, even without being selected above."
	>
		<TagInput bind:value={target.providerTags} suggestions={availableTags} {tones} />
	</Field>

	{#if target.type === 'matrix'}
		<Field label="Homeserver" helper="Change it only if this target uses another homeserver.">
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
			label="Access token"
			helper="Optional when a password login is configured. Used until the homeserver rejects it."
			placeholder="syt_xxx"
			bind:value={target.accessToken}
		/>

		<Field label="User ID" helper="With a password, lets Antenne log back in on its own.">
			<Input
				bind:value={target.userId}
				placeholder="@antenne:matrix.example.com"
				class="font-fc-mono"
			/>
		</Field>

		<SecretField editable copyable={false} label="Password" bind:value={target.password as string} />

		<Field
			label="Device ID"
			helper="Optional. Keep it stable so the homeserver reissues for one device."
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
			label="Refresh token"
			helper="Optional. Used when the access token expires."
			placeholder="syr_xxx"
			bind:value={target.refreshToken as string}
		/>

		<Field label="Room IDs" helper="At least one, or this target has nowhere to post.">
			<StringListInput
				bind:value={target.roomIds}
				placeholder="!room:matrix.example.com"
				addLabel="Add room"
			/>
		</Field>
	{:else if target.type === 'discord'}
		<SecretField
			editable
			copyable={false}
			label="Webhook URL"
			helper="Required. The URL itself is the credential."
			placeholder="https://discord.com/api/webhooks/…"
			bind:value={target.webhookUrl}
		/>
	{:else if target.type === 'smtp'}
		<Field label="Host">
			<Input bind:value={target.host} placeholder="smtp.example.com" required class="font-fc-mono" />
		</Field>

		<Field label="Port">
			<Input type="number" min="1" step="1" bind:value={target.port} required />
		</Field>

		<div class="flex flex-col gap-1">
			<Switch label="Implicit TLS" bind:checked={target.secure} />
			<p class="text-fc-xs text-fc-fg-muted">
				Off connects in the clear and upgrades with STARTTLS when offered.
			</p>
		</div>

		<Field label="Username">
			<Input bind:value={target.username} autocomplete="off" />
		</Field>

		<SecretField editable copyable={false} label="Password" bind:value={target.password as string} />

		<Field label="From address">
			<Input
				type="email"
				bind:value={target.fromAddress}
				placeholder="antenne@example.com"
				required
				class="font-fc-mono"
			/>
		</Field>

		<Field label="To addresses" helper="At least one recipient.">
			<StringListInput
				bind:value={target.toAddresses}
				type="email"
				placeholder="ops@example.com"
				addLabel="Add recipient"
			/>
		</Field>

		<Field label="Subject prefix">
			<Input bind:value={target.subjectPrefix} placeholder="[Antenne]" />
		</Field>
	{:else if target.type === 'ntfy'}
		<Field label="Server URL">
			<Input
				type="url"
				bind:value={target.serverUrl}
				placeholder="https://ntfy.sh"
				required
				class="font-fc-mono"
			/>
		</Field>

		<Field label="Topic">
			<Input bind:value={target.topic} placeholder="my-alerts" required class="font-fc-mono" />
		</Field>

		<SecretField
			editable
			copyable={false}
			label="Access token"
			bind:value={target.accessToken as string}
		/>

		<Field label="Priority" helper="1 to 5. ntfy's default is 3.">
			<Input type="number" min="1" max="5" step="1" bind:value={target.priority as number} />
		</Field>

		<Field label="Notification tags" helper="ntfy renders these as emoji.">
			<TagInput bind:value={target.tags} placeholder="rotating_light" />
		</Field>

		<Field label="Click URL">
			<Input type="url" bind:value={target.clickUrl as string} class="font-fc-mono" />
		</Field>
	{:else if target.type === 'perception'}
		<Field label="URL">
			<Input type="url" bind:value={target.url} required class="font-fc-mono" />
		</Field>

		<Field label="Workspace">
			<Input bind:value={target.workspace} required class="font-fc-mono" />
		</Field>

		<SecretField
			editable
			copyable={false}
			label="Signing secret"
			helper="Required — Perception rejects unsigned events."
			bind:value={target.secretValue as string}
		/>

		<div class="flex flex-col gap-1">
			<Switch label="Include the raw payload" bind:checked={target.includeRawPayload} />
			<p class="text-fc-xs text-fc-fg-muted">
				Off strips the third-party body and says so in the envelope.
			</p>
		</div>
	{:else}
		<Field label="URL">
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
			label="Signing secret"
			helper="Signs timestamp.eventId.body as x-antenne-signature-256. Empty sends unsigned."
			bind:value={target.secretValue as string}
		/>

		<div class="flex flex-col gap-1">
			<Switch label="Include the raw payload" bind:checked={target.includeRawPayload} />
			<p class="text-fc-xs text-fc-fg-muted">
				Off strips the third-party body and says so in the envelope.
			</p>
		</div>
	{/if}
</div>
