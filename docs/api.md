# Nook — API

Every HTTP route the Bun server handles, and the WebSocket protocol the Nook Pool speaks.

Routing is a chain of `if` statements in `src/server.ts`, evaluated top to bottom, so order
matters: pool routes are matched first, then the health check and OpenAPI documents, then the auth
endpoints, then the authenticated `/api/*` surface, then the dashboard, and finally an unmatched
`POST` falls through to the webhook receivers.

Nook also serves its own spec, but it documents the **pool only**:

- `GET /docs` — Scalar reference, the same path as every other Facile backend
- `GET /docs/openapi.json` — OpenAPI document for the pool endpoints
- `GET /api/docs` and `GET /api/openapi` — the previous paths, kept as aliases

Both are reachable without a session.

## Authentication

The dashboard API uses a session cookie, `nook_session`, obtained from `POST /api/login` or the
`/dashboard/login` form. It is `HttpOnly`, `SameSite=Lax`, `Secure` over HTTPS, and signed with
HMAC-SHA256 over `<username>:<issuedAt>:<password>`.

**When `NOOK_ADMIN_PASSWORD` is unset, `isAuthenticated` returns true unconditionally** and the
entire API is open. That is the documented default, not a bug, and the process warns about it at
startup.

The pool endpoints do not use the session. They authenticate with `POOL_SECRET` at registration and
with the returned token on the WebSocket. The exception is `GET /api/pool/stats`, which is behind
the session.

Errors are JSON with a `code` and a message; codes seen in the router are `unauthenticated`,
`invalid_argument`, `rate_limited`, `not_found` and `internal`.

## Public routes

| Method | Path | What it does |
|---|---|---|
| GET | `/health` | Returns `{"status":"ok"}` as soon as the server is listening |
| GET | `/docs/openapi.json` | OpenAPI document for the pool, with the server URL derived from the request. Also served at `/api/openapi` |
| GET | `/api/docs` | Scalar reference page |
| GET | `/favicon.ico` | Serves the dashboard favicon |
| GET | `/` | 302 to `/dashboard` |

## Authentication routes

| Method | Path | What it does |
|---|---|---|
| POST | `/api/login` | Body `{"password": "…"}`. Sets the session cookie. Returns `{"ok":true}` immediately when no password is configured |
| POST | `/api/logout` | Clears the session cookie |
| POST | `/dashboard/login` | Form-encoded equivalent, redirects to `/dashboard/providers` on success and back to `/dashboard?error=invalid` or `?error=rate-limit` on failure |

Failed attempts are rate limited per client IP. Once limited, `POST /api/login` returns
`rate_limited` and the form redirects with `?error=rate-limit`.

## Dashboard API

Everything below requires a session when one is configured.

### Session and settings

| Method | Path | What it does |
|---|---|---|
| GET | `/api/session` | `{authenticated, passwordRequired, username}` |
| GET | `/api/settings` | `{settings, runtime, matrixConfigured}`. Secrets are redacted from `settings` |
| PUT | `/api/settings` | Replaces the settings. Redacted secrets in the body are merged back from the stored values, so the dashboard can round-trip without ever seeing them. Returns the same shape as GET |

`runtime` is `{serverPort, settingsPath, usingPersistedSettings, dashboardBuilt, lastReloadedAt}`.

A `PUT` normalizes and validates before persisting, then re-applies every provider manager
immediately. Rejections come back as `invalid_argument` with the reason — most often a
non-positive timeout or two enabled webhook providers claiming the same path.

### Testing

| Method | Path | What it does |
|---|---|---|
| POST | `/api/test-alert` | Sends a test alert through the full pipeline, including routing filters. Fails when no delivery target exists |
| POST | `/api/test-delivery-target` | Body `{"targetId": "…"}`. Sends directly to one target, bypassing routing |
| POST | `/api/test-provider-connection` | Body `{"providerId": "…"}`. IMAP providers only; any other type returns an error |

`test-alert` respects `providerIds` and `providerTags`, so it can legitimately succeed while
delivering to nothing. `test-delivery-target` is the one that proves a target's credentials work.

### Events

| Method | Path | What it does |
|---|---|---|
| GET | `/api/events?limit=&offset=&q=&source=` | Paged event log with `{events, total, stats}`. Default limit 50 |
| GET | `/api/events/provider/:providerId?limit=&offset=` | Same, filtered to one provider, with that provider's stats |
| GET | `/api/events/target/:targetId?limit=&offset=` | Same, filtered to one delivery target |
| GET | `/api/events/stream` | Server-sent events, one `data:` frame per new event as it is pushed |
| POST | `/api/events/:eventId/replay` | Body `{"targetId": "…"}`. Re-sends a stored envelope to one target |

The log keeps the last 500 entries. `source` filters on `webhook`, `website`, `imap`, `rss`, `test`
or `system`. Replay requires the stored entry to have an envelope.

### Spaces

| Method | Path | What it does |
|---|---|---|
| GET | `/api/spaces` | List spaces |
| POST | `/api/spaces` | Create a space |
| GET | `/api/spaces/:id` | One space |
| PUT | `/api/spaces/:id` | Update name or description |
| DELETE | `/api/spaces/:id` | Delete a space |
| GET | `/api/spaces/:id/members` | List members |
| POST | `/api/spaces/:id/members` | Add a member |
| PUT | `/api/spaces/:id/members/:userId` | Change a member's role |
| DELETE | `/api/spaces/:id/members/:userId` | Remove a member |

Roles are `owner`, `admin` and `member`. Members are keyed by an opaque user id; spaces are not
wired to any identity provider.

## Webhook receivers

Any `POST` that matches no earlier route is handed to the webhook dispatcher, which looks for an
enabled webhook provider whose configured `path` matches. Paths are set per provider in the
dashboard; defaults are listed in [configuration.md](configuration.md).

| Provider type | Default path |
|---|---|
| `uptime-kuma` | `/webhook/uptime-kuma` |
| `docuseal` | `/webhook/docuseal` |
| `dokploy` | `/webhook/dokploy` |
| `git` | `/webhook/git` |
| `vikunja` | `/webhook/vikunja` |
| `ardoise` | `/webhook/ardoise` |
| `sablier` | `/webhook/sablier` |
| `perception` | `/webhook/perception` |
| `opus` | `/webhook/opus` |
| `nuage` | `/webhook/nuage` |
| `vision`, `plume` | fall through to `/webhook/dokploy` — set an explicit path |

Authentication per request is either the provider's own `authenticate` implementation (signature
verification, for the types that support it) or a constant-time comparison of the configured
`secretHeader` against `secretValue`. A provider with no `secretValue` accepts anything; a provider
with a secret but no header name rejects everything.

A successful receipt normalizes the payload into an `AlertContext`, which becomes a
`NookEventEnvelope` and goes through the delivery pipeline.

## Outbound deliveries

Nook is a client as well as a server. HTTP and Perception delivery targets receive a `POST` with
the envelope as the JSON body and these headers:

| Header | Value |
|---|---|
| `Content-Type` | `application/json` |
| `User-Agent` | `Nook/1.0 uptime-bot` |
| `x-nook-event-id` | The envelope id |
| `x-nook-timestamp` | Unix seconds |
| `x-nook-schema-version` | `1` |
| `x-nook-signature-256` | `sha256=<hmac>`, present only when the target has a secret |

The HMAC is SHA-256 over `<timestamp>.<event_id>.<raw_body>` keyed with the target's secret.
Verify it against the raw body before parsing.

Failed deliveries are retried after 5 s, 30 s and 300 s, three attempts maximum.

## Pool protocol

The Nook Pool is a WebSocket event bus with a durable, offset-based queue. Client apps use the
`@facile/pool` library rather than speaking this directly.

### Registration

```
POST /api/pool/register
{
  "app": "Opus",
  "instance_id": "<optional>",
  "secret": "<POOL_SECRET>",
  "events": { "emit": ["task.created"], "listen": ["task.created"] }
}
→ 200 {"token": "…", "app_id": "…"}
→ 400 {"error": "app is required"} | {"error": "invalid json"}
→ 403 {"error": "invalid secret"}
```

The identity is `app` when no `instance_id` is given, `app:instance_id` otherwise — that is how one
Opus deployment maintains a separate pool identity per workspace. `emit` and `listen` both default
to `["*"]`.

### Connecting

```
GET /api/pool/ws?token=<token>          Upgrade: websocket
→ 401 {"error": "token required"}
→ 403 {"error": "invalid token"}
```

Only one live connection per `app_id`. A second one replaces the first, which is closed with code
`1000` and reason `replaced by new connection`.

### Messages

| Direction | Type | Shape |
|---|---|---|
| server → client | `welcome` | `{type, epoch, app_id}` |
| client → server | `subscribe` | `{type, channels: {"<name>": {last_offset}}}` |
| server → client | `subscribed` | `{type, channels: {"<name>": {offset}}}` |
| server → client | `event` | `{type, id, channel, offset, payload, timestamp, sender}` |
| client → server | `ack` | `{type, channel, offset}` |
| server → client | `error` | `{type, code, message}` |

`epoch` is a UUID regenerated on every server boot. A client that sees a new epoch must treat its
stored offsets as invalid and resubscribe from scratch — this is what makes losing
`pool-events.db` survivable.

Subscribe with `last_offset` to resume; the server replays everything after it on that channel.
Acknowledge as you go: retention is interest-based, and unacknowledged events are kept.

### Stats

| Method | Path | Auth | What it does |
|---|---|---|---|
| GET | `/api/pool/stats` | session | Pool epoch, registered apps and live connections |

The payload Nook relays over the pool is opaque to it. Nook does not author suite event envelopes
and cannot supply `actor_email` — it moves other apps' payloads without inspecting them. See
[architecture.md](architecture.md).
