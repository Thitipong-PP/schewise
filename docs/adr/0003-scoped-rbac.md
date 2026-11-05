<!-- SPDX-License-Identifier: Apache-2.0 -->

# ADR 0003: hierarchical scoped authorization

- Status: accepted
- Date: 2026-11-05
- Updates: ADR 0002's single-role authorization and audit table

## Context

Step 2A needs organizational hierarchy, additive scoped permissions, safe
delegation, explainable decisions, durable audits, and last-admin protection.
Role-management HTTP APIs and UI belong to Step 2B. Target connections do not
exist yet, so their ownership must be an explicit extension boundary.

## Decision

Use `internal/rbac` with a fixed permission catalog and a single `Can` gate,
backed by persisted role/permission/assignment data. `Explain` shares the same
evaluation; `CanAssign` composes permission checks. Resolve ancestry from
application metadata, never browser claims. Combine grants by union and deny by
default. Limit system-level permissions and roles containing them to root
assignments.

Create one permanent root, a depth-six unit tree, four immutable system roles,
and optional custom roles. Assignment presence is membership. Use a scoped
connection ownership adapter without introducing a connection table or
connector.

Serialize RBAC and account-privilege writes with a transaction advisory lock
before permission/count reads. Use READ COMMITTED after that lock, not a stale
pre-lock snapshot. Commit mutations and `audit_log` records together; protect
root/system-role/path/last-admin invariants in the database too. Role edits
cannot introduce permissions the editor lacks or create invalid system-level
assignments. Deactivation deletes sessions and serializes with login through the
user row lock.

Use consistent read-only snapshots for impact previews and recheck every actual
mutation. Keep the existing auth route contract and display-only role label
while replacing all role-name authorization checks.

## Consequences

The migration preserves old users, password hashes, sessions, role assignments,
and audit history, but is forward-only because a single-role schema cannot
represent the new state. Metadata backups and a reviewed recovery plan are
required before upgrading an existing deployment. A global metadata write lock
favors correctness and simplicity at this foundation stage; finer-grained locks
can be considered only with equivalent concurrency tests.

The permission engine and repositories are usable internally, while management
endpoints/UI, groups, expiry, deny rules, composite roles, inheritance flags,
and finer database permissions remain out of scope. See the
[authorization guide](../authorization.md) for semantics and extension
contracts.
