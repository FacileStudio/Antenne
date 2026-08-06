<script lang="ts">
	import { Dialog as SheetPrimitive } from 'bits-ui';
	import type { Snippet } from 'svelte';
	import Icon from '../Icon.svelte';
	import { cn } from '$lib/utils';

	let {
		open = $bindable(false),
		title = '',
		description = '',
		class: className = '',
		children,
		footer
	}: {
		open?: boolean;
		title?: string;
		description?: string;
		class?: string;
		children?: Snippet;
		footer?: Snippet;
	} = $props();
</script>

<SheetPrimitive.Root bind:open>
	<SheetPrimitive.Portal>
		<SheetPrimitive.Overlay
			class="fixed inset-0 z-50 bg-black/40 backdrop-blur-[2px] data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:animate-in data-[state=open]:fade-in-0"
		/>
		<SheetPrimitive.Content
			class={cn(
				'fixed inset-y-0 right-0 z-50 grid w-full max-w-lg grid-rows-[auto_1fr_auto] border-l border-border bg-background shadow-2xl outline-none',
				'data-[state=closed]:animate-out data-[state=closed]:slide-out-to-right data-[state=open]:animate-in data-[state=open]:slide-in-from-right data-[state=open]:duration-200',
				className
			)}
		>
			<div class="flex items-start justify-between gap-3 border-b border-border px-6 py-4">
				<div class="min-w-0">
					<SheetPrimitive.Title class="truncate font-heading text-base font-semibold">
						{title}
					</SheetPrimitive.Title>
					{#if description}
						<SheetPrimitive.Description class="mt-0.5 text-xs text-muted-foreground">
							{description}
						</SheetPrimitive.Description>
					{/if}
				</div>
				<SheetPrimitive.Close
					class="rounded-md p-1 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
					aria-label="Close"
				>
					<Icon icon="solar:close-circle-linear" size={18} />
				</SheetPrimitive.Close>
			</div>

			<div class="grid content-start gap-5 overflow-y-auto px-6 py-5">
				{@render children?.()}
			</div>

			{#if footer}
				<div class="flex flex-wrap justify-end gap-2 border-t border-border px-6 py-4">
					{@render footer()}
				</div>
			{/if}
		</SheetPrimitive.Content>
	</SheetPrimitive.Portal>
</SheetPrimitive.Root>
