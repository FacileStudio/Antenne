<script lang="ts">
	import type { Snippet } from 'svelte';
	import { page } from '$app/state';
	import AppSidebar from '$lib/components/AppSidebar.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import { dashboard } from '$lib/dashboard.svelte';

	let { children }: { children: Snippet } = $props();

	let mobileNavOpen = $state(false);

	const TITLES: { prefix: string; label: string }[] = [
		{ prefix: '/providers', label: 'Providers' },
		{ prefix: '/delivery', label: 'Delivery' },
		{ prefix: '/targets', label: 'Delivery target' },
		{ prefix: '/activity', label: 'Activity' },
		{ prefix: '/spaces', label: 'Spaces' },
		{ prefix: '/settings', label: 'Settings' }
	];

	const title = $derived(
		TITLES.find((entry) => page.url.pathname.startsWith(entry.prefix))?.label ?? 'Nook'
	);

	$effect(() => {
		void dashboard.load();
	});
</script>

<div class="flex min-h-dvh bg-background">
	<div class="sticky top-0 hidden h-dvh lg:block">
		<AppSidebar />
	</div>

	{#if mobileNavOpen}
		<div class="fixed inset-0 z-40 flex lg:hidden">
			<button
				type="button"
				class="absolute inset-0 bg-black/40"
				aria-label="Close the navigation"
				onclick={() => (mobileNavOpen = false)}
			></button>
			<div class="relative z-10 h-dvh">
				<AppSidebar onnavigate={() => (mobileNavOpen = false)} />
			</div>
		</div>
	{/if}

	<div class="flex min-w-0 flex-1 flex-col">
		<header
			class="sticky top-0 z-30 flex items-center justify-between gap-4 border-b border-border bg-background/80 px-6 py-4 backdrop-blur-sm max-lg:px-4"
		>
			<div class="flex items-center gap-3">
				<Button
					variant="ghost"
					size="icon"
					class="lg:hidden"
					onclick={() => (mobileNavOpen = true)}
					aria-label="Open the navigation"
				>
					<Icon icon="solar:hamburger-menu-linear" size={18} />
				</Button>
				<h1 class="font-heading text-2xl font-semibold tracking-tight">{title}</h1>
			</div>

			<div class="flex flex-wrap items-center justify-end gap-2">
				<ThemeToggle />
				<Button
					variant="outline"
					onclick={() => dashboard.sendTestAlert()}
					disabled={dashboard.testing || dashboard.loading}
				>
					<Icon icon="solar:plain-linear" size={14} />
					<span class="max-sm:sr-only">
						{dashboard.testing ? 'Sending…' : 'Send test alert'}
					</span>
				</Button>
				<Button onclick={() => dashboard.save()} disabled={dashboard.saving || dashboard.loading}>
					<Icon icon="solar:diskette-linear" size={14} />
					<span class="max-sm:sr-only">{dashboard.saving ? 'Saving…' : 'Save settings'}</span>
				</Button>
			</div>
		</header>

		<main class="flex min-w-0 flex-1 flex-col gap-8 p-6 max-lg:p-4">
			{#if dashboard.loading}
				<div class="rounded-xl border border-border bg-card px-4 py-10 text-center text-sm text-muted-foreground">
					Loading the dashboard…
				</div>
			{:else}
				{@render children()}
			{/if}
		</main>
	</div>
</div>
