<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import ProviderList from '$lib/components/ProviderList.svelte';
</script>

<div class="flex flex-col gap-10">
	<div class="flex flex-col gap-2">
		<h1 class="text-fc-2xl font-semibold text-fc-fg">{t('providers.title')}</h1>
		<p class="text-fc-sm text-fc-fg-muted">
			{t('providers.subtitle')}
		</p>
	</div>

	<ProviderList />
</div>
