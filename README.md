# Nook

Lightweight uptime monitor that sends alerts to Matrix rooms. Built with Bun.

## Features

- Monitors HTTP endpoints on configurable intervals
- Alerts on downtime (after 2 consecutive failures) and recovery
- Multi-room Matrix notifications
- Uptime Kuma webhook receiver for external alerts
- YAML configuration with fallback parser

## Setup

### Environment Variables

| Variable | Required | Description |
|---|---|---|
| `MATRIX_HOMESERVER` | Yes | Matrix server URL |
| `MATRIX_ACCESS_TOKEN` | Yes | Bot access token |
| `MATRIX_ROOM_IDS` | Yes | Comma-separated room IDs (also accepts `MATRIX_ROOM_ID`) |
| `CONFIG_PATH` | No | Config file path (default: `/data/config.yml`) |
| `WEBHOOK_SECRET` | No | Secret for authenticating incoming webhooks |
| `WEBHOOK_ENABLED` | No | `true` or `false` — overrides config and auto-detection |
| `WEBHOOK_PORT` | No | Webhook listener port (default: `9090`) |
| `TIMEOUT` | No | HTTP request timeout in seconds (default: `10`) |
| `EVERY` | No | Default check interval in seconds (default: `60`) |

### Configuration

The config file is a simple list of URLs to monitor:

```yaml
- https://example.com
- https://api.example.com
- https://status.example.com
```

All other settings (timeout, interval, webhook) are configured via environment variables.

### Run

```sh
bun run index.ts
```

### Docker

```sh
docker build -t nook .
docker run -d \
  -e MATRIX_HOMESERVER=https://matrix.example.com \
  -e MATRIX_ACCESS_TOKEN=syt_... \
  -e MATRIX_ROOM_IDS='!room1:example.com,!room2:example.com' \
  -v ./config.yml:/data/config.yml:ro \
  nook
```

## Webhook-Only Mode (Uptime Kuma → Matrix)

Nook can run without any `config.yml` as a pure Matrix notifier for Uptime Kuma. When no config file is found (or the config has no `sites`), the webhook server auto-enables.

### Minimal Setup

Just set the Matrix environment variables and deploy — no config file needed:

```sh
docker run -d \
  -e MATRIX_HOMESERVER=https://matrix.example.com \
  -e MATRIX_ACCESS_TOKEN=syt_... \
  -e MATRIX_ROOM_IDS='!room:example.com' \
  -e WEBHOOK_SECRET=some-random-secret \
  nook
```

### Uptime Kuma Configuration

1. In Uptime Kuma, go to **Settings → Notifications → Add Notification**
2. Type: **Webhook**
3. URL: `http://nook:9090/webhook/uptime-kuma` (use the Docker network hostname)
4. Method: `POST`
5. Content Type: `application/json`
6. Add header: `x-webhook-secret` = your `WEBHOOK_SECRET` value

### How It Decides

Environment variables take precedence over config file values.

| Config state | Webhook default |
|---|---|
| `WEBHOOK_ENABLED=true` or `false` | Respected (highest priority) |
| No config file | Enabled |
| Config with no `sites` | Enabled |
| Config with `sites`, no `webhook` key | Disabled |
| Explicit `webhook.enabled: true/false` in YAML | Respected |

### Endpoints

- `POST /webhook/uptime-kuma` — receives Uptime Kuma payloads and forwards alerts to Matrix
- `GET /health` — returns `200 ok`

If `WEBHOOK_SECRET` is set, webhook requests must include a matching `x-webhook-secret` header.

## Project Structure

```
index.ts           Entry point
src/types.ts       Type definitions
src/config.ts      YAML parsing and config loading
src/matrix.ts      Matrix API client
src/monitor.ts     Site health checking
src/webhook.ts     Webhook HTTP server
```
