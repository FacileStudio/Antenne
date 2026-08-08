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
| `HEARTBEAT_URL` | Dead man's switch. Empty disables it. See below |
| `HEARTBEAT_INTERVAL` | How often to ping it. Default `5m`, rejected under `1m` |

Everything else — providers, delivery targets, credentials, routing — lives in the database
and is edited through the dashboard.

Full reference: [docs/configuration.md](docs/configuration.md).

## Target health

Antenne watches its own delivery targets, because a channel that has quietly stopped working
is the one failure it cannot otherwise report. Each target is probed **read-only** — nothing is
delivered — on its own schedule: Matrix and ntfy every 5 minutes, Discord and Perception every
10, SMTP every 30 because each probe there is a full TLS handshake plus a credential
presentation, and repeated auth failures feed fail2ban and provider lockouts.

A **generic webhook is never probed** and stays `unknown`. There is no safe read-only check for
an arbitrary URL: `GET` may be unimplemented, may mean something else, or may be answered by a
proxy that never reaches the handler, so both the pass and the fail are meaningless. `unknown`
is an honest answer, not a failure — a green dot nobody can justify is worse than a grey one.

States are `ok`, `pending` (failing, below the threshold), `down`, `flapping`, `stale`,
`unknown`, and `probe_error` — the last meaning *our* check broke, not the target.

You are told on the **second** consecutive failure, once, and then on a fixed four-hour repeat
until it recovers; recovery is announced with the downtime. A failure the retry queue fixes
within seconds raises nothing, deliberately. Recovery needs two consecutive successes *and* ten
minutes, because the symmetric rule makes a target that fails every other probe announce itself
forever. A target that changes state too often is marked `flapping` and its per-transition
notices are replaced by one "unstable".

An alert about a target is delivered through the **other** healthy targets, never the sick one.
When nothing can carry it, it goes to the log at error level, which Journal receives.

Probing **stops** on a credential the far side has rejected outright. That is a safety property:
Discord blocks an IP for about an hour after roughly 10k invalid requests in 10 minutes, so a
loop of dead-webhook probes is how one broken target takes the whole instance offline. The
dashboard's *Check now* button is the only way past that suspension.

Two endpoints back this: `GET /api/delivery-health` for every target's state, and
`POST /api/delivery-health/{id}/check` to probe one now (10 per minute). A **declined handshake
is a 200** with the reason in the body — the diagnostic worked, the far side declined. Non-2xx
is reserved for our own faults: 429 for that budget, 404, 500.

### The dead man's switch

None of the above can report that the process is dead, or that every channel is down at once.
That needs something outside Antenne. Set `HEARTBEAT_URL` and Antenne pings it on a schedule;
whatever receives it alarms when the pings **stop**. It works with healthchecks.io, Dead Man's
Snitch or Grafana OnCall today, and — since it is only an HTTP ping to a configurable URL — with
a self-hosted Sonde once that exists.

The ping is withheld unless the pipeline has actually completed a cycle, so a wedged process
goes quiet rather than reporting itself healthy. Jitter is downward-only, because symmetric
jitter sends some pings late and spends the receiver's grace time on nothing. Give the receiver
a period above `HEARTBEAT_INTERVAL` and real grace.

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
