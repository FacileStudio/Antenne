<script lang="ts">
	import { OptionCards, SettingsRow, SettingsSection, icons } from '@facile/muse';
	import { setTheme, theme, type ThemeMode } from '$lib/theme.svelte';

	const options = [
		{ value: 'system', label: 'System', icon: icons.monitor },
		{ value: 'light', label: 'Light', icon: icons.sun },
		{ value: 'dark', label: 'Dark', icon: icons.moon }
	];
</script>

<div class="flex flex-col gap-10">
	<SettingsSection
		title="Theme"
		description="Applied to this browser only. Antenne stores nothing about it server-side."
	>
		<!-- `value` is not bound: setTheme owns the write, since it also has to touch the two
		     root classes and localStorage. -->
		<SettingsRow
			label="Colour scheme"
			description="System follows the operating system's preference."
			stacked
		>
			<OptionCards
				{options}
				value={theme.mode}
				name="theme"
				label="Colour scheme"
				onSelect={(value) => setTheme(value as ThemeMode)}
			/>
		</SettingsRow>
	</SettingsSection>
</div>
