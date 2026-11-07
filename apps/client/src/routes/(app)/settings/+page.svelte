<script lang="ts">
	import { browser } from '$app/environment';
	import { Alert, Button, SettingsRow, SettingsSection, StatusDot, icons } from '@facile/muse';
	import { dashboard } from '$lib/dashboard.svelte';
	import { t } from '$lib/i18n/index.svelte';

	const origin = $derived(browser ? window.location.origin : '—');
</script>

<div class="flex flex-col gap-10">
	<SettingsSection title={t('profile.account')} description={t('profile.accountText')}>
		<SettingsRow label={t('profile.signedInAs')}>
			<span class="text-fc-sm font-medium text-fc-fg">
				{dashboard.name?.trim() || dashboard.username}
				{#if dashboard.email && dashboard.email !== (dashboard.name?.trim() || dashboard.username)}
					<span class="text-fc-fg-muted font-normal">({dashboard.email})</span>
				{/if}
			</span>
		</SettingsRow>

		{#if dashboard.role}
			<SettingsRow label={t('profile.role')}>
				<span class="text-fc-sm text-fc-fg">{dashboard.role}</span>
			</SettingsRow>
		{/if}

		<SettingsRow label={t('profile.authentication')} description={t('profile.authenticationText')}>
			<StatusDot tone="success" label={t('profile.sso')} />
		</SettingsRow>

		<SettingsRow label={t('profile.instance')}>
			<span class="font-fc-mono text-fc-sm text-fc-fg">{origin}</span>
		</SettingsRow>
	</SettingsSection>

	<SettingsSection title={t('profile.session')} description={t('profile.sessionText')}>
		<SettingsRow
			label={t('profile.logout')}
			description={t('profile.logoutText')}
		>
			<Button variant="outline" icon={icons.logout} onclick={() => dashboard.logout()}>
				{t('profile.logout')}
			</Button>
		</SettingsRow>
	</SettingsSection>
</div>
