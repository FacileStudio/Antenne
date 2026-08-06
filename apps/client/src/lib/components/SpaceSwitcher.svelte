<script lang="ts">
	import { DropdownMenu } from 'bits-ui';
	import Icon from './Icon.svelte';
	import { spaces } from '$lib/spaces.svelte';
	import { cn } from '$lib/utils';

	$effect(() => {
		void spaces.load();
	});
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger
		disabled={spaces.loading}
		class="flex w-full items-center gap-2 rounded-md border border-sidebar-border bg-background px-2.5 py-2 text-sm transition-colors hover:bg-sidebar-accent disabled:opacity-50"
	>
		<Icon icon="solar:buildings-2-linear" size={15} class="text-muted-foreground" />
		<span class="min-w-0 flex-1 truncate text-left">
			{spaces.current?.name ?? 'No space'}
		</span>
		<Icon icon="solar:alt-arrow-down-linear" size={12} class="text-muted-foreground" />
	</DropdownMenu.Trigger>

	<DropdownMenu.Portal>
		<DropdownMenu.Content
			sideOffset={4}
			class="z-50 min-w-[13.5rem] rounded-md border border-border bg-popover p-1 text-popover-foreground shadow-lg outline-none"
		>
			<DropdownMenu.Group>
				<DropdownMenu.GroupHeading class="px-2 py-1.5 text-xs text-muted-foreground">
					Spaces
				</DropdownMenu.GroupHeading>

			{#each spaces.spaces as space (space.id)}
				<DropdownMenu.Item
					onSelect={() => spaces.select(space.id)}
					class={cn(
						'flex cursor-pointer items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-none',
						'data-highlighted:bg-accent data-highlighted:text-accent-foreground',
						space.id === spaces.currentId && 'font-medium'
					)}
				>
					<Icon icon="solar:buildings-2-linear" size={14} class="text-muted-foreground" />
					<span class="truncate">{space.name}</span>
				</DropdownMenu.Item>
			{/each}

				{#if spaces.spaces.length === 0}
					<p class="px-2 py-1.5 text-xs text-muted-foreground">No space yet.</p>
				{/if}
			</DropdownMenu.Group>

			<DropdownMenu.Separator class="my-1 h-px bg-border" />

			<DropdownMenu.Item
				class="flex cursor-pointer items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-none data-highlighted:bg-accent data-highlighted:text-accent-foreground"
			>
				{#snippet child({ props })}
					<a href="/spaces" {...props}>
						<Icon icon="solar:settings-linear" size={14} class="text-muted-foreground" />
						Manage spaces
					</a>
				{/snippet}
			</DropdownMenu.Item>
		</DropdownMenu.Content>
	</DropdownMenu.Portal>
</DropdownMenu.Root>
