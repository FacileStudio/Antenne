# Antenne — Architecture

How an alert gets from a provider to a delivery target, how the Antenne Pool relays events between
suite apps, and where the state actually lives.

## Runtime topology

Antenne is one Go binary serving the dashboard API, the webhook receivers, the pool and the built
SvelteKit client from a single port, backed by one PostgreSQL database.

```
                    ┌──────────────────────────────────────────┐
  third parties ───▶│ /webhook/<provider>   (root, never /api)  │
  pool clients ────▶│ /pool/register, /pool/ws                  │
  browser ─────────▶│ /api/*  ·  /health  ·  everything else    │
                    └──────────────────┬───────────────────────┘
                                       │
                    ┌──────────────────▼───────────────────────┐
                    │ internal/app — the alert pipeline         │
                    └──────────────────┬───────────────────────┘
                                       │
       ┌───────────────┬───────────────┼───────────────┬────────────────┐
       ▼               ▼               ▼               ▼                ▼
   providers/      settings/       events/        delivery/          pool/
   (pollers)       (config)     (activity log)    (senders)        (WS bus)
                                       │
                                       ▼
                                  PostgreSQL
```

[`tronc`](https://github.com/FacileStudio/tronc) supplies the chassis: the chi router and its
middleware chain (request id, recovery, real IP, CORS, request logging), the
`{"error":{"code","message"}}` envelope, bounded JSON decoding, the health endpoints, the SPA
handler and the shared configuration. `tronc/migrate` applies the embedded goose migrations at
startup, so a distroless container with no shell can still migrate itself.

## The alert pipeline

Every event — a webhook, a website transition, a new email, a feed item, a system notice —
takes the same path through `internal/app`:

1. **Deduplicate.** An identical message from the same provider within 30 seconds is dropped.
   Two sources reporting one outage is one alert.
2. **Build the envelope.** `internal/envelope` assembles the schema-version-1 document and
   derives its id by hashing the provider, the event type, the occurrence time and the
   payload — so the same source event replayed twice carries the same id and a downstream
   consumer can deduplicate.
3. **Log it.** The entry lands in `events` with a flattened `search_text` column, and is fanned
   out to every open SSE subscriber.
4. **Route it.** `delivery.Summarize` resolves which targets opted in.
5. **Deliver it.** Every sendable target is attempted in parallel; each attempt is recorded
   with its status, error and duration.
6. **Retry.** A failed attempt is queued for three more tries at 5s, 30s and 5min. The queue is
   in-memory on purpose: a retry is only useful within minutes of the original attempt, and
   the event log already records anything that outlives the process.

### Routing is opt-in, with no fallthrough

A target receives an event only if it names the provider id in `providerIds` **or** shares one
of the provider's tags through `providerTags`. Both default to empty and there is no
catch-all, so a freshly created target receives nothing until one of the two is set.

Antenne's own events carry the reserved provider id `__antenne_system__`. Enabling a system event in
Settings is not enough — a target has to list that id explicitly.

The delivery log line reports a `filtered` count. `filtered` equal to the target count means
the routing rules match nothing.

## The settings model

Providers and delivery targets are polymorphic: fifteen provider types and six target types,
each with its own fields. They are stored as a row carrying the indexed columns every type
shares — id, type, name, enabled, position, tags — plus a `jsonb` `config` for the rest.

The wire format is the dashboard's contract and is unchanged from the pre-Go service, so a
config exported from the old dashboard imports here untouched. `Provider.MarshalJSON` and
`DeliveryTarget.MarshalJSON` emit only the fields belonging to the type, keeping the JSON as
narrow as the TypeScript union it replaced.

A save replaces both lists wholesale rather than diffing them. Ordering is part of the state,
and a partial write would leave the runtime routing to targets the operator just deleted.

### Secrets

Secrets never leave the server. `settings.Redact` replaces each one with `••••••••` on the way
out; `settings.MergeSecrets` rewrites that marker back to the stored value on the way in,
before normalization sees it. A dashboard that never held a secret therefore cannot erase it,
and an exported config carries none.

## Providers

- **Webhook providers** are passive. `modules/webhooks` holds one file per provider, each
  owning its payload shape, its message rendering and its envelope mapping. The intake reads
  the raw body once and keeps it raw, because signature verification has to hash exactly what
  the sender signed.
- **Pollers** are active. `modules/providers` drives websites, IMAP inboxes and RSS feeds. Each
  restarts its watchers on every settings save, bound to the process context — never to a
  request context, which would cancel them the moment the save responded.

A website needs two consecutive failures before it is called down: one timeout is noise. Any
status under 400 counts as healthy, because several monitored hosts redirect their canonical
path or locale and `res.ok` would misread that as downtime.

The first IMAP and RSS poll only records what is already there, so a restart re-seeds instead
of replaying the whole inbox into Matrix.

## The Antenne Pool

A WebSocket bus with a durable, offset-based queue in Postgres. A client registers over HTTP
with the shared secret, receives a token, and connects with it.

| Frame | Direction | Meaning |
|---|---|---|
| `welcome` | server | the boot epoch and the client's app id |
| `subscribe` | client | channels and the client's last known offset |
| `subscribed` | server | the confirmed cursor per channel, after replay |
| `event` | both | an emission, or a relayed one |
| `ack` | both | the assigned offset, or the client's cursor |
| `ping` / `pong` | both | liveness |
| `error` | server | a code and a message |

The **epoch** is a fresh id on every boot, sent in `welcome`. A client can tell a restart from
a reconnect and resync from its own cursor rather than trusting offsets across a gap.

**One live connection per app.** A second replaces the first with close code 1000 `replaced by
new connection`, so a reconnect arriving before the old socket's close is noticed does not
deliver everything twice.

**Retention is interest-based, not time-based.** Events stay until every active listener has
acked them, past a 48h floor kept for replay. A listener not seen for 14 days stops pinning
the log. A 30-day ceiling backstops everything, and crossing it while a listener is still
behind logs loudly rather than dropping silently.

An app never receives its own emissions back.

## Authentication

Antenne owns its users and sessions and has no OIDC. The implementation sits behind
`auth.Authenticator`; handlers read an `authcontext.Identity` and never see a cookie, a token
or a user row. Replacing the implementation with the suite's shared auth package changes no
handler.

Session tokens are opaque random values, stored only as a SHA-256 hash, so a database leak
does not hand over live sessions. `ANTENNE_ADMIN_PASSWORD` remains the source of truth for the
bootstrap account: changing it and restarting re-seeds the hash and revokes that user's
sessions, which is the recovery path.

## URL layout

The layout is not uniform, and each exception is load-bearing:

- `/webhook/*` and `/pool/*` stay at the root. They are registered in Dokploy, DocuSeal, the
  git forges and every pool client, and moving them under `/api` breaks all of them silently.
- `/api/*` carries its own 404 and 405. Without them the SPA catch-all answers an unknown API
  path with 200 and HTML, which a machine client reads as a successful write.
- `/health` and `/ready` are mounted at both the root and `/api`, so an external monitor
  reaching only one of the two cannot read the SPA shell as green.
- `/dashboard` and `/dashboard/*` 301 to the equivalent root path, for links made before the
  client moved to the root.
