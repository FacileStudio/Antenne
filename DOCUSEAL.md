# Connecting DocuSeal to Nook

This guide assumes Nook is reachable from DocuSeal through a public URL or a tunnel.
These values can come from env bootstrap or from the dashboard after first boot.

## Nook Setup

### Environment Variables

Set these in your Nook deployment:

| Variable | Required | Value |
|---|---|---|
| `MATRIX_HOMESERVER` | Yes | Your Matrix server URL |
| `MATRIX_ACCESS_TOKEN` | Yes | Bot access token |
| `MATRIX_ROOM_IDS` | Yes | Comma-separated room IDs |
| `DOCUSEAL_WEBHOOK_ENABLED` | Recommended | `true` if you also monitor sites and want DocuSeal enabled |
| `DOCUSEAL_WEBHOOK_SECRET` | Recommended | Random shared secret value |
| `DOCUSEAL_WEBHOOK_SECRET_HEADER` | No | Secret header name, default `x-docuseal-secret` |
| `DOCUSEAL_WEBHOOK_PATH` | No | Custom path, default `/webhook/docuseal` |
| `WEBHOOK_PORT` | No | Webhook listener port, default `9090` |

If Nook runs without `sites`, the webhook server auto-enables and DocuSeal is enabled by default.

## DocuSeal Setup

DocuSeal's webhook documentation says secrets are sent as custom request headers. Nook maps that directly to a provider-specific header and value.

1. Open `https://console.docuseal.com/webhooks`
2. Create a new webhook
3. Set URL to `https://your-nook-domain.example/webhook/docuseal`
4. Select the events you want, such as:
   - `form.completed`
   - `form.declined`
   - `form.started`
   - `template.updated`
5. Click `Add Secret`
6. Set:
   - key: `x-docuseal-secret`
   - value: the same value as `DOCUSEAL_WEBHOOK_SECRET`

## What Nook Sends to Matrix

- `form.completed`: template name, signer, submission id, optional external id, submission link, document links
- `form.declined`: template name, signer, decline reason, submission link
- `form.started` and `form.viewed`: informational signer activity
- `template.*` and `submission.*`: compact lifecycle messages

## Verify

1. Trigger a DocuSeal webhook event, or use a test submission
2. Check your Matrix room for a `[DocuSeal]` message
3. Check Nook logs for a line like `[webhook] docuseal: form.completed`
