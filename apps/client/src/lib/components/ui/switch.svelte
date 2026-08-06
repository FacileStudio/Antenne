<script lang="ts">
	import { Switch as SwitchPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils';

	let {
		checked = $bindable(false),
		disabled = false,
		size = 'default',
		class: className = '',
		onCheckedChange = undefined,
		'aria-label': ariaLabel = undefined
	}: {
		checked?: boolean;
		disabled?: boolean;
		size?: 'default' | 'sm';
		class?: string;
		/** For the list rows, which persist the change rather than hold it. */
		onCheckedChange?: (checked: boolean) => void;
		'aria-label'?: string;
	} = $props();
</script>

<SwitchPrimitive.Root
	bind:checked
	{disabled}
	{onCheckedChange}
	aria-label={ariaLabel}
	class={cn(
		'peer inline-flex shrink-0 cursor-pointer items-center rounded-full border border-transparent transition-colors',
		'focus-visible:ring-[3px] focus-visible:ring-ring/40 focus-visible:outline-none',
		'disabled:cursor-not-allowed disabled:opacity-50',
		'data-[state=checked]:bg-primary data-[state=unchecked]:bg-input',
		size === 'sm' ? 'h-4 w-7' : 'h-5 w-9',
		className
	)}
>
	<SwitchPrimitive.Thumb
		class={cn(
			'pointer-events-none block rounded-full bg-background shadow-sm ring-0 transition-transform',
			size === 'sm'
				? 'size-3 translate-x-0.5 data-[state=checked]:translate-x-3.5'
				: 'size-4 translate-x-0.5 data-[state=checked]:translate-x-4.5'
		)}
	/>
</SwitchPrimitive.Root>
