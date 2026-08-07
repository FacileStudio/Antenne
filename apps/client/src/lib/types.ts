export type WebhookProviderType =
	| 'uptime-kuma'
	| 'docuseal'
	| 'dokploy'
	| 'git'
	| 'vikunja'
	| 'ardoise'
	| 'sablier'
	| 'perception'
	| 'opus'
	| 'vision'
	| 'plume'
	| 'nuage';

export type ProviderType = WebhookProviderType | 'imap' | 'website' | 'rss';
export type DeliveryTargetType = 'matrix' | 'discord' | 'webhook' | 'smtp' | 'ntfy' | 'perception';

export interface ProviderBase {
	id: string;
	type: ProviderType;
	name: string;
	enabled: boolean;
	tags: string[];
}

export interface WebsiteProvider extends ProviderBase {
	type: 'website';
	url: string;
	interval: number;
}

export interface RssProvider extends ProviderBase {
	type: 'rss';
	feedUrl: string;
	pollInterval: number;
	includeContent: boolean;
}

export interface ImapProvider extends ProviderBase {
	type: 'imap';
	host: string;
	port: number;
	secure: boolean;
	username: string;
	password: string | null;
	pollInterval: number;
	includeEmailBody: boolean;
}

export interface WebhookProvider extends ProviderBase {
	type: WebhookProviderType;
	path: string;
	secretHeader: string | null;
	secretValue: string | null;
}

export type Provider = WebsiteProvider | RssProvider | ImapProvider | WebhookProvider;

export interface TargetBase {
	id: string;
	type: DeliveryTargetType;
	name: string;
	enabled: boolean;
	providerIds: string[];
	providerTags: string[];
}

export interface MatrixTarget extends TargetBase {
	type: 'matrix';
	homeserver: string;
	accessToken: string;
	refreshToken: string | null;
	userId: string;
	password: string | null;
	deviceId: string | null;
	roomIds: string[];
}

export interface DiscordTarget extends TargetBase {
	type: 'discord';
	webhookUrl: string;
}

export interface WebhookTarget extends TargetBase {
	type: 'webhook';
	url: string;
	secretValue: string | null;
	includeRawPayload: boolean;
}

export interface SmtpTarget extends TargetBase {
	type: 'smtp';
	host: string;
	port: number;
	secure: boolean;
	username: string;
	password: string | null;
	fromAddress: string;
	toAddresses: string[];
	subjectPrefix: string;
}

export interface NtfyTarget extends TargetBase {
	type: 'ntfy';
	serverUrl: string;
	topic: string;
	accessToken: string | null;
	priority: number | null;
	tags: string[];
	clickUrl: string | null;
}

export interface PerceptionTarget extends TargetBase {
	type: 'perception';
	url: string;
	workspace: string;
	secretValue: string | null;
	includeRawPayload: boolean;
}

export type DeliveryTarget =
	| MatrixTarget
	| DiscordTarget
	| WebhookTarget
	| SmtpTarget
	| NtfyTarget
	| PerceptionTarget;

export interface SystemEvents {
	startup: boolean;
	providerEnabled: boolean;
	providerDisabled: boolean;
	providerAdded: boolean;
	providerRemoved: boolean;
	deliveryFailure: boolean;
}

export interface AppSettings {
	delivery: { items: DeliveryTarget[] };
	system: { events: SystemEvents };
	timeout: number;
	providers: { port: number; items: Provider[] };
	// Always present client-side, so the settings form can bind straight into
	// it. The API tolerates an absent block and normalizes what it receives.
	pool: { secret: string; logActivity: boolean };
}

export interface RuntimeStatus {
	serverPort: number;
	settingsPath: string;
	usingPersistedSettings: boolean;
	dashboardBuilt: boolean;
	lastReloadedAt: string;
}

export interface DeliveryAttempt {
	targetId: string;
	targetName: string;
	targetType: string;
	status: 'success' | 'error';
	error?: string;
	durationMs: number;
}

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
	envelope?: unknown;
	deliveries: DeliveryAttempt[];
}

export interface EventStats {
	totalEvents: number;
	lastEventAt: string | null;
	deliverySuccessCount: number;
	deliveryErrorCount: number;
}

export type SpaceRole = 'owner' | 'admin' | 'member';

export interface Space {
	id: string;
	name: string;
	description: string;
	createdAt: string;
	updatedAt: string;
}

export interface SpaceMember {
	spaceId: string;
	userId: string;
	role: SpaceRole;
	joinedAt: string;
}

export interface PoolStats {
	epoch: string;
	connections: number;
	apps: { app_id: string; app: string; channels: string[] }[];
}

export interface SettingsPayload {
	settings: AppSettings;
	runtime: RuntimeStatus;
	matrixConfigured: boolean;
}

export interface SessionPayload {
	authenticated: boolean;
	passwordRequired: boolean;
	username: string;
}

/** The reserved provider id Antenne's own events carry. */
export const SYSTEM_PROVIDER_ID = '__antenne_system__';

/** What a stored secret reads as. Sending it back keeps the stored value. */
export const REDACTED = '••••••••';

export const providerTypes: ProviderType[] = [
	'website',
	'imap',
	'rss',
	'ardoise',
	'docuseal',
	'dokploy',
	'git',
	'nuage',
	'opus',
	'perception',
	'plume',
	'sablier',
	'uptime-kuma',
	'vikunja',
	'vision'
];

export const deliveryTargetTypes: DeliveryTargetType[] = [
	'matrix',
	'discord',
	'webhook',
	'smtp',
	'ntfy',
	'perception'
];

const PROVIDER_LABELS: Record<string, string> = {
	website: 'Website',
	imap: 'IMAP',
	rss: 'RSS',
	docuseal: 'DocuSeal',
	dokploy: 'Dokploy',
	git: 'Git',
	vikunja: 'Vikunja',
	ardoise: 'Ardoise',
	sablier: 'Sablier',
	opus: 'Opus',
	perception: 'Perception',
	vision: 'Vision',
	plume: 'Plume',
	nuage: 'Nuage',
	'uptime-kuma': 'Uptime Kuma'
};

export function providerLabel(type: ProviderType): string {
	return PROVIDER_LABELS[type] ?? type;
}

const TARGET_LABELS: Record<DeliveryTargetType, string> = {
	matrix: 'Matrix',
	discord: 'Discord webhook',
	webhook: 'HTTP webhook',
	smtp: 'SMTP',
	ntfy: 'ntfy',
	perception: 'Perception'
};

export function targetLabel(type: DeliveryTargetType): string {
	return TARGET_LABELS[type];
}

const WEBHOOK_SECRET_HEADERS: Record<WebhookProviderType, string> = {
	'uptime-kuma': 'x-webhook-secret',
	docuseal: 'x-docuseal-secret',
	dokploy: 'x-dokploy-secret',
	git: 'x-hub-signature-256',
	vikunja: 'x-vikunja-signature',
	ardoise: 'x-ardoise-signature-256',
	sablier: 'x-sablier-signature',
	perception: 'x-perception-secret',
	opus: 'x-opus-signature',
	vision: 'x-vision-signature-256',
	plume: 'x-plume-signature-256',
	nuage: 'x-nuage-signature-256'
};

export function isWebhookProvider(provider: Provider): provider is WebhookProvider {
	return provider.type in WEBHOOK_SECRET_HEADERS;
}

export function createProvider(type: ProviderType): Provider {
	const base = { id: crypto.randomUUID(), name: providerLabel(type), enabled: true, tags: [] };

	if (type === 'website') {
		return { ...base, type, url: 'https://', interval: 60 };
	}
	if (type === 'rss') {
		return { ...base, type, name: 'RSS Feed', feedUrl: '', pollInterval: 300, includeContent: false };
	}
	if (type === 'imap') {
		return {
			...base,
			type,
			name: 'IMAP Inbox',
			host: '',
			port: 993,
			secure: true,
			username: '',
			password: null,
			pollInterval: 60,
			includeEmailBody: false
		};
	}

	return {
		...base,
		type,
		path: `/webhook/${type}`,
		secretHeader: WEBHOOK_SECRET_HEADERS[type],
		secretValue: null
	};
}

function matrixDeviceId(): string {
	return `ANTENNE${crypto.randomUUID().replaceAll('-', '').slice(0, 12).toUpperCase()}`;
}

export function createTarget(type: DeliveryTargetType): DeliveryTarget {
	const base = {
		id: crypto.randomUUID(),
		name: targetLabel(type),
		enabled: true,
		providerIds: [] as string[],
		providerTags: [] as string[]
	};

	switch (type) {
		case 'matrix':
			return {
				...base,
				type,
				homeserver: 'https://matrix.org',
				accessToken: '',
				refreshToken: null,
				userId: '',
				password: null,
				deviceId: matrixDeviceId(),
				roomIds: []
			};
		case 'discord':
			return { ...base, type, webhookUrl: '' };
		case 'smtp':
			return {
				...base,
				type,
				host: '',
				port: 587,
				secure: true,
				username: '',
				password: null,
				fromAddress: '',
				toAddresses: [],
				subjectPrefix: '[Antenne]'
			};
		case 'ntfy':
			return {
				...base,
				type,
				serverUrl: 'https://ntfy.sh',
				topic: '',
				accessToken: null,
				priority: 3,
				tags: ['rotating_light'],
				clickUrl: null
			};
		case 'perception':
			return { ...base, type, url: '', workspace: '', secretValue: null, includeRawPayload: false };
		default:
			return { ...base, type: 'webhook', url: '', secretValue: null, includeRawPayload: false };
	}
}
