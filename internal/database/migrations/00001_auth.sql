-- +goose Up
CREATE TABLE roles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL UNIQUE
);
INSERT INTO roles (name) VALUES ('Administrator');

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text NOT NULL UNIQUE CHECK (email = lower(btrim(email))),
    password_hash text NOT NULL,
    role_id uuid NOT NULL REFERENCES roles(id),
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Anonymous sessions exist only to synchronize CSRF before login.
-- Authentication tokens are stored as SHA-256 digests, never in plaintext.
CREATE TABLE sessions (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    user_id uuid REFERENCES users(id) ON DELETE CASCADE,
    csrf_token text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);
CREATE INDEX sessions_user_id_idx ON sessions(user_id);
CREATE INDEX sessions_expires_at_idx ON sessions(expires_at);

CREATE TABLE audit_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    occurred_at timestamptz NOT NULL DEFAULT now(),
    request_id text NOT NULL,
    actor_id uuid REFERENCES users(id) ON DELETE SET NULL,
    target_id uuid,
    action text NOT NULL,
    outcome text NOT NULL,
    remote_ip text NOT NULL
);
CREATE INDEX audit_events_occurred_at_idx ON audit_events(occurred_at);
CREATE INDEX audit_events_actor_id_idx ON audit_events(actor_id);

-- +goose Down
-- This explicit down migration removes authentication and audit data.
DROP TABLE audit_events;
DROP TABLE sessions;
DROP TABLE users;
DROP TABLE roles;
