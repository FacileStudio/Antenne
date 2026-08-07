# Antenne

Alert aggregation and the suite's event bus — Antenne watches providers, routes what they emit to
delivery targets, and hosts the Antenne Pool that other Facile apps sync through.

It is a Go service on the [`tronc`](https://github.com/FacileStudio/tronc) chassis: chi, GORM
and PostgreSQL, with goose migrations embedded in the binary. One container serves the API,
the webhook receivers, the pool and a SvelteKit dashboard. Provider and delivery
configuration is edited entirely from that dashboard; environment variables cover the server
itself and its bootstrap credentials.

Live at [antenne.facile.studio](https://antenne.facile.studio).

## What it does

- Monitors websites on an interval and reports up and down transitions
- Polls IMAP inboxes and RSS feeds for new items
- Receives webhooks from Uptime Kuma, DocuSeal, Dokploy, Git forges and Vikunja, and from the
  suite apps Ardoise, Sablier, Perception, Opus, Vision, Plume and Nuage
- Delivers alerts to Matrix, Discord, HTTP webhooks, SMTP, ntfy and Perception, routed per
  target by provider id or provider tag
- Signs structured event envelopes with HMAC-SHA256 for HTTP and Perception targets
- Retries failed deliveries three times with backoff, and keeps a replayable event log
- Hosts the Antenne Pool: a WebSocket bus with a durable offset-based queue that other suite apps
  register with and sync through

## Stack

| Layer | Tech |
|---|---|
| API | Go 1.25, `tronc` (chi), GORM, PostgreSQL 16, `tronc/migrate` (goose) |
| Client | SvelteKit 2, Svelte 5 runes, Tailwind v4, shadcn-svelte over bits-ui, Solar icons |
| Deploy | Docker Compose, distroless image, the dashboard baked in |

## Quick start

```sh
docker compose up -d
```

Compose starts Postgres and the app. `POSTGRES_PASSWORD`, `ANTENNE_ADMIN_PASSWORD` and
`ANTENNE_SESSION_SECRET` are required. The service listens on `9090` and serves the API, the
dashboard, the health check, the webhook receivers and the pool from that one port. Compose
uses `expose` rather than `ports`, so publish the port or put Traefik in front of it.

### Local development

```sh
mise run install    # client dependencies
mise run dev        # the API, with secrets injected from Casier
mise run client     # the dashboard dev server, proxying /api to the API
```

Configuration for the developer loop comes from [Casier](https://casier.facile.studio);
`.casier.toml` pins this repo to the `antenne` project. `mise run dev-offline` runs from the
cached values. See [docs/development.md](docs/development.md).

## Configuration

| Variable | What it does |
|---|---|
| `DATABASE_URL` | Postgres connection string. Required |
| `PORT` | HTTP port for the API, dashboard, health check, webhooks and pool. Default `9090` |
| `ANTENNE_ADMIN_PASSWORD` | When set, the dashboard and its API require login. Required in production |
| `ANTENNE_SESSION_SECRET` | Keeps sessions alive across a restart. Required in production |
| `ANTENNE_SECRET` | Shared secret apps present when registering with the pool |

Everything else — providers, delivery targets, credentials, routing — lives in the database
and is edited through the dashboard.

Full reference: [docs/configuration.md](docs/configuration.md).

## Structure

```
Dockerfile   Multi-stage build: the client with Bun, then the Go binary
apps/
  api/       Go service — internal/ holds the pipeline and plumbing, modules/ the domains
  client/    SvelteKit dashboard, built into apps/client/build
docs/        Architecture, configuration, development, deployment, API
```

## Documentation

| Doc | What's in it |
|---|---|
| [Architecture](docs/architecture.md) | Runtime topology, the alert pipeline, the pool protocol, the settings model |
| [Configuration](docs/configuration.md) | Every environment variable and the settings model |
| [Development](docs/development.md) | Local setup, Casier, the quality gate, tests |
| [Deployment](docs/deployment.md) | Docker, Postgres, volumes, Traefik routing |
| [API](docs/api.md) | HTTP endpoints, webhook receivers, the pool WebSocket protocol |

A running instance also serves a generated OpenAPI reference at `/docs`.

---

Part of the [Facile Suite](https://facile.studio) — self-hosted tools for creative studios
and freelancers. One login, zero cloud dependency.
