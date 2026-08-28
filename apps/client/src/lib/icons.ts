import type { DeliveryTargetType, ProviderType } from './types';

const PROVIDER_ICONS: Record<string, string> = {
	system: 'solar:settings-bold-duotone',
	website: 'solar:global-bold-duotone',
	imap: 'solar:letter-bold-duotone',
	rss: 'solar:feed-bold-duotone',
	sonde: 'solar:radar-bold-duotone',
	'uptime-kuma': 'solar:heart-pulse-bold-duotone',
	docuseal: 'solar:document-bold-duotone',
	dokploy: 'solar:server-bold-duotone',
	git: 'solar:code-bold-duotone',
	vikunja: 'solar:checklist-bold-duotone',
	ardoise: 'solar:bill-list-bold-duotone',
	sablier: 'solar:hourglass-bold-duotone',
	perception: 'solar:eye-bold-duotone',
	opus: 'solar:widget-2-bold-duotone',
	vision: 'solar:panorama-bold-duotone',
	plume: 'solar:pen-new-square-bold-duotone',
	nuage: 'solar:cloud-bold-duotone',
	filet: 'solar:code-scan-bold-duotone'
};

const DELIVERY_ICONS: Record<string, string> = {
	matrix: 'mdi:matrix',
	discord: 'mdi:discord',
	webhook: 'solar:link-bold-duotone',
	smtp: 'solar:letter-bold-duotone',
	ntfy: 'solar:bell-bold-duotone',
	perception: 'solar:eye-bold-duotone'
};

const SOURCE_ICONS: Record<string, string> = {
	website: 'solar:global-linear',
	webhook: 'solar:link-round-angle-linear',
	imap: 'solar:letter-linear',
	rss: 'solar:feed-linear',
	test: 'solar:test-tube-linear',
	system: 'solar:server-linear'
};

/** Apps that connect to the pool, for the settings page's connected list. */
const APP_ICONS: Record<string, string> = {
	Sonde: 'solar:radar-bold-duotone',
	Opus: 'solar:widget-2-bold-duotone',
	Sablier: 'solar:hourglass-bold-duotone',
	Ardoise: 'solar:bill-list-bold-duotone',
	Plume: 'solar:pen-new-square-bold-duotone',
	Glouton: 'solar:chef-hat-bold-duotone',
	Vision: 'solar:panorama-bold-duotone',
	Mycelium: 'solar:global-bold-duotone'
};

export const FALLBACK_ICON = 'solar:widget-2-bold-duotone';

export function providerIcon(type: ProviderType | string): string {
	return PROVIDER_ICONS[type] ?? FALLBACK_ICON;
}

export function deliveryIcon(type: DeliveryTargetType | string): string {
	return DELIVERY_ICONS[type] ?? FALLBACK_ICON;
}

export function sourceIcon(source: string): string {
	return SOURCE_ICONS[source] ?? SOURCE_ICONS.system;
}

export function appIcon(name: string): string {
	// The pool identifies an instance as `App:instance`, so match on the app
	// half rather than the whole identity.
	const app = name.split(':')[0];
	return APP_ICONS[app] ?? 'solar:server-square-bold-duotone';
}
