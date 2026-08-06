<script lang="ts">
	import { page } from '$app/state';
	import Icon from './Icon.svelte';
	import NookLogo from './NookLogo.svelte';
	import SpaceSwitcher from './SpaceSwitcher.svelte';
	import { dashboard } from '$lib/dashboard.svelte';
	import { cn } from '$lib/utils';

	let { onnavigate }: { onnavigate?: () => void } = $props();

	const sections = [
		{ href: '/delivery', label: 'Delivery', icon: 'solar:delivery-linear', match: ['/delivery', '/targets'] },
		{ href: '/providers', label: 'Providers', icon: 'solar:link-linear' },
		{ href: '/activity', label: 'Activity', icon: 'solar:chart-2-linear' },
		{ href: '/spaces', label: 'Spaces', icon: 'solar:buildings-2-linear' },
		{ href: '/settings', label: 'Settings', icon: 'solar:settings-linear' }
	];

	function isActive(section: { href: string; match?: string[] }): boolean {
		const prefixes = section.match ?? [section.href];
		return prefixes.some(
			(prefix) => page.url.pathname === prefix || page.url.pathname.startsWith(prefix + '/')
		);
	}
</script>

<aside
	class="flex h-full w-60 shrink-0 flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground"
>
	<div class="flex items-center gap-2.5 px-5 pt-7 pb-5">
		<NookLogo size={26} class="text-foreground" />
		<span class="font-heading text-xl font-bold tracking-tight">Nook</span>
	</div>

	<div class="px-3 pb-3">
		<SpaceSwitcher />
	</div>

	<nav class="flex flex-1 flex-col gap-0.5 px-3" aria-label="Sections">
		{#each sections as section (section.href)}
			<a
				href={section.href}
				onclick={onnavigate}
				aria-current={isActive(section) ? 'page' : undefined}
				class={cn(
					'flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors',
					isActive(section)
						? 'bg-sidebar-primary font-medium text-sidebar-primary-foreground'
						: 'text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground'
				)}
			>
				<Icon icon={section.icon} size={16} />
				{section.label}
			</a>
		{/each}
	</nav>

	{#if dashboard.passwordRequired}
		<div class="border-t border-sidebar-border p-3">
			<button
				type="button"
				class="flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
				onclick={() => dashboard.logout()}
			>
				<Icon icon="solar:logout-2-linear" size={16} />
				Log out
			</button>
		</div>
	{/if}
</aside>
