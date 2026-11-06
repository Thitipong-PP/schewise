-- SPDX-License-Identifier: Apache-2.0

-- +goose Up
-- is_active already exists from migration 00002; preserve its values/guards.
ALTER TABLE users ADD COLUMN last_login_at timestamptz;
ALTER TABLE sessions ADD COLUMN id uuid NOT NULL DEFAULT gen_random_uuid() UNIQUE;
ALTER TABLE sessions ADD COLUMN user_agent text NOT NULL DEFAULT '' CHECK (length(user_agent) <= 256);
ALTER TABLE sessions ADD COLUMN ip text;
ALTER TABLE sessions ADD COLUMN last_seen_at timestamptz;
UPDATE sessions SET last_seen_at=created_at;
ALTER TABLE sessions ALTER COLUMN last_seen_at SET NOT NULL;
ALTER TABLE sessions ALTER COLUMN last_seen_at SET DEFAULT now();

CREATE TABLE password_resets (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash)=32),
    created_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL DEFAULT (now()+interval '24 hours'),
    used_at timestamptz,
    revoked_at timestamptz,
    CHECK (used_at IS NULL OR revoked_at IS NULL)
);
-- New links revoke every outstanding link under the shared metadata lock.
CREATE UNIQUE INDEX password_resets_pending_user_idx ON password_resets(user_id)
WHERE used_at IS NULL AND revoked_at IS NULL;

-- +goose Down
-- Session metadata and recovery history must not disappear implicitly.
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Account settings are forward-only; use a reviewed recovery plan'; END $$;
-- +goose StatementEnd
