<!-- SPDX-License-Identifier: Apache-2.0 -->

# Scoped authorization

Steps 2A and 2B implement application metadata, authorization in Go, and
permission-gated administration APIs and UI. Authentication uses the same
engine. Later steps add invitations, account settings, deactivation, and
PostgreSQL connections using the same engine. Data and AI features remain
planned.

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
permissions for planned capabilities as well as implemented connection/schema
access. Data queries, tagging, and AI functionality remain unimplemented.

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
sideways. Walk upward from the resource's owner, include assignments at each
unit, and stop after the first `inherit_access=false` unit. Include a direct
connection assignment too. A built-in root Administrator assignment always
applies, even across boundaries; a custom all-permission clone does not bypass.
Multiple applicable assignments combine by union, so a narrower Viewer cannot
remove a Developer grant within the remaining chain. No matching grant means
deny. Unknown permissions, missing/inactive users, invalid resources,
cancellation, and store/resolver failures never grant access. Errors remain
distinct from ordinary denials for callers to handle. SQL-backed decisions read
identity, ancestry, and grants from one consistent snapshot so a concurrent move
and assignment edit cannot produce permissions that never coexisted.

`Explain` uses the same evaluation and returns every granting assignment's ID,
role ID/name, scope, and inheritance marker, or an explicit denial reason with
an empty grant list. `user:manage`, `role:manage`, and `audit:read` count only
from assignments at the root. A role containing any such permission can only be
assigned at that root.

`CanAssign` requires `unit:manage_members` at the target scope and every
permission in the proposed role at that scope. `CanRevoke` adds the
own-unit-subtree restriction and protection for the last active root
Administrator. Users meeting the same management/permission checks may revoke
their own assignments; Developers cannot. The store, HTTP revoke path, and
read-only UI projections use this single gate. Managers can delegate Developer
or Viewer inside their subtree, but cannot change assignments at an ancestor or
grant Administrator. Developers cannot delegate. Role editing requires root
`role:manage` and all proposed permissions at root, preventing an editor from
escalating an already assigned role. Adding system-level permissions to a role
already assigned below root is rejected.

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

## Inheritance boundaries

`SetInheritAccess` requires `unit:manage_members` at the unit. The root rejects
all changes and a database constraint keeps its flag true. Other units default
to true. A change is rejected if the actor would lose membership-management
permission at that unit, unless they hold a built-in root Administrator
assignment. Writes recheck under the existing shared lock and commit their
`unit.set_inherit_access` audit record atomically. Moves retain the flag.

`DryRunInheritAccess` covers the unit subtree and owned connections using the
same projected-reader mechanism as move previews. Reports include user email,
before/after roles and permissions, gained/lost permissions, connections, and
self-lockout blockers. Previews do not mutate units or audit history. A blocked
inheritance proposal still explains its potential impact; blocked deletion
continues to have no executable changes.

`Explain` reports matching assignments that were cut off, the blocking unit, and
`blocked_by_inherit_access` when that explains denial. Direct grants and root
Administrator bypasses can still allow the permission while other grants are
blocked. Membership/access lists include applicable assignments only. Tree/list
visibility uses the same updated `Can` rules.

Migration `00003_unit_inherit_access.sql` adds only the flag and root
constraint, leaving existing units inheriting and all prior rows intact. Its
downgrade is refused because dropping boundaries would silently broaden access.

## Connection ownership extension

The code scope registry accepts only `unit` and `connection`.
`ConnectionResolver` resolves a connection to exactly one owning unit and lists
connections in specified units using the repository's query/transaction handle.
Step 5 supplies the default SQL resolver and a real connection table. A
connection must exist before it can receive a scoped assignment; a generated FK
validates that invariant and cascades only its direct assignments on deletion.
Unit deletion is blocked while it owns connections; move/inheritance previews
include them. Profile ownership is immutable through editing and follows unit
moves. All metadata writes join the shared lock protocol. See
[connections](connections.md) for connection permissions and target safety.

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

## Administration API and UI (Step 2B)

All administration routes require a live session. Mutations also require the
existing exact HTTPS Origin and synchronizer CSRF token. Every endpoint uses
`Can`, `CanAssign`, or `CanRevoke`, directly or through authorized read
projections; the UI never authorizes by role name. Unit operations first check
`unit:read`, so unreadable and nonexistent units return the same 403 error
envelope. Responses use `{ "error": "Readable reason", "code": "stable_code" }`
for failures.

| Area               | Routes beneath `/api/v1`                                                | Authorization                                                                                           |
| ------------------ | ----------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| Capabilities       | `GET /access/context`                                                   | Effective permissions from `Can`                                                                        |
| Units              | `GET /units`, `/units/tree`, `/units/{id}`                              | `unit:read`, filtered forest                                                                            |
| Unit edits         | `POST /units`, `PATCH/DELETE /units/{id}`                               | Read plus `unit:manage`; root/delete constraints                                                        |
| Moves              | `POST /units/{id}/move`                                                 | Read source/destination; manage source, current parent, destination                                     |
| Impact             | `POST /units/{id}/dry-run/move`, `/dry-run/delete`                      | Same authorization as corresponding mutation                                                            |
| Roles              | `GET/POST /roles`, `PATCH/DELETE /roles/{id}`, `POST /roles/{id}/clone` | Root `role:manage`; writes must hold all proposed permissions                                           |
| Catalog            | `GET /permissions`                                                      | Readable units or root `role:manage`                                                                    |
| Delegation choices | `GET /units/{id}/assignable-roles`                                      | Unit read/manage-members; filtered by `CanAssign`                                                       |
| Membership         | `GET /units/{id}/members`, `/assignments`, `/access`                    | `unit:read`; assignments are direct only; access excludes inactive/no-access users                      |
| User directory     | `GET /users?unit_id=&q=`                                                | Root `user:manage` or read/manage-members at supplied unit; active IDs/emails only, at most 200 matches |
| User access        | `GET /users/{id}/assignments`, `/access`                                | Self, root `user:manage`, or manageable readable units; filtered to inspectable units                   |
| Assignment edits   | `POST /assignments`, `DELETE /assignments/{id}`                         | Readable origin plus `CanAssign` for grants / `CanRevoke` for revocations; optional note on create      |
| Explain            | `POST /access/explain`                                                  | Read target unit plus self, root `user:manage`, or unit manage-members                                  |
| Audit              | `GET /audit`                                                            | Root `audit:read`; actor/action/from/to filters and stable keyset pagination                            |

`PATCH /units/{id}/inherit-access` calls `SetInheritAccess` with a required,
non-null `inherit_access` boolean. `POST /units/{id}/dry-run/inherit-access`
returns the impact before confirmation. Both require a readable unit and
`unit:manage_members`. The revoke route calls `CanRevoke` through the authorized
assignment lookup and again under the store's write lock.

Step 5 enables real connection scopes for assignments and Explain. The OpenAPI
document specifies request/response schemas and query parameters. PATCH requests
replace editable metadata fields; the UI submits the complete form. Unit delete
never cascades into child units, assignments, or connections. Confirmed
connection delete has its own narrowly scoped cascade described in
[connections](connections.md).

Organization is the protected `/` page. `/roles` manages custom roles with a
permission catalog grouped by area and distinct-user/assignment counts.
`/units/{id}` lists direct and inherited members, scoped role assignment with an
optional note, and revocation. `/explain` supports optional filters. Resource
alone lists active users with effective access; adding a permission filters that
server-evaluated list. User alone lists accessible units within the actor's
inspectable scope. Resource plus User lists that user's effective permissions
and applicable assignments at the unit. Adding Permission requests the existing
Explain result. Permission is disabled until a resource is selected, and at
least Resource or User is required. Clearing Resource also clears Permission.
Reset clears all filters.

Without a selected resource, root user managers use the existing global
directory; subtree managers initialize the existing directory with a readable
manageable unit. Other readers can select themselves. This changes no API
authorization or directory visibility. Results remain filtered by Go and all
origin/blocker redaction rules still apply. `/audit` supports filters and
pagination. Menus and controls use API-evaluated permissions and capability
flags; direct HTTP requests are independently enforced.

### Inherited-source privacy

Read projections evaluate origin visibility **per assignment**. A caller who can
read Payments but cannot read Engineering sees Bob's Engineering grant as
`source: "inherited_hidden"`, with the role and granted permissions. The origin
unit name/ID, scope, assignment ID, note, and timestamp are omitted. The UI says
"Inherited from a higher-level unit you don't have access to" and provides no
revoke action. An Administrator with origin read access sees the full source.
Explain uses the same projection. A blocking unit that the requester cannot read
is `source: "blocked_hidden"`, with no unit name/ID; its assignment is limited
to the role and granted permissions. A readable blocker is `blocked` with its
unit name/ID. These decisions are independent for each assignment and blocker. A
readable origin outside the caller's delegation authority includes a read-only
explanation. Visible forests also omit inaccessible parent IDs. Audit contents
require the independent global `audit:read` permission and are not a scoped
membership projection.

### Generated contracts and verification

`make api-generate` derives `docs/openapi.json` from real Go DTOs/routes, then
regenerates `apps/web/src/lib/api-types.ts`. `make api-check` rejects stale
outputs; it runs in `make check` and CI. Handlers validate operation DTO types
against the manifest and strictly reject unknown JSON fields. No API types are
copied by hand into the administration client.

HTTP integration tests use isolated PostgreSQL schemas for 401/403, hidden vs
missing resources, per-assignment redaction, Explain, escalation, immutable
roles, role deletion while assigned, last-admin protection, unit dry-run/move/
delete, and audit filtering. Frontend Node tests cover source presentation,
forest ordering, move choices, and client failure/CSRF request behavior.

## Deferred capabilities

Invitations, account settings, and deactivation APIs/UI remain deferred. Groups,
permission expiry, deny rules, composite roles, and schema/table/column-level
permissions are deliberately absent.

## Invitations

Root `user:manage` gates listing, creation, revocation, and regeneration.
`CanAssign` gates the role/scope on creation and regeneration, and live issuer
permissions are rechecked again during acceptance under the shared write lock.
Invitation links never grant the issuer authority they do not have. Units and
roles can be deleted while an invite is pending; acceptance then returns the
generic unavailable response without consuming the invite. Invite records do not
count as active role assignments. The Invites UI offers only server-approved
assignable roles, while the API independently enforces each operation.

## Account administration and self access

The `/admin/users` family always requires live root `Can(user:manage)`,
including search and reset-link creation. Deactivation/reactivation delegates to
the existing `SetUserActive` mutation, keeping shared locking and the
last-active root-Administrator guard. Reset issuance grants no role/assignment
and never accepts a replacement password from an administrator.

Self-service routes require a live session owned by the current account and do
not accept a target user ID. They work without unit permissions. `MyAccess`
reuses the existing `effective`, `inspect`, and `assignmentView` projections in
a single snapshot. Only readable units are returned; inherited sources retain
the same per-assignment redaction and all rows are read-only. An account with no
readable units receives an empty list rather than losing access to settings.
