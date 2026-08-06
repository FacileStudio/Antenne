<script lang="ts">
	import PageHeader from '$lib/components/PageHeader.svelte';
	import ProviderList from '$lib/components/ProviderList.svelte';
</script>

<section class="grid gap-5">
	<PageHeader eyebrow="Providers" title="Sources" />
	<ProviderList />
</section>
