<!-- SPDX-License-Identifier: Apache-2.0 -->

# PostgreSQL connections and schema snapshots

Step 5 adds PostgreSQL profiles and read-only catalog introspection. Profiles
belong to exactly one organization unit. The application's metadata database is
separate from targets. No table data preview, SQL execution UI, inferred
relationships, classification, AI requests, scheduled sync, or annotation
editing is implemented.

## Encryption and destination policy

Set `SCHEWISE_SECRET_KEY` to a base64-encoded, random 32-byte key before
creating profiles. Generate it once with `openssl rand -base64 32`, put it in
your ignored `.env` or secret manager, and preserve it with metadata backups.
`make init` generates one for new installations; it preserves existing `.env`
files. Add the setting manually to an older installation. The API reads process
environment, not dotenv files. Invalid keys fail initialization. An absent key
is allowed only while there are no profiles; creates fail until configured.
Existing ciphertexts are authenticated at startup, so a missing or wrong
original key fails before the API starts listening. Automatic key rotation is
not provided.

Passwords use AES-256-GCM with a fresh 12-byte nonce and connection UUID as
authenticated additional data. API responses, audit details, errors, snapshots,
and Zap events never include passwords or encrypted password fields. Password
replacement is a separate endpoint; forms never prefill it. If a catalog string
contains the exact nonempty connection password, syncing fails safely rather
than storing the secret or silently rewriting the schema facts. Do not put
credentials in database comments, names, or default expressions.

`SCHEWISE_CONNECTION_HOST_ALLOW` and `SCHEWISE_CONNECTION_HOST_DENY` accept
comma-separated exact hostnames, IP addresses, or CIDRs. Empty allow means allow
all; deny takes precedence. Every resolved IP is checked again at dial time;
Unix sockets, connection strings, and multiple hosts are not accepted in the
host field. Hostname allow entries allow their resolved addresses unless a deny
rule blocks them. TLS verification modes use the server process's system CA
store. `require`, `prefer`, and `allow` follow libpq's non-verifying semantics;
use `verify-full` with a trusted server certificate for production.

**SSRF:** allowing connection managers to choose destinations lets them contact
internal hosts and ports reachable by the Go server. Default allow-all is useful
for self-hosting, not an isolation boundary. Configure host policies and network
egress controls for your deployment. Metadata connection environment variables,
password files, client certificates, and runtime parameters are not inherited by
target profiles. No target content is sent to an external provider.

## Read-only operation

The pgx session sets `default_transaction_read_only=on`, a five-second connect
timeout, and `statement_timeout` (`SCHEWISE_STATEMENT_TIMEOUT`, default `30s`,
allowed `1s`–`5m`). Introspection uses a repeatable-read, read-only transaction.
The overall operation deadline is the statement timeout plus 15 seconds. Targets
have no persistent pool. A cross-process transaction advisory lock permits only
one test/sync/profile mutation/delete per connection; conflicts return 409. Slow
network work does not hold the global RBAC mutation lock. Authorization is
rechecked under that lock before committing metadata and its audit event.

The fixed queries read `pg_catalog` only. They exclude `information_schema` and
reserved `pg_*` schemas, and collect schemas; tables, partitioned tables, views
and materialized views; formatted column types and modifiers; identity and
generated expressions; declared keys/checks/FKs and actions; indexes including
expressions/included columns; and object comments. No user relation is scanned,
sampled, counted, or analyzed. Estimated row counts use `pg_class.reltuples`;
`-1` displays as "Not analyzed" and is never confused with zero. Limits are
10,000 relations, 100,000 columns, and 32 MiB of encoded structural content.

A least-privilege target account needs CONNECT and schema USAGE for catalog
inspection, without SELECT on application tables:

```sql
CREATE ROLE schewise_reader LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION;
-- Set a unique password securely, e.g. with psql's \password schewise_reader.
GRANT CONNECT ON DATABASE your_database TO schewise_reader;
GRANT USAGE ON SCHEMA public, your_schema TO schewise_reader;
ALTER ROLE schewise_reader SET default_transaction_read_only = on;
```

The server must allow that account through `pg_hba.conf`; TLS/server certificate
configuration belongs to the target installation. Role-level read-only defaults
are an additional guard, not a substitute for granting minimal privileges.
Permission, timeout, authentication, missing database, TLS, and network failures
return fixed friendly messages. Test/sync target failures return a 200 operation
result with `success: false` and a safe category; request/authorization/conflict
failures use the normal error envelope. Raw target errors are discarded.

## History and statistics

Migration `00006_postgres_connections.sql` adds profiles and snapshots plus a
generated connection FK on the existing polymorphic assignments table.
Connection assignments must point to real rows and cascade only when that
connection is deleted. Unit ownership is protected by a restrictive FK.

Canonical structural JSON is sorted by schema/object name, column ordinal,
constraint name, and index name. Array order within composite keys and indexes
is meaningful and preserved. SHA-256 covers structure and object comments only;
volatile counts and observation timestamps live in `stats`. Every successful
sync updates `last_synced_at`. If the latest structural hash is unchanged, only
that latest snapshot's stats are overwritten. Structural/comment changes create
a new snapshot. Earlier snapshots keep their last observed statistics.
`created_snapshot` distinguishes these outcomes. These are schema observations,
not data backups, diffs, migration plans, or rollback guarantees.

## Permissions, API, and deletion

All decisions call the existing `Can()`/`CanAssign()`/`CanRevoke()` engine. Unit
inheritance boundaries and the root Administrator bypass apply normally.
Create/update/delete/password replacement require `connection:manage` at the
owning **unit**. Ownership cannot be changed through profile editing; unit moves
retain their existing impact review. Test/sync and sensitive host details
require `connection:manage` at the connection. Metadata requires
`connection:read`; content and history require `schema:read` at the connection.
Connection-scoped grants apply only there. Unreadable owning units are omitted
from metadata responses. Missing/unreadable connections return the same 403 to
unauthorized actors. No UI decision grants API authority.

Routes under `/api/v1` (see [OpenAPI](openapi.json) for generated bodies/types):

- `GET/POST /connections`, `GET /units/{id}/connections`.
- `GET/PATCH/DELETE /connections/{id}`.
- `POST /connections/{id}/password`, `/test`, `/sync`, `/dry-run/delete`.
- `GET /connections/{id}/schema` for the latest snapshot.
- `GET /connections/{id}/snapshots` (keyset pagination, default 50, maximum
  200).
- `GET /connections/{id}/snapshots/{sid}` for a specific immutable structure.
- Existing `/assignments` and `/access/explain` accept real connection scopes.

Deleting a unit with connections is blocked. Move/inheritance previews enumerate
real owned connections. Connection deletion requires a dry-run review in the UI
and typing the exact name; the DELETE body is `{ "name": "connection name" }`.
The server recomputes impact, rejects an in-progress operation with 409, and
atomically removes the profile, password, snapshots, and direct connection
assignments. Unit assignments and other profiles are untouched. The impact lists
users whose direct connection grants will be removed; they may also have unit
grants. No SQL is sent to the target. The durable audit retains the actor,
connection UUID/name, snapshot count, and assignment count after deletion. Other
connection audits intentionally omit host/credential/catalog data.

## Synthetic demo and tests

After starting the application and configuring its encryption key:

```sh
make demo-up
```

This starts only `demo-db` on `127.0.0.1:5433` and registers/syncs "Commerce
demo" in a Demo unit through the domain services. It does not migrate
application metadata; initialize the app first. Repeated registration preserves
the existing profile. The Docker target has its own `demo-postgres-data` volume,
commerce schema, composite keys, declared FKs, checks, indexes,
views/materialized views, partitioning, comments, and a few synthetic records.
Its reader has no SELECT on user tables. Set `DEMO_POSTGRES_PASSWORD` and
`DEMO_DATABASE_PASSWORD` for unique local demo credentials (`make init` includes
them); fallback passwords are intentionally public synthetic fixtures, bound
only to localhost.

The default registered host is suitable for a native Go API. For the Compose
API, run `SCHEWISE_DEMO_HOST=demo-db SCHEWISE_DEMO_PORT=5432 make demo-up` so
the stored destination resolves inside the Compose network. The command needs
local Go and the configured installation user's current root permissions. Host
policy must permit the selected destination. Open Connections or the Demo unit
to view and sync it. Never point the demo at a real database.

`make integration-test` starts the synthetic target, runs all Go race tests,
creates isolated metadata schemas, and creates/drops uniquely named **test
databases on the demo server**. It never introspects the application's metadata
schema. A remote test setup can provide `SCHEWISE_TEST_DATABASE_URL` and
`SCHEWISE_TEST_TARGET_ADMIN_URL` (the latter requires CREATEDB; defaults to the
explicit test URL in CI). Cleanup drops only fixtures created by each test.
Normal `make check` skips SQL tests without explicit integration configuration.

Sources:
[PostgreSQL index catalog](https://www.postgresql.org/docs/18/catalog-pg-index.html),
[constraint catalog](https://www.postgresql.org/docs/18/catalog-pg-constraint.html),
and
[pgx connection configuration](https://pkg.go.dev/github.com/jackc/pgx/v5/pgconn).
