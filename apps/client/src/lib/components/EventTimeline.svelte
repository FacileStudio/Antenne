<script lang="ts">
	import Icon from './Icon.svelte';
	import Badge from './ui/badge.svelte';
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

	function previewFor(entry: EventEntry): string {
		const message = plainText(entry.message);
		if (!message) return '';

		const lines = message.split('\n').filter(Boolean);
		const title = titleFor(entry);

		if (lines.length > 1 && lines[0] === title) return lines.slice(1).join(' ');
		return message === title ? '' : message;
	}
</script>

<div class="overflow-hidden rounded-xl border border-border bg-card">
	{#if events.length === 0}
		<p class="px-5 py-10 text-center text-sm text-muted-foreground">No events recorded yet.</p>
	{:else}
		<div class="divide-y divide-border">
			{#each events as entry (entry.id)}
				<article class="flex items-start gap-3 px-4 py-3 transition-colors hover:bg-muted/40">
					<span class="mt-0.5 rounded-lg bg-muted p-1.5 text-muted-foreground">
						<Icon icon={sourceIcon(entry.source)} size={14} />
					</span>

					<div class="grid min-w-0 flex-1 gap-1.5">
						<div class="flex items-baseline justify-between gap-3">
							<span class="truncate text-sm leading-tight font-semibold">{titleFor(entry)}</span>
							<time
								class="shrink-0 text-xs whitespace-nowrap text-muted-foreground tabular"
								datetime={entry.timestamp}
							>
								{formatRelative(entry.timestamp)}
							</time>
						</div>

						<div class="flex flex-wrap items-center gap-1.5">
							<Badge variant="outline">{sourceLabel(entry.source)}</Badge>
							{#if entry.providerName && entry.providerName !== 'Nook' && entry.providerName !== 'system'}
								<Badge>{entry.providerName}</Badge>
							{/if}
						</div>

						{#if previewFor(entry)}
							<p class="line-clamp-2 text-sm leading-relaxed text-muted-foreground">
								{previewFor(entry)}
							</p>
						{/if}

						{#if entry.deliveries.length > 0}
							<div class="flex flex-wrap items-center gap-1.5">
								{#each entry.deliveries as attempt (attempt.targetId)}
									<Badge
										variant={attempt.status === 'success' ? 'success' : 'destructive'}
										title={attempt.error ?? `${attempt.durationMs}ms`}
									>
										<Icon
											icon={attempt.status === 'success'
												? 'solar:check-circle-linear'
												: 'solar:close-circle-linear'}
											size={11}
										/>
										{attempt.targetName}
									</Badge>
								{/each}
							</div>
						{/if}

						{#if entry.envelope && onreplay}
							<button
								type="button"
								class="mt-0.5 inline-flex w-fit items-center gap-1.5 rounded-md px-2 py-1 text-xs text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
								onclick={() => onreplay?.(entry.id)}
								title="Send this event again"
							>
								<Icon icon="solar:refresh-linear" size={12} />
								Replay
							</button>
						{/if}
					</div>
				</article>
			{/each}
		</div>
	{/if}
</div>
