<script lang="ts">
	import type { Snippet } from 'svelte';
	import { page } from '$app/state';
	import { Divider, PageTransition, Tabs, icons } from '@facile/muse';

	let { children }: { children: Snippet } = $props();

	/*
	 * Sections are routes, not local state: /settings/bus survives a reload, a bookmark and
	 * browser-back. CHARTE §14.
	 */
	const sections = [
		{ id: '/settings', label: 'Profile', icon: icons.userCircle, href: '/settings' },
		{ id: '/settings/appearance', label: 'Appearance', icon: icons.palette, href: '/settings/appearance' },
		{ id: '/settings/bus', label: 'Bus', icon: icons.plug, href: '/settings/bus' },
		{ id: '/settings/events', label: 'Events', icon: icons.notification, href: '/settings/events' },
		{ id: '/settings/advanced', label: 'Advanced', icon: icons.shield, href: '/settings/advanced' }
	];

	/*
	 * Longest match wins, otherwise every child path also matches the bare `/settings` tab and
	 * the pill lands on Profile whatever section is open.
	 */
	const active = $derived(
		[...sections]
			.sort((left, right) => right.id.length - left.id.length)
			.find(({ id }) => page.url.pathname === id || page.url.pathname.startsWith(id + '/'))?.id ??
			'/settings'
	);
</script>

<div class="flex flex-col gap-8">
	<div class="flex flex-col gap-1">
		<h1 class="text-fc-2xl font-semibold text-fc-fg">Settings</h1>
		<p class="text-fc-sm text-fc-fg-muted">
			This instance's configuration. Changes are drafts until the Save button up top writes them.
		</p>
	</div>

	<!-- gap-4, not tighter: the rule separates the header from the body, and pulled up under the
	     strip it reads as an underline welded to the active pill. -->
	<div class="flex flex-col gap-4">
		<Tabs items={sections} value={active} label="Settings sections" />
		<Divider class="my-0" />
	</div>

	<PageTransition key={active} distance={8} duration={0.25}>
		{@render children()}
	</PageTransition>
</div>
