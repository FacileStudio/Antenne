<script lang="ts">
	import DeliveryList from '$lib/components/DeliveryList.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
</script>

<section class="grid gap-5">
	<PageHeader eyebrow="Delivery" title="Targets" />
	<DeliveryList />
</section>
