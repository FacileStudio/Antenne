# MCP

`POST /mcp` is a remote [MCP](https://modelcontextprotocol.io) server (spec 2026-07-28,
stateless) for any MCP client holding a Registre access token bound to
`https://antenne.facile.studio/mcp`, with scope `antenne.read` and, for the write tools,
`antenne.write`. `/.well-known/oauth-protected-resource/mcp` is its RFC 9728 metadata. A
person's first call creates their account, as a first web sign-in would, and every account
is an administrator.

| Tool | Does | Confirmation |
|---|---|---|
| `antenne_list_events` | the activity log, newest first, filtered and paged | none, read-only |
| `antenne_get_event` | one event with its payload | none, read-only |
| `antenne_list_providers` | the providers, without credentials | none, read-only |
| `antenne_list_targets` | the delivery targets with routing and health, without addresses or credentials | none, read-only |
| `antenne_pool_status` | the apps connected to the bus, without tokens | none, read-only |
| `antenne_test_provider` | an IMAP provider's login, tried once | none, read-only |
| `antenne_check_target` | the check-now probe, under the dashboard's allowance per person | none |
| `antenne_test_target` | the fixed test message to one target | none |
| `antenne_set_enabled` | a provider or target on or off | none |
| `antenne_replay_event` | a logged event sent again to a target | always |

No tool returns the pool secret, a pool token or a provider or target credential. A replay asks
the person through MRTR when the client can elicit; otherwise it returns `confirm_url`, a link
to `/pending`, and nothing is sent until the person confirms there. The confirm page reads
`GET /api/mcp/pending` and posts `/api/mcp/pending/{id}/confirm` or `/dismiss`, from the
person's own browser session only (porte `mcpauth.RequireOwnSession`). A bearer token, a CLI
session, an API key or a delegated token gets 403, and so does a cookie minted before porte
v0.20.0: the page then asks the person to sign in again in the browser. The mobile and
desktop apps cannot confirm. A target edited after the question closes the action. Erasing a
person erases their pending actions.

## Left out on purpose

MCP clients cannot create or delete a provider or a delivery target. These stay on the
dashboard and its REST API (`PUT /api/settings`):

- **Creating one takes a credential.** A provider holds an IMAP password or a webhook signing
  secret. A target holds a Discord webhook URL, a Matrix access token or password, an
  SMTP password, an ntfy token, or the signing secret of an HTTP or Perception target. A tool that accepts one puts the secret in the model's context and
  in the client's transcript. No tool returns a credential either, so a model could not check
  what it wrote.
- **Deleting one removes a credential that cannot be read back.** The dashboard never shows
  the stored secret (it holds `••••••••`), so a deleted target or provider can only be
  rebuilt by someone who has the secret elsewhere. A deleted webhook provider also breaks the
  URL a third party posts to.
- **Editing routing, tags or credentials** is left out for the same reasons.
  `antenne_set_enabled` covers the one change an agent needs during an incident: turning a
  noisy provider or a broken target off and on.

The pool secret and pool tokens have no tool at all.

Back to the [API](api.md).
