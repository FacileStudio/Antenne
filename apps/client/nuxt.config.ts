export default defineNuxtConfig({
  compatibilityDate: "2026-04-25",
  ssr: false,
  devtools: { enabled: false },
  app: {
    baseURL: "/dashboard/",
    head: {
      title: "Nook Dashboard",
      meta: [
        { name: "viewport", content: "width=device-width, initial-scale=1" },
      ],
      link: [
        { rel: "icon", type: "image/webp", href: "/dashboard/nook-avatar.webp" },
      ],
    },
  },
  nitro: {
    prerender: {
      routes: ["/"],
    },
    output: {
      dir: "../api/dashboard/.output",
      publicDir: "../api/dashboard/.output/public",
    },
  },
  css: ["~/assets/main.css"],
});
