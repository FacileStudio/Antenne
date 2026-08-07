# Antenne Roadmap

## Core Upgrades

- [x] Standardize `status`, `dueDate`, `completedDate` fields across all webhook providers
- [x] Enrich IMAP normalized payload with `from`, `to`, `subject`, `snippet`, `messageId`
- [x] Tag-based delivery routing — route providers to targets by tag, not all-or-nothing
- [x] Retry + exponential backoff on delivery failures (3 attempts: 5s / 30s / 5min)
- [x] Event deduplication window — collapse repeated identical alerts
- [x] Real-time activity feed via SSE
- [x] Full-text search and filtering in activity log
- [x] Webhook replay — re-deliver any logged event to any target from the dashboard
- [x] Replace the flat JSON file with a real database — landed as Postgres, not SQLite,
      to match the rest of the Go suite (goose migrations through `tronc/migrate`)
- [ ] Adopt the shared auth package once it ships — `auth.Authenticator` is the seam

## New Webhook Providers

- [ ] Grafana — alert firing/resolved with severity and metric labels
- [ ] Prometheus Alertmanager — label-based routing, more granular than Grafana
- [ ] Coolify — deployment events (popular Dokploy alternative)
- [ ] Sentry — error spikes, issue lifecycle, release tracking
- [ ] Gitea / Forgejo — extend existing `git.ts` pattern
- [ ] Woodpecker CI / Drone — CI pipeline events for non-GitHub shops

## New Pull Providers

- [x] RSS / Atom feeds — generic content monitoring, one config, infinite sources
- [ ] SSL certificate expiry — fire N days before expiry
- [ ] DNS record watcher — fire on any record change
- [ ] Cron heartbeat monitor — expect a ping on a schedule, alert if missing

## New Delivery Targets

- [ ] Slack
- [ ] Telegram
- [ ] Gotify — self-hosted ntfy alternative
- [ ] Pushover

## Perception Integration

- [x] Standardize `status`, `dueDate`, `completedDate` in all provider `normalizedPayload`s
- [x] Perception ingest generically extracts and populates `events.status`, `events.due_at`, `events.completed_at`
- [x] perception-js `IngestOptions` extended with `status`, `dueDate`, `completedDate`
