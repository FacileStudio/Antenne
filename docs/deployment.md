# Antenne — Deployment

Building the image, the port that must not drift, and putting Antenne behind Traefik on la ruche.

## The image

`Dockerfile` at the repository root is a three-stage build — it has to be there, because it
builds both `apps/client` and `apps/api` and needs the whole repo as its context:

1. `oven/bun:1` builds the SvelteKit client.
2. `golang:1.25-alpine` builds a static binary with `CGO_ENABLED=0`.
3. `gcr.io/distroless/static-debian12` carries the binary and the built client.

`CLIENT_DIR=/client` is set in the image on purpose. A relative `./client` would resolve
against the working directory, and the dashboard would silently not be served — the API
answers, the healthcheck stays green, and only a browser sees the failure.

It runs as `:nonroot`. The binary writes nothing to disk — every byte of state lives in
Postgres — so there is no volume whose ownership could fight the process.

## Compose

```sh
docker compose up -d
```

Two services: `postgres` and `app`. `POSTGRES_PASSWORD`, `ANTENNE_ADMIN_PASSWORD` and
`PUBLIC_URL` use the `${VAR:?message}` form, so an unset one fails the compose file
loudly instead of booting an open dashboard with a Postgres nobody can authenticate to.

`app` waits on `service_healthy`, so it does not race the database on a cold start.

### The healthcheck

```yaml
test: ["CMD", "/antenne", "healthcheck"]
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

One volume, `antenne-postgres`, holding everything: settings, providers, targets, the event log,
spaces and the pool queue. Nothing else is written to disk.

It bakes in `POSTGRES_PASSWORD` at first init. Changing the variable later without wiping the
volume gives SASL authentication failures against a healthy-looking database container.

## Traefik

The compose file carries the labels for la ruche: an HTTP router redirecting to HTTPS, an
HTTPS router with the Let's Encrypt resolver, and one service on port 9090.

```yaml
traefik.http.routers.antenne-secure.rule: "Host(`antenne.facile.studio`)"
traefik.http.services.antenne-svc.loadbalancer.server.port: "9090"
```

There is **no `stripprefix`**, and there must not be. The API lives at `/api` inside the
binary, the webhook receivers and the pool live at the root, and the dashboard is the catch-all.
Stripping a prefix at the edge would break at least one of the three.

The container joins both `default` and `dokploy-network`; Traefik reaches it over the latter,
which is why no host port is published.

## Health

Both are mounted at the root and under `/api`:

```sh
curl https://antenne.facile.studio/health
curl https://antenne.facile.studio/api/health
```

`/health` reports the process; `/ready` also checks the database. An external monitor should
watch `/api/health` as well as `/health` — a 200 carrying HTML is worse than a 404, because
every monitor reads it as green.

## Logs

Structured JSON on stdout. With `JOURNAL_URL` and `JOURNAL_TOKEN` both set, the same records
are teed to [Journal](https://journal.facile.studio) under the app name `antenne`. Use a per-app
key from Journal's Keys page; Antenne is not on Journal's docker network, so the URL is
`https://journal.facile.studio/api` — with the `/api` suffix, which the SPA catch-all on the
other end would otherwise answer with a silent 200.

### Dashboard errors

The API's log covers the server. What the dashboard itself throws in someone's browser — a
component that failed to render, a load that rejected — never reaches stdout, so it ships
separately: with `JOURNAL_BROWSER_URL` and `JOURNAL_BROWSER_KEY` both set, `hooks.client.ts`
reports it to the same Journal instance with `meta.source = "browser"`.

`JOURNAL_BROWSER_KEY` is a **public** key, minted in Journal under **Settings → API** with this
deployment's origin on its allowlist. It is served to every browser that loads the dashboard,
which is expected — a public key is not a secret, and the origin allowlist, the 60/min rate
limit and the daily quota are what protect the instance. Do not put the server's
`JOURNAL_TOKEN` here.

The two pairs are separate variables because they are reached from different places.
`JOURNAL_URL` may be a docker-internal address; `JOURNAL_BROWSER_URL` has to be the public
origin, and it too must carry the `/api` suffix. Antenne **refuses to boot** without it: the
browser SDK never throws, so the failure would otherwise be a dashboard reporting diligently
into Journal's HTML catch-all with nothing anywhere to show it.

The key is not baked into the bundle. The client is `adapter-static` and has no environment to
read, so the API hands the pair over on `GET /api/auth/config`, which the dashboard already
calls before it renders — rotating the key is a restart, not a rebuild. With either variable
unset the key is absent from that payload and the client stays inert.

Every report carries a `release`, which is SvelteKit's own build version — `kit.version.name` in
`svelte.config.js`, an ISO timestamp. Nothing has to be passed at build time for it, and
pivoting Journal on `meta.release` is how you tell which deploy started producing an error.
