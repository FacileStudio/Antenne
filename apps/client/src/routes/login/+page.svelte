<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { Alert, AuthScreen } from '@facile/muse';
	import { api } from '$lib/backend';
	import AntenneLogo from '$lib/components/AntenneLogo.svelte';
	import { rememberLocale, setSessionLocale, t } from '$lib/i18n/index.svelte';
	import { rememberReturn } from '$lib/return-path';

	let failed = $state(false);

	onMount(async () => {
		failed = page.url.searchParams.has('error');
		rememberReturn(page.url.searchParams.get('next'));
		try {
			const session = await api.session();
			if (session.authenticated) {
				setSessionLocale(session.locale ?? null);
				await goto('/providers', { replaceState: true });
			}
		} catch {}
	});
</script>

<svelte:head>
	<title>{t('login.title')}</title>
</svelte:head>

<AuthScreen appName="Antenne" href="/api/auth/oidc" onLocaleChange={rememberLocale}>
	{#snippet logo()}
		<AntenneLogo size={40} class="text-fc-fg" />
	{/snippet}
	{#snippet tagline()}
		{t('login.tagline')}
	{/snippet}
	{#snippet notice()}
		{#if failed}
			<Alert tone="danger">{t('login.failed')}</Alert>
		{/if}
	{/snippet}
</AuthScreen>
