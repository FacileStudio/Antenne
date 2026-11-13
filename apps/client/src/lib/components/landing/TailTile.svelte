<script lang="ts">
	import { Caret, Stage, Terminal, Tile, grow, type Act } from '@facile/mise-en-scene';
	import { t } from '$lib/i18n/index.svelte';

	type Line = { id: number; time: string; source: string; provider: string; title: string; status: string; ok: boolean };

	const COMMAND = 'antenne tail';

	const lines = $derived<Line[]>([
		{ id: 1, time: '14:02:07', source: 'webhook', provider: 'Sonde', title: t('landing.demo.down'), status: '2 ok', ok: true },
		{ id: 2, time: '14:02:09', source: 'webhook', provider: 'Dokploy', title: t('landing.demo.deploy'), status: '1 failed', ok: false },
		{ id: 3, time: '14:03:40', source: 'rss', provider: 'Releases', title: 'antenne-cli v0.5.1', status: '1 ok', ok: true }
	]);

	let typed = $state('');
	let following = $state(false);
	let shown = $state(0);

	function reset() {
		typed = '';
		following = false;
		shown = 0;
	}

	function poster() {
		typed = COMMAND;
		following = true;
		shown = lines.length;
	}

	async function script(act: Act) {
		await act.wait(600);
		await act.type(COMMAND, (v) => (typed = v), 70);
		await act.wait(400);
		following = true;
		for (let i = 1; i <= lines.length; i++) {
			await act.wait(1300);
			shown = i;
		}
		await act.wait(2400);
		reset();
		await act.wait(400);
	}
</script>

<Tile title={COMMAND} body={t('landing.cli.tail')} level={3}>
	<Stage width={560} height={260} compact={{ below: 480, width: 340, height: 260 }} {script} {reset} {poster}>
		{#snippet children({ compact })}
			<Terminal style="top:24px;left:24px;right:24px;bottom:24px">
				<div class="tt-line"><span class="tt-dim">$</span> {typed}{#if !following}<Caret />{/if}</div>
				{#if following}
					<div class="tt-line" transition:grow={{ duration: 260 }}>
						<span class="tt-step">▸</span> <span class="tt-dim">following antenne.facile.studio</span>
					</div>
				{/if}
				{#each lines.slice(0, shown) as line (line.id)}
					<div class="tt-row" class:tt-compact={compact} transition:grow={{ duration: 320 }}>
						<span class="tt-dim">{compact ? line.time.slice(0, 5) : line.time}</span>
						{#if !compact}<span>{line.source}</span><span>{line.provider}</span>{/if}
						<span class="tt-title">{line.title}</span>
						<span class={line.ok ? 'tt-ok' : 'tt-bad'}>{line.status}</span>
					</div>
				{/each}
				{#if following}<div class="tt-line"><Caret /></div>{/if}
			</Terminal>
		{/snippet}
	</Stage>
</Tile>

<style>
	.tt-line,
	.tt-row {
		line-height: 22px;
		white-space: nowrap;
	}
	.tt-row {
		display: grid;
		grid-template-columns: 72px 64px 72px minmax(0, 1fr) auto;
		gap: 12px;
	}
	.tt-compact {
		grid-template-columns: 44px minmax(0, 1fr) auto;
		gap: 10px;
	}
	.tt-title {
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.tt-dim {
		color: var(--color-fc-on-inverse-muted);
	}
	.tt-step {
		color: var(--color-fc-info);
	}
	.tt-ok {
		color: var(--color-fc-success);
	}
	.tt-bad {
		color: var(--color-fc-danger);
	}
</style>
