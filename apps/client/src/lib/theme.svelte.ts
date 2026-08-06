import { browser } from '$app/environment';

const STORAGE_KEY = 'nook-theme';

type Theme = 'light' | 'dark';

function initial(): Theme {
	if (!browser) return 'light';
	const stored = localStorage.getItem(STORAGE_KEY);
	if (stored === 'light' || stored === 'dark') return stored;
	return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

class ThemeStore {
	current = $state<Theme>('light');

	/**
	 * Called from the root layout on mount. app.html already applied the class
	 * before the first paint; this only syncs the reactive copy, so the toggle
	 * renders the right icon without a flash.
	 */
	hydrate() {
		this.current = initial();
		this.apply();
	}

	toggle() {
		this.current = this.current === 'dark' ? 'light' : 'dark';
		if (browser) localStorage.setItem(STORAGE_KEY, this.current);
		this.apply();
	}

	private apply() {
		if (!browser) return;
		document.documentElement.classList.toggle('dark', this.current === 'dark');
	}
}

export const theme = new ThemeStore();
