<!-- SPDX-License-Identifier: Apache-2.0 -->

# ADR 0004: permission-gated administration and generated contracts

- Status: accepted
- Date: 2026-11-05
- Extends: ADR 0003 with the Step 2B HTTP and presentation boundary

## Context

The scoped engine needs usable unit, role, assignment, explanation, and audit
workflows. A manager can read a descendant without reading the ancestor that
granted another member access. HTTP responses must explain inheritance without
disclosing hidden ancestor identifiers or names. Frontend types must stay in
step with backend contracts.

## Decision

Keep authorization in Go. HTTP handlers and authorized read projections call
`Can` or `CanAssign`; writes also recheck inside the existing locked repository
transactions. Gate unit existence by `unit:read` before returning details or
mutation errors. Retain the session, Origin, and synchronizer-CSRF boundary.

Use read-only repeatable-read projections for capabilities and access views.
Cache reader results only inside that transaction. Project each assignment
independently: unreadable sources become `inherited_hidden`, with no source
unit, scope, assignment ID, note, or timestamp. Visible assignments outside
delegation authority are read-only. Apply the same projection to Explain.

Maintain a Go DTO/route manifest used by Fiber and a small OpenAPI 3.1
generator. Use `openapi-typescript` for generated TypeScript and `openapi-fetch`
for typed requests/responses, including mandatory CSRF headers. Check generated
artifacts in CI. Next.js presents server-evaluated capabilities and never
duplicates the permission engine. Use native dialogs for impact confirmation and
keep the existing monochrome visual conventions.

## Consequences

Managers receive useful explanations without ancestor disclosure. Read
capabilities are UI hints, not authority or durable approval tokens; writes
remain authoritative. Role lists/counts require global `role:manage`; other
users' access inspection requires global `user:manage` or scoped membership
management. Audit history requires global `audit:read`.

Contract changes require `make api-generate` and committed generated files. The
generator covers the administration API and the identity/CSRF reads it uses,
rather than claiming to document all legacy authentication routes. Connection
resources, invitations, account settings, and data features remain outside this
step. No additional database migration is required beyond 2A.
