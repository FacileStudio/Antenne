<script lang="ts">
	import { goto } from '$app/navigation';

	// The dashboard has no landing page of its own; providers is where an
	// operator starts. A redirect keeps the URL honest instead of rendering
	// another page's content at /.
	$effect(() => {
		void goto('/providers', { replaceState: true });
	});
</script>
