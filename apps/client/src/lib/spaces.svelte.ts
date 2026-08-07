import { browser } from '$app/environment';
import { api } from './backend';
import type { Space } from './types';

const STORAGE_KEY = 'antenne-current-space-id';

class SpacesStore {
	spaces = $state<Space[]>([]);
	currentId = $state<string | null>(null);
	loading = $state(false);
	loaded = $state(false);

	get current(): Space | null {
		return this.spaces.find((space) => space.id === this.currentId) ?? null;
	}

	async load(force = false) {
		if (this.loaded && !force) return;

		this.loading = true;
		try {
			const payload = await api.spaces();
			this.spaces = payload.spaces;

			const restored = browser ? localStorage.getItem(STORAGE_KEY) : null;
			if (restored && this.spaces.some((space) => space.id === restored)) {
				this.currentId = restored;
			} else if (!this.currentId && this.spaces.length > 0) {
				this.select(this.spaces[0].id);
			}
			this.loaded = true;
		} catch {
			// The switcher is chrome: a spaces outage must not take the whole
			// dashboard down with it.
		} finally {
			this.loading = false;
		}
	}

	select(id: string) {
		this.currentId = id;
		if (browser) localStorage.setItem(STORAGE_KEY, id);
	}

	async create(name: string, description: string) {
		const payload = await api.createSpace(name, description);
		this.spaces = [...this.spaces, payload.space];
		this.select(payload.space.id);
		return payload.space;
	}

	async update(id: string, updates: { name?: string; description?: string }) {
		const payload = await api.updateSpace(id, updates);
		this.spaces = this.spaces.map((space) => (space.id === id ? payload.space : space));
		return payload.space;
	}

	async remove(id: string) {
		await api.deleteSpace(id);
		this.spaces = this.spaces.filter((space) => space.id !== id);
		if (this.currentId === id) {
			const next = this.spaces[0]?.id ?? null;
			this.currentId = next;
			if (browser) {
				if (next) localStorage.setItem(STORAGE_KEY, next);
				else localStorage.removeItem(STORAGE_KEY);
			}
		}
	}
}

export const spaces = new SpacesStore();
