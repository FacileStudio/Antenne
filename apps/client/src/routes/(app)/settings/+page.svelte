<script lang="ts">
	import { browser } from '$app/environment';
	import { ProfileSection } from '@facile/muse';
	import { dashboard } from '$lib/dashboard.svelte';

	const origin = $derived(browser ? window.location.origin : '—');
</script>

<ProfileSection
	name={dashboard.name?.trim() || dashboard.username || '—'}
	email={dashboard.email || undefined}
	avatar={dashboard.avatarUrl || undefined}
	role={dashboard.role || undefined}
>
	<p class="font-fc-mono text-fc-sm text-fc-fg-muted">{origin}</p>
</ProfileSection>
