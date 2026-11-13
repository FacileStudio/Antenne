<script lang="ts">
	import { Avatar, Badge, Button, Card, EmptyState, Tooltip, icons } from '@facile/muse';
	import { t } from '$lib/i18n/index.svelte';
	import { sourceIcon } from '$lib/icons';
	import type { EventEntry } from '$lib/types';
	import { formatRelative, plainText } from '$lib/utils';

	let {
		events,
		onreplay = undefined
	}: { events: EventEntry[]; onreplay?: (eventId: string) => void } = $props();

	const SOURCE_KEYS = {
		website: 'source.website',
		webhook: 'source.webhook',
		imap: 'source.imap',
		rss: 'source.rss',
		test: 'source.test',
		system: 'source.system'
	} as const;

	function sourceLabel(source: string): string {
		return t(SOURCE_KEYS[source as keyof typeof SOURCE_KEYS] ?? 'source.system');
	}

	function titleFor(entry: EventEntry): string {
		const event = plainText(entry.event);
		const lines = plainText(entry.message).split('\n').filter(Boolean);

		// A system event's message is the sentence; its `event` field is an
		// internal label, so the message reads better as the headline.
		if (entry.source === 'system' && lines.length > 0) return lines[0];
		if (event) return event;
		if (lines.length > 0) return lines[0];
		return sourceLabel(entry.source);
	}

	/** Who caused the event, by name or else by address, or '' when the source did not say. */
	function actorOf(entry: EventEntry): string {
		const actor = entry.envelope?.actor;
		return actor?.name?.trim() || actor?.email?.trim() || '';
	}

	function previewFor(entry: EventEntry): string {
		const message = plainText(entry.message);
		if (!message) return '';

		const lines = message.split('\n').filter(Boolean);
		const title = titleFor(entry);

		if (lines.length > 1 && lines[0] === title) return lines.slice(1).join(' ');
		return message === title ? '' : message;
	}
</script>

{#if events.length === 0}
	<EmptyState
		icon="solar:inbox-linear"
		title={t('events.emptyTitle')}
		description={t('events.emptyText')}
	/>
{:else}
	<!-- `p-0` because the rows carry the padding: rows reach both edges of the
	     card so the hover tone spans it. -->
	<Card class="overflow-hidden p-0">
		<div class="flex flex-col gap-1">
			{#each events as entry (entry.id)}
				{@const actor = actorOf(entry)}
				<article class="flex items-start gap-3 p-4 transition-colors hover:bg-fc-surface">
					<span class="mt-0.5 rounded-fc-sm bg-fc-surface p-1.5 text-fc-fg-muted">
						<iconify-icon icon={sourceIcon(entry.source)} width="14" height="14" class="block"
						></iconify-icon>
					</span>

					<div class="flex min-w-0 flex-1 flex-col gap-2">
						<div class="flex items-baseline justify-between gap-3">
							<span class="truncate text-fc-sm font-semibold text-fc-fg">{titleFor(entry)}</span>
							<time
								class="shrink-0 text-fc-xs whitespace-nowrap text-fc-fg-muted tabular-nums"
								datetime={entry.timestamp}
							>
								{formatRelative(entry.timestamp)}
							</time>
						</div>

						<div class="flex flex-wrap items-center gap-2">
							<Badge>{sourceLabel(entry.source)}</Badge>
							{#if entry.providerName && entry.providerName !== 'Antenne' && entry.providerName !== 'system'}
								<Badge tone="info">{entry.providerName}</Badge>
							{/if}
							{#if actor}
								<span class="flex min-w-0 items-center gap-1.5 text-fc-xs text-fc-fg-muted">
									<Avatar src={entry.actorAvatarUrl || undefined} name={actor} size="sm" class="h-5 w-5" />
									<span class="truncate">{actor}</span>
								</span>
							{/if}
						</div>

						{#if previewFor(entry)}
							<p class="line-clamp-2 text-fc-sm text-fc-fg-muted">
								{previewFor(entry)}
							</p>
						{/if}

						{#if entry.deliveries.length > 0}
							<div class="flex flex-wrap items-center gap-2">
								{#each entry.deliveries as attempt (attempt.targetId)}
									<Tooltip text={attempt.error ?? `${attempt.durationMs}ms`} class="contents">
										<Badge tone={attempt.status === 'success' ? 'success' : 'danger'}>
											<iconify-icon
												icon={attempt.status === 'success'
													? 'solar:check-circle-linear'
													: 'solar:close-circle-linear'}
												width="12"
												height="12"
												class="block"
											></iconify-icon>
											{attempt.targetName}
										</Badge>
									</Tooltip>
								{/each}
							</div>
						{/if}

						{#if entry.envelope && onreplay}
							<Tooltip text={t('events.replayHelp')} class="contents">
								<Button
									variant="ghost"
									size="sm"
									icon={icons.refresh}
									class="-ml-3.5 w-fit"
									onclick={() => onreplay?.(entry.id)}
								>
									{t('events.replay')}
								</Button>
							</Tooltip>
						{/if}
					</div>
				</article>
			{/each}
		</div>
	</Card>
{/if}
