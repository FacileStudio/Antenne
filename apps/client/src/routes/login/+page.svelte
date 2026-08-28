<script lang="ts">
	import { goto } from '$app/navigation';
	import AntenneLogo from '$lib/components/AntenneLogo.svelte';
	import { page } from '$app/state';
	import { api } from '$lib/backend';
	import { dashboard } from '$lib/dashboard.svelte';

	/*
	 * The auth screen imports no component library, by suite convention: it is the one view
	 * that has to render before anything else is trusted to work. The class strings below are
	 * the suite's canonical set, shared byte-for-byte across the Facile repos, and they speak
	 * the semantic aliases (`bg-background`, `border-input`, …) that app.css maps onto muse's
	 * tokens — so the palette follows the design system without the markup depending on it.
	 */
	const inputClass =
		'flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50';
	const labelClass = 'text-sm font-medium leading-none';
	const primaryButtonClass =
		'inline-flex h-10 w-full items-center justify-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50';

	let email = $state('');
	let password = $state('');
	let error = $state('');
	let submitting = $state(false);
	let passwordRequired = $state(false);
	let ssoEnabled = $state(false);

	/*
	 * A failed single sign-on lands back here as a redirect carrying the reason
	 * in ?error, because the callback is a browser navigation and writing a JSON
	 * error envelope into somebody's address bar is the app asking a human to
	 * read a wire format. This block sits outside every conditional below on
	 * purpose: the one deployment that can only ever produce this error is an
	 * SSO-only one, and gating the message on the password form would be the
	 * deployment that shows nothing.
	 */
	let ssoError = $derived(page.url.searchParams.get('error') ?? '');

	$effect(() => {
		void api
			.authConfig()
			.then((config) => {
				passwordRequired = config.password_required && !config.sso_only;
				ssoEnabled = config.oidc_enabled;
				if (passwordRequired && !ssoEnabled) {
					email = config.email;
				}
			})
			.catch(() => {});
	});

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		submitting = true;
		error = '';

		try {
			await api.login(email, password);
			dashboard.loaded = false;
			await goto('/providers', { replaceState: true });
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Sign-in failed';
			password = '';
		} finally {
			submitting = false;
		}
	}
</script>

<div class="flex min-h-dvh bg-background text-foreground">
	<aside class="hidden w-1/2 flex-col justify-between bg-foreground p-12 text-background lg:flex">
		<div class="flex items-center gap-2.5">
			<AntenneLogo size={26} />
			<span class="font-heading text-xl font-semibold tracking-tight">Antenne</span>
		</div>

		<div>
			<h2 class="font-heading text-4xl leading-tight font-semibold tracking-tight">
				Monitor everything.<br />Miss nothing.
			</h2>
			<p class="mt-4 max-w-sm text-sm leading-relaxed opacity-60">
				Event routing and alert delivery for your infrastructure.
			</p>
		</div>

		<p class="text-xs opacity-40">© {new Date().getFullYear()} Antenne by Facile.</p>
	</aside>

	<main class="flex w-full flex-col items-center justify-center p-8 lg:w-1/2">
		<div class="w-full max-w-sm">
			<div class="mb-8 flex items-center gap-2.5 lg:hidden">
				<AntenneLogo size={24} class="text-foreground" />
				<span class="font-heading text-lg font-semibold tracking-tight">Antenne</span>
			</div>

			<h1 class="font-heading text-2xl font-semibold tracking-tight">Welcome back</h1>
			<p class="mt-1.5 text-sm text-muted-foreground">
				{#if !passwordRequired && ssoEnabled}
					Sign in with your Facile account to access Antenne.
				{:else if ssoEnabled}
					Sign in with your Facile account or password credentials.
				{:else}
					Sign in to your Antenne dashboard.
				{/if}
			</p>

			{#if ssoError}
				<p class="mt-6 text-sm text-destructive" role="alert">{ssoError}</p>
			{/if}

			{#if ssoEnabled}
				<a href="/api/auth/oidc" class="{primaryButtonClass} mt-8" data-sveltekit-reload>
					<!-- The literal rather than muse's icons.key: this view imports no
					component library on purpose, and the string is muse's own mapping,
					so the mark stays in the linear set every other icon is drawn from. -->
					<iconify-icon icon="solar:key-linear" width="18" class="mr-2"></iconify-icon>
					Sign in with Facile
				</a>
			{/if}

			{#if passwordRequired}
				{#if ssoEnabled}
					<div class="my-6 flex items-center gap-3">
						<span class="h-px flex-1 bg-border"></span>
						<span class="text-xs text-muted-foreground">or</span>
						<span class="h-px flex-1 bg-border"></span>
					</div>
				{/if}

				<form class="grid gap-4 {ssoEnabled ? '' : 'mt-8'}" onsubmit={submit}>
					<div class="grid gap-1.5">
						<label class={labelClass} for="email">Email</label>
						<input
							id="email"
							class={inputClass}
							type="email"
							bind:value={email}
							autocomplete="username"
							required
						/>
					</div>

					<div class="grid gap-1.5">
						<label class={labelClass} for="password">Password</label>
						<input
							id="password"
							class={inputClass}
							type="password"
							bind:value={password}
							placeholder="••••••••"
							autocomplete="current-password"
							required
						/>
					</div>

					{#if error}
						<p class="text-sm text-destructive" role="alert">{error}</p>
					{/if}

					<button type="submit" class={primaryButtonClass} disabled={submitting}>
						{submitting ? 'Signing in…' : 'Sign in'}
					</button>
				</form>
			{/if}
		</div>
	</main>
</div>
