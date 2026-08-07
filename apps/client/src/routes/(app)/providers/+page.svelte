<script lang="ts">
	import ProviderList from '$lib/components/ProviderList.svelte';
</script>

<div class="flex flex-col gap-10">
	<div class="flex flex-col gap-2">
		<h1 class="text-fc-2xl font-semibold text-fc-fg">Providers</h1>
		<p class="text-fc-sm text-fc-fg-muted">
			The sources Antenne watches, polls or receives on.
		</p>
	</div>

	<ProviderList />
</div>
