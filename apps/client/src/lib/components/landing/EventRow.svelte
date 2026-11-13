<script lang="ts">
	import { Bar, grow } from '@facile/mise-en-scene';
	import { Badge, Icon, icons } from '@facile/muse';
	import { sourceIcon } from '$lib/icons';
	import { t } from '$lib/i18n/index.svelte';

	type Delivery = { name: string; ok: boolean };

	let {
		title,
		provider = '',
		deliveries = [],
		replayable = false,
		replay = $bindable()
	}: { title?: string; provider?: string; deliveries?: Delivery[]; replayable?: boolean; replay?: HTMLElement } = $props();
</script>

<div class="er-row">
	{#if title}
		<span class="er-chip"><iconify-icon icon={sourceIcon('webhook')} width="14" height="14" class="block"></iconify-icon></span>
		<div class="er-main">
			<div class="er-top">
				<span class="er-title">{title}</span>
				<span class="er-time">{t('common.justNow')}</span>
			</div>
			<div class="er-badges">
				<Badge tone="info">{provider}</Badge>
				{#each deliveries as delivery (delivery.name)}
					<Badge tone={delivery.ok ? 'success' : 'danger'}>
						<iconify-icon icon={delivery.ok ? 'solar:check-circle-linear' : 'solar:close-circle-linear'} width="12" height="12" class="er-ico"></iconify-icon>
						{delivery.name}
					</Badge>
				{/each}
			</div>
			{#if replayable}
				<div transition:grow={{ duration: 360 }}>
					<span bind:this={replay} class="er-replay">
						<Icon icon={icons.refresh} size={14} />
						{t('events.replay')}
					</span>
				</div>
			{/if}
		</div>
	{:else}
		<span class="er-chip er-blank"></span>
		<div class="er-main er-gap">
			<Bar w="42%" />
			<div class="er-badges"><Bar w={60} h={18} r={9} /><Bar w={52} h={18} r={9} /></div>
		</div>
	{/if}
</div>

<style>
	.er-row {
		display: flex;
		align-items: flex-start;
		gap: 12px;
		padding: 14px 16px;
	}
	.er-chip {
		display: flex;
		flex: none;
		margin-top: 1px;
		padding: 6px;
		border-radius: 8px;
		background: color-mix(in oklch, var(--color-fc-fg) 7%, transparent);
		color: var(--color-fc-fg-muted);
	}
	.er-blank {
		width: 26px;
		height: 26px;
	}
	.er-main {
		display: flex;
		flex: 1;
		flex-direction: column;
		gap: 8px;
		padding-bottom: 2px;
		min-width: 0;
	}
	.er-gap {
		gap: 12px;
		padding-top: 6px;
	}
	.er-top {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 12px;
	}
	.er-title {
		overflow: hidden;
		font-size: 14px;
		font-weight: 700;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.er-time {
		flex: none;
		color: var(--color-fc-fg-muted);
		font-size: 12px;
	}
	.er-badges {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px;
	}
	.er-replay {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		width: fit-content;
		height: 30px;
		padding: 0 12px;
		margin-left: -12px;
		border-radius: 999px;
		color: var(--color-fc-fg);
		font-size: 13px;
		font-weight: 700;
	}
	.er-ico {
		display: inline-block;
		margin-right: 4px;
		vertical-align: -2px;
	}
</style>
