<!-- SPDX-License-Identifier: Apache-2.0 -->

# Scoped authorization (Step 2A)

This milestone implements application metadata and authorization in Go. It does
not expose role, unit, assignment, or deactivation HTTP endpoints or management
UI. Existing authentication endpoints are integrated with the engine.

## Units, roles, and assignments

One permanent `Organization` root is created by the migration. Root depth is
one; maximum depth is six. Units carry a parent, name, description, free-form
kind label, and ordered ancestor UUIDs. Create/move operations maintain the
paths; database constraints also reject inconsistent paths and a second root.

Roles contain a flat permission set. Four system roles cannot be edited or
deleted, including their permissions, but can be cloned into custom roles:

| Role          | Permissions                                      |
| ------------- | ------------------------------------------------ |
| Administrator | All catalog permissions                          |
| Manager       | All non-system permissions                       |
| Developer     | Manager permissions except `unit:manage_members` |
| Viewer        | `unit:read`, `connection:read`, `schema:read`    |

The catalog is defined in `internal/rbac/model.go`: `user:manage`,
`role:manage`, `audit:read`, `unit:read`, `unit:manage`, `unit:manage_members`,
`connection:read`, `connection:manage`, `schema:read`, `docs:edit`,
`tag:manage`, `ai:run`, `data:preview`, and `data:query`. These names reserve
permissions for future features; they do not implement connectors, queries, or
AI functionality.

An assignment links a user and role to a unit or connection, with an optional
note, creator, and timestamp. Membership is derived from assignments; there is
no membership table, expiry, deny policy, group, or role composition. Duplicate
user/role/scope assignments are rejected.

## Authorization boundary

`Engine.Can(ctx, userID, permission, resource)` is the authorization gate.
Resources contain only a scope type and UUID, never a caller-supplied parent
path. The engine resolves the owning unit and its ancestors from trusted
metadata. A unit assignment grants permissions at that unit and below it;
connection assignments apply only to that connection. Nothing flows upward or
sideways. Multiple matching assignments combine by union, so adding a narrower
Viewer cannot remove an inherited Developer grant. No matching grant means deny.
Unknown permissions, missing/inactive users, invalid resources, cancellation,
and store/resolver failures never grant access. Errors remain distinct from
ordinary denials for callers to handle. SQL-backed decisions read identity,
ancestry, and grants from one consistent snapshot so a concurrent move and
assignment edit cannot produce permissions that never coexisted.

`Explain` uses the same evaluation and returns every granting assignment's ID,
role ID/name, scope, and inheritance marker, or an explicit denial reason with
an empty grant list. `user:manage`, `role:manage`, and `audit:read` count only
from assignments at the root. A role containing any such permission can only be
assigned at that root.

`CanAssign` requires `unit:manage_members` at the target scope and every
permission in the proposed role at that scope. Revocation uses the same rule.
Managers can delegate Developer or Viewer inside their subtree, but cannot
change assignments at an ancestor or grant Administrator. Developers cannot
delegate. Role editing requires root `role:manage` and all proposed permissions
at root, preventing an editor from escalating an already assigned role. Adding
system-level permissions to a role already assigned below root is rejected.

## Repositories and invariants

`Repository` supplies unit create/update/move/delete/chain reads, role
create/update/delete/clone/reads, assignment grant/revoke/reads, active-user
changes, audit reads, and move/delete impact previews. Mutations check live
permissions inside a database transaction, serialize with `rbac.Lock`, and write
`audit_log` before commit. Failed audit writes roll back metadata changes.
Committed operations and failed mutation attempts also produce redacted Zap
logs; metadata notes, credentials, and raw driver errors are not logged.

The last **active** built-in Administrator assignment at root cannot be revoked,
and its user cannot be deactivated. An inactive Administrator or a custom clone
does not satisfy this invariant. Database triggers reinforce the root, path,
system-role, scope, and last-admin invariants. A database owner can still bypass
controls or alter audit history; this is not tamper-proof storage.

Moving a unit requires `unit:manage` on it, its current parent, and its proposed
parent. This prevents a subtree manager from moving their own scope out of its
boundary or importing unmanaged units. Moves reject self/descendant parents and
any resulting depth above six, then rebuild the entire subtree's ancestor paths.
Delete is non-cascading: a unit must have no child units, direct assignments, or
owned connections. Inherited access alone does not make a unit nonempty. Root
move/delete is always forbidden.

## Connection ownership extension

The code scope registry accepts only `unit` and `connection`.
`ConnectionResolver` resolves a connection to exactly one owning unit and lists
connections in specified units using the repository's query/transaction handle.
No connection table, foreign key, adapter implementation, wire protocol, or real
database connection is introduced here. The default engine fails closed for
connection authorization and assignment until an ownership adapter exists. Tests
use a synthetic adapter for connection inheritance and impact reporting. Future
connection ownership mutations must join the shared lock protocol and preserve
exactly one owning unit.

## Impact previews

`DryRunMove` reports per-user, per-resource permission gains/losses over the
moved units and their connections. Direct grants that remain effective are not
reported as losses. `DryRunDelete` reports blockers and affected connections; a
blocked deletion has no executable access changes. An empty-unit deletion
reports loss of access to the removed resource, including inherited access.
Neither preview changes metadata or writes a mutation audit record.

Previews run against read-only repeatable-read snapshots. They are descriptive
reports, not approval tokens or promises about future state; mutations always
recheck live permissions and constraints under the write lock.

## Migration and recovery

Migration 00002 upgrades the actual Step 1 schema, which used `users.role_id`
rather than `users.is_admin`. Every old role assignment becomes a root
assignment. Administrator retains all permissions; unknown legacy roles become
custom roles with no inferred permissions. The old role column is dropped,
`is_active` defaults to true, and existing password hashes and sessions are
preserved. Authentication audit records move to `audit_log`, retaining IDs,
timestamps, and request/outcome/IP metadata. The seeded roles use stable UUIDs.
A conflicting legacy Manager/Developer/Viewer name fails rather than silently
promoting that custom role.

The migration is forward-only: multiple scopes, roles, and deactivation cannot
be represented safely by the old schema. Back up application metadata before
upgrading an existing installation. Do not restart an old binary against the
upgraded schema; recovery requires a reviewed backup/forward migration plan.
Normal API startup applies migrations. Tests use isolated schemas and do not
upgrade the developer's application schema implicitly.

## Deferred to Step 2B and later

Unit/role/assignment/deactivation HTTP APIs and UI are deferred. Groups,
permission expiry, deny rules, composite roles, inheritance flags, and
schema/table/column-level permissions are deliberately absent.
