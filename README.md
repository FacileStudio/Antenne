# Nook

Nook is a Bun service that monitors HTTP endpoints, receives external webhooks, and forwards alerts to Matrix. It now ships with a Nuxt dashboard for configuring sites, Matrix delivery, and webhook providers without hand-editing config every time.

## What It Does

- monitors sites on intervals
- forwards downtime and recovery alerts to Matrix
- receives Uptime Kuma and DocuSeal webhooks
- serves a dashboard at `/dashboard`
- keeps env and YAML as bootstrap defaults, with dashboard-saved settings overriding them later

## Config Model

Nook uses two layers:

1. bootstrap defaults from env vars and optional `config.yml`
2. persistent dashboard settings in `/data/nook.json`

If `/data/nook.json` exists, it wins. If it does not, Nook falls back to env plus `config.yml`.

## Environment Variables

### Core

| Variable | Required | Description |
|---|---|---|
| `PORT` | No | HTTP port for API, dashboard, health, and webhooks. Falls back to `WEBHOOK_PORT` or `9090`. |
| `NOOK_SETTINGS_PATH` | No | JSON settings file path. Default: `/data/nook.json` |
| `CONFIG_PATH` | No | Legacy YAML bootstrap file path. Default: `/data/config.yml` |

### Dashboard Auth

| Variable | Required | Description |
|---|---|---|
| `NOOK_ADMIN_PASSWORD` | No | If set, dashboard API requires login |
| `NOOK_ADMIN_USERNAME` | No | Login label only. Default: `admin` |

### Matrix Bootstrap

| Variable | Required | Description |
|---|---|---|
| `MATRIX_HOMESERVER` | No | Matrix homeserver URL |
| `MATRIX_ACCESS_TOKEN` | No | Matrix bot access token |
| `MATRIX_ROOM_IDS` | No | Comma-separated room IDs |
| `MATRIX_ROOM_ID` | No | Single room fallback |

### Monitoring Bootstrap

| Variable | Required | Description |
|---|---|---|
| `TIMEOUT` | No | Default timeout in seconds |
| `EVERY` | No | Default site interval in seconds |

### Webhook Bootstrap

| Variable | Required | Description |
|---|---|---|
| `WEBHOOK_ENABLED` | No | Global default for providers |
| `WEBHOOK_PORT` | No | Legacy port fallback if `PORT` is not set |
| `WEBHOOK_SECRET` | No | Legacy shared secret for Uptime Kuma |
| `UPTIME_KUMA_WEBHOOK_ENABLED` | No | Enable or disable Uptime Kuma provider |
| `UPTIME_KUMA_WEBHOOK_PATH` | No | Default path: `/webhook/uptime-kuma` |
| `UPTIME_KUMA_WEBHOOK_SECRET` | No | Uptime Kuma secret value |
| `UPTIME_KUMA_WEBHOOK_SECRET_HEADER` | No | Default: `x-webhook-secret` |
| `DOCUSEAL_WEBHOOK_ENABLED` | No | Enable or disable DocuSeal provider |
| `DOCUSEAL_WEBHOOK_PATH` | No | Default path: `/webhook/docuseal` |
| `DOCUSEAL_WEBHOOK_SECRET` | No | DocuSeal secret value |
| `DOCUSEAL_WEBHOOK_SECRET_HEADER` | No | Default: `x-docuseal-secret` |

## Legacy `config.yml`

This is still valid as bootstrap input:

```yaml
timeout: 10
every: 60
sites:
  - https://example.com
  - name: API
    url: https://api.example.com
    every: 30
```

After saving from the dashboard, the resulting JSON settings file becomes the active config source.

## Routes

- `GET /health`
- `GET /dashboard`
- `GET /api/settings`
- `PUT /api/settings`
- `POST /api/test-alert`
- `POST /api/login`
- `POST /api/logout`
- `POST /webhook/uptime-kuma`
- `POST /webhook/docuseal`

## Local Development

```sh
bun install
bun run dashboard:build
bun run dev
```

For dashboard-only work:

```sh
bun run dashboard:dev
```

## Docker

```sh
docker build -t nook .
docker run -d \
  -p 9090:9090 \
  -e PORT=9090 \
  -e NOOK_ADMIN_PASSWORD=change-me \
  -e MATRIX_HOMESERVER=https://matrix.example.com \
  -e MATRIX_ACCESS_TOKEN=syt_xxx \
  -e MATRIX_ROOM_IDS='!room:example.com' \
  -v ./data:/data \
  nook
```

Then open `http://localhost:9090/dashboard`.

## Webhook Setup

### Uptime Kuma

- URL: `http://<nook-host>:9090/webhook/uptime-kuma`
- header: `x-webhook-secret`
- value: whatever you set in `UPTIME_KUMA_WEBHOOK_SECRET`

### DocuSeal

- URL: `https://<nook-host>/webhook/docuseal`
- header: `x-docuseal-secret`
- value: whatever you set in `DOCUSEAL_WEBHOOK_SECRET`

DocuSeal guide: [DOCUSEAL.md](DOCUSEAL.md)
Uptime Kuma guide: [UPTIME-KUMA.md](UPTIME-KUMA.md)
