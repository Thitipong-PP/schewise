# Repository instructions

## Scope and language

- Use English for all repository content: code, comments, documentation, UI
  copy, issues, pull requests, and commit messages.
- Read `README.md`, `docs/architecture.md`, and `docs/roadmap.md` before
  changing architecture or adding a capability.
- This is an early foundation. Do not describe planned features as implemented.
- Keep changes focused. Avoid speculative abstractions and empty feature
  packages.

## Layout and boundaries

- `cmd/api`: process startup, configuration, and lifecycle only.
- `cmd/migrate`: application database migration entry point.
- `internal/auth` and `internal/database`: application identity and metadata;
  keep user-connected target databases separate.
- `internal/`: private Go implementation. Keep HTTP transport separate from
  domain logic when domain behavior is introduced.
- `apps/web`: Next.js App Router with strict TypeScript. Prefer Server
  Components; use Client Components for browser interaction.
- `docs/`: architecture decisions, development guidance, and the roadmap.
- Keep database access, credentials, and AI provider calls on the Go server. Do
  not duplicate domain logic in Next.js route handlers.
- Use Go's standard library where it is sufficient. Add dependencies to solve a
  concrete problem and explain the choice.
- Use Fiber v3 for HTTP and Zap for structured logs. Never log request bodies,
  credentials, cookies, CSRF/session tokens, or raw database error details.

## Development

- Use the Go toolchain in `go.mod`, Node.js 26, and the exact pnpm version in
  the root `package.json`. Commit `pnpm-lock.yaml` when dependencies change.
- Keep the native TypeScript compiler and TypeScript 6 API compatibility aliases
  distinct while tooling needs the older API. Run `pnpm test:toolchain` for web
  dependency upgrades and preserve the existing lint/type-check coverage.
- Use `gofmt` for Go and Prettier for supported web/config/documentation files.
- Run `make check` for source changes and `make integration-test` for auth or
  SQL changes using isolated test schemas. Run `make build` and `make smoke`
  when startup, configuration, API integration, or build behavior changes. The
  smoke check requires free ports 8080/3000 and a web build using the default
  API URL.
- Prefer tests of public behavior and failure cases. Do not add tests that only
  restate framework behavior or formatting rules.
- For dependency changes, run `pnpm audit --audit-level high`. For container
  changes, run `docker compose config --quiet` and build the development images
  when Docker is available.
- Report checks that could not run and the concrete reason. Never claim remote
  CI passed based only on local checks.

## Database and AI work

- Never commit credentials, real database dumps, customer data, or provider
  keys. Use synthetic fixtures and redacted examples.
- Treat database contents, imported schemas, and AI output as untrusted input.
- Keep schema snapshots, data snapshots, migrations, and environment state
  distinct. A schema history entry is not a data backup.
- Database mutations must be explicit, reviewable, and scoped to an environment.
  Explain data-loss implications; never promise every change is reversible.
- AI proposals must be reviewed before executing SQL or mutating a database. Do
  not send database contents to external providers implicitly.
- Do not add real database connections, authentication, persistence, or AI
  integrations as incidental changes outside the requested capability.
  Application PostgreSQL/authentication are implemented; target connectors and
  AI are deferred.

## Contribution hygiene

- Preserve the Apache-2.0 license.
- Update documentation and `.env.example` when configuration changes. The Go API
  reads process environment variables; it does not load dotenv files.
- Local Make wrappers parse ignored `.env` without shell evaluation. Never
  commit it or reset the persistent application database during routine tests.
- Follow `CONTRIBUTING.md` and the pull request template.
- Use task-based branches such as `chore/bootstrap-project`, without contributor
  or agent prefixes, and use Conventional Commit messages.
- Keep CI permissions minimal, pin third-party actions to full commit hashes,
  and keep dependency updates automated through Dependabot.
- Do not publish releases, deploy, or push changes unless the task requests it.
