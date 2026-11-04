# Contributing to Schewise

Thanks for helping build Schewise. The project is at the foundation stage; check
the [roadmap](docs/roadmap.md) before starting a new capability.

## Before you start

- Use English for code, documentation, UI copy, issues, pull requests, and
  commits.
- Follow the [Code of Conduct](CODE_OF_CONDUCT.md).
- For bugs, include reproducible steps, expected behavior, actual behavior, and
  relevant tool versions. Remove credentials and real database records from
  logs.
- Open a feature proposal before implementing a large architectural change or
  new dependency. Small fixes and documentation improvements can go directly to
  a PR.
- Report vulnerabilities privately as described in [SECURITY.md](SECURITY.md).

## Development workflow

1. Fork the repository and create a focused branch.
2. Follow the [README](README.md) to install dependencies and run both services.
3. Make a focused change. Add behavioral tests for new backend logic or fixes.
4. Run `make check`. For integration, startup, or build changes, also run
   `make build` and `make smoke` with ports 8080/3000 available and the default
   API URL. Run container checks when changing Docker files.
5. Update documentation and configuration examples if behavior changes. Commit
   the pnpm lockfile when dependencies change. A `go.sum` will be needed when
   external Go modules are introduced; the current API uses only the standard
   library.
6. Open a pull request using the template. Explain the problem, resulting
   behavior, validation, and any limitations.

Use `make fmt` to apply formatting. See [development](docs/development.md) for
direct commands and troubleshooting. No database, AI account, or deployment
access is required to contribute to the starter.

## Branches and commits

Use task-based branch names in the form `<type>/<short-kebab-case-description>`,
such as `feat/schema-history`, `fix/api-timeout`, or `chore/bootstrap-project`.
Do not add contributor names or agent-specific prefixes to branch names.

Use Conventional Commits with a descriptive English subject, for example
`feat: add schema history`, `fix: handle API connection failures`, or
`chore: bootstrap Go and Next.js project foundation`. Use a scope when it helps,
such as `fix(api): handle connection timeouts`. Keep each commit focused on one
coherent change. These conventions are documented but not enforced by
automation.

## Review expectations

Keep unrelated changes in separate pull requests. Prefer clear names, small
packages, and explicit behavior. Avoid introducing frameworks, services, or
abstractions for features that have not been designed yet. For architectural
changes, record the decision using the [ADR template](docs/adr/template.md).

Never include real credentials, database dumps, or customer data. Use synthetic
fixtures. AI-assisted contributions must be understood, verified, and reviewed
by their author like any other contribution.

By submitting a contribution, you agree that it is licensed under the
repository's Apache-2.0 license. Only contribute material you are entitled to
share under that license.
