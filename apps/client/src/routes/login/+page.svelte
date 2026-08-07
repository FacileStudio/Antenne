<script lang="ts">
	import { goto } from '$app/navigation';
	import AntenneLogo from '$lib/components/AntenneLogo.svelte';
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

	let password = $state('');
	let error = $state('');
	let submitting = $state(false);

	// An instance with no admin password configured has nothing to log into.
	$effect(() => {
		void api
			.publicSession()
			.then((session) => {
				if (!session.passwordRequired || session.authenticated) {
					void goto('/providers', { replaceState: true });
				}
			})
			.catch(() => {});
	});

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		submitting = true;
		error = '';

		try {
			await api.login(password);
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
				Sign in as <strong class="font-medium text-foreground">{dashboard.username}</strong>
			</p>

			<form class="mt-8 grid gap-4" onsubmit={submit}>
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
		</div>
	</main>
</div>
