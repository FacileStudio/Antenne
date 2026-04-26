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
        { rel: "icon", type: "image/png", href: "/dashboard/nook-avatar.png" },
      ],
    },
  },
  nitro: {
    prerender: {
      routes: ["/"],
    },
  },
  css: ["~/assets/main.css"],
});
