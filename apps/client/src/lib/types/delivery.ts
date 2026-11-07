import { t } from '../i18n/index.svelte';
export type DeliveryTargetType = 'matrix' | 'discord' | 'webhook' | 'smtp' | 'ntfy' | 'perception';

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

export interface DeliveryAttempt {
	targetId: string;
	targetName: string;
	targetType: string;
	status: 'success' | 'error';
	error?: string;
	durationMs: number;
}

export const deliveryTargetTypes: DeliveryTargetType[] = [
	'matrix',
	'discord',
	'webhook',
	'smtp',
	'ntfy',
	'perception'
];

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

/** A target with no provider and no tag selected can never receive anything. */
export function isSilent(target: DeliveryTarget): boolean {
	return target.providerIds.length === 0 && target.providerTags.length === 0;
}

export function routingSummary(target: DeliveryTarget): string {
	const providerCount = t('delivery.routingProviders', { count: target.providerIds.length });
	if (target.providerTags.length === 0) return providerCount;
	return `${providerCount} · ${t('delivery.routingTags', { count: target.providerTags.length })}`;
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

const NULLABLE_DELIVERY_FIELDS = [
	'accessToken',
	'refreshToken',
	'password',
	'deviceId',
	'secretValue',
	'clickUrl',
	'webhookUrl'
] as const;

export function hydrateDeliveryTarget(target: DeliveryTarget): DeliveryTarget {
	const draft = target as unknown as Record<string, unknown>;
	for (const key of NULLABLE_DELIVERY_FIELDS) {
		if (draft[key] === null) draft[key] = '';
	}
	if (!draft.providerIds) draft.providerIds = [];
	if (!draft.providerTags) draft.providerTags = [];
	return target;
}


