<!-- SPDX-License-Identifier: Apache-2.0 -->

# ADR 0006: read-only PostgreSQL catalog snapshots

- Status: Accepted
- Scope: Step 5

## Context

Schewise needs its first real target connector without mixing application
identity/authorization storage with customer databases. Declared database facts
must remain separate from future annotations, and changing row statistics must
not create structural history entries. Connection deletion must remove local
access/history deliberately while leaving the target untouched.

## Decision

Use the existing pgx dependency for fixed `pg_catalog` queries in read-only
transactions/sessions. Normalize structure/comments into JSON and hash them;
store volatile estimates separately and refresh the latest stats on every sync.
Keep encrypted passwords server-side with AES-256-GCM, per-record random nonces,
UUID-bound authentication data, and a base64 environment key checked at startup.

Retain the unit/connection scope registry and add a real SQL ownership resolver.
A generated FK validates only connection assignments and cascades them alongside
snapshots when the confirmed profile is deleted. Unit deletion stays
restrictive. Per-connection transaction advisory locks serialize remote work and
deletion across processes; the existing global RBAC lock serializes
authorization and metadata/audit commits after network I/O. Enforce destination
policy at resolution and dial time. Use generated OpenAPI/TypeScript contracts
for the new UI.

## Consequences

No additional runtime dependency is needed. Operators must preserve the original
encryption key and configure trusted destinations/egress controls for their
network. Snapshot history cannot recover table data. Password rotation, other
engines, annotations, inferred relationships, data preview, SQL execution, AI,
and scheduling remain separate work. See [connections](../connections.md) for
limits, failure categories, least-privilege setup, and synthetic tests.
