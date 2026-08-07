import { browser } from '$app/environment';

export type ThemeMode = 'system' | 'light' | 'dark';

const STORAGE_KEY = 'antenne-theme';

function stored(): ThemeMode {
	if (!browser) return 'system';
	const raw = localStorage.getItem(STORAGE_KEY);
	return raw === 'light' || raw === 'dark' || raw === 'system' ? raw : 'system';
}

export const theme = $state({ mode: stored() });

/**
 * Writes both classes, and `system` writes neither.
 *
 * muse's tokens flip on `prefers-color-scheme` scoped to `:root:not(.light)`, so the
 * `.light` class is the only thing that lets someone force light on a dark OS. A store
 * that only ever adds `.dark` silently strands exactly those users.
 */
export function setTheme(mode: ThemeMode) {
	theme.mode = mode;
	if (!browser) return;
	const root = document.documentElement;
	root.classList.toggle('dark', mode === 'dark');
	root.classList.toggle('light', mode === 'light');
	localStorage.setItem(STORAGE_KEY, mode);
}

/** Re-applies the stored choice on hydration; app.html already painted it. */
export function applyStoredTheme() {
	setTheme(theme.mode);
}
