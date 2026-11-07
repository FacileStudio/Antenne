<script lang="ts">
	import type { Snippet } from 'svelte';
	import { page } from '$app/state';
	import { SettingsLayout, icons } from '@facile/muse';
	import { dashboard } from '$lib/dashboard.svelte';
	import { t } from '$lib/i18n/index.svelte';

	let { children }: { children: Snippet } = $props();

	const sections = $derived([
		{ id: 'profile', label: t('settings.profile'), icon: icons.userCircle, href: '/settings' },
		{ id: 'preferences', label: t('settings.preferences'), icon: icons.palette, href: '/settings/preferences' },
		{ id: 'bus', label: t('settings.bus'), icon: icons.plug, href: '/settings/bus' },
		{ id: 'events', label: t('settings.events'), icon: icons.notification, href: '/settings/events' },
		{ id: 'advanced', label: t('settings.advanced'), icon: icons.shield, href: '/settings/advanced' }
	]);

	const active = $derived(
		[...sections]
			.sort((left, right) => right.href.length - left.href.length)
			.find(({ href }) => page.url.pathname === href || page.url.pathname.startsWith(href + '/'))?.id
	);

	let previousSerialized = '';
	$effect(() => {
		if (!dashboard.loaded) return;
		const current = JSON.stringify({
			bus: dashboard.settings.bus,
			system: dashboard.settings.system,
			timeout: dashboard.settings.timeout,
			providersPort: dashboard.settings.providers.port
		});
		if (!previousSerialized) {
			previousSerialized = current;
			return;
		}
		if (current !== previousSerialized) {
			previousSerialized = current;
			dashboard.autoSave(500);
		}
		return () => {
			dashboard.flushSave();
		};
	});
</script>

<SettingsLayout {sections} {active} backHref="/settings" description={t('settings.subtitle')} navLabel={t('settings.sections')}>
	{@render children()}
</SettingsLayout>
