# Nook

Nook is a Bun service that watches modular providers, forwards alerts to Matrix, Discord, HTTP webhooks, SMTP, or ntfy, and ships with a Nuxt dashboard for configuring everything without hand-editing config every time.

The Docker image is runtime-only and serves a prebuilt dashboard from `dashboard/.output/public`, so container builds do not need to install npm packages.

## What It Does

- monitors websites on intervals
- polls IMAP inboxes for newly received emails
- receives Uptime Kuma, DocuSeal, and Dokploy webhooks
- delivers alerts to Matrix, Discord webhooks, HTTP webhooks, SMTP, and ntfy
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
| `MATRIX_REFRESH_TOKEN` | No | Matrix refresh token for renewing expired access tokens |
| `MATRIX_USER_ID` | No | Matrix user ID for password login, for example `@nook:matrix.example.com` |
| `MATRIX_PASSWORD` | No | Matrix password used to log back in and regenerate tokens |
| `MATRIX_DEVICE_ID` | No | Stable Matrix device ID to reuse during re-login |
| `MATRIX_ROOM_IDS` | No | Comma-separated room IDs |
| `MATRIX_ROOM_ID` | No | Single room fallback |

If `MATRIX_USER_ID` and `MATRIX_PASSWORD` are set, Nook will request refresh tokens on login and re-authenticate automatically when Matrix returns `M_UNKNOWN_TOKEN`.

### ntfy Bootstrap

| Variable | Required | Description |
|---|---|---|
| `NTFY_ENABLED` | No | Enable ntfy target bootstrap |
| `NTFY_SERVER_URL` | No | ntfy base URL. Default: `https://ntfy.sh` |
| `NTFY_TOPIC` | No | Topic name to publish to |
| `NTFY_ACCESS_TOKEN` | No | Bearer token for protected topics |
| `NTFY_PRIORITY` | No | Notification priority, usually `1` to `5` |
| `NTFY_TAGS` | No | Comma-separated tags |
| `NTFY_CLICK_URL` | No | Optional click-through URL |

### Monitoring Bootstrap

| Variable | Required | Description |
|---|---|---|
| `TIMEOUT` | No | Default timeout in seconds |
| `EVERY` | No | Default website provider interval in seconds |

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
| `DOKPLOY_WEBHOOK_ENABLED` | No | Enable or disable Dokploy provider |
| `DOKPLOY_WEBHOOK_PATH` | No | Default path: `/webhook/dokploy` |
| `DOKPLOY_WEBHOOK_SECRET` | No | Dokploy secret value |
| `DOKPLOY_WEBHOOK_SECRET_HEADER` | No | Default: `x-dokploy-secret` |

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

Saved dashboard settings use a single provider list. Each provider can be a website, IMAP inbox, or webhook source.

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
bun run dashboard:build
bun run dev
```

For dashboard-only work:

```sh
bun run dashboard:dev
```

Before committing dashboard changes meant for deployment, rebuild the static bundle:

```sh
bun run dashboard:build
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

Because the image copies the prebuilt dashboard bundle, rebuild `dashboard/.output/public` locally before shipping dashboard UI changes.

## Webhook Setup

### Uptime Kuma

- URL: `http://<nook-host>:9090/webhook/uptime-kuma`
- header: `x-webhook-secret`
- value: whatever you set in `UPTIME_KUMA_WEBHOOK_SECRET`

### DocuSeal

- URL: `https://<nook-host>/webhook/docuseal`
- header: `x-docuseal-secret`
- value: whatever you set in `DOCUSEAL_WEBHOOK_SECRET`

### Dokploy

- URL: `https://<nook-host>/webhook/dokploy`
- header: `x-dokploy-secret`
- value: whatever you set in `DOKPLOY_WEBHOOK_SECRET`

DocuSeal guide: [DOCUSEAL.md](DOCUSEAL.md)
Uptime Kuma guide: [UPTIME-KUMA.md](UPTIME-KUMA.md)
