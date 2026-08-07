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

| Route | Body | Notes |
|---|---|---|
| `POST /api/login` | `{"password":"…"}` | sets the `antenne_session` cookie |
| `POST /api/logout` | — | revokes the session and expires the cookie |
| `GET /api/session/public` | — | public: whether a password is required at all |
| `GET /api/session` | — | authenticated: the current caller |

The session cookie is `HttpOnly`, `SameSite=Lax`, and `Secure` whenever the request arrived
over TLS — including through Traefik, which terminates it and forwards plain HTTP.

Failed logins are rate-limited per client IP: five within ten minutes blocks that IP for ten
minutes. The bucket keys on `X-Forwarded-For`, which is only safe because the reverse proxy
overwrites it.

With no `ANTENNE_ADMIN_PASSWORD` configured, every caller is served as the admin and `/api/login`
answers `{"ok":true}` without doing anything.

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
| `GET /api/spaces` · `POST /api/spaces` | list, create |
| `GET`/`PUT`/`DELETE /api/spaces/{id}` | one space; `GET` includes its members |
| `GET`/`POST /api/spaces/{id}/members` | list, add |
| `PUT`/`DELETE /api/spaces/{id}/members/{userId}` | update the role, remove |

Roles are `owner`, `admin`, `member`. Adding an existing member returns that membership
unchanged rather than erroring.

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
| `POST /pool/register`, `POST /api/antenne/register` | exchange the shared secret for a token |
| `GET /pool/ws`, `GET /api/antenne/ws` | the WebSocket, `?token=…` |
| `GET /api/antenne/stats` | authenticated: epoch, live connections, connected apps |

```jsonc
// POST /pool/register
{ "app": "sablier", "instance_id": "optional",
  "secret": "…", "events": { "emit": ["*"], "listen": ["*"] } }
// → { "token": "…", "app_id": "…" }
```

Re-registering an app keeps its existing token: a restarting client must not invalidate the
token another instance is using.

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
