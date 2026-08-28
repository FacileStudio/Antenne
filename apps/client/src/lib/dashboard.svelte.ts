import { toast } from '@facile/muse';
import { api, ApiError, isUnauthenticated, redirectToLogin } from './backend';
import type { AppSettings, DeliveryTarget, Provider, RuntimeStatus } from './types';

function emptySettings(): AppSettings {
	return {
		delivery: { items: [] },
		system: {
			events: {
				startup: true,
				providerEnabled: false,
				providerDisabled: false,
				providerAdded: false,
				providerRemoved: false,
				deliveryFailure: false
			}
		},
		timeout: 10,
		providers: { port: 9090, items: [] },
		bus: { secret: '', logActivity: true }
	};
}

/**
 * The API omits an empty pool block and an unset secret. The settings form
 * binds straight into both, so they are filled in on the way in.
 */
function withPool(settings: AppSettings): AppSettings {
	return {
		...settings,
		bus: {
			secret: settings.bus?.secret ?? '',
			logActivity: settings.bus?.logActivity ?? true
		}
	};
}

function describe(error: unknown, fallback: string): string {
	if (error instanceof ApiError) return error.message;
	if (error instanceof Error) return error.message;
	return fallback;
}

/**
 * The dashboard's single source of truth.
 *
 * Settings are edited in place and saved as one document, which is the API's
 * contract: PUT /api/settings replaces the whole thing. Every mutation therefore
 * reassigns rather than mutating in place, so `$state` sees the change.
 */
class DashboardStore {
	settings = $state<AppSettings>(emptySettings());
	runtime = $state<RuntimeStatus | null>(null);
	matrixConfigured = $state(false);
	passwordRequired = $state(false);
	username = $state('');
	email = $state('');
	name = $state('');
	role = $state('');

	loading = $state(true);
	saving = $state(false);
	testing = $state(false);
	loaded = $state(false);

	async load(force = false) {
		if (this.loaded && !force) {
			this.loading = false;
			return;
		}

		this.loading = true;
		try {
			const [payload, session] = await Promise.all([api.settings(), api.session()]);
			this.settings = withPool(payload.settings);
			this.runtime = payload.runtime;
			this.matrixConfigured = payload.matrixConfigured;
			this.passwordRequired = session.passwordRequired;
			this.username = session.username;
			this.email = session.email;
			this.name = session.name ?? session.username;
			this.role = session.role ?? '';
			this.loaded = true;
		} catch (error) {
			if (isUnauthenticated(error)) {
				await redirectToLogin();
				return;
			}
			toast.danger(describe(error, 'Failed to load settings'));
		} finally {
			this.loading = false;
		}
	}

	async save(options: { silent?: boolean } = {}) {
		this.saving = true;
		try {
			const payload = await api.saveSettings(this.settings);
			this.settings = withPool(payload.settings);
			this.runtime = payload.runtime;
			this.matrixConfigured = payload.matrixConfigured;
			if (!options.silent) toast.success('Settings saved. Runtime reloaded.');
		} catch (error) {
			if (isUnauthenticated(error)) {
				await redirectToLogin();
				return;
			}
			toast.danger(describe(error, 'Failed to save settings'));
		} finally {
			this.saving = false;
		}
	}

	async sendTestAlert() {
		this.testing = true;
		try {
			await api.testAlert();
			toast.success('Test alert sent.');
		} catch (error) {
			toast.danger(describe(error, 'Failed to send the test alert'));
		} finally {
			this.testing = false;
		}
	}

	async logout() {
		try {
			await api.logout();
		} finally {
			this.loaded = false;
			await redirectToLogin();
		}
	}

	setProviders(providers: Provider[]) {
		this.settings = { ...this.settings, providers: { ...this.settings.providers, items: providers } };
	}

	setTargets(targets: DeliveryTarget[]) {
		this.settings = { ...this.settings, delivery: { items: targets } };
	}

	/** Every tag in use, for the filter bar and the tag input's suggestions. */
	get availableTags(): string[] {
		const tags = new Set<string>();
		for (const provider of this.settings.providers.items) {
			for (const tag of provider.tags) tags.add(tag);
		}
		return [...tags].sort((left, right) => left.localeCompare(right));
	}

	exportConfig() {
		const blob = new Blob([JSON.stringify(this.settings, null, 2)], { type: 'application/json' });
		const url = URL.createObjectURL(blob);
		const anchor = document.createElement('a');
		anchor.href = url;
		anchor.download = 'antenne-config.json';
		anchor.click();
		URL.revokeObjectURL(url);
	}

	async importConfig(file: File) {
		let parsed: AppSettings;
		try {
			parsed = JSON.parse(await file.text()) as AppSettings;
		} catch {
			toast.danger('That file is not valid JSON.');
			return;
		}

		this.settings = parsed;
		await this.save({ silent: true });
		toast.success('Config imported and saved.');
	}
}

export const dashboard = new DashboardStore();
