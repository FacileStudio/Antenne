# Antenne

Modular alert aggregation service and the suite's event bus — watches providers, receives
webhooks, and delivers notifications to Matrix, Discord, HTTP, SMTP, ntfy and Perception.
Ships with a SvelteKit dashboard for configuration.

## Tech Stack

- **API**: Go 1.25, chi via [`tronc`](https://github.com/FacileStudio/tronc), GORM + PostgreSQL 16
- **Migrations**: goose through `tronc/migrate`, embedded in the binary and applied at startup
- **Client**: SvelteKit 2 / Svelte 5 runes, Tailwind v4, and
  [`muse`](https://github.com/FacileStudio/muse) (`@facile/muse`) — the suite's design system.
  Antenne owns no primitives of its own; Solar icons come from the `iconify-icon` element
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

`casier.yml` pins this repo to the `antenne` project and its `dev` environment, so `mise run
dev` wraps the API in `casier run` and the process starts with `DATABASE_URL`,
`ANTENNE_ADMIN_PASSWORD`, `PUBLIC_URL` and `ANTENNE_SECRET` already in its environment.
`casier run` is network-first with a last-known-good cache, so a Casier outage falls back to
the previously fetched values and says so on stderr. Add a value with
`casier secrets set -p antenne -e dev KEY value`.

Production values live in Casier's `prod` environment and are pushed into Dokploy with
`casier push dokploy -p antenne -e prod <composeId>`; the container reads them from the Dokploy
env, never from Casier at boot. Casier owns and versions the values; Dokploy is the bootstrap
tier.

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
      envelope/             The Antenne event envelope, its HMAC signing and its text rendering
      app/                  The alert pipeline: dedup, envelope, log, route, deliver, retry
      server/               Dashboard API, webhook intake
    modules/
      auth/                 Users, sessions, RequireAuth
      settings/             Providers, delivery targets, normalization, redaction, storage
      delivery/             Senders (Matrix, Discord, HTTP, SMTP, ntfy, Perception) + retry
      events/               Activity log and the SSE fan-out
      providers/            The pollers Antenne drives: website, IMAP, RSS
      webhooks/             The intake registry and one file per provider
      bus/                  The Antenne Pool: router.go, the WebSocket server and its queue
      spaces/               Teams and members
  client/                   SvelteKit dashboard, built to apps/client/build
```

## Routes

The URL layout is not uniform, and that is deliberate — see the conventions below.

- `GET /health`, `/ready` and `/api/health`, `/api/ready`
- `GET /docs`, `/docs/openapi.json` — the generated Scalar reference
- `/api/*` — the dashboard API, session-authenticated
- `/webhook/<provider>` — **at the root**, third-party receivers
- `/antenne/register`, `/antenne/ws` — **at the root** and mirrored under `/api`
- `/dashboard`, `/dashboard/*` — 301 to the equivalent root path
- everything else — the SPA

## Conventions

- **Webhook paths and the pool endpoints stay at the root.** They are registered in Dokploy,
  DocuSeal, the git forges and every pool client; moving them under `/api` breaks all of them
  silently.
- **The `/api` subtree has its own 404.** Without it the SPA catch-all answers an unknown API
  path with 200 and HTML, and a machine client reads that as a successful write.
- **Delivery routing is opt-in with no fallthrough.** A target receives an event only if it
  names the provider id or shares one of its tags. Antenne's own events need the reserved
  `__antenne_system__` id listed explicitly — enabling a system event is not enough.
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
- **The dashboard is muse, and muse owns the palette.** `app.css` is two lines of muse plus a
  `@theme inline` block aliasing the suite's semantic names (`bg-background`, `border-input`, …)
  onto `fc-*`. That alias block is what keeps the shared raw-element login screen rendering, so
  do not delete it, and do not redeclare a colour anywhere. The two lines that look optional are
  not: without `@source '../node_modules/@facile/muse/src'` every muse component renders
  unstyled, and without `optimizeDeps.exclude: ['@facile/muse']` in `vite.config.ts` the dev
  server will not boot — while `vite build` and the whole gate stay green.
- **Reach for a muse component before writing markup.** Feedback is `toast`, confirmation is
  `ConfirmModal`, an empty list is `EmptyState`, a credential is `SecretField`, a settings row is
  `SettingsRow`. Antenne keeps only what muse has no answer for: the provider and delivery
  editors, the event timeline and the tag scale.
- **Tags are the one categorical colour Antenne owns.** muse stops at six chart slots and tags
  are user-created, so `tags.ts` derives a hue from the name and `[data-tag-tone]` in `app.css`
  paints it. Tags are not muse `Badge` tones.
- Icons are the `<iconify-icon>` custom element, registered once in the root layout. Use muse's
  `icons` map for UI chrome; `apps/client/src/lib/icons.ts` maps provider, delivery-target and
  source types to their own marks.
- **Settings sections are routes, not state.** `/settings/<section>` is a real URL so a reload
  and the back button both work; the tab strip lives in `settings/+layout.svelte`. Settings is
  reached from the sidebar's user card, never from a nav row, and the danger zone sits at the
  bottom of Advanced rather than in a tab of its own.
- **Every route needs a registry entry.** Each module owns a `Documentation` value feeding the
  reference at `/docs`; `TestEveryRouteIsDocumented` fails the build on a route without one.

## Authentication

Antenne runs on [`porte`](https://github.com/FacileStudio/porte), the suite's auth kit. porte
owns the credential — `porte_identities`, `porte_sessions`, one hashed opaque token, an
`HttpOnly` `__Host-` cookie for the dashboard and a bearer token for the CLI. Antenne keeps
what porte has no opinion about: what a user is, and who bootstraps the instance.

- `auth.Store` implements `porte.UserStore` and `porte.PasswordUserStore` over Antenne's own
  `users` table. porte/pg's `porte_users` is deliberately **not** used — `users` is referenced
  across this codebase and carries the role.
- `auth.RequireAuth(sessions, resolver)` wraps porte's middleware and hydrates
  `authcontext.Identity` from `users`. porte hands on a user id and nothing else.
- Handlers still read `authcontext.Identity` and never see a cookie, a token or a user row.
- **There is no open mode.** Booting with neither `ANTENNE_ADMIN_PASSWORD` nor `OIDC_ISSUER` is
  a refusal, in every environment. Antenne used to serve a synthetic admin to every caller when
  no password was set, which made any auth assertion written against such an instance worthless.
- **The password is re-seeded, never migrated.** `ANTENNE_ADMIN_PASSWORD` stays the source of
  truth: changing it and restarting re-hashes the credential into `porte_identities` and revokes
  that account's sessions. Do **not** copy Journal's `adoptExistingPasswords` — Antenne's old
  digests were bcrypt over a prehash and no argon2 verifier accepts them.
- **Every account is an administrator**, because Antenne has one tier: the dashboard holds the
  provider credentials, the delivery secrets and the pool secret. Who may sign in through SSO is
  therefore Authentik's decision, expressed as a group binding on the `antenne` application —
  an application with no binding is open to every Authentik user.
- **The client must send `X-Facile-CSRF` on every mutating request.** porte reads the cookie
  before the `Authorization` header and refuses a cookie-authenticated write without it. Forget
  it and every read succeeds while every write 403s, which reads as "the app works, saving is
  broken". `backend.ts` sets it for any method outside GET/HEAD/OPTIONS. Verify with a browser,
  never with a bearer token — bearer is exempt from the rule by construction.
- `/api/events/stream` is an `EventSource`, which can set no headers, so it is cookie-only
  forever. It is a GET, so the CSRF rule does not reach it.

## Environment Variables

| Variable | What it does | Default |
|---|---|---|
| `DATABASE_URL` | Postgres connection string | **required** — the API exits 1 without it |
| `APP_ENV` | `development` / `staging` / `production`. Never gates security behaviour | `development` |
| `PORT` | HTTP port for the API, dashboard, health, webhooks and pool | `9090` |
| `LOG_LEVEL` | `debug` / `info` / `warn` / `error` | `info` |
| `CORS_ALLOWED_ORIGINS` | Comma-separated origins, read through `tronc/env`. `ALLOWED_ORIGINS` and `DOMAINS` remain accepted fallbacks | unset denies every cross-origin caller |
| `ANTENNE_ADMIN_USERNAME` | Login label, and the local part of the bootstrap account's email | `admin` |
| `ANTENNE_ADMIN_PASSWORD` | The bootstrap password login. Required unless `OIDC_ISSUER` is set | — |
| `PUBLIC_URL` | The https origin Antenne is reached on; porte reads the cookie's `Secure` flag and `__Host-` prefix off it | **required in production** |
| `OIDC_ISSUER` / `OIDC_CLIENT_ID` / `OIDC_CLIENT_SECRET` | Single sign-on against Authentik. Unset disables SSO | — |
| `OIDC_REDIRECT_URL` / `OIDC_SUCCESS_URL` | Default to `PUBLIC_URL` + `/api/auth/oidc/callback` and `PUBLIC_URL` + `/` | derived |
| `SSO_ONLY` | Suppresses the password route entirely | `false` |
| `ANTENNE_SECRET` | Shared secret pool clients register with. The dashboard value wins | `pool` |
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

Compose brings up Postgres and the app. `POSTGRES_PASSWORD`, `ANTENNE_ADMIN_PASSWORD` and
`PUBLIC_URL` are required and fail the compose file loudly when unset. The container
probes itself with `/antenne healthcheck`, which answers before any config is read, so a missing
`DATABASE_URL` cannot break the probe.
