# Development guide

## Tooling and commands

Use Go 1.27.3 (the preferred toolchain in `go.mod`), Node.js 26, and pnpm
12.16.1. The root Go module covers the backend; the pnpm workspace covers
JavaScript tools and `apps/web`. There are no third-party Go modules yet, so
`go.sum` is absent.

`make setup` runs `go mod download` and `pnpm install --frozen-lockfile`. The Go
command may download the preferred toolchain automatically. Install pnpm using
`npm install --global pnpm@12.16.1` if it is not already available.

| Task                | Make command     | Direct command                                                        |
| ------------------- | ---------------- | --------------------------------------------------------------------- |
| Run API             | `make dev-api`   | `go run ./cmd/api`                                                    |
| Run web             | `make dev-web`   | `pnpm dev:web`                                                        |
| Format              | `make fmt`       | `gofmt -w cmd internal` and `pnpm format`                             |
| Check Go formatting | `make fmt-check` | `gofmt -l cmd internal` (must print no files) and `pnpm format:check` |
| Lint                | `make lint`      | `go vet ./...` and `pnpm lint`                                        |
| Test API            | `make test`      | `go test -race -count=1 ./...`                                        |
| Type check web      | part of `check`  | `pnpm typecheck`                                                      |
| Build               | `make build`     | `go build -trimpath -o bin/schewise-api ./cmd/api` and `pnpm build`   |
| Integration check   | `make smoke`     | `bash scripts/smoke.sh`                                               |
| Dependency audit    | n/a              | `pnpm audit --audit-level high`                                       |

`pnpm typecheck` generates Next.js route types before running TypeScript, so it
also works in a fresh checkout before a production build. The race detector
needs CGO and a C compiler (Xcode command-line tools on macOS; GCC/Clang on
Linux).

## Configuration

The API uses process environment variables and does not read dotenv files:

```sh
SCHEWISE_HTTP_ADDR=127.0.0.1:8081 go run ./cmd/api
```

Next.js reads `apps/web/.env.local` automatically:

```sh
cp apps/web/.env.example apps/web/.env.local
# Set API_BASE_URL=http://127.0.0.1:8081 in that file if changing the API port.
pnpm dev:web
```

Restart Next.js after changing configuration. Rewrites are resolved at dev
startup or production build time, so production changes require `pnpm build`
again. Never put database credentials or AI provider keys in `NEXT_PUBLIC_*`
variables.

## Production-build smoke check

This tests the local production build; it does not deploy anything:

```sh
# Stop any running development services first.
make check
make build
make smoke
```

Use the default `API_BASE_URL=http://127.0.0.1:8080` for the web build. Remove
local overrides from the shell and `.env.local` before this check. The script
starts the API and Next.js on loopback ports 8080/3000, checks the page and both
direct/proxied health responses, verifies unsupported mutations return 405, and
stops its child processes. It needs Bash, curl, Node.js, and the build output. A
successful smoke check verifies server integration; it is not a browser
interaction test.

To run built services manually in separate terminals:

```sh
./bin/schewise-api
pnpm --filter @schewise/web start
```

## Docker development

```sh
docker compose config --quiet
docker compose up --build
```

The API listens on `0.0.0.0:8080` inside its container. Next.js forwards to
`http://api:8080` through the Compose network. Published host ports bind to
`127.0.0.1`. The API health check gates web startup. No database service is
started.

The API mounts the repository and caches Go modules and build output in named
volumes. At startup it compiles and then executes the API binary directly, so
container stop signals reach the server for graceful shutdown. It does not use a
file watcher: run `docker compose restart api` after Go edits. The development
build omits embedded VCS metadata so differences between the host checkout's
ownership and the container user do not prevent startup. The web app mounts
`apps/web/src` for hot reload; other files are copied into the image, so rebuild
after changing dependencies, Next.js configuration, or other files outside
`src`. Dependencies are installed inside the Linux image, not copied from host
`node_modules`.

```sh
docker compose logs -f
docker compose down
```

These images are development-only and are not hardened release artifacts. CD,
registries, TLS, and hosting are outside this milestone.

## CI and GitHub repository settings

The `CI` workflow checks:

- **Go:** formatting, vet, race tests, and binary build.
- **Web and integration:** frozen dependency installation, repository
  formatting, ESLint, generated route types/TypeScript, dependency audit,
  production build, and local API/web smoke check.
- **Development containers:** Compose validation, both image builds, startup,
  API/web requests through the container network, and clean API shutdown.

Workflow permissions are read-only. Actions use full commit hashes; Dependabot
proposes updates for Actions, npm, Go modules, and Docker base images.
JavaScript dependencies are locked; third-party Go dependencies should add a
committed `go.sum` when introduced. Dependabot does not replace periodic Go/Node
toolchain and vulnerability review.

The following require GitHub repository settings and are not enabled by files in
this scaffold:

1. Enable private vulnerability reporting and verify the link in `SECURITY.md`.
2. Enable available dependency graph, Dependabot alerts, and secret scanning
   features.
3. After the first successful CI run, add a `main` ruleset requiring PRs and the
   `Go`, `Web and integration`, and `Development containers` checks. Decide
   review and bypass requirements to fit the number of active maintainers.
4. Optionally require code-owner review; `CODEOWNERS` alone only requests
   reviews.

## Dependency compatibility and initial audit

As of 2026-11-04, `pnpm audit` reports one moderate advisory in transitive
development dependencies:
[GHSA-g3hv-75wg-m249](https://github.com/advisories/GHSA-g3hv-75wg-m249) in
`minimatch`, brought in by the ESLint toolchain. The installed graph has no high
or critical advisories in that audit. CI blocks high and critical findings and
continues to display lower-severity findings.

ESLint is pinned to 9.39.5 because the current Next.js lint plugins' peer ranges
do not yet support ESLint 10. Upgrading minimatch across major versions with a
blanket override can break consumers using its older API. Follow upstream lint
dependency updates rather than suppressing the advisory or forcing that
override.

## Troubleshooting

- **Connection check fails:** confirm `curl http://127.0.0.1:8080/api/v1/health`
  works, then test the same path on port 3000. Check `API_BASE_URL` and restart
  or rebuild Next.js if it changed.
- **Port already in use:** stop the existing service or configure another
  API/web port for development. Smoke checks require the defaults.
- **Go download fails:** ensure `proxy.golang.org` and `sum.golang.org` are
  reachable, or install the preferred Go version manually.
- **pnpm lockfile differs:** use the version in `packageManager`; run
  `pnpm install` intentionally and commit the resulting lockfile change.
- **Docker cannot connect:** start Docker Engine/Desktop and confirm
  `docker info` succeeds before building or running Compose.
