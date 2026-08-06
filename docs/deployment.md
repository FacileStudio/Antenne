# Nook — Deployment

Building the image, the port that must not drift, and putting Nook behind Traefik on la ruche.

## The image

`apps/api/Dockerfile` is a three-stage build:

1. `oven/bun:1` builds the SvelteKit client.
2. `golang:1.25-alpine` builds a static binary with `CGO_ENABLED=0`.
3. `gcr.io/distroless/static-debian12` carries the binary and the built client.

`CLIENT_DIR=/client` is set in the image on purpose. A relative `./client` would resolve
against the working directory, and the dashboard would silently not be served — the API
answers, the healthcheck stays green, and only a browser sees the failure.

The image is **not** the `:nonroot` distroless variant: `/data` is a named volume, which Docker
creates owned by root, and a non-root process could not read the legacy settings import.

## Compose

```sh
docker compose up -d
```

Two services: `postgres` and `app`. `POSTGRES_PASSWORD`, `NOOK_ADMIN_PASSWORD` and
`NOOK_SESSION_SECRET` use the `${VAR:?message}` form, so an unset one fails the compose file
loudly instead of booting an open dashboard with a Postgres nobody can authenticate to.

`app` waits on `service_healthy`, so it does not race the database on a cold start.

### The healthcheck

```yaml
test: ["CMD", "/nook", "healthcheck"]
```

The binary probes itself, which is the only option in a container with no shell and no curl.
`healthcheck.Handle` runs before any config is read, so a missing `DATABASE_URL` produces a
clear startup failure rather than a probe that can never pass.

### The port must not drift

`PORT` is pinned to `9090` literally, in both `expose` and the container environment. It is not
interpolated from a variable, and the healthcheck does not reference one either.

This is deliberate. The webhook URLs registered in Dokploy, DocuSeal and the git forges point
at that port, and so does every pool client's base URL. The previous compose file
interpolated `${PORT:-9090}` into both `expose` and the healthcheck; changing the port from the
dashboard then made Docker mark a perfectly healthy container unhealthy and restart-loop it.

Changing the listener port is a four-place change: the dashboard setting, `PORT`, `expose`, and
the Traefik service port.

## Volumes

| Volume | Holds |
|---|---|
| `nook-postgres` | the database — settings, providers, targets, the event log, spaces, the pool queue |
| `nook-data` | only the legacy `nook.json`, read once during the upgrade |

The Postgres volume bakes in `POSTGRES_PASSWORD` at first init. Changing the variable later
without wiping the volume gives SASL authentication failures against a healthy-looking database
container.

`nook-data` can be dropped once the import has run and the settings are confirmed in the
dashboard.

## Traefik

The compose file carries the labels for la ruche: an HTTP router redirecting to HTTPS, an
HTTPS router with the Let's Encrypt resolver, and one service on port 9090.

```yaml
traefik.http.routers.nook-secure.rule: "Host(`nook.facile.studio`)"
traefik.http.services.nook-svc.loadbalancer.server.port: "9090"
```

There is **no `stripprefix`**, and there must not be. The API lives at `/api` inside the
binary, the webhook receivers and the pool live at the root, and the dashboard is the catch-all.
Stripping a prefix at the edge would break at least one of the three.

The container joins both `default` and `dokploy-network`; Traefik reaches it over the latter,
which is why no host port is published.

## Health

Both are mounted at the root and under `/api`:

```sh
curl https://nook.facile.studio/health
curl https://nook.facile.studio/api/health
```

`/health` reports the process; `/ready` also checks the database. An external monitor should
watch `/api/health` as well as `/health` — a 200 carrying HTML is worse than a 404, because
every monitor reads it as green.

## Logs

Structured JSON on stdout. With `JOURNAL_URL` and `JOURNAL_TOKEN` both set, the same records
are teed to [Journal](https://journal.facile.studio) under the app name `nook`. Use a per-app
key from Journal's Keys page; Nook is not on Journal's docker network, so the URL is
`https://journal.facile.studio/api` — with the `/api` suffix, which the SPA catch-all on the
other end would otherwise answer with a silent 200.

## Upgrading from the Bun release

1. Add a Postgres and set `DATABASE_URL`.
2. Keep `/data` mounted and leave `NOOK_SETTINGS_PATH` pointing at `nook.json`.
3. Deploy. Migrations run at startup; the legacy file is imported on that first boot and the
   import is logged with the provider and target counts.
4. Check the dashboard, then drop `nook-data`.

The dashboard moves from `/dashboard` to the root — old links 301. The webhook paths and the
pool endpoints do not move.
