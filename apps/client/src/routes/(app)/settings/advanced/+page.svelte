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
	<SettingsSection title="Runtime tuning" description="Applied automatically on runtime reload.">
		<SettingsRow
			label="Request timeout"
			description="Seconds a website check waits before it counts as down."
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
			label="Listener port"
			description="Takes effect on the next restart. Every published webhook URL and every pool client points at it, so the deployment has to agree."
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

	<SettingsSection title="Runtime" description="What this process is actually doing.">
		<SettingsRow label="Listening on">
			<span class="font-fc-mono text-fc-sm text-fc-fg">{runtime?.serverPort ?? '—'}</span>
		</SettingsRow>
		<SettingsRow label="Settings store">
			<span class="truncate font-fc-mono text-fc-sm text-fc-fg">{runtime?.settingsPath ?? '—'}</span>
		</SettingsRow>
		<SettingsRow label="Source">
			<span class="text-fc-sm text-fc-fg">
				{runtime?.usingPersistedSettings ? 'Saved from the dashboard' : 'Defaults'}
			</span>
		</SettingsRow>
		<SettingsRow label="Last reload">
			<span class="text-fc-sm text-fc-fg">{formatDateTime(runtime?.lastReloadedAt)}</span>
		</SettingsRow>
	</SettingsSection>

	<SettingsSection
		title="Configuration"
		description="The whole settings document as one JSON file."
	>
		<SettingsRow
			label="Export"
			description="Secrets come out redacted, so the file is safe to keep and to commit."
		>
			<Button variant="outline" icon={icons.download} onclick={() => dashboard.exportConfig()}>
				Export config
			</Button>
		</SettingsRow>
	</SettingsSection>

	<!-- Danger zone last, and never a tab of its own: CHARTE §14. Import lives here rather than
	     beside Export because it is not the inverse of a download — it replaces the live
	     document and saves it on the spot. -->
	<SettingsSection
		title="Danger zone"
		description="Irreversible from the dashboard. Keep an export first."
	>
		<SettingsRow
			label="Import a configuration"
			description="Replaces every provider, delivery target and system setting, then saves immediately."
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
					Choose a file…
				</Button>
			</div>
		</SettingsRow>
	</SettingsSection>
</div>

<ConfirmModal
	bind:open={confirmingImport}
	tone="danger"
	title="Replace this instance's configuration?"
	description={`Importing ${pendingImport?.name ?? 'this file'} overwrites every provider, every delivery target and every system setting, and saves them straight away. Providers that are not in the file stop polling and their webhook paths start answering 404; delivery targets that are not in the file stop receiving alerts. Redacted secrets in the file are restored from what is stored, so a target the file does not name loses its credentials with it.`}
	confirmLabel="Replace configuration"
	onConfirm={applyImport}
	onCancel={clearImport}
/>
