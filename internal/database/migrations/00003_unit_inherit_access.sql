-- SPDX-License-Identifier: Apache-2.0

-- +goose Up
ALTER TABLE units ADD COLUMN inherit_access boolean NOT NULL DEFAULT true;
ALTER TABLE units ADD CONSTRAINT units_root_inherits CHECK (parent_id IS NOT NULL OR inherit_access);

-- +goose Down
-- A downgrade would silently re-enable grants blocked by inheritance boundaries.
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Inheritance boundaries require a reviewed recovery plan; dropping them would broaden access'; END $$;
-- +goose StatementEnd
