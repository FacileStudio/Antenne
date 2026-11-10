# Changelog

## [Unreleased]

### Changed

- Licence: Antenne is now licensed under `AGPL-3.0-only` (see `LICENSE`); before this, the repository carried no licence.
- build(deps): tronc v0.17.1 to v0.21.0, porte v0.13.0 to v0.18.0 (`porte/directory` v0.3.0 to v0.4.0), `@facile/muse` v0.16.1 to v0.18.0. Every entry between the pins is additive.
- refactor(auth): a federated sign-in resolves through `porte/kit`'s store over Antenne's `users` table, and the sign-in routes mount through `kit.Mount`. The linking rules, the picture backfill and the "every account is an administrator" rule are unchanged; the route list and every unauthenticated status code are identical. The role and locale identity, the session middleware and the bootstrap password store stay in Antenne.
- fix(auth): a Registre erasure now deletes the account and its space memberships in one transaction, under the account lock, through `porte/kit`'s `EraseAccount`. The lookup used to run outside the transaction that deleted the rows. A test seeds a person in every table that names them and proves none does after the erasure.
- refactor(webhooks): the channels the git and filet providers publish (`git.push`, `git.pull_request`, `git.issue`, `git.event`, `filet.check.failed`, `filet.check.passed`) come from `antenne-client/go/catalog` v1.0.0, so the suite has one definition of each name. The names on the wire are unchanged. Antenne keeps its own alert envelope: it is a different shape from `antenne-client/go/envelope`, not a copy of it.
- refactor(server): the check-now limit runs on tronc's `middleware.Limiter` with the strict bound from tronc's `docs/ratelimit.md` (burst 5, then one check every 12 seconds, never more than 10 in a minute, 50 000 addresses remembered at most), keyed on the host of `RemoteAddr`. After the burst a caller waits 12 seconds instead of being blocked for a minute; `Retry-After` carries the real wait. The login throttle stays app-side: it counts failures and resets on success.
- refactor(client): the theme store and the 401 redirect come from `@facile/client` v0.1.0, under the same `antenne-theme` key, so a stored choice survives. A redirect to the login page now carries `?next=`. The fetch kernel stays: `@facile/client`'s `ApiError` does not carry `Retry-After`, which the check-now button shows.
- fix(deps): the API's `go` directive moves to 1.26.10, which clears the critical standard-library and toolchain advisories douane reports for `go 1.26`. `golang.org/x/crypto` moves to the commit that drops the unmaintained `openpgp` package (GO-2026-5932, no release carries it yet; Antenne never imported it), `golang.org/x/text` to v0.44.0 and `go-jose` to v4.1.9.

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
