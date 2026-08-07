<script lang="ts">
	import type { Snippet } from 'svelte';
	import { page } from '$app/state';
	import {
		Button,
		MobileNav,
		PageTransition,
		SideBar,
		SpaceSwitcher,
		Spinner,
		Topbar,
		icons
	} from '@facile/muse';
	import { dashboard } from '$lib/dashboard.svelte';
	import { spaces } from '$lib/spaces.svelte';

	let { children }: { children: Snippet } = $props();

	let collapsed = $state(false);
	let scroller: HTMLElement | null = $state(null);

	/*
	 * No Settings row: settings is reached from the user card at the bottom of the rail and
	 * from the avatar in MobileNav. See CHARTE §14.
	 */
	const links = [
		{ href: '/providers', label: 'Providers', icon: icons.plug },
		{ href: '/delivery', label: 'Delivery', icon: icons.mail, match: ['/delivery', '/targets'] },
		{ href: '/activity', label: 'Activity', icon: icons.dashboard },
		{ href: '/spaces', label: 'Spaces', icon: icons.usersGroup }
	];

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

	const user = $derived({ name: dashboard.username });
	const switcherSpaces = $derived(spaces.spaces.map((space) => ({ id: space.id, name: space.name })));

	$effect(() => {
		void dashboard.load();
		void spaces.load();
	});

	$effect(() => {
		if (page.url.pathname) scroller?.scrollTo({ top: 0 });
	});
</script>

<div class="flex h-dvh w-full overflow-hidden bg-fc-page">
	<div class="hidden h-full shrink-0 p-3 md:block">
		<SideBar
			icon="solar:translation-2-bold-duotone"
			title="Antenne"
			bind:collapsed
			pages={navPages}
			spaces={switcherSpaces}
			activeSpaceId={spaces.currentId}
			onSpaceSelect={(id) => id && spaces.select(id)}
			manageSpacesHref="/spaces"
			{user}
			userHref="/settings"
			userActive={onSettings}
			class="h-full"
		/>
	</div>

	<!-- `min-w-0` so a wide table scrolls itself instead of pushing the shell sideways, and
	     `overscroll-contain` because <main> is the only scroller. -->
	<main bind:this={scroller} class="min-w-0 flex-1 overflow-auto overscroll-contain pb-28 md:pb-0">
		<!--
			Settings are one document: PUT /api/settings replaces the whole thing, so a provider
			edited on one page and a target edited on another are saved together or not at all.
			That is why the two actions live in the shell rather than on a page — they are not
			this page's actions, they are the document's.
		-->
		<Topbar>
			<div class="flex min-w-0 flex-1 items-center gap-3">
				<span class="text-fc-md font-semibold text-fc-fg md:hidden">Antenne</span>
				{#if switcherSpaces.length > 0}
					<div class="min-w-0 max-w-56 flex-1 md:hidden">
						<SpaceSwitcher
							spaces={switcherSpaces}
							activeId={spaces.currentId}
							onSelect={(id) => id && spaces.select(id)}
							manageHref="/spaces"
						/>
					</div>
				{/if}
			</div>

			<div class="flex shrink-0 items-center gap-2">
				<Button
					variant="outline"
					size="sm"
					icon={icons.bolt}
					onclick={() => dashboard.sendTestAlert()}
					disabled={dashboard.testing || dashboard.loading}
				>
					<span class="max-sm:sr-only">{dashboard.testing ? 'Sending…' : 'Test alert'}</span>
				</Button>
				<Button
					size="sm"
					icon={icons.check}
					onclick={() => dashboard.save()}
					disabled={dashboard.saving || dashboard.loading}
				>
					<span class="max-sm:sr-only">{dashboard.saving ? 'Saving…' : 'Save'}</span>
				</Button>
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

	<MobileNav items={navPages} {user} profileHref="/settings" profileActive={onSettings} />
</div>
