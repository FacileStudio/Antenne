# Antenne — API

Every HTTP route the server handles, and the WebSocket protocol the Antenne Pool speaks.

A generated reference is served by the running instance at **`/docs`**, with the OpenAPI 3.1
document at `/docs/openapi.json`. It is built from a route registry each module owns
(`Documentation` in `modules/*`, `*Documentation` in `internal/server`), and
`TestEveryRouteIsDocumented` fails the build when a route is added without an entry. This page
is the prose version: the reference is the machine-readable one.

Errors use the suite envelope:

```json
{ "error": { "code": "invalid_argument", "message": "timeout must be a positive number" } }
```

`invalid_argument` 400 · `unauthenticated` 401 · `permission_denied` 403 · `not_found` 404 ·
`already_exists` 409 · `rate_limited` 429 · `internal` 500.

## Health

| Route | Notes |
|---|---|
| `GET /health`, `GET /api/health` | process liveness |
| `GET /ready`, `GET /api/ready` | also checks the database |

Mounted at both prefixes so an external monitor cannot mistake the SPA shell for a green check.

## Authentication

Antenne runs on [`porte`](https://github.com/FacileStudio/porte), which owns every route below
except `GET /api/session`.

| Route | Body | Notes |
|---|---|---|
| `GET /api/auth/config` | — | public: whether SSO is available, mandatory, and whether a password form applies |
| `POST /api/auth/login` | `{"email":"…","password":"…"}` | sets the cookie **and** returns `{"user_id","token"}` — a client wanting the bearer token reads the body, not the `Set-Cookie` |
| `POST /api/auth/logout` | — | revokes the session and expires the cookie. Idempotent |
| `GET /api/auth/oidc` | — | starts the Authentik flow. Only when `OIDC_ISSUER` is set |
| `GET /api/auth/oidc/callback` | — | a browser navigation: both outcomes are a 302, read `Location` |
| `POST /api/auth/oidc/exchange` | `{"code":"…"}` | a CLI trading its one-time login code for a bearer token |
| `GET /api/session` | — | authenticated: the current caller |
| `GET /api/me/invitations` | — | authenticated: `{"count","href"}`, the caller's pending Registre space invitations for the user menu. Never fails: `{"count":0}` when Registre cannot answer. Mounted only when `OIDC_ISSUER` and `OIDC_CLIENT_SECRET` are set |

**Two transports, one session model.** The dashboard rides an `HttpOnly` cookie, named
`__Host-session` over https; a CLI or a script sends `Authorization: Bearer`. Same table, same
hash, same revocation.

**A cookie-authenticated write must carry `X-Facile-CSRF`.** Any non-empty value counts — the
header's presence is the whole signal, because a browser will not attach it to a simple
cross-site request. Omit it and every read still succeeds while every write answers 403, which
reads as "the app works, saving is broken". Bearer callers are exempt by construction, so a
test that authenticates with a token does not exercise this rule at all.

`GET /api/events/stream` is consumed by an `EventSource`, which can set no headers. It is
therefore cookie-only, permanently — and being a GET, the CSRF rule does not reach it.

Refused logins are throttled per client address: five within ten minutes blocks that address
for ten minutes. With `OIDC_ISSUER` set the password route is not registered at all, rather than
rejected, so there is no endpoint left to probe.

## Settings

| Route | Notes |
|---|---|
| `GET /api/settings` | the settings, the runtime status, and whether Matrix is usable |
| `PUT /api/settings` | replaces the whole document |

Both return:

```json
{ "settings": { … }, "runtime": { … }, "matrixConfigured": true }
```

Secrets read as `••••••••`. Sending the marker back keeps the stored value; anything else
replaces it; an empty string clears it. A rejected `PUT` leaves the current settings untouched
— see [configuration.md](configuration.md) for what gets rejected.

## Actions

| Route | Body | Notes |
|---|---|---|
| `POST /api/test-alert` | — | a test alert through the whole pipeline, including the routing rules |
| `POST /api/test-delivery-target` | `{"targetId":"…"}` | straight to one target, bypassing routing |
| `POST /api/test-provider-connection` | `{"providerId":"…"}` | IMAP only |

`test-alert` respects routing, so it is silent unless a target lists `__antenne_system__`.
`test-delivery-target` is the one that answers "does this target work at all".

## Events

| Route | Notes |
|---|---|
| `GET /api/events` | `limit`, `offset`, `q`, `source` |
| `GET /api/events/provider/{providerId}` | scoped to one provider |
| `GET /api/events/target/{targetId}` | scoped to one delivery target |
| `GET /api/events/stream` | server-sent events, one `data:` frame per new entry |
| `POST /api/events/{eventId}/replay` | `{"targetId":"…"}` — re-send a logged envelope |

```json
{ "events": [ … ], "total": 128, "stats": { "totalEvents": 128, "lastEventAt": "…",
  "deliverySuccessCount": 121, "deliveryErrorCount": 7 } }
```

`stats` describes the whole filtered set, not the returned page, so the counters do not change
as the operator scrolls. `q` matches every search term against a flattened per-event text
column covering the event, the provider, the envelope's message and normalized string fields,
and each delivery attempt's target and error.

The log keeps the most recent 5000 entries; older ones are pruned hourly.

## Spaces

| Route | Notes |
|---|---|
| `GET /api/spaces` | list, with `manageUrl`, Registre's spaces page |
| `GET /api/spaces/{id}` | one space with its members |
| `GET /api/spaces/{id}/members` | list |
| `POST /api/registre/sync` | Registre's nudge: anonymous, answers 202 and pulls the snapshot |

Registre owns spaces. Antenne applies Registre's snapshot at boot, every five minutes and
two seconds after a nudge, so creating, renaming and staffing a space happen at `manageUrl`.
Roles are `owner`, `admin`, `member`.

`{userId}` is percent-decoded server-side, so a client sending `encodeURIComponent(email)`
matches a stored address. chi routes on the raw path and would otherwise never match.

## Webhooks

```
POST /webhook/<provider>
```

At the root, never under `/api` — these paths are registered in Dokploy, DocuSeal, the git
forges and Vikunja. The path and the secret header are per-provider settings; the defaults are
in [configuration.md](configuration.md).

The body is read once under a 4 MB cap and kept raw, because signature verification hashes
exactly what the sender signed. Authentication runs before the JSON is parsed.

| Answer | Meaning |
|---|---|
| `200 {"ok":true}` | accepted; delivered if the provider produced a message |
| `401` | the secret or signature did not verify |
| `400` | the body is not JSON |
| `404` | no enabled provider is configured on that path |

A recognised event that is not worth an alert — a bare heartbeat — is accepted, logged, and not
delivered.

## Outbound event envelope

HTTP and Perception targets receive the envelope, signed:

| Header | |
|---|---|
| `x-antenne-event-id` | the envelope id |
| `x-antenne-timestamp` | Unix seconds |
| `x-antenne-schema-version` | `1` |
| `x-antenne-signature-256` | `sha256=` + HMAC-SHA256 of `timestamp.eventId.body` |

The signature binds all three, so a receiver that also bounds the timestamp's age can reject a
replay. A target with no secret configured gets the first three headers and no signature.

`payload.raw` is `null` unless the provider marked the payload safe to forward **and** the
target has `includeRawPayload`. When it is stripped, `privacy.rawPayloadAllowed` goes to
`false` with it, so the receiver can tell the difference between "no raw payload" and "raw
payload withheld".

## The pool

| Route | Notes |
|---|---|
| `POST /antenne/register`, `POST /api/antenne/register` | exchange the shared secret for a token |
| `GET /antenne/ws`, `GET /api/antenne/ws` | the WebSocket, `?token=…` |
| `GET /api/antenne/stats` | authenticated: epoch, live connections, connected apps |

```jsonc
// POST /antenne/register
{ "app": "sablier", "instance_id": "optional",
  "secret": "…", "events": { "emit": ["*"], "listen": ["*"] } }
// → { "token": "…", "app_id": "…" }
```

### Tokens

- **Every registration issues a new token.** Registering under a name that already exists
  replaces that app's token, revokes the old one and closes the connection that used it. The
  app id stays the same, so subscriptions and retention carry over. Knowing the secret never
  returns a token someone else is using.
- **A token is bound to the secret it was issued under.** `pool_apps.secret_fingerprint` holds
  an HMAC-SHA256 keyed by the effective secret, never the secret itself. The socket refuses a
  token whose fingerprint is not the current secret's, with the same `403 invalid token`.
- **Changing the effective secret revokes every token.** The effective secret is the dashboard
  value when set, else `ANTENNE_SECRET`, else `pool`. A dashboard save that changes it closes
  every live connection at once (close reason `secret rotated`); a restart with a new env value
  refuses every old token. A save that leaves the secret as it was disconnects nobody.
- **Identity is `app`, or `app:instance_id`.** Two processes that share an identity replace
  each other's token on every reconnect. Give each running instance its own `instance_id`.

Clients do not cache tokens. Both antenne-client implementations (Go from the first release,
TypeScript) register before every connection attempt, so a client whose token was revoked
registers again on its next reconnect and needs no change. A client holding a rotated-away
secret gets `403 invalid secret` and keeps retrying under its reconnect policy.

### Frames

```jsonc
// server, on connect
{ "type": "welcome", "epoch": "…", "app_id": "…" }

// client
{ "type": "subscribe", "channels": { "tasks": { "last_offset": 42 } } }
// server, after replaying anything newer
{ "type": "subscribed", "channels": { "tasks": { "offset": 57 } } }

// either direction
{ "type": "event", "id": "…", "channel": "tasks", "offset": 58,
  "payload": { … }, "timestamp": 1786022937782, "sender": "sablier" }

// server to the emitter, or client reporting its cursor
{ "type": "ack", "channel": "tasks", "offset": 58 }

{ "type": "ping" }  →  { "type": "pong" }
{ "type": "error", "code": "UNKNOWN_TYPE", "message": "…" }
```

On subscribe, the server resumes from whichever cursor is further along: the client's
`last_offset` or the server-side ack cursor, which is what lets a client that lost its offsets
recover. An app never receives its own emissions back.

The `epoch` changes on every boot. A client seeing a new one knows it is looking at a restarted
server and should resync rather than trust its offsets across the gap.

One live connection per app: a second closes the first with code 1000 `replaced by new
connection`.

Acking is what pins retention. Events are kept until every active listener has acked them, past
a 48h floor; a listener not seen for 14 days stops pinning; a 30-day ceiling backstops
everything and logs loudly when a listener is still behind it.
