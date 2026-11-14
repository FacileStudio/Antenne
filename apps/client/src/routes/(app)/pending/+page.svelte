<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { Alert, Button, EmptyState, Spinner, icons, toast } from '@facile/muse';
	import { ApiError, api } from '$lib/backend';
	import type { PendingAction } from '$lib/types';
	import PendingCard from '$lib/components/PendingCard.svelte';
	import { t } from '$lib/i18n/index.svelte';
	import { rememberReturn } from '$lib/return-path';

	let actions = $state<PendingAction[]>([]);
	let loading = $state(true);
	let busy = $state('');
	let signInAgain = $state(false);

	const focused = $derived(page.url.searchParams.get('action') ?? '');
	const ordered = $derived([
		...actions.filter((action) => action.id === focused),
		...actions.filter((action) => action.id !== focused)
	]);

	onMount(load);

	const OWN_SESSION_REFUSAL = "only the person's own session can confirm actions";

	/**
	 * Reports whether err is porte's refusal of a session that is not the person's own
	 * browser session, and shows the sign-in prompt if so. Signing in again fixes only
	 * this one; any other 403 is the confirmed action's own refusal.
	 */
	function refusedSession(err: unknown): boolean {
		signInAgain =
			err instanceof ApiError &&
			err.status === 403 &&
			err.code === 'permission_denied' &&
			err.message === OWN_SESSION_REFUSAL;
		return signInAgain;
	}

	/** The server's own words for any other refusal, else the fallback text. */
	function reason(err: unknown, fallback: string): string {
		return err instanceof ApiError && err.status === 403 ? err.message : fallback;
	}

	/** Comes back to this page once the fresh browser sign-in returns. */
	function signInAgainHere() {
		rememberReturn(page.url.pathname + page.url.search);
	}

	/** Loads the open actions. */
	async function load() {
		loading = true;
		try {
			actions = (await api.pending()).actions;
		} catch (err) {
			if (!refusedSession(err)) toast.danger(reason(err, t('pending.loadFailed')));
		}
		loading = false;
	}

	/** Runs one action once, then drops it from the list. */
	async function confirm(action: PendingAction) {
		busy = action.id;
		try {
			await api.confirmPending(action.id);
			toast.success(t('pending.sent'));
			actions = actions.filter((item) => item.id !== action.id);
		} catch (err) {
			if (!refusedSession(err)) {
				toast.danger(err instanceof Error ? err.message : t('pending.failed'));
				await load();
			}
		}
		busy = '';
	}

	/** Closes one action without running it. */
	async function dismiss(action: PendingAction) {
		busy = action.id;
		try {
			await api.dismissPending(action.id);
			toast.success(t('pending.dismissed'));
			actions = actions.filter((item) => item.id !== action.id);
		} catch (err) {
			if (!refusedSession(err)) toast.danger(err instanceof Error ? err.message : t('pending.failed'));
		}
		busy = '';
	}
</script>

<svelte:head>
	<title>{t('pending.title')}</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-2xl flex-col gap-6">
	<div class="flex flex-col gap-2">
		<h1 class="text-fc-2xl font-semibold text-fc-fg">{t('pending.heading')}</h1>
		<p class="text-fc-sm text-fc-fg-muted">{t('pending.intro')}</p>
	</div>

	{#if signInAgain}
		<Alert tone="warning" title={t('pending.signInAgain.title')}>
			<p>{t('pending.signInAgain.body')}</p>
			<Button href="/api/auth/oidc" onclick={signInAgainHere} variant="primary" class="mt-3 w-full sm:w-auto [@media(pointer:coarse)]:h-11">
				{t('pending.signInAgain.action')}
			</Button>
		</Alert>
	{:else if loading}
		<div class="flex h-48 items-center justify-center"><Spinner /></div>
	{:else if actions.length === 0}
		<EmptyState icon={icons.check} title={t('pending.none')} description={t('pending.noneBody')} />
	{:else}
		<section class="flex flex-col gap-4">
			{#each ordered as action (action.id)}
				<PendingCard {action} busy={busy === action.id} onconfirm={confirm} ondismiss={dismiss} />
			{/each}
		</section>
	{/if}
</div>
