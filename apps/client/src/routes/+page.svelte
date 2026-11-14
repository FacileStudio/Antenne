<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { Button, DownloadSection, SignInWithFacile, icons } from '@facile/muse';
	import { api } from '$lib/backend';
	import HeroDemo from '$lib/components/landing/HeroDemo.svelte';
	import LandingShell from '$lib/components/landing/LandingShell.svelte';
	import TailTile from '$lib/components/landing/TailTile.svelte';
	import { CLI_COMMANDS, CLI_RELEASES } from '$lib/downloads';
	import { setSessionLocale, t } from '$lib/i18n/index.svelte';

	const SIGN_IN = '/api/auth/oidc';

	const FEATURES = [
		{ id: 'sources', icon: 'solar:inbox-linear' },
		{ id: 'routing', icon: 'solar:tag-linear' },
		{ id: 'retry', icon: 'solar:refresh-circle-linear' },
		{ id: 'bus', icon: 'solar:transfer-horizontal-linear' }
	] as const;

	let ready = $state(false);

	onMount(async () => {
		try {
			const session = await api.session();
			if (session.authenticated) {
				setSessionLocale(session.locale ?? null);
				await goto('/providers', { replaceState: true });
				return;
			}
		} catch {}
		ready = true;
	});
</script>

<svelte:head>
	<title>{t('landing.title')}</title>
</svelte:head>

{#if ready}
	<LandingShell>
		<section class="mx-auto flex w-full max-w-fc-xl flex-col items-center gap-10 px-4 pt-12 sm:px-6 md:pt-20">
			<div class="flex max-w-fc-md flex-col items-center gap-6 text-center">
				<h1 class="font-fc-title text-fc-3xl font-bold text-balance">{t('landing.hero.title')}</h1>
				<p class="text-fc-md text-balance text-fc-fg-muted">{t('landing.hero.line')}</p>
				<div class="flex flex-wrap items-center justify-center gap-3">
					<SignInWithFacile href={SIGN_IN} size="lg" />
					<Button href="#cli" variant="secondary" size="lg" icon={icons.download}>{t('landing.hero.cli')}</Button>
				</div>
			</div>
			<div class="w-full">
				<HeroDemo />
			</div>
		</section>

		<section id="features" aria-labelledby="features-title" class="mx-auto flex w-full max-w-fc-xl scroll-mt-24 flex-col gap-8 px-4 sm:px-6">
			<h2 id="features-title" class="font-fc-title text-fc-2xl font-bold">{t('landing.features.title')}</h2>
			<ul class="grid gap-4 sm:grid-cols-2">
				{#each FEATURES as feature (feature.id)}
					<li class="flex flex-col gap-3 rounded-fc-lg bg-fc-component p-6">
						<span class="flex size-10 items-center justify-center rounded-fc-md bg-fc-surface text-fc-fg" aria-hidden="true">
							<iconify-icon icon={feature.icon} width="20" height="20" class="block"></iconify-icon>
						</span>
						<h3 class="text-fc-md font-bold">{t(`landing.feature.${feature.id}.title`)}</h3>
						<p class="text-fc-sm text-fc-fg-muted">{t(`landing.feature.${feature.id}.body`)}</p>
					</li>
				{/each}
			</ul>
		</section>

		<section id="cli" aria-labelledby="cli-title" class="mx-auto flex w-full max-w-fc-xl scroll-mt-24 flex-col gap-8 px-4 sm:px-6">
			<h2 id="cli-title" class="font-fc-title text-fc-2xl font-bold">{t('landing.cli.title')}</h2>
			<div class="grid items-start gap-4 lg:grid-cols-2">
				<TailTile />
				<DownloadSection mode="device" allHref="/downloads" commands={CLI_COMMANDS} releasesHref={CLI_RELEASES} headingLevel={3} />
			</div>
		</section>

		<section class="mx-auto flex w-full max-w-fc-md flex-col items-center gap-6 px-4 text-center sm:px-6">
			<h2 class="font-fc-title text-fc-2xl font-bold">{t('landing.cta.title')}</h2>
			<SignInWithFacile href={SIGN_IN} size="lg" />
		</section>
	</LandingShell>
{/if}
