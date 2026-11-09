<!-- SPDX-License-Identifier: Apache-2.0 -->

# Step 5 implementation and validation

Implemented on `feat/postgres-schema-introspection`. This report covers only
PostgreSQL profiles and read-only schema introspection; the operational contract
is in [connections](connections.md). No dependency was added.

## Migration

`internal/database/migrations/00006_postgres_connections.sql` is the only new
migration. It adds `connections`, `schema_snapshots`, and the generated
connection-only FK on `assignments`. Previous migrations are unchanged. Unit
ownership is restrictive; confirmed connection deletion cascades its snapshots
and direct connection assignments inside one audited transaction. Downgrade is
intentionally refused to avoid silently destroying credential/history metadata.

## Validation

- `make check`: generated-contract checks, formatting, Go vet, ESLint, race
  tests, frontend toolchain/administration tests, and TypeScript pass.
- The full SQL suite (`scripts/connection-integration.mjs`, the runner used by
  `make integration-test`) passes with the race detector. Metadata fixtures use
  isolated schemas; target fixtures use uniquely named databases on the separate
  synthetic demo server. No introspection or migration was run against the main
  application's schema during this validation.
- Additional catalog assertions and a real pending network sync verify that
  structural facts match the fixture and an active sync blocks deletion.
- `make build` passes with the default API URL. A second web build using
  `API_BASE_URL=http://127.0.0.1:8082` supports isolated preview/smoke checks.
- `make smoke` passes on isolated ports 8082/3002, including secure cookies,
  CSRF, login/logout, protected redirects, and Next-to-Go proxying. Default
  ports 8080/3000 were occupied by the user's existing stack and left running.
- `docker compose config --quiet` and development image builds pass. The demo
  service starts healthy; `make demo-up` registers and syncs through domain
  services in a temporary metadata schema, using its catalog-only reader.
- Browser checks cover desktop and 320px width, form focus, no document-wide
  overflow, schema/column search, declared FK navigation, sync/stat refresh,
  deletion preview/name confirmation, Viewer redaction, and connection-only
  access/home navigation. No browser deletion or credential replacement was
  performed; automated API/store tests cover those mutations.
- Remote CI was not run or assumed to pass.

Tests cover encryption/authentication of ciphertext and wrong/missing keys,
credential/environment isolation, sanitized target error categories, secret
catalog rejection, hash ordering, statistics-only refresh without a new
snapshot, structural/constraint/comment history, composite keys/FKs, object
kinds/comments/indexes, read-only enforcement, snapshot pagination, permission
redaction, inheritance, real ownership/deletion blockers and move impacts,
confirmed cascade scope, busy-state privacy, and rollback when auditing fails.

## Existing test changes

| File                                         | Change and reason                                                                                                                                                                                                       |
| -------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `internal/rbac/store_integration_test.go`    | The connection delegation fixture now inserts a real metadata row for the new FK. The default-resolver test expects a missing-row denial instead of an unavailable extension, because SQL ownership is now implemented. |
| `internal/httpapi/auth_integration_test.go`  | The shared HTTP fixture initializes the connection service with a synthetic encryption key so new endpoints participate in existing authorization tests.                                                                |
| `internal/httpapi/admin_integration_test.go` | The generic endpoint authorization matrix substitutes the new snapshot UUID path parameter. Its 401/403 expectations are preserved.                                                                                     |
| `apps/web/scripts/admin.test.mjs`            | Adds schema/column search, exact FK destination, and unknown-versus-zero estimate cases; existing expectations are unchanged.                                                                                           |

New Go tests live in `internal/connections/connections_test.go`,
`internal/connections/integration_test.go`, and
`internal/httpapi/connections_integration_test.go`. Temporary preview code,
credentials, screenshots, and database schemas are excluded from Git. Only
intentional synthetic demo SQL and test fixtures are committed.

## Changed files

The following manifest includes this report and both generated contracts.

- `.env.example`
- `Makefile`
- `README.md`
- `apps/web/scripts/admin.test.mjs`
- `apps/web/src/app/(workspace)/connections/[id]/page.tsx`
- `apps/web/src/app/(workspace)/connections/page.tsx`
- `apps/web/src/app/(workspace)/page.tsx`
- `apps/web/src/app/(workspace)/units/[id]/connections/page.tsx`
- `apps/web/src/app/(workspace)/workspace.css`
- `apps/web/src/components/admin/connections.tsx`
- `apps/web/src/components/admin/schema-browser.tsx`
- `apps/web/src/components/admin/shell.tsx`
- `apps/web/src/components/admin/unit-detail.tsx`
- `apps/web/src/lib/api-types.ts`
- `apps/web/src/lib/schema-model.ts`
- `cmd/api/main.go`
- `cmd/demo/main.go`
- `compose.yaml`
- `deploy/postgres/demo-init.sql`
- `deploy/postgres/demo.sql`
- `docs/adr/0006-postgres-connections.md`
- `docs/architecture.md`
- `docs/authentication.md`
- `docs/authorization.md`
- `docs/connections.md`
- `docs/development.md`
- `docs/openapi.json`
- `docs/roadmap.md`
- `docs/step5-validation.md`
- `internal/apicontract/operations.go`
- `internal/apicontract/schema.go`
- `internal/config/config.go`
- `internal/connections/catalog.go`
- `internal/connections/connections_test.go`
- `internal/connections/crypto.go`
- `internal/connections/integration_test.go`
- `internal/connections/model.go`
- `internal/connections/service.go`
- `internal/connections/snapshots.go`
- `internal/connections/target.go`
- `internal/database/migrations/00006_postgres_connections.sql`
- `internal/httpapi/admin.go`
- `internal/httpapi/admin_integration_test.go`
- `internal/httpapi/auth_integration_test.go`
- `internal/httpapi/connections.go`
- `internal/httpapi/connections_integration_test.go`
- `internal/httpapi/handler.go`
- `internal/rbac/access_views.go`
- `internal/rbac/connections.go`
- `internal/rbac/repository.go`
- `internal/rbac/sql.go`
- `internal/rbac/store_integration_test.go`
- `internal/rbac/views.go`
- `scripts/connection-integration.mjs`
- `scripts/init-dev.mjs`
