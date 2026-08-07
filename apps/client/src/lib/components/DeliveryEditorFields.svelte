<script lang="ts">
	import SecretInput from './SecretInput.svelte';
	import StringListInput from './StringListInput.svelte';
	import TagInput from './TagInput.svelte';
	import ToggleRow from './ToggleRow.svelte';
	import Field from './ui/field.svelte';
	import Input from './ui/input.svelte';
	import {
		providerLabel,
		SYSTEM_PROVIDER_ID,
		type DeliveryTarget,
		type Provider
	} from '$lib/types';
	import { cn } from '$lib/utils';

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

<ToggleRow label="Enabled" bind:checked={target.enabled} />

<Field label="Name">
	<Input bind:value={target.name} />
</Field>

<Field
	label="Providers"
	hint="Routing is opt-in with no fallthrough: a target with nothing selected here and no tag below receives nothing. Antenne's own events need “Antenne system”."
>
	<div class="grid gap-1.5">
		{#each options as option (option.id)}
			<label
				class={cn(
					'flex cursor-pointer items-center gap-3 rounded-lg border px-3 py-2 text-sm transition-colors',
					target.providerIds.includes(option.id)
						? 'border-foreground/30 bg-muted'
						: 'border-border hover:bg-muted/50'
				)}
			>
				<input
					type="checkbox"
					class="size-4 accent-foreground"
					checked={target.providerIds.includes(option.id)}
					onchange={() => toggleProvider(option.id)}
				/>
				<span class="grid min-w-0 gap-0.5">
					<span class="truncate font-medium">{option.name}</span>
					<span class="text-xs text-muted-foreground">{option.typeLabel}</span>
				</span>
			</label>
		{/each}
	</div>
</Field>

<Field
	label="Provider tags"
	hint="A provider carrying any of these tags reaches this target, even without being selected above."
>
	<TagInput bind:value={target.providerTags} suggestions={availableTags} {tones} />
</Field>

{#if target.type === 'matrix'}
	<Field label="Homeserver" hint="Change it only if this target uses another homeserver.">
		<Input type="url" bind:value={target.homeserver} placeholder="https://matrix.org" />
	</Field>

	<Field
		label="Access token"
		hint="Optional when a password login is configured. Used until the homeserver rejects it."
	>
		<SecretInput bind:value={target.accessToken} placeholder="syt_xxx" />
	</Field>

	<Field label="User ID" hint="With a password, lets Antenne log back in on its own.">
		<Input bind:value={target.userId} placeholder="@antenne:matrix.example.com" />
	</Field>

	<Field label="Password">
		<SecretInput bind:value={target.password as string} />
	</Field>

	<Field label="Device ID" hint="Optional. Keep it stable so the homeserver reissues for one device.">
		<Input bind:value={target.deviceId as string} placeholder="ANTENNEABC123" />
	</Field>

	<Field label="Refresh token" hint="Optional. Used when the access token expires.">
		<SecretInput bind:value={target.refreshToken as string} placeholder="syr_xxx" />
	</Field>

	<Field label="Room IDs" required>
		<StringListInput bind:value={target.roomIds} placeholder="!room:matrix.example.com" addLabel="Add room" />
	</Field>
{:else if target.type === 'discord'}
	<Field label="Webhook URL" required>
		<SecretInput bind:value={target.webhookUrl} placeholder="https://discord.com/api/webhooks/…" />
	</Field>
{:else if target.type === 'smtp'}
	<Field label="Host" required>
		<Input bind:value={target.host} placeholder="smtp.example.com" />
	</Field>

	<Field label="Port" required>
		<Input type="number" min="1" step="1" bind:value={target.port} />
	</Field>

	<ToggleRow
		label="Implicit TLS"
		bind:checked={target.secure}
		hint="Off connects in the clear and upgrades with STARTTLS when offered."
	/>

	<Field label="Username">
		<Input bind:value={target.username} autocomplete="off" />
	</Field>

	<Field label="Password">
		<SecretInput bind:value={target.password as string} />
	</Field>

	<Field label="From address" required>
		<Input type="email" bind:value={target.fromAddress} placeholder="antenne@example.com" />
	</Field>

	<Field label="To addresses" required>
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
	<Field label="Server URL" required>
		<Input type="url" bind:value={target.serverUrl} placeholder="https://ntfy.sh" />
	</Field>

	<Field label="Topic" required>
		<Input bind:value={target.topic} placeholder="my-alerts" />
	</Field>

	<Field label="Access token">
		<SecretInput bind:value={target.accessToken as string} />
	</Field>

	<Field label="Priority" hint="1 to 5. ntfy's default is 3.">
		<Input type="number" min="1" max="5" step="1" bind:value={target.priority as number} />
	</Field>

	<Field label="Notification tags" hint="ntfy renders these as emoji.">
		<TagInput bind:value={target.tags} placeholder="rotating_light" />
	</Field>

	<Field label="Click URL">
		<Input type="url" bind:value={target.clickUrl as string} />
	</Field>
{:else if target.type === 'perception'}
	<Field label="URL" required>
		<Input type="url" bind:value={target.url} />
	</Field>

	<Field label="Workspace" required>
		<Input bind:value={target.workspace} />
	</Field>

	<Field label="Signing secret" required hint="Perception rejects unsigned events.">
		<SecretInput bind:value={target.secretValue as string} />
	</Field>

	<ToggleRow
		label="Include the raw payload"
		bind:checked={target.includeRawPayload}
		hint="Off strips the third-party body and says so in the envelope."
	/>
{:else}
	<Field label="URL" required>
		<Input type="url" bind:value={target.url} placeholder="https://example.com/hooks/antenne" />
	</Field>

	<Field
		label="Signing secret"
		hint="Signs timestamp.eventId.body as x-antenne-signature-256. Empty sends unsigned."
	>
		<SecretInput bind:value={target.secretValue as string} />
	</Field>

	<ToggleRow
		label="Include the raw payload"
		bind:checked={target.includeRawPayload}
		hint="Off strips the third-party body and says so in the envelope."
	/>
{/if}
