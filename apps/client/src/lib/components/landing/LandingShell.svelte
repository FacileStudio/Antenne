<script lang="ts">
	import type { Snippet } from 'svelte';
	import { LandingFooter, LandingHeader, SignInWithFacile, registerIcons } from '@facile/muse';
	import { ANTENNE_ICON, ANTENNE_MARK } from '$lib/components/AntenneLogo.svelte';
	import { rememberLocale, t } from '$lib/i18n/index.svelte';

	registerIcons({ [ANTENNE_ICON]: ANTENNE_MARK });

	let { children }: { children: Snippet } = $props();

	const links = $derived([
		{ href: '/#features', label: t('landing.nav.features') },
		{ href: '/downloads', label: t('landing.nav.downloads') }
	]);
</script>

<div class="min-h-dvh overflow-x-clip bg-fc-page text-fc-fg">
	<LandingHeader appName="Antenne" icon={ANTENNE_ICON} {links} skipHref="#main">
		{#snippet actions()}
			<SignInWithFacile href="/api/auth/oidc" size="sm" />
		{/snippet}
	</LandingHeader>

	<main id="main" class="flex flex-col gap-24 pb-24">
		{@render children()}
	</main>

	<LandingFooter
		appName="Antenne"
		icon={ANTENNE_ICON}
		tagline={t('login.tagline')}
		columns={[{ title: 'Antenne', links }]}
		legal={{ onLocaleChange: rememberLocale }}
	/>
</div>
