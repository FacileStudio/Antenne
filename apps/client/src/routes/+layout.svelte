<script lang="ts">
	import type { Snippet } from 'svelte';
	import { browser } from '$app/environment';
	import { Toaster, provideLocale } from '@facile/muse';
	import { bindLocale, localeSource } from '$lib/i18n/index.svelte';
	import { theme } from '$lib/theme.svelte';
	import '../app.css';

	let { children }: { children: Snippet } = $props();

	const i18n = provideLocale(localeSource);
	bindLocale(() => i18n.locale);

	/*
	 * muse renders <iconify-icon> elements and the custom element has to be registered by
	 * the consumer, or every glyph in the library stays an empty box. Imported dynamically
	 * because the package calls customElements.define at module scope.
	 */
	if (browser) {
		void import('iconify-icon');
		theme.restore();
	}
</script>

{@render children()}

<!-- One Toaster for the whole app, outside the router, so a navigation cannot unmount a
     toast mid-flight. The extra bottom padding keeps the stack clear of MobileNav. -->
<Toaster class="pb-28 md:pb-6" />
