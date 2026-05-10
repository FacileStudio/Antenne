# Nook × Perception Integration

## What Exists Today

Nook sends a signed `NookEventEnvelope` (schema v1) to Perception via HTTP POST.
Perception stores it, infers field schemas, resolves entities, queues embeddings.
All 7 webhook providers + IMAP + website monitoring flow through the pipe.

The pipe works. The data arriving is thin.

---

## Current Gaps

### 1. `events.status` always NULL

Perception has a `status` column in the `events` table — never populated.
Every provider has implicit status buried in the event type string or a provider-specific payload field.

Uptime Kuma already sends `status: "up" | "down"` in its normalizedPayload. Nobody else does.

### 2. `events.due_at` and `events.completed_at` always NULL

Both columns exist in the schema. Nothing writes to them.
Vikunja sends `dueDate`. DocuSeal sends `completedAt`. Sablier has a stop timestamp.
These sit in `dynamic_facts` and never reach the top-level columns used for filtering and chat queries.

### 3. IMAP events are structurally opaque

Email arrives with generic `messageText`. No `from` / `to` / `subject` structure.
Perception can't resolve sender entities, track threads, or categorize by subject.

---

## Implemented Fixes

### Nook Side — normalized payload standardization

All webhook providers now include these fields in `normalizedPayload` where applicable:

| Field | Type | Meaning |
|---|---|---|
| `status` | `string` | Current state in provider-native terms (e.g. `"up"`, `"down"`, `"completed"`, `"merged"`) |
| `dueDate` | `string \| null` | ISO 8601 deadline, if the event has one |
| `completedDate` | `string \| null` | ISO 8601 completion timestamp, if the event represents completion |

Per-provider breakdown:

| Provider | status | dueDate | completedDate |
|---|---|---|---|
| uptime-kuma | `"up"` / `"down"` / `"unknown"` (was already there) | — | — |
| vikunja | derived from `taskDone` + event name | ✅ from `task.due_date` | set when task marked done |
| dokploy | from deployment payload `status` field | — | — |
| git | derived from event type (`"pushed"`, `"merged"`, `"opened"`, `"closed"`) | — | — |
| docuseal | from `submissionStatus` or event type | — | `completedAt` or `declinedAt` |
| charles | derived from event type (create/update/delete) | — | — |
| sablier | `"started"` / `"stopped"` | — | `stoppedAt` when timer stops |

IMAP normalized payload now includes:

| Field | Source |
|---|---|
| `from` | Parsed `From` header |
| `to` | Parsed `To` header (first address) |
| `subject` | Email subject line |
| `snippet` | First ~200 chars of plain text body |
| `messageId` | `Message-ID` header (dedup / threading) |

### Perception Side — generic column uplift

`ingest.ts` now checks `normalizedPayload` for common field names and populates the top-level columns:

- `status` ← first of: `status`, `state`
- `due_at` ← first of: `dueDate`, `due_date`, `due_at`, `deadline`
- `completed_at` ← first of: `completedDate`, `completed_date`, `completed_at`, `completedAt`

Logic is generic — not Nook-specific. Any ingest source that sends these field names benefits automatically.

---

## New Provider Opportunities

### Webhook providers (easy, same pattern as existing)

- **Grafana** — alert firing/resolved, with severity and metric labels
- **Prometheus Alertmanager** — same domain, more granular label routing
- **Coolify** — Dokploy competitor, popular in self-hosted space
- **Sentry** — error spikes, issue lifecycle, release events
- **Gitea / Forgejo** — already have `git.ts` pattern, minor extension
- **Woodpecker CI / Drone** — CI pipeline events for non-GitHub shops

### Pull providers

- **RSS / Atom feeds** — generic content monitoring, one config, infinite sources
- **SSL certificate expiry** — passive check, fire N days before expiry
- **DNS record watcher** — fire on any record change
- **Cron heartbeat monitor** — expect a ping by schedule, alert if missing

### Delivery targets

- **Slack** — most-wanted missing target
- **Telegram** — popular for self-hosted communities
- **Gotify** — self-hosted ntfy alternative
- **Pushover** — solid mobile push without requiring ntfy infrastructure

---

## Other Upgrade Opportunities

| Item | Value | Effort |
|---|---|---|
| Retry + exponential backoff on delivery failure | High — zero retries today | Medium |
| Event deduplication window | High — noisy providers spam all targets | Low |
| Tag-based delivery routing | High — all-or-nothing per provider today | Medium |
| Webhook replay from dashboard | Medium — re-deliver any logged event | Low |
| Real-time activity feed via SSE | Medium — currently requires manual refresh | Medium |
| Full-text search in activity log | Medium — 500 events, no search | Low |
| SQLite for persistence | High at scale — flat JSON hurts | Medium |
