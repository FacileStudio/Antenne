# Connecting Uptime Kuma to Nook

This guide assumes both Uptime Kuma and Nook are deployed through Dokploy on the same server.
You can set these values from env first, then adjust them later in the Nook dashboard.

## Nook Setup

### Environment Variables

Set these in Dokploy's environment settings for the Nook service:

| Variable | Required | Value |
|---|---|---|
| `MATRIX_HOMESERVER` | Yes | Your Matrix server URL (e.g. `https://matrix.facile.studio`) |
| `MATRIX_ACCESS_TOKEN` | Yes | Bot access token |
| `MATRIX_ROOM_IDS` | Yes | Comma-separated room IDs (e.g. `!abc:matrix.example.com`) |
| `UPTIME_KUMA_WEBHOOK_SECRET` | Recommended | A random secret string shared with Uptime Kuma |
| `WEBHOOK_ENABLED` | No | `true` or `false` — overrides auto-detection |
| `WEBHOOK_PORT` | No | Webhook listener port (default: `9090`) |
| `UPTIME_KUMA_WEBHOOK_PATH` | No | Custom Uptime Kuma webhook path |
| `UPTIME_KUMA_WEBHOOK_SECRET_HEADER` | No | Secret header name (default: `x-webhook-secret`) |

No `config.yml` is needed. Nook can bootstrap the Uptime Kuma provider from env alone, and the dashboard can later persist overrides into `/data/nook.json`.

### Network

Keep Nook internal — do not expose it publicly. Uptime Kuma reaches it through the Docker network.

## Uptime Kuma Setup

1. Open your Uptime Kuma dashboard
2. Go to **Settings > Notifications > Setup Notification**
3. Set the following:
   - **Notification Type**: Webhook
   - **Post URL**: `http://<nook-service-name>:9090/webhook/uptime-kuma`
   - **Request Body**: select **JSON**
4. If you set `UPTIME_KUMA_WEBHOOK_SECRET` on Nook, add a custom header:
   - **Header Name**: `x-webhook-secret`
   - **Header Value**: the same secret you set in Nook's environment

Replace `<nook-service-name>` with the actual service name in Dokploy (visible in the service's network settings, e.g. `nook-a1b2c3`).

## Verify

1. In Uptime Kuma, click **Test** on the notification
2. Check your Matrix room for a message from Nook
3. Check Nook's logs in Dokploy for `[webhook] received alert`

## Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/webhook/uptime-kuma` | Receives Uptime Kuma alert payloads |
| `GET` | `/health` | Returns `200 ok` |
