<script lang="ts">
	import DeliveryList from '$lib/components/DeliveryList.svelte';
</script>

<div class="flex flex-col gap-10">
	<div class="flex flex-col gap-2">
		<h1 class="text-fc-2xl font-semibold text-fc-fg">Delivery</h1>
		<p class="text-fc-sm text-fc-fg-muted">
			Where Antenne sends what it sees. A target receives an event only if it names the provider or
			shares one of its tags.
		</p>
	</div>

	<DeliveryList />
</div>
