import type { DeliveryAttempt } from './delivery';

export type EventSource = 'webhook' | 'website' | 'imap' | 'rss' | 'test' | 'system';

export interface EventEntry {
	id: string;
	timestamp: string;
	source: EventSource;
	providerId: string | null;
	providerName: string;
	providerType: string;
	event: string;
	message: string | null;
	envelope?: { actor?: { name?: string | null; email?: string | null } | null };
	deliveries: DeliveryAttempt[];
	actorAvatarUrl?: string;
}

export interface EventStats {
	totalEvents: number;
	lastEventAt: string | null;
	deliverySuccessCount: number;
	deliveryErrorCount: number;
}
