<script lang="ts">
	import { DownloadSection } from '@facile/muse';
	import LandingShell from '$lib/components/landing/LandingShell.svelte';
	import { CLI_COMMANDS, CLI_RELEASES } from '$lib/downloads';
	import { t } from '$lib/i18n/index.svelte';
</script>

<svelte:head>
	<title>{t('downloads.title')} · Antenne</title>
</svelte:head>

<LandingShell>
	<section aria-labelledby="downloads-title" class="mx-auto flex w-full max-w-fc-xl flex-col gap-8 px-4 pt-12 sm:px-6 md:pt-20">
		<div class="flex max-w-fc-md flex-col gap-4">
			<h1 id="downloads-title" class="font-fc-title text-fc-3xl font-bold text-balance">{t('downloads.title')}</h1>
			<p class="text-fc-md text-fc-fg-muted">{t('downloads.line')}</p>
		</div>
		<DownloadSection commands={CLI_COMMANDS} releasesHref={CLI_RELEASES} />
	</section>
</LandingShell>
