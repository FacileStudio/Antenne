<script lang="ts">
	import { Badge, Button, Card, icons } from '@facile/muse';
	import type { PendingAction } from '$lib/types';
	import { deliveryIcon, sourceIcon } from '$lib/icons';
	import { t } from '$lib/i18n/index.svelte';
	import { formatDateTime } from '$lib/utils';

	let {
		action,
		busy,
		onconfirm,
		ondismiss
	}: {
		action: PendingAction;
		busy: boolean;
		onconfirm: (action: PendingAction) => void;
		ondismiss: (action: PendingAction) => void;
	} = $props();

	/** Names the client that asked: the host of a client URL, else its id. */
	function clientName(clientId: string): string {
		if (clientId === 'bulle') return 'Bulle';
		try {
			return new URL(clientId).host;
		} catch {
			return clientId || '?';
		}
	}
</script>

<Card class="flex flex-col gap-4">
	<div class="flex flex-wrap items-center justify-between gap-2">
		<Badge tone="warning" class="shrink-0 whitespace-nowrap">{t('pending.kind.replay')}</Badge>
		<span class="text-fc-xs text-fc-fg-muted">
			{t('pending.askedBy', { client: clientName(action.client_id) })} · {formatDateTime(action.created_at)}
		</span>
	</div>

	{#if action.event && action.target}
		<div class="flex flex-col gap-3 rounded-fc-md bg-fc-surface p-4">
			<div class="flex items-center gap-3">
				<iconify-icon icon={sourceIcon(action.event.source)} width="24" height="24" class="shrink-0 text-fc-fg-muted"></iconify-icon>
				<div class="flex min-w-0 flex-1 flex-col gap-1">
					<p class="break-words text-fc-md font-medium text-fc-fg">{action.event.title || action.event.event}</p>
					<p class="text-fc-sm text-fc-fg-muted">{action.event.provider} · {formatDateTime(action.event.at)}</p>
				</div>
			</div>
			<div class="flex items-center gap-3">
				<iconify-icon icon={deliveryIcon(action.target.type)} width="24" height="24" class="shrink-0 text-fc-fg-muted"></iconify-icon>
				<p class="min-w-0 flex-1 break-words text-fc-sm text-fc-fg">{t('pending.to', { target: action.target.name })}</p>
				<Badge class="shrink-0 whitespace-nowrap">{action.target.label}</Badge>
			</div>
		</div>
		<p class="text-fc-sm text-fc-fg-muted">{t('pending.replayNote')}</p>
	{:else}
		<p class="break-words text-fc-sm text-fc-fg-muted">{action.summary}</p>
	{/if}

	{#if action.state === 'changed'}
		<p class="text-fc-sm text-fc-warning">{t('pending.changed')}</p>
	{:else if action.state === 'missing'}
		<p class="text-fc-sm text-fc-warning">{t('pending.missing')}</p>
	{/if}

	<div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
		<Button variant="ghost" class="w-full sm:w-auto [@media(pointer:coarse)]:h-11" disabled={busy} onclick={() => ondismiss(action)}>
			{t('pending.dismiss')}
		</Button>
		{#if action.state === 'ready'}
			<Button
				variant="primary"
				class="w-full sm:w-auto [@media(pointer:coarse)]:h-11"
				disabled={busy}
				icon={icons.check}
				onclick={() => onconfirm(action)}
			>
				{t('pending.confirmReplay')}
			</Button>
		{/if}
	</div>
</Card>
