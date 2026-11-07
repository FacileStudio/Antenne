<script lang="ts">
	import { PreferencesSection, type Locale, type Theme } from '@facile/muse';
	import { rememberLocale } from '$lib/i18n/index.svelte';
	import { setTheme, theme } from '$lib/theme.svelte';
</script>

<PreferencesSection
	theme={theme.mode}
	onThemeChange={(next: Theme) => setTheme(next)}
	onLocaleChange={(next: Locale) => rememberLocale(next)}
/>
