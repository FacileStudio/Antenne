import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	/*
	 * muse and @facile/client ship uncompiled source, including `.svelte.ts` rune modules.
	 * Vite's dev-only dependency optimizer hands those to esbuild without the TypeScript transform, so
	 * `utils/toast.svelte.ts` dies on its first type annotation and `vite dev` refuses to
	 * start — while `vite build`, which never runs the optimizer, is perfectly happy.
	 * Excluding the packages leaves them to the svelte plugin, which is what compiles them.
	 */
	optimizeDeps: {
		exclude: ['@facile/muse', '@facile/client']
	},
	server: {
		// The client's API base URL is relative, so `bun run dev` needs this
		// proxy to reach the Go API. The port is hardcoded rather than read from
		// process.env: this project declares node types without installing them,
		// so touching process here turns a svelte-check warning into an error.
		proxy: {
			'/api': { target: 'http://localhost:9090', changeOrigin: true, ws: true }
		}
	}
});
