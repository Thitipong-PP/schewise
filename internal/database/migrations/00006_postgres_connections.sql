-- SPDX-License-Identifier: Apache-2.0

-- +goose Up
CREATE TABLE connections (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id uuid NOT NULL REFERENCES units(id) ON DELETE RESTRICT,
    name text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 120),
    engine text NOT NULL DEFAULT 'postgres' CHECK (engine='postgres'),
    host text NOT NULL,
    port integer NOT NULL CHECK (port BETWEEN 1 AND 65535),
    database text NOT NULL,
    username text NOT NULL,
    ssl_mode text NOT NULL CHECK (ssl_mode IN ('disable','allow','prefer','require','verify-ca','verify-full')),
    encrypted_password bytea NOT NULL,
    nonce bytea NOT NULL CHECK (octet_length(nonce)=12),
    status text NOT NULL DEFAULT 'unknown' CHECK (status IN ('unknown','ok','error')),
    last_error text,
    last_synced_at timestamptz,
    created_by uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE(unit_id,name)
);
CREATE TABLE schema_snapshots (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id uuid NOT NULL REFERENCES connections(id) ON DELETE CASCADE,
    taken_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    content_hash text NOT NULL,
    content jsonb NOT NULL CHECK (jsonb_typeof(content)='object'),
    stats jsonb NOT NULL CHECK (jsonb_typeof(stats)='object')
);
CREATE INDEX schema_snapshots_connection_idx ON schema_snapshots(connection_id,taken_at DESC,id DESC);

-- A polymorphic assignment keeps its existing scope fields. A generated FK
-- enforces real connection ownership and cascades only connection assignments.
ALTER TABLE assignments ADD COLUMN connection_id uuid GENERATED ALWAYS AS
    (CASE WHEN scope_type='connection' THEN scope_id ELSE NULL END) STORED
    REFERENCES connections(id) ON DELETE CASCADE;

-- +goose Down
-- Connection credentials, access grants and schema history cannot be discarded
-- by an implicit downgrade. This snapshot history is not a target data backup.
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'PostgreSQL connections are forward-only; use a reviewed recovery plan'; END $$;
-- +goose StatementEnd
