# Changelog

## [Unreleased]

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
