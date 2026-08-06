import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
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
