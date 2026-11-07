<script lang="ts">
	import {
		Button,
		ConfirmModal,
		Input,
		SettingsRow,
		SettingsSection,
		icons
	} from '@facile/muse';
	import { dashboard } from '$lib/dashboard.svelte';
	import { t } from '$lib/i18n/index.svelte';
	import { formatDateTime } from '$lib/utils';

	let importInput = $state<HTMLInputElement | null>(null);
	let pendingImport = $state<File | null>(null);
	let confirmingImport = $state(false);

	const runtime = $derived(dashboard.runtime);

	function pickImport(event: Event) {
		pendingImport = (event.currentTarget as HTMLInputElement).files?.[0] ?? null;
		confirmingImport = pendingImport !== null;
	}

	function clearImport() {
		pendingImport = null;
		confirmingImport = false;
		if (importInput) importInput.value = '';
	}

	async function applyImport() {
		const file = pendingImport;
		if (!file) return;
		// importConfig saves the document immediately, so this is the one place in settings
		// that writes without the shell's Save button.
		await dashboard.importConfig(file);
		clearImport();
	}
</script>

<div class="flex flex-col gap-10">
	<SettingsSection title={t('advanced.tuning')} description={t('advanced.tuningText')}>
		<SettingsRow
			label={t('advanced.timeout')}
			description={t('advanced.timeoutText')}
			for="timeout"
			stacked
		>
			<Input
				id="timeout"
				type="number"
				min="1"
				step="1"
				bind:value={dashboard.settings.timeout}
				onchange={() => dashboard.autoSave(0)}
			/>
		</SettingsRow>

		<SettingsRow
			label={t('advanced.port')}
			description={t('advanced.portText')}
			for="listener-port"
			stacked
		>
			<Input
				id="listener-port"
				type="number"
				min="1"
				step="1"
				bind:value={dashboard.settings.providers.port}
				onchange={() => dashboard.autoSave(0)}
			/>
		</SettingsRow>
	</SettingsSection>

	<SettingsSection title={t('advanced.runtime')} description={t('advanced.runtimeText')}>
		<SettingsRow label={t('advanced.listening')}>
			<span class="font-fc-mono text-fc-sm text-fc-fg">{runtime?.serverPort ?? '—'}</span>
		</SettingsRow>
		<SettingsRow label={t('advanced.store')}>
			<span class="truncate font-fc-mono text-fc-sm text-fc-fg">{runtime?.settingsPath ?? '—'}</span>
		</SettingsRow>
		<SettingsRow label={t('advanced.source')}>
			<span class="text-fc-sm text-fc-fg">
				{runtime?.usingPersistedSettings ? t('advanced.sourceSaved') : t('advanced.sourceDefaults')}
			</span>
		</SettingsRow>
		<SettingsRow label={t('advanced.lastReload')}>
			<span class="text-fc-sm text-fc-fg">{formatDateTime(runtime?.lastReloadedAt)}</span>
		</SettingsRow>
	</SettingsSection>

	<SettingsSection
		title={t('advanced.configuration')}
		description={t('advanced.configurationText')}
	>
		<SettingsRow
			label={t('advanced.export')}
			description={t('advanced.exportText')}
		>
			<Button variant="outline" icon={icons.download} onclick={() => dashboard.exportConfig()}>
				{t('advanced.exportButton')}
			</Button>
		</SettingsRow>
	</SettingsSection>

	<!-- Danger zone last, and never a tab of its own: CHARTE §14. Import lives here rather than
	     beside Export because it is not the inverse of a download — it replaces the live
	     document and saves it on the spot. -->
	<SettingsSection
		title={t('advanced.danger')}
		description={t('advanced.dangerText')}
	>
		<SettingsRow
			label={t('advanced.import')}
			description={t('advanced.importText')}
			for="import-config"
			stacked
		>
			<div class="flex w-full flex-wrap items-center gap-3">
				<!--
					sr-only, never hidden: display:none takes the input out of the tab order and
					leaves the control mouse-only. The button is the pointer affordance; keyboard
					and screen-reader users reach the input itself.
				-->
				<input
					bind:this={importInput}
					id="import-config"
					type="file"
					accept=".json,application/json"
					disabled={dashboard.saving}
					onchange={pickImport}
					class="sr-only"
				/>
				<Button
					variant="danger"
					icon={icons.upload}
					disabled={dashboard.saving}
					onclick={() => importInput?.click()}
				>
					{t('advanced.chooseFile')}
				</Button>
			</div>
		</SettingsRow>
	</SettingsSection>
</div>

<ConfirmModal
	bind:open={confirmingImport}
	tone="danger"
	title={t('advanced.importTitle')}
	description={t('advanced.importWarning', { name: pendingImport?.name ?? t('advanced.thisFile') })}
	confirmLabel={t('advanced.replace')}
	onConfirm={applyImport}
	onCancel={clearImport}
/>
