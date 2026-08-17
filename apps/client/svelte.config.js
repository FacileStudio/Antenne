import adapterStatic from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		// SvelteKit stamps every build with a version and hands it to the app as
		// `version` from $app/environment; hooks.client.ts ships it to Journal as
		// `release`, so a stack trace names the deploy that produced it. The
		// default is Date.now(), which is unique and sortable but unreadable in a
		// log — an ISO timestamp is both. Only equality is ever compared.
		version: { name: new Date().toISOString() },
		// A static SPA: the Go binary serves the build directory and falls back
		// to index.html, so client-side routing works without a Node runtime.
		adapter: adapterStatic({
			pages: 'build',
			assets: 'build',
			fallback: 'index.html',
			precompress: false
		}),
		prerender: {
			entries: []
		}
	},
	vitePlugin: {
		dynamicCompileOptions: ({ filename }) =>
			filename.includes('node_modules') ? undefined : { runes: true }
	}
};

export default config;
