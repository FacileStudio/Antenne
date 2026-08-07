# Antenne — Configuration

Antenne is configured in two places that behave very differently: a handful of environment
variables for the process itself, and a settings document in Postgres for everything else.

## Environment variables

Read once at startup. Every error `env.Load` returns is a reason not to boot.

| Variable | What it does | Default |
|---|---|---|
| `DATABASE_URL` | Postgres connection string | **required** — the API exits 1 without it |
| `APP_ENV` | `development` / `staging` / `production`. Never gates security behaviour on its own | `development` |
| `PORT` | HTTP port for the API, dashboard, health check, webhooks and pool | `9090` |
| `LOG_LEVEL` | `debug` / `info` / `warn` / `error` | `info` |
| `CORS_ALLOWED_ORIGINS` | Comma-separated origins. `ALLOWED_ORIGINS`, `DOMAINS`, `DOMAIN`, `CORS_ORIGINS`, `TRUSTED_ORIGINS` and `CLIENT_ORIGIN` remain accepted fallbacks | unset denies every cross-origin caller |
| `ANTENNE_ADMIN_USERNAME` | Login label, and the local part of the bootstrap account's email | `admin` |
| `ANTENNE_ADMIN_PASSWORD` | Enables login. Unset leaves the dashboard and its API open to anyone who can reach the server | — |
| `ANTENNE_SESSION_SECRET` | Required in production so sessions survive a restart | — |
| `ANTENNE_SECRET` | Shared secret pool clients register with. A dashboard value wins over it | `pool` |
| `TIMEOUT` | Website check timeout, in seconds | `10` |
| `JOURNAL_URL` / `JOURNAL_TOKEN` | Ship logs to Journal. Both must be set for anything to ship | — |

`APP_ENV=production` refuses to start without `ANTENNE_ADMIN_PASSWORD` and `ANTENNE_SESSION_SECRET`.
That is the one place `APP_ENV` changes behaviour, and it fails closed.

### PORT is not tronc's default

`troncenv.Core` defaults `PORT` to `8080`. Antenne deliberately keeps `9090`: the webhook URLs
registered in Dokploy, DocuSeal and the git forges all point at it, every pool client's base
URL points at it, and the dashboard's dev proxy targets it. Compose pins it explicitly, and so
does `.env.example`.

### Schemeless CORS origins are dropped

`troncenv.CORSOrigins` reads `DOMAIN` among its fallbacks, and deployments set that to a bare
hostname for Traefik rather than to an origin. Antenne filters out anything without an `http://`
or `https://` prefix, because a schemeless value never matches an `Origin` header and only
misleads anything that reads the list back as a URL.

### Development configuration comes from Casier

`.casier.toml` pins this repo to the `antenne` project and its `dev` environment, so `mise run
dev` wraps the API in `casier run` and the process starts with everything above already in its
environment. Nothing has to exist on disk; a local `.env` still works and simply wins nothing,
because the API reads the process environment either way.

`casier run` is network-first with a last-known-good cache, so a Casier outage falls back to
the previously fetched values and says so on stderr — it never starts the API with an empty
environment. `mise run dev-offline` forces the cached path.

```sh
casier secrets set -p antenne -e dev DATABASE_URL postgres://antenne:antenne@localhost:5432/antenne
casier sync push -p antenne -e dev -f apps/api/.env.example
mise run secrets    # list what Casier holds
```

### Production: Casier is the source, Dokploy is the injector

The `prod` environment holds every production value, and Dokploy is driven from it:

```sh
DOKPLOY_URL=https://gare.facile.studio/api DOKPLOY_API_KEY=… \
  casier push dokploy -p antenne -e prod MmZp9bJ4XggnCCArKOp3z
```

The running container still reads its configuration from the Dokploy compose env, **never from
Casier at boot**. That split is deliberate: an app that fetched its own configuration from
Casier at startup could not start while Casier was down, and Casier needs a configured
environment of its own to start at all. So Casier owns the values, versions them and tracks
their rotation age; Dokploy remains the bootstrap tier that hands them to the process.

Rotating a production secret is therefore: `casier secrets set -p antenne -e prod KEY value`,
then `casier push dokploy`, then redeploy.

## The settings document

Everything else — providers, delivery targets, credentials, routing, system events, the pool
secret — lives in Postgres and is edited through the dashboard. `GET /api/settings` returns it;
`PUT /api/settings` replaces it whole.

```jsonc
{
  "timeout": 10,
  "providers": {
    "port": 9090,
    "items": [
      {
        "id": "…", "type": "website", "name": "Landing",
        "enabled": true, "tags": ["public"],
        "url": "https://facile.studio", "interval": 60
      },
      {
        "id": "…", "type": "dokploy", "name": "Dokploy",
        "enabled": true, "tags": ["infra"],
        "path": "/webhook/dokploy",
        "secretHeader": "x-dokploy-secret", "secretValue": "••••••••"
      }
    ]
  },
  "delivery": {
    "items": [
      {
        "id": "…", "type": "matrix", "name": "Ops room", "enabled": true,
        "providerIds": ["__antenne_system__"], "providerTags": ["infra"],
        "homeserver": "https://matrix.org", "accessToken": "••••••••",
        "roomIds": ["!ops:matrix.example.com"]
      }
    ]
  },
  "system": { "events": { "startup": true, "deliveryFailure": false } },
  "pool": { "secret": "••••••••", "logActivity": true }
}
```

Only the fields belonging to a type appear on that type.

### Secrets read as `••••••••`

Every stored secret is redacted on the way out. Sending the marker back keeps the stored value;
sending anything else replaces it; sending an empty string clears it. So an exported config
carries no credentials, and a dashboard that never saw one cannot erase it.

### Provider defaults

| Type | Default path | Default secret header |
|---|---|---|
| `uptime-kuma` | `/webhook/uptime-kuma` | `x-webhook-secret` |
| `docuseal` | `/webhook/docuseal` | `x-docuseal-secret` |
| `dokploy` | `/webhook/dokploy` | `x-dokploy-secret` |
| `git` | `/webhook/git` | `x-hub-signature-256` |
| `vikunja` | `/webhook/vikunja` | `x-vikunja-signature` |
| `ardoise` | `/webhook/ardoise` | `x-ardoise-signature-256` |
| `sablier` | `/webhook/sablier` | `x-sablier-signature` |
| `perception` | `/webhook/perception` | `x-perception-secret` |
| `opus` | `/webhook/opus` | `x-opus-signature` |
| `vision` | `/webhook/vision` | `x-vision-signature-256` |
| `plume` | `/webhook/plume` | `x-plume-signature-256` |
| `nuage` | `/webhook/nuage` | `x-nuage-signature-256` |

`ardoise`, `vision`, `plume` and `nuage` verify an HMAC-SHA256 of the body prefixed with
`sha256=`. `opus` verifies the bare hex digest with no prefix. The rest compare the header
against the configured secret in constant time. An empty secret accepts unauthenticated calls
on that path.

### Rejected documents

A `PUT` is refused, with the current settings left untouched, when the timeout is not positive,
a webhook path does not start with `/`, two enabled webhook providers share a path, a delivery
target has an unknown type, or the listener port is not a valid TCP port.

### The listener port

`providers.port` is what the process binds on the next restart, and the only setting that needs
one. Everything else re-applies live on save. Changing it means changing the deployment's
`PORT`, the compose healthcheck and every registered webhook URL — which is why compose pins
`9090` literally instead of interpolating it.
