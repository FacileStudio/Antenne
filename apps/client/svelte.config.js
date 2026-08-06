import adapterStatic from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
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
