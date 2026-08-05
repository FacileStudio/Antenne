# Nook — Development

Local setup, the dashboard build step that catches everyone out, and how to test.

## Prerequisites

| Tool | Version | Why |
|---|---|---|
| Bun | 1.x | The runtime, the package manager, the test runner and the SQLite driver |
| Docker | any recent | Only for a container run |

There is no Node dependency, no `mise.toml`, no `scripts/check.sh` and no git hooks. The API has no
runtime dependencies at all — `apps/api/package.json` declares only `@types/bun` and TypeScript as
dev dependencies. Everything the service needs comes from Bun's standard library: `Bun.serve`,
`Bun.file`, `bun:sqlite`, `node:crypto`.

## Setup

```sh
bun install
bun run client:build
bun run dev
```

`bun run dev` runs `apps/api/index.ts` directly. There is no build step for the API and no watch
mode wired up in the scripts.

Point the settings file somewhere local rather than at `/data`:

```sh
NOOK_SETTINGS_PATH=./data/nook.json bun run dev
```

The directory is created on the first save. Set `NOOK_ADMIN_PASSWORD` too unless you want the
dashboard open, and note the startup warning if you do not.

## The dashboard build step

This is the constraint that shapes the whole workflow: **the Docker image never builds the
dashboard from source.** It copies a prebuilt bundle. So the generated output has to be committed
and current before anything ships.

```sh
bun run client:dev     # Nuxt dev server, for iterating on the UI
bun run client:build   # bunx nuxi generate — regenerate the static bundle
```

`apps/client/nuxt.config.ts` sets `ssr: false`, `app.baseURL: "/dashboard/"` and redirects Nitro's
output to `../api/dashboard/.output`, so `client:build` writes straight into
`apps/api/dashboard/.output/public`, which is what the API serves and what the image copies.

Ship a dashboard change without running `client:build` and the deployed UI stays on the previous
bundle, with no error anywhere to tell you.

At startup the API checks whether `dashboard/.output/public/index.html` exists and logs
`[startup] dashboard: built` or `not built`. Check that line first when the dashboard 404s.

## Tests

```sh
bun test
```

The root `test` script is `bun test apps/api/src/*.test.ts`. Two suites exist: `delivery.test.ts`
covers target selection and the delivery pipeline, `server.test.ts` covers routing and auth.

Typecheck the API:

```sh
bun run typecheck
```

## Reading the startup log

Nook is deliberately loud on boot, and the log is the fastest diagnostic available:

```
[startup] settings path: /data/nook.json
[startup] persisted settings: yes
[startup] dashboard: built
[startup] server port: 9090
[startup] delivery targets: 3
[startup] matrix rooms: !abc:example.org
[startup] providers: 7 total
[startup] websites: 2
[startup] imap: 1
[startup] rss: 1
[startup] webhooks: 3
[startup] pool: enabled (epoch=…)
[startup] listening on 9090
```

Two security warnings follow when they apply: one when `NOOK_ADMIN_PASSWORD` is unset, one about
`x-forwarded-for` being trusted for rate limiting. Neither is optional to fix in production.

Delivery produces one summary line per event, listing how many targets succeeded, failed and were
filtered out, with the target names. A `filtered` count equal to the number of targets means the
routing rules match nothing — check `providerIds` and `providerTags` on the targets.

## Conventions

- One file per webhook provider in `apps/api/src/webhooks/providers/`, registered by type in
  `webhooks/index.ts`. A new provider needs a `WebhookProviderDefinition`, an entry in that map, a
  `WebhookProviderType` union member, and defaults in `provider-config.ts`.
- Delivery targets are sender functions in `delivery-senders.ts`, selected by the type guards in
  `delivery-config.ts`.
- HTTP and Perception deliveries always carry a signed envelope; the other targets carry the HTML
  message.
- Pull providers expose an `apply(settings)` manager so a settings change reconfigures them without
  a restart. Anything new that polls should follow that shape.
- Settings mutations go through `SettingsStore.update`, never by writing the file directly. The
  store normalizes, validates, persists and re-applies in one call.
- No inline comments in code.
