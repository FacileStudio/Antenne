# Nook

Modular alert aggregation service and the suite's event bus — watches providers, receives
webhooks, and delivers notifications to Matrix, Discord, HTTP, SMTP, ntfy and Perception.
Ships with a SvelteKit dashboard for configuration.

## Tech Stack

- **API**: Go 1.25, chi via [`tronc`](https://github.com/FacileStudio/tronc), GORM + PostgreSQL 16
- **Migrations**: goose through `tronc/migrate`, embedded in the binary and applied at startup
- **Client**: SvelteKit 2 / Svelte 5 runes, Tailwind v4, shadcn-svelte primitives over bits-ui,
  Solar icons inlined at build time
- **Docker**: multi-stage — the client is built with Bun, then copied into a distroless Go image
- **Deploy**: one container serving the API, the webhook receivers, the pool and the dashboard

## Commands

Run from the repository root. Development configuration comes from
[Casier](https://casier.facile.studio), not from a `.env`.

| Task | Command |
|---|---|
| Run the API | `mise run dev` |
| Run the API offline | `mise run dev-offline` |
| Run the dashboard | `mise run client` |
| Apply migrations only | `mise run migrate` |
| Quality gate | `mise run check` |
| Go only | `mise run check-go` |
| Format Go | `mise run format` |

`.casier.toml` pins this repo to the `nook` project and its `dev` environment, so `mise run
dev` wraps the API in `casier run` and the process starts with `DATABASE_URL`,
`NOOK_ADMIN_PASSWORD`, `NOOK_SESSION_SECRET` and `POOL_SECRET` already in its environment.
`casier run` is network-first with a last-known-good cache, so a Casier outage falls back to
the previously fetched values and says so on stderr. Add a value with
`casier secrets set -p nook -e dev KEY value`.

Production values are injected by Dokploy; Casier covers the developer loop.

## Project Structure

```
Dockerfile                  Multi-stage build: the client with Bun, then the Go binary
apps/
  api/
    main.go                 Entry point: config, migrations, router, graceful shutdown
    migrations/             goose SQL, embedded
    internal/
      env/                  Config; embeds troncenv.Core
      database/             The GORM handle
      authcontext/          Identity on the request context — the auth seam
      ratelimit/            Fixed-window counter for login attempts
      envelope/             The Nook event envelope, its HMAC signing and its text rendering
      app/                  The alert pipeline: dedup, envelope, log, route, deliver, retry
      server/               Dashboard API, webhook intake
    modules/
      auth/                 Users, sessions, RequireAuth
      settings/             Providers, delivery targets, normalization, redaction, storage
      delivery/             Senders (Matrix, Discord, HTTP, SMTP, ntfy, Perception) + retry
      events/               Activity log and the SSE fan-out
      providers/            The pollers Nook drives: website, IMAP, RSS
      webhooks/             The intake registry and one file per provider
      pool/                 The Nook Pool: WebSocket server and its Postgres-backed queue
      spaces/               Teams and members
  client/                   SvelteKit dashboard, built to apps/client/build
```

## Routes

The URL layout is not uniform, and that is deliberate — see the conventions below.

- `GET /health`, `/ready` and `/api/health`, `/api/ready`
- `GET /docs`, `/docs/openapi.json` — the generated Scalar reference
- `/api/*` — the dashboard API, session-authenticated
- `/webhook/<provider>` — **at the root**, third-party receivers
- `/pool/register`, `/pool/ws` — **at the root** and mirrored under `/api`
- `/dashboard`, `/dashboard/*` — 301 to the equivalent root path
- everything else — the SPA

## Conventions

- **Webhook paths and the pool endpoints stay at the root.** They are registered in Dokploy,
  DocuSeal, the git forges and every pool client; moving them under `/api` breaks all of them
  silently.
- **The `/api` subtree has its own 404.** Without it the SPA catch-all answers an unknown API
  path with 200 and HTML, and a machine client reads that as a successful write.
- **Delivery routing is opt-in with no fallthrough.** A target receives an event only if it
  names the provider id or shares one of its tags. Nook's own events need the reserved
  `__nook_system__` id listed explicitly — enabling a system event is not enough.
- **Secrets are redacted on the way out and merged back on the way in.** The dashboard only
  ever holds `••••••••`; `settings.MergeSecrets` restores the stored value before
  normalization.
- **Long-running work hangs off the process context, never a request context.** Re-applying
  the runtime under `r.Context()` starts every poller and has it cancelled when the response
  is written.
- Webhook providers live in `apps/api/modules/webhooks/provider_*.go`, one file per provider,
  registered in `webhooks.Registry`.
- Delivery targets are sender methods in `modules/delivery`.
- Provider and delivery config is edited through the dashboard, not through env vars.
- Icons are inlined at build time from `@iconify-json/solar`; add a name to
  `apps/client/scripts/build-icons.ts` and run `bun run icons`.
- **Every route needs a registry entry.** Each module owns a `Documentation` value feeding the
  reference at `/docs`; `TestEveryRouteIsDocumented` fails the build on a route without one.

## Authentication

Nook owns a `users` table, an opaque session token hashed at rest, and a password login. No
OIDC and no shared identity engine yet — that is deliberate, and the whole implementation
sits behind one interface so it can be replaced without touching a handler:

- `auth.Authenticator` is the seam. `auth.Service` is Nook's implementation of it.
- `auth.RequireAuth(authenticator)` is the only thing routers use.
- Handlers read `authcontext.Identity` and never see a cookie, a token or a user row.

When the suite's unified auth package lands, it supplies a new `Authenticator` and its own
login routes. `Identity` keeps its shape — `UserID` / `Email` / `Name` / `Role` / `ClientIP`,
with the suite's `USER` / `ADMIN` hierarchy — so nothing downstream changes.

`NOOK_ADMIN_PASSWORD` stays the source of truth for the bootstrap account: changing it and
restarting re-seeds the password and revokes that user's sessions, which is the recovery path.

## Environment Variables

| Variable | What it does | Default |
|---|---|---|
| `DATABASE_URL` | Postgres connection string | **required** — the API exits 1 without it |
| `APP_ENV` | `development` / `staging` / `production`. Never gates security behaviour | `development` |
| `PORT` | HTTP port for the API, dashboard, health, webhooks and pool | `9090` |
| `LOG_LEVEL` | `debug` / `info` / `warn` / `error` | `info` |
| `CORS_ALLOWED_ORIGINS` | Comma-separated origins, read through `tronc/env`. `ALLOWED_ORIGINS` and `DOMAINS` remain accepted fallbacks | unset denies every cross-origin caller |
| `NOOK_ADMIN_USERNAME` | Login label, and the local part of the bootstrap account's email | `admin` |
| `NOOK_ADMIN_PASSWORD` | Enables login. Unset leaves the dashboard and its API open | — (required in production) |
| `NOOK_SESSION_SECRET` | Required in production so sessions survive a restart | — |
| `POOL_SECRET` | Shared secret pool clients register with. The dashboard value wins | `pool` |
| `TIMEOUT` | Website check timeout, in seconds | `10` |
| `JOURNAL_URL` / `JOURNAL_TOKEN` | Ship logs to Journal. Both must be set | — |

`PORT` deliberately does **not** follow `troncenv.Core`'s 8080 default: the published webhook
URLs and every pool client point at 9090, and the client's dev proxy targets it too.

## Testing

Go tests live beside the source (`modules/settings/settings_test.go`). Run `mise run check`
for the full gate: gofmt, vet, `go test ./...`, then `svelte-check` on the client.

Postgres is the only database, tests included — the gate fails the build if a SQLite or MySQL
driver appears in `go.mod`.

## Docker

```sh
docker compose up -d
```

Compose brings up Postgres and the app. `POSTGRES_PASSWORD`, `NOOK_ADMIN_PASSWORD` and
`NOOK_SESSION_SECRET` are required and fail the compose file loudly when unset. The container
probes itself with `/nook healthcheck`, which answers before any config is read, so a missing
`DATABASE_URL` cannot break the probe.
