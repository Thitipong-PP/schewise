# ADR 0001: local-first Go and Next.js monorepo

- Status: accepted
- Date: 2026-11-04

## Context

Schewise needs an open-source foundation for future database history,
environment management, mock data, schema mapping, harmonization, and
AI-assisted workflows. The first milestone is runnable local services and
contributor tooling, with no deployment target or database engine selected.

## Decision

Use one root Go module with a standard-library HTTP API under `cmd/api` and
`internal/`. Put a Next.js App Router application with strict TypeScript in
`apps/web`, managed by a pnpm workspace. Proxy browser API requests through
Next.js to Go. Use Make as a convenience interface with documented direct
commands.

Provide optional development containers, pinned CI actions, a dependency
lockfile, automated dependency updates, community templates, and repository-wide
agent instructions. Keep all repository content in English and preserve
Apache-2.0.

Defer domain implementations, database drivers, application persistence,
authentication, AI providers, release automation, and CD.

## Consequences

API and UI changes share a review and integration check. The foundation remains
small and can grow around real use cases. Go owns backend behavior while Next.js
owns presentation. Contributors need Go, Node.js, and pnpm, or Docker Compose.

The initial API is unauthenticated and intended for localhost. Production
operations, access control, backup guarantees, and deployment design remain
future work. The Git analogy does not imply automatic or lossless database
rollback.
