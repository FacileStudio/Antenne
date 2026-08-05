# Nook — Configuration

Nook is configured in two places that behave very differently: a handful of environment variables
for the process itself, and a JSON settings file for everything else.

## The split

**Environment variables cover core server settings and dashboard auth only.** Providers, delivery
targets, credentials and routing rules live in the settings file and are edited exclusively
through the dashboard at `/dashboard`. There is no environment variable for a Matrix room, an SMTP
password or a webhook path, and adding one is not the intended extension point.

Nook does not use `tronc`, so there is no `troncenv.Core` block and no fail-fast on a missing
variable. Every variable below is optional and the process boots without any of them.

## Environment variables

### Core

| Variable | Required | Default | What it does |
|---|---|---|---|
| `PORT` | no | `9090` | Seeds the default listen port. Serves API, dashboard, health check and webhooks |
| `WEBHOOK_PORT` | no | — | Legacy alias, read only when `PORT` is unset |
| `NOOK_SETTINGS_PATH` | no | `/data/nook.json` | Settings file path. The event log and spaces files are derived from it |
| `TIMEOUT` | no | `10` | Seeds the default provider timeout, in seconds. Ignored when not a positive number |

`PORT` and `TIMEOUT` only seed *defaults*. Once a settings file exists, its `providers.port` and
`timeout` win — see the trap below.

### Dashboard authentication

| Variable | Required | Default | What it does |
|---|---|---|---|
| `NOOK_ADMIN_PASSWORD` | no | — | When set, the dashboard and every `/api/*` route require a session. **When unset, both are open to anyone who can reach the port** |
| `NOOK_ADMIN_USERNAME` | no | `admin` | Login label, and part of the session token payload. Not checked at login |
| `NOOK_SESSION_SECRET` | no | random per boot | HMAC key for the session cookie. Leave it unset and every restart signs users out |

The session cookie is `nook_session`, `HttpOnly`, `SameSite=Lax`, and `Secure` when the request
looks like HTTPS. Login is rate limited per client IP.

### Pool

| Variable | Required | Default | What it does |
|---|---|---|---|
| `POOL_SECRET` | no | `pool` | Shared secret apps present to `POST /api/pool/register`. Overridden by `pool.secret` in the settings file when that is set |

The default is the literal string `pool`, and the server logs
`[pool] using default secret 'pool' — set POOL_SECRET in production` when it is in use. Any app
that knows the secret can register and read every channel it subscribes to, so treat it as a
credential.

## The port trap

The listen port is read from the **settings file**, not the environment:

```ts
const serverPort = store.getSettings().providers.port;
```

`PORT` seeds `providers.port` only when no settings file exists yet. Change the port in the
dashboard and `nook.json` wins from then on, with the environment variable silently ignored.

This matters because both the compose `expose` and the compose healthcheck interpolate
`${PORT:-9090}`. If the two drift apart, the container fails its healthcheck while the service is
perfectly healthy on a different port. Keep `PORT` and `providers.port` in sync, or leave the port
alone in the dashboard and manage it purely from the environment.

A port change is also the one setting that needs a process restart. Everything else is applied
live on `PUT /api/settings`.

## The settings file

Written to `NOOK_SETTINGS_PATH`, pretty-printed JSON, rewritten in full on every save. Three
sibling files are derived from the same path: `nook-events.json` (the event log, capped at 500
entries), `nook-spaces.json` (spaces and members), and `pool-events.db` (the pool's SQLite queue,
in the same directory).

Top-level shape:

```json
{
  "delivery": { "items": [] },
  "system": {
    "events": {
      "startup": true,
      "providerEnabled": false,
      "providerDisabled": false,
      "providerAdded": false,
      "providerRemoved": false,
      "deliveryFailure": false
    }
  },
  "timeout": 10,
  "providers": { "port": 9090, "items": [] },
  "pool": { "secret": "…", "logActivity": true }
}
```

Every write goes through `normalizeAppSettings`, which coerces types, fills gaps from the defaults
and rejects invalid input. Two rejections are worth knowing: a non-positive `timeout`, and two
enabled webhook providers sharing a path.

### `system.events`

Nook can alert about itself. Each flag emits a system event through the normal delivery pipeline,
under the reserved provider id `__nook_system__`.

| Flag | Default | Fires when |
|---|---|---|
| `startup` | `true` | The service finishes booting |
| `providerEnabled` | `false` | A provider is switched on |
| `providerDisabled` | `false` | A provider is switched off |
| `providerAdded` | `false` | A provider is created |
| `providerRemoved` | `false` | A provider is deleted |
| `deliveryFailure` | `false` | One or more targets fail for an event |

System events are routed like any other, which means a target only receives them if it lists
`__nook_system__` in its `providerIds`. Enabling the flag alone is not enough.

### `providers.items`

Each entry has `id`, `type`, `name`, `enabled` and `tags`. Type-specific fields:

| Type | Fields |
|---|---|
| `website` | URL and `interval` in seconds, default `60` |
| `imap` | Host, port, credentials and mailbox. The only type with a connection test |
| `rss` | Feed URL and poll interval |
| webhook types | `path` and, optionally, `secretHeader` plus `secretValue` |

Webhook types are `uptime-kuma`, `docuseal`, `dokploy`, `git`, `vikunja`, `ardoise`, `sablier`,
`perception`, `opus`, `vision`, `plume` and `nuage`. Default paths and secret headers:

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
| `nuage` | `/webhook/nuage` | `x-nuage-signature-256` |
| `vision`, `plume` | `/webhook/dokploy` | `x-dokploy-secret` |

`vision` and `plume` have no dedicated default and fall through to the `dokploy` values. Give them
an explicit path, or the second one you enable will collide and the save will be rejected.

A provider with no `secretValue` accepts unauthenticated requests. A provider with a `secretValue`
but no `secretHeader` rejects everything.

### `delivery.items`

Each target has `id`, `type`, `name`, `enabled`, `providerIds` and `providerTags`, plus fields
specific to its type — see [architecture.md](architecture.md) for the full field list per type.

`providerIds` and `providerTags` are the routing rule and both default to empty, which means **a
new target receives nothing until you attach at least one provider id or tag to it**. There is no
catch-all.

### `pool`

| Field | Default | What it does |
|---|---|---|
| `secret` | — | Overrides `POOL_SECRET` |
| `logActivity` | `true` | Set to `false` to stop mirroring pool traffic into the event log |

## Secrets on disk

`nook.json` holds every credential in plaintext: Matrix access tokens and passwords, SMTP
passwords, ntfy tokens, HTTP and Perception target secrets, webhook provider secrets and the pool
secret. It is a secret file living on the `/data` volume. Back it up as one, and do not commit a
copy.
