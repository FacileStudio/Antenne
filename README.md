# Nook

Nook is a Bun service that watches modular providers, forwards alerts to Matrix, Discord, HTTP webhooks, SMTP, ntfy, and Perception, and ships with a Nuxt dashboard for configuring everything.

The Docker image is runtime-only and serves a prebuilt dashboard from `dashboard/.output/public`, so container builds do not need to install npm packages.

## What It Does

- monitors websites on intervals
- polls IMAP inboxes for newly received emails
- receives Uptime Kuma, DocuSeal, and Dokploy webhooks
- delivers alerts to Matrix, Discord webhooks, HTTP webhooks, SMTP, ntfy, and Perception
- can send signed structured event envelopes for downstream memory/reasoning systems
- serves a dashboard at `/dashboard`

## Config Model

All provider and delivery target configuration is managed through the dashboard and persisted in `/data/nook.json`. Environment variables are only used for core server settings and dashboard auth.

## Environment Variables

### Core

| Variable | Required | Description |
|---|---|---|
| `PORT` | No | HTTP port for API, dashboard, health, and webhooks. Default: `9090`. |
| `NOOK_SETTINGS_PATH` | No | JSON settings file path. Default: `/data/nook.json` |
| `TIMEOUT` | No | Default timeout in seconds |

### Dashboard Auth

| Variable | Required | Description |
|---|---|---|
| `NOOK_ADMIN_PASSWORD` | No | If set, dashboard API requires login |
| `NOOK_ADMIN_USERNAME` | No | Login label only. Default: `admin` |

## Event Envelopes

HTTP webhook targets and the dedicated Perception target both send the structured event envelope.

Envelope deliveries use fixed signing headers:

- `x-nook-event-id`
- `x-nook-timestamp`
- `x-nook-schema-version`
- `x-nook-signature-256`

The signature is HMAC-SHA256 over:

```text
<timestamp>.<event_id>.<raw_body>
```

The event envelope uses `schemaVersion: 1` but the TypeScript type is named `NookEventEnvelope` so callers do not carry version suffix noise around the codebase.

## Routes

- `GET /health`
- `GET /dashboard`
- `GET /api/settings`
- `PUT /api/settings`
- `POST /api/test-alert`
- `POST /api/login`
- `POST /api/logout`
- `POST <provider webhook path>`

## Local Development

```sh
bun install
bun run client:build
bun run dev
```

For dashboard-only work:

```sh
bun run client:dev
```

Before committing dashboard changes meant for deployment, rebuild the static bundle:

```sh
bun run client:build
```

## Docker

```sh
docker build -t nook .
docker run -d \
  -p 9090:9090 \
  -e PORT=9090 \
  -e NOOK_ADMIN_PASSWORD=change-me \
  -v ./data:/data \
  nook
```

Then open `http://localhost:9090/dashboard` to configure providers and delivery targets.

Because the image copies the prebuilt dashboard bundle, rebuild `dashboard/.output/public` locally before shipping dashboard UI changes.

## Webhook Setup

Configure webhook providers via the dashboard. The default webhook paths are:

- Uptime Kuma: `/webhook/uptime-kuma`
- DocuSeal: `/webhook/docuseal`
- Dokploy: `/webhook/dokploy`

DocuSeal guide: [DOCUSEAL.md](DOCUSEAL.md)
Uptime Kuma guide: [UPTIME-KUMA.md](UPTIME-KUMA.md)
