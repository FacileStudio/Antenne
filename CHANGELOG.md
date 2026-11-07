# Changelog

## [Unreleased]

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
