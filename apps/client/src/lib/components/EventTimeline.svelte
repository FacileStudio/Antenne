<script lang="ts">
	import { Avatar, Badge, Button, Card, EmptyState, icons } from '@facile/muse';
	import { sourceIcon } from '$lib/icons';
	import type { EventEntry } from '$lib/types';
	import { formatRelative, plainText } from '$lib/utils';

	let {
		events,
		onreplay = undefined
	}: { events: EventEntry[]; onreplay?: (eventId: string) => void } = $props();

	const SOURCE_LABELS: Record<string, string> = {
		website: 'Website',
		webhook: 'Webhook',
		imap: 'Inbox',
		rss: 'RSS',
		test: 'Test',
		system: 'System'
	};

	function sourceLabel(source: string): string {
		return SOURCE_LABELS[source] ?? 'System';
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
		title="No events recorded yet"
		description="Every alert Antenne receives, and every delivery it attempts, lands here."
	/>
{:else}
	<!-- `p-0` because the rows carry the padding: a divider has to reach both edges of the
	     card, and a divider inset by `p-5` reads as a stray line. -->
	<Card class="overflow-hidden p-0">
		<div class="divide-y divide-fc-border">
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
									<Badge
										tone={attempt.status === 'success' ? 'success' : 'danger'}
										title={attempt.error ?? `${attempt.durationMs}ms`}
									>
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
								{/each}
							</div>
						{/if}

						{#if entry.envelope && onreplay}
							<Button
								variant="ghost"
								size="sm"
								icon={icons.refresh}
								class="-ml-3.5 w-fit"
								title="Send this event again"
								onclick={() => onreplay?.(entry.id)}
							>
								Replay
							</Button>
						{/if}
					</div>
				</article>
			{/each}
		</div>
	</Card>
{/if}
