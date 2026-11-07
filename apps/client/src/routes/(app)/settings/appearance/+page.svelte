<script lang="ts">
	import { OptionCards, SettingsRow, SettingsSection, icons } from '@facile/muse';
	import { t } from '$lib/i18n/index.svelte';
	import { setTheme, theme, type ThemeMode } from '$lib/theme.svelte';

	const options = $derived([
		{ value: 'system', label: t('appearance.system'), icon: icons.monitor },
		{ value: 'light', label: t('appearance.light'), icon: icons.sun },
		{ value: 'dark', label: t('appearance.dark'), icon: icons.moon }
	]);
</script>

<div class="flex flex-col gap-10">
	<SettingsSection
		title={t('appearance.theme')}
		description={t('appearance.themeText')}
	>
		<!-- `value` is not bound: setTheme owns the write, since it also has to touch the two
		     root classes and localStorage. -->
		<SettingsRow
			label={t('appearance.scheme')}
			description={t('appearance.schemeText')}
			stacked
		>
			<OptionCards
				{options}
				value={theme.mode}
				name="theme"
				label={t('appearance.scheme')}
				onSelect={(value) => setTheme(value as ThemeMode)}
			/>
		</SettingsRow>
	</SettingsSection>
</div>
