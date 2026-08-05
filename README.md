# Nook

Alert aggregation and the suite's event bus — Nook watches providers, routes what they emit to
delivery targets, and hosts the Nook Pool that other Facile apps sync through.

It is a Bun service with no HTTP framework: one `Bun.serve` handler, a hand-written router, and a
prebuilt Nuxt dashboard served as static files. Provider and delivery target configuration lives
entirely in `/data/nook.json`, edited through that dashboard — environment variables cover only
core server settings and dashboard auth.

Live at [nook.facile.studio](https://nook.facile.studio).

## What it does

- Monitors websites on an interval and reports up and down transitions
- Polls IMAP inboxes and RSS feeds for new items
- Receives webhooks from Uptime Kuma, DocuSeal, Dokploy, Git forges and Vikunja, and from the
  suite apps Ardoise, Sablier, Perception, Opus, Vision, Plume and Nuage
- Delivers alerts to Matrix, Discord, HTTP webhooks, SMTP, ntfy and Perception, routed per target
  by provider id or provider tag
- Signs structured event envelopes with HMAC-SHA256 for HTTP and Perception targets
- Retries failed deliveries three times with backoff, and keeps a replayable event log
- Hosts the Nook Pool: a WebSocket bus with a persistent offset-based queue that other suite apps
  register with and sync through
- Serves a dashboard at `/dashboard` covering every provider, target, space and event

## Stack

| Layer | Tech |
|---|---|
| API | Bun 1, `Bun.serve` with a hand-written router, no framework |
| Client | Nuxt 4 with SSR off, statically generated, Vue 3, Tailwind CSS 4, Vue Flow |
| Storage | `nook.json` plus sibling JSON files for events and spaces, SQLite (`bun:sqlite`) for the pool queue |
| Deploy | Docker Compose, single container, prebuilt dashboard baked into the image |

## Quick start

```sh
docker compose up -d
```

The service listens on `PORT` (default `9090`) and serves the API, dashboard, health check and
webhook receivers from that single port. Open `http://localhost:9090/dashboard` to configure
providers and delivery targets. The compose file uses `expose` rather than `ports`, so publish the
port or put Traefik in front of it.

### Local development

```sh
bun install
bun run client:build
bun run dev
```

For dashboard-only work, `bun run client:dev` runs Nuxt in dev mode. The static bundle has to be
rebuilt with `bun run client:build` before any dashboard change can ship — the runtime image does
not build it.

## Configuration

| Variable | What it does |
|---|---|
| `PORT` | HTTP port for the API, dashboard, health check and webhooks. Default `9090` |
| `NOOK_SETTINGS_PATH` | Path to the settings file. Default `/data/nook.json` |
| `NOOK_ADMIN_PASSWORD` | When set, the dashboard and its API require login. Unset leaves both open |
| `NOOK_SESSION_SECRET` | Signing key for the dashboard session cookie; random per boot when unset |
| `POOL_SECRET` | Shared secret apps present when registering with the pool |

Everything else — providers, delivery targets, credentials, routing — lives in the settings file
and is edited through the dashboard.

Full reference: [docs/configuration.md](docs/configuration.md).

## Structure

```
apps/
  api/       Bun service — src/ holds the router, providers, delivery, pool and settings
  client/    Nuxt dashboard, generated into apps/api/dashboard/.output/public
docs/        Architecture, configuration, development, deployment, API
```

## Documentation

| Doc | What's in it |
|---|---|
| [Architecture](docs/architecture.md) | Runtime topology, the alert pipeline, the pool protocol, the settings model |
| [Configuration](docs/configuration.md) | Every environment variable, and the `nook.json` model |
| [Development](docs/development.md) | Local setup, the dashboard build step, tests |
| [Deployment](docs/deployment.md) | Docker, the variable port trap, volumes, Traefik routing |
| [API](docs/api.md) | HTTP endpoints, webhook receivers, the pool WebSocket protocol |

---

Part of the [Facile Suite](https://facile.studio) — self-hosted tools for creative studios
and freelancers. One login, zero cloud dependency.
