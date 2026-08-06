<script lang="ts" module>
	import { tv, type VariantProps } from 'tailwind-variants';

	export const badgeVariants = tv({
		base: 'inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[0.7rem] leading-none font-medium whitespace-nowrap',
		variants: {
			variant: {
				default: 'border-border bg-muted text-muted-foreground',
				outline: 'border-border bg-transparent text-muted-foreground',
				success: 'border-success/25 bg-success/10 text-success',
				destructive: 'border-destructive/25 bg-destructive/10 text-destructive'
			}
		},
		defaultVariants: { variant: 'default' }
	});

	export type BadgeVariant = VariantProps<typeof badgeVariants>['variant'];
</script>

<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '$lib/utils';

	let {
		variant = 'default',
		title = undefined,
		class: className = '',
		children
	}: { variant?: BadgeVariant; title?: string; class?: string; children?: Snippet } = $props();
</script>

<span class={cn(badgeVariants({ variant }), className)} {title}>
	{@render children?.()}
</span>
