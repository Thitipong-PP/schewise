-- SPDX-License-Identifier: Apache-2.0

-- +goose Up
ALTER TABLE users ADD COLUMN display_name text NOT NULL DEFAULT '' CHECK (length(display_name) <= 120);

-- Role/scope references intentionally retain their IDs after deletion. Acceptance
-- revalidates them under the shared metadata lock; stale invites remain unused.
CREATE TABLE invites (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text NOT NULL CHECK (email = lower(btrim(email)) AND length(email) <= 254),
    role_id uuid NOT NULL,
    scope_type text NOT NULL CHECK (scope_type IN ('unit', 'connection')),
    scope_id uuid,
    token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
    note text NOT NULL DEFAULT '' CHECK (length(note) <= 2000),
    created_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL DEFAULT (now() + interval '7 days'),
    used_at timestamptz,
    revoked_at timestamptz,
    used_by_user_id uuid REFERENCES users(id) ON DELETE RESTRICT,
    CHECK (scope_type = 'unit' OR scope_id IS NOT NULL),
    CHECK (scope_id IS NULL OR scope_id <> '00000000-0000-0000-0000-000000000000'),
    CHECK ((used_at IS NULL) = (used_by_user_id IS NULL)),
    CHECK (used_at IS NULL OR revoked_at IS NULL)
);
CREATE INDEX invites_email_idx ON invites(email);
CREATE INDEX invites_created_idx ON invites(created_at, id);

-- Expiry is time-dependent and cannot appear in an immutable partial index.
-- Serialize the active-email check with all account/RBAC/invite mutations.
-- +goose StatementBegin
CREATE FUNCTION validate_active_invite() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    PERFORM pg_advisory_xact_lock(736395103);
    IF NEW.used_at IS NULL AND NEW.revoked_at IS NULL AND NEW.expires_at > clock_timestamp()
        AND EXISTS (SELECT 1 FROM invites WHERE email=NEW.email AND id<>NEW.id
            AND used_at IS NULL AND revoked_at IS NULL AND expires_at>clock_timestamp()) THEN
        RAISE EXCEPTION 'active invite already exists' USING ERRCODE='23505';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER invites_validate_active BEFORE INSERT OR UPDATE ON invites
FOR EACH ROW EXECUTE FUNCTION validate_active_invite();
-- +goose StatementEnd

-- +goose Down
-- Invite history and display names must not be discarded implicitly.
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Invitations are forward-only; use a reviewed recovery plan'; END $$;
-- +goose StatementEnd
