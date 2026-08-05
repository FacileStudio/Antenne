# Nook — Deployment

Building the image, the runtime-only constraint, the variable port, and putting Nook behind Traefik
on la ruche.

## The image is runtime-only

`apps/api/Dockerfile` is a two-stage build, but the second stage installs almost nothing:

```
client-build (oven/bun:1)   bun install --frozen-lockfile  →  bunx nuxi generate
runtime      (oven/bun:1)   bun install --production       →  copy apps/api + prebuilt dashboard
```

The runtime stage runs `bun install --production` against `apps/api/package.json`, which declares
no runtime dependencies — the API is pure Bun standard library. The dashboard is **copied**, not
built, from `client-build`'s output at `apps/api/dashboard/.output/public`.

Two consequences:

- The image contains no `node_modules` worth speaking of, and container builds pull no npm
  packages for the service itself. Builds are fast and offline-friendly.
- The dashboard bundle must exist and be current before the image is built. `bun run client:build`
  is a release step, not an optional one. See [development.md](development.md).

The image sets `NOOK_SETTINGS_PATH=/data/nook.json`, declares `VOLUME ["/data"]` and runs
`bun run index.ts`.

## Docker Compose

```sh
docker compose up -d
```

One service, `app`, built from `apps/api/Dockerfile`:

- `expose: "${PORT:-9090}"` — the port is visible on the compose network only, never published to
  the host. That is correct behind Traefik and wrong for a direct local run; add a `ports:` mapping
  if you need one.
- `volumes: app_data:/data`, named `nook_data`. This volume holds `nook.json`,
  `nook-events.json`, `nook-spaces.json` and `pool-events.db`. Everything Nook knows lives here,
  including every credential in plaintext.
- `restart: unless-stopped`.
- A healthcheck that fetches `http://localhost:${PORT:-9090}/health` with Bun and exits non-zero on
  a non-OK response. 10 s interval, 5 s timeout, 3 retries, 30 s start period.

Environment passed through: `NOOK_SETTINGS_PATH`, `PORT`, `NOOK_SESSION_SECRET`,
`NOOK_ADMIN_PASSWORD`, `NOOK_ADMIN_USERNAME`. `POOL_SECRET` is not in the compose file — add it, or
set `pool.secret` in the dashboard, otherwise the pool runs on the default secret `pool`.

## The variable port, and why the healthcheck cares

Nook reads its listen port from `providers.port` in the settings file, not from the environment.
`PORT` only seeds that value the first time, when no settings file exists yet.

Both `expose` and the healthcheck interpolate `${PORT:-9090}`. So if someone changes the port in
the dashboard:

- the service starts listening on the new port,
- the healthcheck keeps probing the old one and fails,
- Docker marks a perfectly healthy container unhealthy, and Dokploy will restart it in a loop.

Two safe options. Either leave the port alone in the dashboard and drive it entirely from `PORT`,
or change both together and redeploy. The startup log prints `[startup] server port: <n>` and
`[startup] listening on <n>` — check those against the compose file after any settings change that
touches the port.

The port is also the only setting that requires a restart. Everything else applies live on
`PUT /api/settings`.

## Behind Traefik on la ruche

One hostname, routed to the container's port, no path rewriting. Nook serves the dashboard, the
API, the health check, the webhook receivers and the pool WebSocket all from that same port and
distinguishes them by path, so splitting them across routers gains nothing and breaks the
dashboard's relative asset paths.

The router must support WebSocket upgrades on `/api/pool/ws`; Traefik does by default.

Set at minimum:

```sh
NOOK_ADMIN_PASSWORD=<a real password>
NOOK_SESSION_SECRET=<openssl rand -hex 32>
POOL_SECRET=<openssl rand -hex 32>
```

Without the first, the dashboard and the entire settings API — including every stored credential —
are readable and writable by anyone who can reach the hostname. The process logs a warning at
startup, but it starts anyway.

Rate limiting on login keys off `x-forwarded-for`, falling back to `cf-connecting-ip` then
`x-real-ip`. The proxy must **overwrite** `x-forwarded-for` rather than append to it, or a client
can forge a fresh identity per attempt and the limiter is decorative. Nook logs a warning about
this at every startup.

Prefer the `dokploy` CLI over SSH plus docker for managing the deployment.

## Data and backups

Everything is on the `/data` volume:

| File | Contents | Loss impact |
|---|---|---|
| `nook.json` | Providers, targets, all credentials, port, pool secret | Total reconfiguration |
| `nook-events.json` | Last 500 events with delivery outcomes | History and replay gone |
| `nook-spaces.json` | Spaces and members | Grouping gone |
| `pool-events.db` | SQLite pool queue and app registrations | Clients resync from a new epoch |

Losing `pool-events.db` is survivable by design: the pool epoch is a fresh UUID on every boot and
is sent in the `welcome` message, so clients detect the reset and resynchronize instead of trusting
stale offsets. Losing `nook.json` is not survivable — back it up, and remember it is a secrets
file.

Pool retention is interest-based, not time-based: events stay until every active listener on a
channel has acknowledged them. A listener that disappears without deregistering pins its channel's
events for 14 days before Nook stops counting it. Cleanup runs once every 24 hours, so the database
grows between passes.

## Health

`GET /health` returns `{"status":"ok"}` as soon as `Bun.serve` is listening. It does not check the
settings file, the dashboard bundle, the pool database or any provider. A green health check with a
missing dashboard bundle serves a 404 at `/dashboard` and reports itself perfectly healthy — verify
the dashboard loads after a deploy, and check `[startup] dashboard: built` in the logs.

`GET /api/pool/stats` (session-authenticated) reports the pool epoch and connection state, which is
the quickest way to confirm client apps are actually attached.
