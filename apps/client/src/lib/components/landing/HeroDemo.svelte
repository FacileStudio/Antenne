<script lang="ts">
	import { tick } from 'svelte';
	import { Bar, Cursor, Roll, Stage, Window, cursorAt, grow, type Act } from '@facile/mise-en-scene';
	import { Icon, icons } from '@facile/muse';
	import AntenneLogo from '$lib/components/AntenneLogo.svelte';
	import EventRow from './EventRow.svelte';
	import { t } from '$lib/i18n/index.svelte';

	let down = $state(false);
	let deploy = $state(false);
	let replayed = $state(false);
	let cursor = $state(cursorAt());
	let replay = $state<HTMLElement>();

	const tracked = $derived(String(1284 + Number(down) + Number(deploy)));

	function reset() {
		down = false;
		deploy = false;
		replayed = false;
		cursor.visible = false;
	}

	function poster() {
		down = true;
		deploy = true;
		replayed = true;
	}

	async function script(act: Act) {
		await act.wait(900);
		down = true;
		await act.wait(1600);
		deploy = true;
		await act.wait(1400);
		await tick();
		cursor.visible = true;
		await act.move(cursor, act.point(replay, 0.5, 0.6));
		await act.click(cursor, () => (replayed = true));
		await act.wait(2200);
		reset();
		await act.wait(600);
	}
</script>

{#snippet feed(compact: boolean)}
	<div class="hd-head">
		<span class="hd-h1">{t('activity.title')}</span>
		<span class="hd-live"><span class="hd-dot"></span><Bar w={44} h={6} /></span>
	</div>
	{#if !compact}
		<div class="hd-stats">
			<div class="hd-stat"><Bar w="50%" h={6} /><Roll value={tracked} class="hd-num" /></div>
			<div class="hd-stat"><Bar w="42%" h={6} /><span class="hd-num">99%</span></div>
			<div class="hd-stat"><Bar w="56%" h={6} /><span class="hd-num">3</span></div>
		</div>
	{/if}
	<div class="hd-card">
		{#if deploy}
			<div transition:grow={{ duration: 420 }}>
				<EventRow
					title={t('landing.demo.deploy')}
					provider="Dokploy"
					deliveries={[{ name: 'Matrix', ok: replayed }]}
					replayable={!replayed}
					bind:replay
				/>
			</div>
		{/if}
		{#if down}
			<div transition:grow={{ duration: 420 }}>
				<EventRow
					title={t('landing.demo.down')}
					provider="Sonde"
					deliveries={[
						{ name: 'Discord', ok: true },
						{ name: 'ntfy', ok: true }
					]}
				/>
			</div>
		{/if}
		<EventRow />
		{#if !compact}<EventRow />{/if}
	</div>
{/snippet}

<Stage
	width={960}
	height={580}
	compact={{ below: 640, width: 358, height: 440 }}
	{script}
	{reset}
	{poster}
	label={t('landing.demo.label')}
>
	{#snippet children({ compact })}
		{#if compact}
			<Window title="Antenne" style="top:12px;left:12px;width:334px;height:372px">
				<div class="hd-body hd-compact">{@render feed(true)}</div>
			</Window>
		{:else}
			<Window style="top:20px;left:90px;width:780px;height:460px">
				{#snippet sidebar()}
					<div class="hd-nav">
						<div class="hd-brand"><AntenneLogo size={20} /><span>Antenne</span></div>
						<div class="hd-link"><Icon icon={icons.plug} size={16} /><Bar w="58%" /></div>
						<div class="hd-link"><Icon icon={icons.mail} size={16} /><Bar w="48%" /></div>
						<div class="hd-link hd-active"><Icon icon={icons.dashboard} size={16} /><span>{t('nav.activity')}</span></div>
						<div class="hd-link"><Icon icon={icons.space} size={16} /><Bar w="40%" /></div>
					</div>
				{/snippet}
				<div class="hd-body">{@render feed(false)}</div>
			</Window>
		{/if}
		<Cursor {cursor} />
	{/snippet}
</Stage>

<style>
	.hd-body {
		--hd-card: var(--color-fc-component);
		display: flex;
		flex: 1;
		flex-direction: column;
		gap: 16px;
		min-height: 0;
		padding: 24px 28px;
		color: var(--color-fc-fg);
		font-family: var(--font-fc-body);
	}
	@media (prefers-color-scheme: dark) {
		:global(:root:not(.light)) .hd-body {
			--hd-card: var(--color-fc-surface);
		}
	}
	:global(:root.dark) .hd-body {
		--hd-card: var(--color-fc-surface);
	}
	.hd-compact {
		gap: 12px;
		padding: 16px 12px;
	}
	.hd-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}
	.hd-h1 {
		font-family: var(--font-fc-title);
		font-size: 22px;
		font-weight: 700;
	}
	.hd-live {
		display: flex;
		align-items: center;
		gap: 8px;
	}
	.hd-dot {
		width: 8px;
		height: 8px;
		border-radius: 999px;
		background: var(--color-fc-success);
	}
	.hd-stats {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 12px;
	}
	.hd-stat {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding: 14px 16px;
		border-radius: 12px;
		background: var(--hd-card);
	}
	.hd-stat :global(.hd-num),
	.hd-num {
		font-size: 20px;
		font-weight: 700;
		font-variant-numeric: tabular-nums;
	}
	.hd-card {
		display: flex;
		flex-direction: column;
		min-height: 0;
		overflow: hidden;
		border-radius: 12px;
		background: var(--hd-card);
	}
	.hd-nav {
		display: flex;
		flex-direction: column;
		gap: 4px;
		width: 196px;
		padding: 14px 10px;
		font-family: var(--font-fc-body);
		font-size: 13px;
	}
	.hd-brand {
		display: flex;
		align-items: center;
		gap: 10px;
		height: 36px;
		padding: 0 12px 8px;
		font-family: var(--font-fc-title);
		font-size: 15px;
		font-weight: 700;
	}
	.hd-link {
		display: flex;
		align-items: center;
		gap: 10px;
		height: 36px;
		padding: 0 12px;
		border-radius: 10px;
		color: var(--color-fc-fg-muted);
	}
	.hd-active {
		background: color-mix(in oklch, var(--color-fc-fg) 7%, transparent);
		color: var(--color-fc-fg);
		font-weight: 700;
	}
</style>
