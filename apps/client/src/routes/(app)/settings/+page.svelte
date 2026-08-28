<script lang="ts">
	import { browser } from '$app/environment';
	import { Alert, Button, SettingsRow, SettingsSection, StatusDot, icons } from '@facile/muse';
	import { dashboard } from '$lib/dashboard.svelte';

	const origin = $derived(browser ? window.location.origin : '—');
</script>

<div class="flex flex-col gap-10">
	<SettingsSection title="Account" description="The identity this browser is signed in with.">
		<SettingsRow label="Signed in as">
			<span class="text-fc-sm font-medium text-fc-fg">
				{dashboard.name?.trim() || dashboard.username}
				{#if dashboard.email && dashboard.email !== (dashboard.name?.trim() || dashboard.username)}
					<span class="text-fc-fg-muted font-normal">({dashboard.email})</span>
				{/if}
			</span>
		</SettingsRow>

		{#if dashboard.role}
			<SettingsRow label="Role">
				<span class="text-fc-sm text-fc-fg">{dashboard.role}</span>
			</SettingsRow>
		{/if}

		<SettingsRow
			label="Authentication"
			description="Antenne authenticates via single sign-on and local credentials."
		>
			{#if dashboard.passwordRequired}
				<StatusDot tone="success" label="Password login enabled" />
			{:else}
				<StatusDot tone="success" label="Single sign-on active" />
			{/if}
		</SettingsRow>

		<SettingsRow label="Instance">
			<span class="font-fc-mono text-fc-sm text-fc-fg">{origin}</span>
		</SettingsRow>
	</SettingsSection>

	<SettingsSection title="Session" description="This browser's session on this instance.">
		<SettingsRow
			label="Log out"
			description="Ends the session and returns to the login screen. Unsaved settings drafts are discarded."
		>
			<Button variant="outline" icon={icons.logout} onclick={() => dashboard.logout()}>
				Log out
			</Button>
		</SettingsRow>
	</SettingsSection>
</div>
