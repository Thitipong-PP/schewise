# Development guide

Use Go 1.27.3, Node.js 26, pnpm 12.16.1, and Docker Compose. `go.mod`, `go.sum`,
and `pnpm-lock.yaml` record dependencies. The race detector needs CGO and a C
compiler (Xcode command-line tools on macOS; GCC/Clang on Linux).

## Commands

Run from the repository root:

| Command                 | Behavior                                                        |
| ----------------------- | --------------------------------------------------------------- |
| `make help`             | List commands                                                   |
| `make init`             | Generate ignored local credentials without overwriting them     |
| `make setup`            | Download Go dependencies and install the frozen pnpm lockfile   |
| `make dev-api`          | Read local environment and run the Go API                       |
| `make dev-web`          | Read local environment and run Next.js                          |
| `make fmt`              | Apply gofmt and Prettier                                        |
| `make fmt-check`        | Check formatting without editing                                |
| `make lint`             | Go vet and ESLint                                               |
| `make test`             | Go race tests; DB tests skip unless explicitly enabled          |
| `make integration-test` | Run all SQL tests in isolated schemas and demo target databases |
| `make api-generate`     | Regenerate OpenAPI and frontend API types                       |
| `make api-check`        | Check generated contracts match backend routes/DTOs             |
| `make check`            | Contracts, formatting, lint, Go/web tests, and TypeScript       |
| `make build`            | Build API/migration binaries and production web app             |
| `make smoke`            | Launch production builds and verify page/API/auth integration   |
| `make auth-smoke`       | Verify the running HTTPS stack with local CA validation         |
| `make migrate`          | Apply application migrations (also done at API startup)         |
| `make migrate-status`   | Show application migration version                              |
| `make docker-up`        | Build/start the complete development stack                      |
| `make docker-down`      | Stop/remove containers; preserve data/CA/cache volumes          |
| `make local-ca`         | Export the HTTPS gateway's public CA certificate                |

`pnpm lint`, `pnpm typecheck`, `pnpm test:toolchain`, `pnpm test:admin`, and
`pnpm build` operate on the web package. `pnpm format` covers supported
repository files; gofmt covers Go. Next.js type generation runs before
TypeScript checking.

## API contracts and frontend checks

After changing administration DTOs or routes, run `make api-generate` and commit
both generated files. `make api-check` compares OpenAPI semantically and the
formatted TypeScript output exactly. The client uses `openapi-fetch` with those
types, including required CSRF headers. These libraries solve contract drift; Go
remains the only authorization/data owner. `pnpm test:admin` uses the existing
Node test runner for access-source privacy, forest/move selectors, request
headers, and error handling. Browser testing still covers rendered interaction,
keyboard focus, and responsive layout.

## TypeScript 7 and ESLint 10 compatibility

The application uses the latest stable TypeScript 7 compiler and ESLint 10.
TypeScript 7 does not expose the JavaScript compiler API needed by
typescript-eslint, so the workspace uses Microsoft's side-by-side arrangement:

- `@typescript/native` aliases `typescript@7.0.2` and supplies the native `tsc`.
- `typescript` aliases `@typescript/typescript6@6.0.2` and supplies the
  TypeScript 6 API for tools that import `typescript`, plus a separate `tsc6`
  command.

`pnpm typecheck` and Next.js production builds use the project-local native
`tsc`. The linter uses the compatibility API. Keep both aliases when upgrading;
replacing the API alias with TypeScript 7 directly makes typescript-eslint fail
at import. See
[Microsoft's migration guidance](https://devblogs.microsoft.com/typescript/announcing-typescript-7-0/#running-side-by-side-with-typescript-6.0)
and
[Next.js TypeScript 7 support](https://nextjs.org/docs/app/api-reference/config/typescript#using-typescript-7).

The latest published `eslint-plugin-import@2.32.0`,
`eslint-plugin-react@7.37.5`, and `eslint-plugin-jsx-a11y@6.10.2` still declare
peer ranges ending at ESLint 9. Their ranges are not overridden or suppressed.
ESLint 10 passes this project's lint and plugin-execution probes, but this is
project-level verification, not a claim of complete upstream ESLint 10 support.
Installation can report those peer warnings until upstream updates the metadata.

`pnpm test:toolchain` exercises both invalid and valid TypeScript/JSX through
the real configuration, checking that TypeScript, React, accessibility, import,
and Next.js rules actually report errors instead of silently dropping
validation. It runs in `make check` and CI alongside normal lint and native type
checking.

## Configuration

`make init` creates `.env` with unique local passwords. Make wrappers use Node's
dotenv parser without shell evaluation; process environment overrides the file.
The Go programs read only process environment. Never log or commit `.env`.

| Variable                  | Default / purpose                                                                            |
| ------------------------- | -------------------------------------------------------------------------------------------- |
| `SCHEWISE_HTTP_ADDR`      | API bind address, default `127.0.0.1:8080`                                                   |
| `SCHEWISE_ORIGIN`         | Exact browser/CSRF HTTPS origin, default `https://localhost:8443`                            |
| `SCHEWISE_ADMIN_EMAIL`    | First-install Administrator email                                                            |
| `SCHEWISE_ADMIN_PASSWORD` | First-install password; never resets an existing account                                     |
| `SCHEWISE_SESSION_TTL`    | Absolute authenticated lifetime; default `24h`, allowed `1m`–`720h`                          |
| `SCHEWISE_DATABASE_URL`   | Optional full application Postgres URL, overrides PG variables                               |
| `PGHOST` / `PGPORT`       | Native app database destination, default `127.0.0.1:5432`                                    |
| `PGDATABASE` / `PGUSER`   | Native application database/name, default `schewise`                                         |
| `PGPASSWORD`              | Application database password; required unless a full URL is set                             |
| `PGSSLMODE`               | Default `disable` for local private Docker networking; configure verified TLS for remote DBs |
| `POSTGRES_PASSWORD`       | Docker database initialization superuser password; not used by API                           |
| `API_BASE_URL`            | Next.js-to-Go destination, default `http://127.0.0.1:8080`                                   |

The supplied Compose configuration provisions database/role `schewise`, uses
`db:5432` for the API, and `api:8080` for Next.js. Only the bootstrap Postgres
account is a superuser. Custom external database deployment requires a Compose
override or independently supplied process configuration.

`apps/web/.env.local` can override Next.js settings. Restart after configuration
changes; Next.js rewrites are read at dev startup or production build time.
Never put secrets in `NEXT_PUBLIC_*` variables.

## Docker and HTTPS

Follow [authentication setup](authentication.md) for CA trust and credentials.
Browser login uses https://localhost:8443, with Secure cookies in every mode.
The CA is not automatically added to the host trust store.

The API mounts the repository, compiles at startup without VCS stamping, then
executes the binary so shutdown signals reach Fiber. Restart it after Go edits.
The web mounts `apps/web/src` for hot reload; rebuild after dependency/config or
other non-source changes. Postgres, Caddy certificates, and Go caches use named
volumes. Routine `docker compose down` preserves them. Do not delete volumes to
fix a failed test or reset a password.

For native API/web development, start the database with
`docker compose up -d --wait db`, then run the two `make dev-*` commands. For
browser HTTPS with native services, use your trusted local reverse proxy with
`/api/*` pointing to port 8080 and other routes to port 3000, preserving Origin.
The supplied gateway targets Compose service names; use the complete Compose
stack for the ready-made HTTPS workflow.

## Database tests and production smoke

```sh
docker compose up -d --wait db
make integration-test
```

Integration tests create random `auth_test_*` and `rbac_test_*` schemas, apply
migrations there, and drop only that schema afterward. They do not clear
application users, sessions, or audits. Alternatively supply an isolated
PostgreSQL URL through `SCHEWISE_TEST_DATABASE_URL`; CI does this and runs the
suite with the race detector. Tests cover rotation, CSRF, permissions,
revocation, expiry, bootstrap idempotency, audit rollback, rate limits, cookie
flags, and log redaction. RBAC coverage adds the Alice–Frank hierarchy fixture,
permission union, scope isolation, delegation, immutable roles, legacy
migration, concurrent last-admin protection, moves/deletes, and dry-run impact.
Connection ownership uses a synthetic adapter; no target database is contacted.

To test production builds locally, keep Postgres running and stop native API/web
servers or `docker compose stop gateway web api` first:

```sh
make check
make integration-test
make build
make smoke
```

By default, ports 8080/3000 must be free and the build must use the default API
destination. To preserve an existing running stack, choose free smoke ports and
build with a matching destination:

```sh
API_BASE_URL=http://127.0.0.1:8082 make build
API_BASE_URL=http://127.0.0.1:8082 SCHEWISE_SMOKE_API_PORT=8082 SCHEWISE_SMOKE_WEB_PORT=3002 make smoke
```

Supply a separate test database URL and synthetic matching bootstrap credentials
when testing in isolation. Smoke uses the configured application database and
normal startup migrations; port overrides do not isolate the database. Rebuild
with the normal destination before switching back to default ports. This smoke
script tests internal HTTP integration using explicit test cookie headers;
actual browser auth uses HTTPS. Run the stack again, export its CA, and run
`make auth-smoke` for certificate-validated HTTPS. Both smoke commands need
valid installation credentials matching the existing Administrator. They write
only session and audit activity. The API's automatic migration/bootstrap
behavior is the same as normal startup.

## CI and repository settings

- **Go:** formatting, vet, race tests including real Postgres integration,
  build.
- **Web and integration:** frozen installation, formatting, ESLint, TypeScript,
  dependency audit, production builds, and auth/page/API smoke with Postgres.
- **Development containers:** generate temporary installation credentials,
  build, start Postgres/API/web/Caddy, trust the CA per process, test HTTPS
  auth, and verify a zero-exit-code API shutdown.

Actions are pinned to full commits and repository permissions are read-only.
Dependabot tracks Actions, npm, Go, and Docker dependencies. Repository
rulesets, private vulnerability reporting, dependency alerts, and optional
code-owner enforcement still require GitHub settings.

The initial npm audit reports one moderate transitive lint dependency advisory,
[GHSA-g3hv-75wg-m249](https://github.com/advisories/GHSA-g3hv-75wg-m249). CI
blocks high/critical findings. Compatibility notes for the current lint/compiler
versions are documented above; do not override dependency metadata or disable
rules to hide compatibility failures.

## Troubleshooting

- **Certificate warning:** export and explicitly trust your local CA; never
  bypass validation. Use `--cacert .local/root.crt` for curl.
- **Login/CSRF failure:** use the exact HTTPS origin and current installation
  password. Changing bootstrap variables does not reset an existing account.
- **Database failure:** inspect `docker compose logs db api`; confirm Postgres
  is healthy and password values still match the initialized volume.
- **Port conflict:** stop the existing service. Smoke checks need their selected
  API/web ports free.
- **Go/network failure:** allow the official Go module proxy/checksum services
  or install the pinned toolchain manually.
- **Lockfile mismatch:** use the pnpm version in `packageManager`, intentionally
  update dependencies, and commit the lockfile.

## PostgreSQL fixtures

`make demo-db` starts the dedicated synthetic target on localhost:5433;
`make demo-up` registers/syncs it using the installation user's root
permissions. `make integration-test` starts that fixture and creates isolated
target databases there, independently from application metadata test schemas.
See [connections](connections.md) for setup, encryption/host-policy settings,
least-privilege SQL, and the Compose-versus-native registered destination.
