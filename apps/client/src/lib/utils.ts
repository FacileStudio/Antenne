import { locale, t } from './i18n/index.svelte';

/** Renders an ISO timestamp as a short absolute date in the session locale. */
export function formatDateTime(iso: string | null | undefined): string {
	if (!iso) return '—';
	const date = new Date(iso);
	if (Number.isNaN(date.getTime())) return iso;
	return date.toLocaleString(locale(), {
		month: 'short',
		day: 'numeric',
		hour: '2-digit',
		minute: '2-digit'
	});
}

/** Renders an ISO timestamp as an age, falling back to a date past a day. */
export function formatRelative(iso: string): string {
	const date = new Date(iso);
	if (Number.isNaN(date.getTime())) return iso;

	const elapsed = Date.now() - date.getTime();
	if (elapsed < 60_000) return t('common.justNow');
	if (elapsed < 3_600_000) return t('common.minutesAgo', { count: Math.floor(elapsed / 60_000) });
	if (elapsed < 86_400_000) return t('common.hoursAgo', { count: Math.floor(elapsed / 3_600_000) });
	return formatDateTime(iso);
}

/**
 * Renders the small HTML subset Antenne's alert messages use as plain text.
 * The dashboard never injects the markup: an alert body is third-party text.
 */
export function plainText(value: string | null | undefined): string {
	if (!value) return '';

	const decoded = value
		.replace(/<br\s*\/?>/gi, '\n')
		.replace(/<\/p>/gi, '\n')
		.replace(/<[^>]+>/g, ' ')
		.replaceAll('&nbsp;', ' ')
		.replaceAll('&amp;', '&')
		.replaceAll('&lt;', '<')
		.replaceAll('&gt;', '>')
		.replaceAll('&quot;', '"')
		.replaceAll('&#39;', "'");

	return decoded
		.split('\n')
		.map((line) => line.replace(/\s+/g, ' ').trim())
		.filter(Boolean)
		.join('\n');
}

/** Mints an id for a new provider or delivery target. */
export function newId(): string {
	return crypto.randomUUID();
}
