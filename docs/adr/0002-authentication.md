# ADR 0002: application PostgreSQL and server-side authentication

- Status: accepted
- Date: 2026-11-04
- Updates: ADR 0001's HTTP/logging choice and deferred application persistence

## Context

The next milestone needs self-hosted administrator bootstrap, restricted account
creation, login/logout/identity, revocable sessions, CSRF, and durable audit.
The owner selected Fiber, Zap, Argon2id, secure cookies, and local HTTPS.

## Decision

Use Fiber v3 and Zap. Store application metadata in PostgreSQL using pgx through
database/sql. Apply embedded Goose SQL migrations with its Postgres session
locker. Never apply application migrations to a user-connected target database.

Seed Administrator only. Bootstrap from environment when users are empty, inside
an advisory-locked transaction. Require an existing Administrator to create
users. Custom role/permission management is deferred.

Use Argon2id PHC hashes, opaque random tokens stored as digests, synchronizer
CSRF tokens in server-side sessions, exact Origin validation, session rotation
on login, and __Host cookies with HttpOnly/Secure/SameSite=Lax. Persist mutation
success audit events in the same transaction, with redacted Zap request logs.

Serve local HTTPS through Caddy at https://localhost:8443. Next.js proxy and
protected rendering consult Go for identity; Go is the permission boundary.

## Consequences

Local development now needs Postgres and explicit local CA trust for browsers.
CLI tests can trust the CA per process. Installation generates unique ignored
secrets and never resets an existing account on restart. CI exercises real
Postgres integration and HTTPS authentication.

Revocation affects subsequent requests without a JWT denylist. Audit
availability is part of mutation availability. Deployment hardening, shared
throttling, retention, MFA, recovery, invitations, and role editing remain
future work.

## References

- [Goose migration locks](https://pressly.github.io/goose/documentation/provider/)
- [OWASP password storage](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
- [OWASP sessions](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
- [Next.js proxy](https://nextjs.org/docs/app/api-reference/file-conventions/proxy)
- [Caddy local HTTPS](https://caddyserver.com/docs/automatic-https#local-https)
