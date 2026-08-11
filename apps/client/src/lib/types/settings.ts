import type { DeliveryTarget } from './delivery';
import type { Provider } from './provider';

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
	bus: { secret: string; logActivity: boolean };
}

export interface RuntimeStatus {
	serverPort: number;
	settingsPath: string;
	usingPersistedSettings: boolean;
	dashboardBuilt: boolean;
	lastReloadedAt: string;
}

export interface BusStats {
	epoch: string;
	connections: number;
	apps: { app_id: string; app: string; channels: string[]; emits: string[] }[];
}

export interface SettingsPayload {
	settings: AppSettings;
	runtime: RuntimeStatus;
	matrixConfigured: boolean;
}

/** The reserved provider id Antenne's own events carry. */
export const SYSTEM_PROVIDER_ID = '__antenne_system__';

/** What a stored secret reads as. Sending it back keeps the stored value. */
export const REDACTED = '••••••••';
