import { browser } from '$app/environment';
import { applyTheme } from '@facile/muse';

export type ThemeMode = 'system' | 'light' | 'dark';

const STORAGE_KEY = 'antenne-theme';

function stored(): ThemeMode {
	if (!browser) return 'system';
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		return raw === 'light' || raw === 'dark' || raw === 'system' ? raw : 'system';
	} catch {
		return 'system';
	}
}

export const theme = $state({ mode: stored() });

export function setTheme(mode: ThemeMode) {
	theme.mode = mode;
	if (!browser) return;
	applyTheme(mode);
	try {
		localStorage.setItem(STORAGE_KEY, mode);
	} catch {
		return;
	}
}

export function applyStoredTheme() {
	if (browser) applyTheme(theme.mode);
}
