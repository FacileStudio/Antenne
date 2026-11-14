# Changelog

## [Unreleased]

### Security

- fix(api): the MCP confirm page accepts only the person's own browser session (porte `mcpauth.RequireOwnSession`). A bearer token, a CLI session, an API key or a delegated token gets 403, and so does a cookie minted before porte v0.20.0: sign in again in the browser once before confirming.
- build(api): tronc v0.23.1, so a second OAuth client of the same person no longer reuses the first client's open replay request.

### Changed

- build(api): tronc v0.23.1, porte v0.20.0, porte/mcpauth v0.3.0, porte/forfait v0.1.3.
- refactor(api): the MCP state key, the confirm page mount, the account check, the events pager and the waiting and declined texts come from tronc and porte instead of local copies.
- feat(api): resolved and expired MCP actions older than 30 days are pruned hourly.
- feat(client): the confirm page asks the person to sign in again in the browser when the API refuses their session, in every locale.
- docs: `docs/mcp.md` holds the MCP reference and says why creating and deleting providers and targets stays off MCP.

## [1.7.2] - 2026-11-14

### Changed

- build(client): mise-en-scene v0.2.2.

### Fixed

- fix(client): on the Activity page, the delivery status icon and the last-event clock icon sit beside their label inside the badge instead of above it.

## [1.7.1] - 2026-11-14

### Added

- feat(client): `app.html` loads Vision's `s.js` on `antenne.facile.studio` only, with `data-mask` so ids in the path are recorded as `:id`. The API adds `https://vision.facile.studio` to the CSP `script-src`; the loader itself is allowed by the inline script hashes of the built client.

## [1.7.0] - 2026-11-14

### Added

- feat(client): a signed-out landing at `/` with a mise-en-scene demo of alerts arriving and a replayed delivery, plus an `antenne tail` terminal tile. A signed-in visitor still goes straight to `/providers`.
- feat(client): a public `/downloads` page listing every way to install the CLI (install script, `facile install antenne`, Homebrew).

### Changed

- build(client): muse v0.23.0 (Magnetik font, device-only `DownloadSection` on the landing) and mise-en-scene v0.2.1.

## [1.6.9] - 2026-11-13

### Changed

- build(client): muse v0.21.0.

### Fixed

- Icon-only controls show a muse tooltip; native `title` tooltips moved to muse `Tooltip`.

## [1.6.8] - 2026-11-13

### Changed

- chore(deploy): healthchecks run every 30 s with a 1 s start interval.

## [1.6.6] - 2026-11-13

### Fixed

- fix(api): `POST /api/auth/oidc/device/exchange` is in the API reference, so the boot no longer warns that it is missing. porte mounts it only when `OIDC_ISSUER` and `OIDC_CLI_AUDIENCE` are set, which the route test never did. `TestEveryRouteIsDocumented` now also builds the router against a stub issuer with a CLI audience, and fails on any single sign-on route left out of the registry.

## [1.6.5] - 2026-11-13

### Fixed

- fix(auth): the account store is built by `kit.NewStore` instead of a `kit.Store` literal, so porte no longer logs "this Store was built as a literal" at every boot when the picture backfill runs. `NewStore` reads the `users` table once and refuses to boot when a column the store names is missing. Every `users` column is NOT NULL with a default or written by the kit, so a test pins `Unwritten()` to none, and another proves the store reads the table when it is built.

## [1.6.4] - 2026-11-12

### Fixed

- fix(auth): an account re-created for a person Registre erased (a sign-in, a first MCP call or a profile sync racing the erasure) is refused while the subject is in the deletion ledger, and the next poll or push erases any such account again without a restart. The directory import no longer links an erased subject's identity to an existing account and drops that person from the snapshot's memberships; it still creates no account. porte v0.19.0 (directory v0.5.0): `Deletions.Holds = HoldsIdentity` and `oidc.Deps.Erased`.

## [1.6.3] - 2026-11-11

### Fixed

- fix(brand): favicons, touch and manifest icons, the link preview image and `site.webmanifest` come from muse v0.19.0's brand generator, and `app.html` carries its head block: the catalog title "Antenne · Alerts and the suite event bus", its description, theme colours and Open Graph and Twitter tags. The sidebar and login mark now uses the catalog's exact `solar:translation-2-bold-duotone` paths and opacity.

## [1.6.2] - 2026-11-11

### Security

- build(client): vite ^7.3.8 (7.3.6 to 7.3.8), with `overrides` for devalue ^5.9.3 (5.9.0 to 5.9.4), nanoid ^3.3.18 (3.3.17 to 3.3.20) and source-map-js ^1.2.2 (1.2.1 to 1.2.2).
- build(api): github.com/go-chi/chi/v5 v5.3.1 to v5.4.0 (CVE-2026-114437; Antenne does not call `ClientIPFromXFF`).

## [1.6.1] - 2026-11-11

### Security

- fix(server): every dashboard response redacts error text the way the MCP server does. The test-target, replay, test-alert, test-provider and settings errors go through `delivery.RedactError`, and stored delivery errors, health details and standing alerts go through `delivery.ScrubURLs` on read, so a Discord webhook URL that an event row or a health row written before v1.6.0 still holds is cut to `scheme://host` in every API response. Stored rows are not rewritten.
- fix(spaces): Registre erasure deletes every `pool_events` row whose payload holds, as an exact string value at any depth, the person's email or one of their identity subjects, in the transaction that deletes the account. A subject Antenne never saw sign in is erased from the bus too. Free text that merely mentions the address is not matched.
- fix(spaces): the Registre import no longer creates a user for a space member who never signed in. Every Antenne account is an admin, so the import had made one admin row per Registre member; a sign-in is now the only way an account is created. Membership stays keyed by address, so a first sign-in lands on it. Migration 00010 adds `space_members.from_registre` so a sync still removes an imported member Registre dropped. Existing rows are left as they are.
- fix(auth): `GET /api/auth/config` sends `username` and `email` only when the password form is offered, so an SSO-only instance no longer tells anonymous callers its bootstrap address.

### Fixed

- fix(api): the maintenance chores (pool retention, session expiry, event-log pruning) run at boot and then hourly. Pool retention ran on a 24-hour ticker that every redeploy reset, so the 48-hour floor and the 30-day ceiling were almost never applied. The first boot of this release trims the backlog.

### Rollback

- Reverting to v1.6.0 is safe: it ignores `from_registre`, and its migrator accepts the newer version. It would again pre-provision admin rows on the next sync.

## [1.6.0] - 2026-11-11

### Added

- feat(mcp): a remote MCP server at `/mcp` on tronc/mcp v0.22.1 and porte/mcpauth v0.2.1, for any MCP client holding a Registre token bound to the resource, with scopes `antenne.read` and `antenne.write`. Ten tools: `antenne_list_events`, `antenne_get_event`, `antenne_list_providers`, `antenne_list_targets`, `antenne_pool_status`, `antenne_test_provider`, `antenne_check_target`, `antenne_test_target`, `antenne_set_enabled` and `antenne_replay_event`. No result carries the pool secret, a pool token or a provider or target credential. A replay always needs the person: an MRTR question when the client can elicit, otherwise a link to the new `/pending` confirm page (muse, fr/en/es/de), which only the person's own session can confirm. A first MCP call creates the account, as a first web sign-in would. `MCP_RESOURCE` defaults to `PUBLIC_URL` + `/mcp` once single sign-on is on.
- feat(mcp): migration 00009 adds `mcp_actions` (tronc/mcp's `PendingSchema`). An erased person's pending actions are deleted with their account.

### Security

- fix(delivery): a failed delivery's error is stored and logged through `SafeErrorText`, so a Discord webhook URL, whose path is its credential, no longer lands in the event log or in Journal.

### Changed

- build(deps): tronc v0.21.0 to v0.22.1, tronc/migrate v0.1.0 to v0.1.1, and `modelcontextprotocol/go-sdk` v1.8.0 joins.
- refactor(server): the delivery health snapshot and the check-now probe are `API.HealthPage` and `API.CheckTarget`, shared by the dashboard and the MCP server. Routes, status codes and `Retry-After` are unchanged.

### Rollback

- Reverting to v1.5.1 is safe: it ignores `mcp_actions`, and its migrator accepts the newer version.

## [1.5.1] - 2026-11-11

### Security

- build(api): Go 1.26.11 (`go` directive, `golang:1.26.11-alpine` builder, mise pin). It fixes GO-2026-6848 to GO-2026-6859 in the standard library and toolchain, which douane reported as unscored gaps and which failed the gate.

## [1.5.0] - 2026-11-11

### Security

- fix(bus): registering never returns an existing app's token. Every registration issues a new token, revokes the app's previous one and closes the connection that used it; the app id is kept, so subscriptions and retention carry over. Before this, anyone holding the shared secret could register under an existing app's name and receive that app's live token.
- fix(bus): a pool token is bound to the secret it was issued under. `pool_apps.secret_fingerprint` (migration 00008) holds an HMAC-SHA256 keyed by the effective secret, never the secret itself, and the socket refuses a token whose fingerprint is not the current secret's. A dashboard save that changes the secret closes every live connection at once, and a restart with a new `ANTENNE_SECRET` refuses every old token, so rotating the secret now revokes every token issued under the old one. Rows from before the migration are bound to the current secret at the first boot, so the deploy disconnects nobody. `docs/configuration.md` has the rotation steps.

### Rollback

- Reverting to v1.4.1 is safe: it ignores the new column. Tokens it hands out meanwhile have no fingerprint, and the next forward boot binds them to the current secret.

## [1.4.1] - 2026-11-10

### Fixed

- build(deps): porte v0.18.0 to v0.18.1. `porte/kit` matches `<issuer>` and `<issuer>/` as one provider in the sign-in lookup, the second-subject refusal, the erasure, the picture backfill and the invitations subject, so an identity stored in the other spelling is found rather than linked twice or left unerased.
- fix(client): `@facile/client` is excluded from Vite's dev dependency optimizer next to muse, because it ships uncompiled `.svelte.ts` source. `vite build` was never affected.

## [1.4.0] - 2026-11-10

### Changed

- Licence: Antenne is now licensed under `AGPL-3.0-only` (see `LICENSE`); before this, the repository carried no licence.
- build(deps): tronc v0.17.1 to v0.21.0, porte v0.13.0 to v0.18.0 (`porte/directory` v0.3.0 to v0.4.0), `@facile/muse` v0.16.1 to v0.18.0. Every entry between the pins is additive.
- refactor(auth): a federated sign-in resolves through `porte/kit`'s store over Antenne's `users` table, and the sign-in routes mount through `kit.Mount`. The linking rules, the picture backfill and the "every account is an administrator" rule are unchanged; the route list and every unauthenticated status code are identical. The role and locale identity, the session middleware and the bootstrap password store stay in Antenne.
- fix(auth): a Registre erasure now deletes the account and its space memberships in one transaction, under the account lock, through `porte/kit`'s `EraseAccount`. The lookup used to run outside the transaction that deleted the rows. A test seeds a person in every table that names them and proves none does after the erasure.
- refactor(webhooks): the channels the git and filet providers publish (`git.push`, `git.pull_request`, `git.issue`, `git.event`, `filet.check.failed`, `filet.check.passed`) come from `antenne-client/go/catalog` v1.0.0, so the suite has one definition of each name. The names on the wire are unchanged. Antenne keeps its own alert envelope: it is a different shape from `antenne-client/go/envelope`, not a copy of it.
- refactor(server): the check-now limit runs on tronc's `middleware.Limiter` with the strict bound from tronc's `docs/ratelimit.md` (burst 5, then one check every 12 seconds, never more than 10 in a minute, 50 000 addresses remembered at most), keyed on the host of `RemoteAddr`. After the burst a caller waits 12 seconds instead of being blocked for a minute; `Retry-After` carries the real wait. The login throttle stays app-side: it counts failures and resets on success.
- refactor(client): the theme store and the 401 redirect come from `@facile/client` v0.1.0, under the same `antenne-theme` key, so a stored choice survives. A redirect to the login page now carries `?next=`. The fetch kernel stays: `@facile/client`'s `ApiError` does not carry `Retry-After`, which the check-now button shows.
- fix(deps): the API's `go` directive moves to 1.26.10, which clears the critical standard-library and toolchain advisories douane reports for `go 1.26`. `golang.org/x/crypto` moves to the commit that drops the unmaintained `openpgp` package (GO-2026-5932, no release carries it yet; Antenne never imported it), `golang.org/x/text` to v0.44.0 and `go-jose` to v4.1.9.
- ci: `filet.yml` is the suite's `preset: facile` plus Antenne's own exceptions (the ignore list, the modules without a `router.go`, and in-body why-comments allowed); filet v0.19.0 reports the same findings before and after. CI pins the filet action and binary to v0.19.0 and gains a douane 0.7.0 job (`-fail critical -scope prod`, sha256-pinned). The pre-push hook runs `filet gate` after the check script.
- build(hooks): the pre-push gate lives in a tracked `lefthook.yml` (the suite's shared `FacileStudio/hooks` remote, then `scripts/check.sh` and `filet gate`), lefthook 2.1.12 is pinned in `mise.toml`, and `mise run hooks` runs `lefthook install --reset-hooks-path`, which moves a clone off `core.hooksPath .githooks`. `.githooks/` is gone.

### Added

- feat(auth): the user menu shows pending Registre space invitations. `GET /api/me/invitations` comes from `porte/kit`'s `MountInvitations`, behind the session, mounted only when single sign-on and the directory credentials are set; the dashboard passes the answer to muse's `SideBar` and `MobileNav`.

## [1.3.4] - 2026-11-10

### Security

- fix(auth): the login throttle and the check-now limit key on the address tronc's real-IP middleware resolved, not on `X-Forwarded-For`, `CF-Connecting-IP` or `X-Real-IP` read by hand. A caller could pick its own bucket by forging the header, and behind a proxy that rewrites the header every visitor shared one. The boot warning about trusting `X-Forwarded-For` is gone with the cause.

## [1.3.3] - 2026-11-07

### Changed

- build(client): bump muse to v0.16.1 and replace the deprecated `outline` button variant with `secondary`.
- build: `GET /version` is baked at build time from the top released heading of `CHANGELOG.md` by `scripts/version.sh`. The `APP_VERSION` build arg is gone from the Dockerfile and compose file, so a stale Dokploy env can no longer show the wrong version. The response is unchanged.

## [1.3.2] - 2026-11-07

### Changed

- build(client): bump muse to v0.14.4 (from v0.14.2). Switching the language in the user menu re-renders the whole UI at once, with no reload. A single 100% DonutChart slice draws as a full ring.

## [1.3.1] - 2026-11-07

### Security

- Registre tokens are sealed at rest. porte to v0.13.0: `porte_identities.access_token` and `refresh_token`
  are encrypted with AES-256-GCM under `PORTE_TOKEN_KEY`. Existing rows are sealed once, in the background,
  after boot, and the count is logged. An invalid `PORTE_TOKEN_KEY` stops the boot. With the key unset nothing changes.
- Losing `PORTE_TOKEN_KEY` makes the stored tokens unreadable: people sign in again, nothing else is lost.
  Do not roll back to a build older than porte v0.13.0 after the first reseal.

## [1.3.0] - 2026-11-07

### Changed

- The sidebar user card opens the shared muse user menu (account, settings, subscription, language, theme, help, sign out). The cog link and the in-settings sign-out are gone.
- Settings use muse `SettingsLayout` with Profile, Preferences, Bus, Events and Advanced. Profile is read-only and links to Facile; language and theme live in Preferences; the configuration import sits in a `DangerZone`. `/settings/appearance` redirects to `/settings/preferences`.
- `@facile/muse` v0.13.0 to v0.14.2.

## [1.2.1] - 2026-11-07

### Changed

- Libraries: porte v0.12.0 to v0.12.1 (GDPR erasure no longer fails on a schema without
  `porte_magic_links`), `@facile/muse` v0.12.1 to v0.13.0 (borderless controls with the light
  `fc-field` fill, `Select` as a custom combobox, badges that never wrap).
- The activity source filter has an accessible name ("Filter by source"), because muse's
  `Select` is now a button that a screen reader announces by its label.

## [1.2.0] - 2026-11-07

### Changed

- Sign-in goes through Registre only. With `OIDC_ISSUER` set porte mounts no password or
  registration route and the bootstrap admin is no longer seeded. The login page is muse
  `AuthScreen` with `SignInWithFacile`; the password form is gone. The previous bootstrap
  account row stays in `users` and is inert.
- `SSO_ONLY` is removed from the code and compose. Delete it from the Dokploy env after deploy.
- porte v0.5.2 to v0.12.0, porte/directory v0.1.0 to v0.3.0, tronc v0.16.0 to v0.17.1, muse
  v0.8.1 to v0.12.1.
- Account linking follows porte: an existing account is linked by a verified email only, and
  refused when it already holds another subject of the same issuer.
- The dashboard is available in fr, en, es and de. The language comes from the Registre
  profile (`users.locale`), then the picked-language cookie, then the browser. es and de need
  native review. Alert texts produced by the server stay in the language of their source.
- tronc security headers: CSP report-only plus enforced anti-framing. The dashboard's two
  inline scripts are allowed by hash; Iconify and the Journal browser origin are in
  `connect-src`. The CSP stays report-only.
- `svelte.config.js` takes the build stamp once, so the page and the bundle agree on
  SvelteKit's global name.
- CI builds the API and the client.

### Added

- Migration `00007_launch.sql`: `porte_login_codes.provider` and `.subject`, `users.locale`,
  and the `deletion_state` table.
- GDPR deletion receiver: Registre erasures are polled and accepted on
  `POST /api/registre/deletions`. A deleted person loses their account, porte rows and space
  memberships. The event log, providers and delivery targets are instance data and stay.
- `GET /version`, set from the `APP_VERSION` build arg.

### Removed

- `POST /api/auth/login` and `/api/auth/register` answer 404 when an issuer is configured.
  Pool registration, webhook intake and bearer tokens are unchanged.
