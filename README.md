# Schewise

Open-source, AI-assisted database management for software engineers.

Schewise aims to bring version-control workflows to databases: understand
schemas, review changes, track history, manage environments, generate mock data,
and map and harmonize data. The long-term vision is a collaborative home for
database projects.

**Status: foundation only.** The Go API, Next.js starter, local development
setup, and CI are available. Database connections, versioning, rollback, mock
databases, schema mapping, harmonization, authentication, and AI integrations
are not implemented yet. See the [roadmap](docs/roadmap.md).

## Quick start

Install:

- [Go](https://go.dev/dl/) 1.27.3 or newer in the 1.27 release line. `go.mod`
  records the minimum language version and preferred toolchain.
- [Node.js](https://nodejs.org/) 26 LTS (`nvm use` reads `.nvmrc`).
- pnpm 12.16.1: `npm install --global pnpm@12.16.1`.
- Make for convenience commands; direct commands are listed below. On Windows,
  WSL2 is recommended for the Make and Bash workflows.

From the repository root:

```sh
make setup
```

Start the API in one terminal:

```sh
make dev-api
# Or: go run ./cmd/api
```

Start the web app in another terminal:

```sh
make dev-web
# Or: pnpm dev:web
```

Open [http://localhost:3000](http://localhost:3000) and select **Check API
connection**. Both processes bind to loopback by default. No database, account,
external service, or API key is required.

```sh
curl http://127.0.0.1:8080/api/v1/health
# {"status":"ok","service":"schewise-api"}

# The same request through Next.js:
curl http://127.0.0.1:3000/api/v1/health
```

The endpoint checks process liveness, not database readiness.

## Local development with Docker

With Docker Engine/Desktop and Compose running:

```sh
docker compose up --build
# Stop with Ctrl+C, then:
docker compose down
```

The same localhost URLs apply. Web source edits reload automatically. Restart
the API service after Go edits: `docker compose restart api`. Rebuild after
dependency or configuration changes. These are development images; no deployment
or image publishing is configured. Details are in the
[development guide](docs/development.md).

## Repository layout

```text
.
├── cmd/api/          # Go executable and development Dockerfile
├── internal/httpapi/ # Go HTTP routes and contract tests
├── apps/web/         # Next.js App Router + TypeScript
├── docs/             # Architecture, development, roadmap, and decisions
├── scripts/          # Integration smoke check
└── .github/          # CI, dependency updates, and contribution templates
```

One Go module and one pnpm workspace keep API and web changes in a single
review. The Go API owns backend behavior; Next.js owns presentation and proxies
`/api/v1/*` to Go. Read the [architecture](docs/architecture.md) for the
rationale.

## Configuration

Defaults work without creating an environment file.

| Variable             | Consumer | Default                 | Purpose                                  |
| -------------------- | -------- | ----------------------- | ---------------------------------------- |
| `SCHEWISE_HTTP_ADDR` | Go API   | `127.0.0.1:8080`        | HTTP listen address                      |
| `API_BASE_URL`       | Next.js  | `http://127.0.0.1:8080` | Server-side destination for API rewrites |

The API reads process environment variables, for example
`SCHEWISE_HTTP_ADDR=127.0.0.1:8081 go run ./cmd/api`. It does not load `.env`
files. For Next.js overrides, copy `apps/web/.env.example` to
`apps/web/.env.local`. `API_BASE_URL` is read at dev startup or production build
time; restart or rebuild after changing it. It is not a `NEXT_PUBLIC_*` browser
variable.

## Checks

```sh
make check       # Go formatting/vet/race tests; Prettier; ESLint; TypeScript
make build       # Build bin/schewise-api and the production web app
make smoke       # Start built services and check API, page, and proxy
```

Stop development servers before `make smoke`; it uses ports 8080 and 3000. Build
with the default `API_BASE_URL` for this check. Use `make help` for all
commands. CI runs on pull requests and pushes to `main`, with read-only
repository permissions. It also audits JavaScript dependencies and checks Docker
image builds, startup, API/web integration, and API shutdown. There is no CD
pipeline yet.

## Contributing

All repository content and collaboration should be in English. Read
[CONTRIBUTING.md](CONTRIBUTING.md), the [Code of Conduct](CODE_OF_CONDUCT.md),
and [AGENTS.md](AGENTS.md). Use the issue templates for bug reports and feature
proposals. Follow [SECURITY.md](SECURITY.md) for sensitive reports.

Licensed under [Apache-2.0](LICENSE).
