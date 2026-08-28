export type WebhookProviderType =
	| 'sonde'
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
	| 'nuage'
	| 'filet';

export type ProviderType = WebhookProviderType | 'imap' | 'website' | 'rss';

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

export interface IMAPProvider extends ProviderBase {
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

export type Provider = WebsiteProvider | RssProvider | IMAPProvider | WebhookProvider;

export const providerTypes: ProviderType[] = [
	'website',
	'imap',
	'rss',
	'sonde',
	'ardoise',
	'docuseal',
	'filet',
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

const PROVIDER_LABELS: Record<string, string> = {
	website: 'Website',
	imap: 'IMAP',
	rss: 'RSS',
	sonde: 'Sonde',
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
	filet: 'filet',
	'uptime-kuma': 'Uptime Kuma'
};

export function providerLabel(type: ProviderType): string {
	return PROVIDER_LABELS[type] ?? type;
}

const WEBHOOK_SECRET_HEADERS: Record<WebhookProviderType, string> = {
	sonde: 'x-sonde-signature-256',
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
	nuage: 'x-nuage-signature-256',
	filet: 'x-filet-signature-256'
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
