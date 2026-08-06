# Nook — Development

Local setup, where the configuration comes from, and the quality gate.

## Prerequisites

- Go 1.25 (pinned in `mise.toml`)
- Bun, for the dashboard
- A PostgreSQL 16 to point `DATABASE_URL` at
- The `casier` CLI, for the development configuration

## Setup

```sh
mise run install     # client dependencies
mise run hooks       # enable the tracked git hooks in this clone
```

## Running

Two processes, in two terminals:

```sh
mise run dev         # the API on :9090, with secrets injected from Casier
mise run client      # the dashboard on :5173, proxying /api to the API
```

`mise run dev` wraps the API in `casier run`, so nothing has to exist on disk —
`.casier.toml` pins this repo to the `nook` project and its `dev` environment. `mise run
dev-offline` runs from the last-known-good cache when Casier is unreachable. See
[configuration.md](configuration.md) for what it supplies and how to add to it.

Migrations are embedded and applied at startup. To apply them without starting the server:

```sh
mise run migrate
```

`go run . migrate up`, `down`, `status` and `baseline` are all available through
`tronc/migrate`.

### Without Casier

The API only reads the process environment, so anything that puts the variables there works:

```sh
cd apps/api
cp .env.example .env      # then export it, or use your own runner
DATABASE_URL=postgres://nook:nook@localhost:5432/nook?sslmode=disable \
NOOK_ADMIN_PASSWORD=dev go run .
```

With no `NOOK_ADMIN_PASSWORD` set the dashboard and its API are open, which is the fastest way
to poke at a fresh install locally. `APP_ENV=production` refuses to boot that way.

## The quality gate

```sh
mise run check       # gofmt, vet, go test, then svelte-check
mise run check-go    # the Go half only
mise run format      # rewrite Go sources in place
```

`scripts/check.sh` is invoked directly rather than through `mise` inside itself: `mise run`
resolves every tool in the merged config before running a task body, so an unrelated broken
tool in a global config would take the gate down with it.

The gate also fails if a non-Postgres database driver appears in `go.mod`. Postgres is the only
database in this suite, tests included — a test on SQLite builds a different schema from the
GORM tags, then passes, proving nothing about the DDL that ships.

## Tests

Go tests sit beside the source:

| File | What it pins |
|---|---|
| `modules/settings/settings_test.go` | the redaction round-trip, path collisions, per-type marshalling |
| `modules/delivery/delivery_test.go` | opt-in routing with no fallthrough, and per-type readiness |
| `internal/envelope/envelope_test.go` | id stability, raw-payload withholding, the signature covering timestamp + id + body |
| `main_test.go` | every route has an entry in the reference registry, and the registry renders |

Adding a route means adding it to the owning module's `Documentation`, or
`TestEveryRouteIsDocumented` fails. The startup log warns about the same thing, so a route
added without one is visible even if the test is skipped.

```sh
cd apps/api && go test ./...
```

## The dashboard

SvelteKit 2 with Svelte 5 runes, Tailwind v4 and shadcn-svelte primitives over bits-ui. It
builds to `apps/client/build`, which the Go binary serves through `tronc/spa`. The Docker build
does this for you; there is no committed build output to keep in sync.

```sh
cd apps/client
bun run check        # svelte-check
bun run build
```

### Icons

Icons are inlined at build time rather than fetched from a CDN or the Iconify API — Nook is
self-hosted with no cloud dependency. `src/lib/icon-data.ts` is generated:

```sh
cd apps/client
bun run icons        # after adding a name to scripts/build-icons.ts
```

The generator fails loudly on a name that does not exist in the collection, which is worth
knowing because Solar has real typos in its icon names (`magnifer`, `pallete`, `spedometer`).

### Running the client against a deployed API

`vite.config.ts` proxies `/api` to `http://localhost:9090`. The port is hardcoded rather than
read from `process.env`: this project declares node types without installing them, so touching
`process` there turns a svelte-check warning into an error and fails the gate.
