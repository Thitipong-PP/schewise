# Roadmap

This document describes intent, not a release schedule or a promise of support.
The roadmap can evolve as use cases are clarified.

## Phase 0: open-source foundation (current)

- [x] Go API with health endpoint, timeouts, and graceful shutdown.
- [x] Next.js App Router starter with an API connection check.
- [x] Local commands and optional Docker Compose development.
- [x] Formatting, linting, type checking, tests, build and integration CI.
- [x] Contribution guides, issue/PR templates, agent guidance, and dependency
      updates.
- [x] Architecture and decision documentation.

## Authentication foundation (implemented before target connections)

- [x] Dedicated application Postgres with Goose migrations.
- [x] UUID users, Administrator-only registration, first-install env bootstrap.
- [x] Argon2id passwords, database sessions, revocation, expiry, and CSRF.
- [x] HTTPS login UI and protected Next.js pages.
- [x] Fiber HTTP, Zap request/lifecycle logs, and transactional audit events.
- [x] Real Postgres integration tests and HTTPS authentication smoke checks.
- [x] Step 2A: organizational unit tree, four system roles, scoped assignments,
      additive authorization, safe delegation, audit transactions, last-admin
      protection, and unit-impact previews in the Go repository layer.
- [x] Step 2B: permission-gated unit/role/assignment APIs and administration UI,
      inherited-source privacy, access explanations, filtered audit log, and
      generated frontend/backend contracts.
- [x] Unit inheritance boundaries, blocked-source explanations, centralized
      revocation rules, and searchable member/user pickers.
- [ ] Invitations, MFA, verification, and recovery.

## Phase 1: connections and schema understanding

- Select the first supported engine and define a connector boundary.
- Define connection profiles and environment metadata.
- Design secret storage for target database connection profiles.
- Introspect schemas and present tables, relations, constraints, and types.
- Define safe access modes and an initial synthetic database fixture.

## Phase 2: history and controlled changes

- Define a canonical schema snapshot format and version history.
- Compare snapshots and detect drift between environments.
- Review migration plans and record applied changes.
- Design environment-specific execution and auditing.
- Define recovery/rollback guarantees, backup requirements, and
  destructive-change handling per database engine. Schema history alone cannot
  recover deleted data.

## Phase 3: mock data and API development

- Generate deterministic, constraint-aware synthetic datasets.
- Expose mock database connections that backend applications can use to develop
  and test APIs.
- Define isolation, reset behavior, supported wire protocols, and fixture
  sharing.

## Phase 4: mapping and harmonization

- Map schemas and types across sources.
- Define transformation rules, validation, lineage, and reproducibility.
- Preview harmonized outputs before writing data.

## AI assistance (cross-cutting, not yet implemented)

- Explain schemas and propose mappings, mock-data rules, and migration plans.
- Keep provider choice and data sharing explicit.
- Evaluate generated output against deterministic validation and synthetic
  cases.
- Require review before generated SQL or transformations mutate databases.

## Long-term: collaboration for database projects

- Shared projects, review workflows, roles, permissions, and audit trails.
- Discoverable database artifacts, history, and reusable mappings/fixtures.
- Consider hosted operation and deployment only after the local workflow is
  solid.

## Open decisions

First target database engine; schema versus data snapshot scope; branch/merge
semantics; target credential storage; AI providers and privacy; mock protocol
compatibility; and future hosting. None are selected by the foundation scaffold.
