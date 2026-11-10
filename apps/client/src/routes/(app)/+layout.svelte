<script lang="ts">
	import type { Snippet } from 'svelte';
	import { browser } from '$app/environment';
	import { page } from '$app/state';
	import {
		MobileNav,
		PageTransition,
		SideBar,
		SpaceSwitcher,
		Spinner,
		Topbar,
		icons,
		type Locale,
		type Theme
	} from '@facile/muse';
	import { api } from '$lib/backend';
	import { dashboard } from '$lib/dashboard.svelte';
	import { locale, rememberLocale, t } from '$lib/i18n/index.svelte';
	import { spaces } from '$lib/spaces.svelte';
	import { setTheme, theme } from '$lib/theme.svelte';
	import { ANTENNE_ICON, ANTENNE_MARK } from '$lib/components/AntenneLogo.svelte';

	let { children }: { children: Snippet } = $props();

	/*
	 * muse's SideBar takes its brand glyph as an Iconify *name*, not as a component, so the
	 * real mark is registered into Iconify's local registry instead of being approximated by
	 * a public set. Browser-only and deferred for the same reason the root layout defers the
	 * element import: `iconify-icon` touches customElements at module scope.
	 */
	if (browser) {
		void import('iconify-icon').then((iconify) => {
			iconify.addIcon(ANTENNE_ICON, ANTENNE_MARK);
		});
	}

	let collapsed = $state(false);
	let scroller: HTMLElement | null = $state(null);

	const links = $derived([
		{ href: '/providers', label: t('nav.providers'), icon: icons.plug },
		{ href: '/delivery', label: t('nav.delivery'), icon: icons.mail, match: ['/delivery', '/targets'] },
		{ href: '/activity', label: t('nav.activity'), icon: icons.dashboard },
		{ href: '/spaces', label: t('nav.spaces'), icon: icons.space }
	]);

	function isActive(href: string, match?: string[]): boolean {
		return (match ?? [href]).some(
			(prefix) => page.url.pathname === prefix || page.url.pathname.startsWith(prefix + '/')
		);
	}

	const navPages = $derived(
		links.map(({ href, label, icon, match }) => ({
			href,
			label,
			icon,
			active: isActive(href, match)
		}))
	);
	const onSettings = $derived(isActive('/settings'));

	/*
	 * Settings has its own PageTransition over the tab strip, so collapsing every section to
	 * one key stops both from replaying on a tab change — the page fading in behind the
	 * section fading in.
	 */
	const routeKey = $derived(onSettings ? '/settings' : page.url.pathname);

	const user = $derived({
		name: dashboard.name?.trim() || dashboard.username?.trim() || dashboard.email || t('common.user'),
		email: dashboard.email || undefined,
		avatar: dashboard.avatarUrl || undefined
	});
	let invitations = $state<{ count: number; href: string } | null>(null);
	const userMenu = $derived({
		subscription: { plan: 'Gratuit', href: 'https://sso.facile.studio/spaces' },
		helpHref: 'https://github.com/FacileStudio/Antenne/tree/main/docs',
		onSignOut: () => dashboard.logout(),
		locale: locale(),
		onLocaleChange: (next: Locale) => rememberLocale(next),
		theme: theme.mode,
		onThemeChange: (next: Theme) => setTheme(next),
		invitations
	});
	const switcherSpaces = $derived(spaces.spaces.map((space) => ({ id: space.id, name: space.name })));

	$effect(() => {
		void dashboard.load();
		void spaces.load();
	});

	$effect(() => {
		api
			.invitations()
			.then((answer) => (invitations = answer?.href ? { count: answer.count, href: answer.href } : null))
			.catch(() => (invitations = null));
	});

	$effect(() => {
		if (page.url.pathname) scroller?.scrollTo({ top: 0 });
	});
</script>

<div class="flex h-dvh w-full overflow-hidden bg-fc-page">
	<div class="hidden h-full shrink-0 p-3 md:block">
		<SideBar
			icon={ANTENNE_ICON}
			title="Antenne"
			bind:collapsed
			pages={navPages}
			spaces={switcherSpaces}
			activeSpaceId={spaces.currentId}
			onSpaceSelect={(id) => spaces.select(id!)}
			personalSpaceLabel={null}
			manageSpacesHref="/spaces"
			{user}
			userHref="/settings"
			userActive={onSettings}
			{userMenu}
			class="h-full"
		/>
	</div>

	<!-- `min-w-0` so a wide table scrolls itself instead of pushing the shell sideways, and
	     `overscroll-contain` because <main> is the only scroller. -->
	<main bind:this={scroller} class="min-w-0 flex-1 overflow-auto overscroll-contain pb-28 md:pb-0">
		<Topbar class="md:hidden">
			<div class="flex min-w-0 flex-1 items-center gap-3">
				<span class="text-fc-md font-semibold text-fc-fg">Antenne</span>
				{#if switcherSpaces.length > 0}
					<div class="min-w-0 max-w-56 flex-1">
						<SpaceSwitcher
							spaces={switcherSpaces}
							activeId={spaces.currentId}
							onSelect={(id) => spaces.select(id!)}
							personalLabel={null}
							manageHref="/spaces"
						/>
					</div>
				{/if}
			</div>
		</Topbar>

		<div class="mx-auto flex max-w-fc-xl flex-col gap-8 px-4 py-8 sm:px-6 md:px-10 md:py-10">
			{#if dashboard.loading}
				<div class="flex justify-center py-20">
					<Spinner />
				</div>
			{:else}
				<PageTransition key={routeKey}>
					{@render children()}
				</PageTransition>
			{/if}
		</div>
	</main>

	<MobileNav items={navPages} {user} profileHref="/settings" profileActive={onSettings} {userMenu} />
</div>
