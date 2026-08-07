<script lang="ts">
	import { browser } from '$app/environment';
	import { Alert, Button, SettingsRow, SettingsSection, StatusDot, icons } from '@facile/muse';
	import { dashboard } from '$lib/dashboard.svelte';

	const origin = $derived(browser ? window.location.origin : '—');
</script>

<div class="flex flex-col gap-10">
	<SettingsSection title="Account" description="The identity this browser is signed in with.">
		<SettingsRow label="Signed in as">
			<span class="text-fc-sm font-medium text-fc-fg">{dashboard.username}</span>
		</SettingsRow>

		<SettingsRow
			label="Authentication"
			description="Antenne owns its own users; there is no shared identity provider yet."
		>
			{#if dashboard.passwordRequired}
				<StatusDot tone="success" label="Password required" />
			{:else}
				<StatusDot tone="warning" label="Open — no admin password" />
			{/if}
		</SettingsRow>

		<SettingsRow label="Instance">
			<span class="font-fc-mono text-fc-sm text-fc-fg">{origin}</span>
		</SettingsRow>
	</SettingsSection>

	{#if !dashboard.passwordRequired}
		<Alert tone="warning" title="This dashboard is unauthenticated">
			<code class="font-fc-mono">ANTENNE_ADMIN_PASSWORD</code> is unset, so the dashboard and its API
			answer to anyone who can reach this address. Set it and restart to close it — the restart also
			re-seeds the bootstrap account.
		</Alert>
	{/if}

	{#if dashboard.passwordRequired}
		<!-- Log out belongs here, not in the rail: CHARTE §14. An instance with no admin password
		     has no session to end, so the section only exists when one does. -->
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
	{/if}
</div>
