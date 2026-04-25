# Connecting Uptime Kuma to Nook

This guide assumes both Uptime Kuma and Nook are deployed through Dokploy on the same server.

## Nook Setup

### Environment Variables

Set these in Dokploy's environment settings for the Nook service:

| Variable | Required | Value |
|---|---|---|
| `MATRIX_HOMESERVER` | Yes | Your Matrix server URL (e.g. `https://matrix.facile.studio`) |
| `MATRIX_ACCESS_TOKEN` | Yes | Bot access token |
| `MATRIX_ROOM_IDS` | Yes | Comma-separated room IDs (e.g. `!abc:matrix.example.com`) |
| `WEBHOOK_SECRET` | Recommended | A random secret string shared with Uptime Kuma |

No `config.yml` is needed. Nook auto-enables the webhook server when no sites are configured.

### Network

Keep Nook internal — do not expose it publicly. Uptime Kuma reaches it through the Docker network.

## Uptime Kuma Setup

1. Open your Uptime Kuma dashboard
2. Go to **Settings > Notifications > Setup Notification**
3. Set the following:
   - **Notification Type**: Webhook
   - **Post URL**: `http://<nook-service-name>:9090/webhook/uptime-kuma`
   - **Request Body**: select **JSON**
4. If you set `WEBHOOK_SECRET` on Nook, add a custom header:
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
