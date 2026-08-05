# Nook — Architecture

How an alert gets from a provider to a delivery target, how the Nook Pool relays events between
suite apps, and where the state actually lives.

## Runtime topology

One process, one port. `Bun.serve` handles HTTP and WebSocket traffic through a single hand-written
request handler; there is no framework and no reverse proxy inside the container.

```
Internet ──▶ Traefik ──▶ Bun.serve (:PORT, default 9090) ──┬──▶ /health          JSON status
                                                            ├──▶ /api/pool/*     pool server
                                                            ├──▶ /api/*          dashboard API
                                                            ├──▶ /dashboard/*    static Nuxt bundle
                                                            └──▶ POST /*         webhook receivers
                                                                       │
                                              ┌────────────────────────┼───────────────────┐
                                              │                        │                   │
                                     /data/nook.json         /data/nook-events.json   /data/pool-events.db
                                     settings + secrets       event log (500 max)     SQLite, WAL

Other Facile apps ──WebSocket──▶ /api/pool/ws ◀──WebSocket── Other Facile apps
```

Every path is served from the same port. That port is not fixed — see
[deployment.md](deployment.md).

## Components

| File | Role |
|---|---|
| `apps/api/index.ts` | Wiring: loads settings, starts the managers, builds the server, owns `sendConfiguredAlert` |
| `src/server.ts` | Top-level router and error boundary |
| `src/server-api.ts` | Authenticated dashboard API |
| `src/server-api-spaces.ts` | Spaces and members CRUD |
| `src/server-auth.ts` | Session cookie, login rate limiting |
| `src/server-dashboard.ts` | Static asset serving for the prebuilt Nuxt bundle |
| `src/monitor.ts`, `src/imap.ts`, `src/rss.ts` | Pull providers, each with an `apply(settings)` manager |
| `src/webhooks/` | Push providers, one file per provider under `providers/` |
| `src/delivery*.ts` | Target selection, senders, HTTP delivery, signing |
| `src/event-envelope.ts` | The structured envelope every HTTP and Perception delivery carries |
| `src/event-log.ts` | In-memory ring of the last 500 events, persisted to disk |
| `src/retry.ts` | Retry queue for failed deliveries |
| `src/settings*.ts` | Load, normalize, persist and hot-apply the settings file |
| `src/pool/` | The Nook Pool: WebSocket server, connection registry, SQLite event queue |

## The alert pipeline

Every alert, whatever its origin, goes through `sendConfiguredAlert` in `apps/api/index.ts`.

1. **Guard.** If no delivery target exists at all, the call throws.
2. **Deduplicate.** A SHA-256 of `providerId` plus message is kept for 30 seconds; a repeat inside
   that window is dropped.
3. **Route.** `getDeliveryRouteSummary` selects targets. A target accepts an alert when it lists
   the provider id in `providerIds`, or when it shares at least one tag with the provider's
   `providerTags`. **There is no implicit fallthrough** — a target that lists neither receives
   nothing. This is the single most common reason a freshly configured target stays silent.
4. **Build the envelope.** `buildNookEventEnvelope` produces a `NookEventEnvelope` and derives a
   stable id by hashing provider, event type, timestamp and a canonically serialized payload, so
   the same event produced twice keeps the same id.
5. **Log.** The event is pushed to the event log before delivery, then updated with the outcome.
6. **Deliver.** Selected targets that are fully configured are sent in parallel. Targets that are
   selected but incomplete are counted and skipped with a warning.
7. **Retry.** Each failure is queued in `src/retry.ts`, which retries after 5 s, 30 s and 300 s,
   three attempts maximum.
8. **Escalate.** When `system.events.deliveryFailure` is on, a failure emits its own system alert,
   guarded against recursion by a module-level flag.

Alerts are HTML strings. Plain text is derived from them for targets that need it.

## Providers

Two families.

**Pull providers** hold a manager with an `apply(settings)` method, re-invoked on every settings
change so the dashboard can reconfigure them live.

| Type | What it does |
|---|---|
| `website` | HTTP check on an interval, default 60 s, alerting on up and down transitions |
| `imap` | Polls a mailbox for newly received messages. The only provider type with a connection test |
| `rss` | Polls a feed for new entries |

**Push providers** are webhook receivers. Each is a `WebhookProviderDefinition` in
`src/webhooks/providers/`, registered by type in `src/webhooks/index.ts`. Twelve types exist:
`uptime-kuma`, `docuseal`, `dokploy`, `git`, `vikunja`, `ardoise`, `sablier`, `perception`, `opus`,
`vision`, `plume`, `nuage`.

Each configured webhook provider owns a path. Paths must be unique among enabled providers —
`normalizeProvidersConfig` throws on a collision, which rejects the settings save. Defaults come
from `getWebhookDefaultPath`; `vision` and `plume` have no dedicated default and fall through to
`/webhook/dokploy`, so give them an explicit path.

Authentication is per provider. A definition may supply its own `authenticate` (signature
verification); otherwise Nook compares `secretHeader` against `secretValue` with a constant-time
comparison. A provider with no `secretValue` accepts anything.

## Event envelope

`NookEventEnvelope` is Nook's own schema, at `schemaVersion: 1`. The TypeScript type carries no
version suffix on purpose.

```
{
  schemaVersion, id, occurredAt, emittedAt, source,
  provider: { id, type, name, tags },
  event:    { type, title, messageHtml, messageText, url },
  actor:    { id, name, email } | null,
  subject:  { type, id, name, url },
  payload:  { normalized, raw },
  privacy:  { containsPii, rawPayloadAllowed }
}
```

`source` is one of `webhook`, `website`, `imap`, `rss`, `test`, `system`. `subject.type` is drawn
from a closed list (`task`, `contract`, `client`, `deployment`, `repository`, `email`, `service`,
`schedule`, `system`, `unknown`). The raw payload is stripped unless `rawPayloadAllowed` is true;
`redactEnvelopeRawPayload` strips it after the fact for storage.

**This is not the suite `enveloppe` contract.** `@facile/events` keys cross-app identity on
`actor_email`, sourced from a Facile SSO identity. Nook's `actor.email` is whatever a third-party
webhook payload happened to contain, and is null for every website, RSS and system event. Nook
anchors the bus without being a participant in the envelope contract that runs over it — it relays
app payloads opaquely and never authors one.

HTTP and Perception deliveries carry the envelope as the request body plus four headers:

- `x-nook-event-id`
- `x-nook-timestamp`
- `x-nook-schema-version`
- `x-nook-signature-256`

The signature is `sha256=` followed by HMAC-SHA256 over `<timestamp>.<event_id>.<raw_body>` using
the target's secret. Deliveries without a configured secret are sent unsigned, with the other three
headers still present.

## Delivery targets

Six types, all defined in `src/types.ts` and constructed by `createDefaultDeliveryTarget`.

| Type | Carries | Key fields |
|---|---|---|
| `matrix` | HTML message | `homeserver`, `accessToken`, `refreshToken`, `userId`, `password`, `deviceId`, `roomIds` |
| `discord` | HTML message | `webhookUrl` |
| `webhook` | Signed envelope | `url`, `secretValue`, `includeRawPayload` |
| `smtp` | HTML message | `host`, `port`, `secure`, `username`, `password`, `fromAddress`, `toAddresses`, `subjectPrefix` |
| `ntfy` | HTML message | `serverUrl`, `topic`, `accessToken`, `priority`, `tags`, `clickUrl` |
| `perception` | Signed envelope | `url`, `workspace`, `secretValue`, `includeRawPayload` |

Every target additionally has `id`, `name`, `enabled`, `providerIds` and `providerTags`.

## The Nook Pool

The pool is a separate subsystem from alert delivery. It is a durable, offset-based event bus that
suite apps use to sync with each other — Opus, Sablier and Mycelium are the current clients, through
the `@facile/pool` library.

```
App ──POST /api/pool/register {app, instance_id, secret, events}──▶ Nook
    ◀──────────────── {token, app_id} ────────────────────────────
App ──WS /api/pool/ws?token=… ──▶ welcome {epoch, app_id}
    ──▶ subscribe {channels: {name: {last_offset}}}
    ◀── subscribed {channels: {name: {offset}}}
    ◀── event {id, channel, offset, payload, timestamp, sender}
    ──▶ ack {channel, offset}
```

- **Registration** returns a token. The identity is `app` or `app:instance_id` when an instance is
  given, which is how Opus keeps one pool connection per workspace. Declared `events.emit` and
  `events.listen` default to `["*"]`.
- **Authentication** is a single shared `POOL_SECRET`, compared in constant time. The default value
  is the literal string `pool`, and the server logs a warning when it is in use.
- **Persistence** is SQLite at `<data dir>/pool-events.db` with WAL journaling. Each channel has a
  monotonic offset; a reconnecting client resumes from its last acknowledged offset.
- **Epoch** is a UUID regenerated on every boot and sent in the `welcome` message, so clients can
  detect a queue reset and resynchronize rather than trusting stale offsets.
- **Retention is interest-based**, not time-based. Events acknowledged by every active listener of
  a channel become eligible for deletion. A listener inactive for more than 14 days stops pinning
  retention and is logged as such. Cleanup runs every 24 hours.
- **One connection per app id.** A second connection replaces the first, which is closed with
  `1000 replaced by new connection`.
- Pool activity is mirrored into the event log as `providerType: "pool"` unless
  `pool.logActivity` is set to `false` in the settings.

## Settings and state

`SettingsStore` holds the whole configuration in memory, normalizes anything written to it, and
persists to `NOOK_SETTINGS_PATH`. A `PUT /api/settings` re-applies every manager immediately;
nothing needs a restart except a port change.

Three files sit side by side, all derived from the settings path:

| File | Contents |
|---|---|
| `nook.json` | Providers, delivery targets, system event flags, timeout, port, pool settings |
| `nook-events.json` | The event log, capped at 500 entries |
| `nook-spaces.json` | Spaces and their members |
| `pool-events.db` | The pool's SQLite queue, in the same directory |

The settings file contains every credential in plaintext — Matrix access tokens, SMTP passwords,
webhook secrets, the pool secret. Treat the volume accordingly.

Spaces are a grouping layer with `owner`, `admin` and `member` roles keyed by user id. They are
stored in their own file and are not wired to any external identity provider.

## Authentication and safety

- The dashboard and the whole `/api/*` surface are gated by a session cookie, but **only when
  `NOOK_ADMIN_PASSWORD` is set**. Unset, everything is open to anyone who can reach the port, and
  the process logs a security warning at startup saying so.
- `NOOK_SESSION_SECRET` signs the session cookie. Unset, it is random per boot, so every restart
  invalidates sessions.
- Login attempts are rate limited per client IP, derived from `x-forwarded-for`, then
  `cf-connecting-ip`, then `x-real-ip`. Startup logs a second warning about this: the reverse proxy
  must overwrite `x-forwarded-for` or the limiter is trivially bypassed.
- `/api/pool/register` and `/api/pool/ws` are deliberately outside the session gate — they
  authenticate with `POOL_SECRET` and a token. `/api/pool/stats` is inside it.
- An uncaught exception exits the process so the supervisor restarts a known-good instance;
  unhandled rejections are logged only.
