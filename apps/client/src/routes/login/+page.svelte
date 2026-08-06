<script lang="ts">
	import { goto } from '$app/navigation';
	import NookLogo from '$lib/components/NookLogo.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Input from '$lib/components/ui/input.svelte';
	import Label from '$lib/components/ui/label.svelte';
	import { api } from '$lib/backend';
	import { dashboard } from '$lib/dashboard.svelte';

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

<div class="flex min-h-dvh">
	<aside
		class="hidden w-1/2 flex-col justify-between bg-foreground p-12 text-background lg:flex"
	>
		<div class="flex items-center gap-2.5">
			<NookLogo size={26} />
			<span class="font-heading text-xl font-bold tracking-tight">Nook</span>
		</div>

		<div>
			<h2 class="font-heading text-4xl leading-tight font-bold tracking-tight">
				Monitor everything.<br />Miss nothing.
			</h2>
			<p class="mt-4 max-w-sm text-sm leading-relaxed opacity-60">
				Event routing and alert delivery for your infrastructure.
			</p>
		</div>

		<p class="text-xs opacity-40">© {new Date().getFullYear()} Nook by Facile.</p>
	</aside>

	<main class="flex w-full flex-col items-center justify-center p-8 lg:w-1/2">
		<div class="w-full max-w-sm">
			<div class="mb-8 flex items-center gap-2.5 lg:hidden">
				<NookLogo size={24} class="text-foreground" />
				<span class="font-heading text-lg font-bold tracking-tight">Nook</span>
			</div>

			<h1 class="font-heading text-2xl font-bold tracking-tight">Welcome back</h1>
			<p class="mt-1.5 text-sm text-muted-foreground">
				Sign in as <strong class="font-medium text-foreground">{dashboard.username}</strong>
			</p>

			<form class="mt-8 grid gap-4" onsubmit={submit}>
				<div class="grid gap-1.5">
					<Label for="password">Password</Label>
					<Input
						id="password"
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

				<Button type="submit" size="lg" disabled={submitting}>
					{submitting ? 'Signing in…' : 'Sign in'}
				</Button>
			</form>
		</div>
	</main>
</div>
