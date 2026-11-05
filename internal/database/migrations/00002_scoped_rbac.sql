-- SPDX-License-Identifier: Apache-2.0

-- +goose Up
ALTER TABLE roles ADD COLUMN description text NOT NULL DEFAULT '';
ALTER TABLE roles ADD COLUMN is_system boolean NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN is_active boolean NOT NULL DEFAULT true;

CREATE TABLE units (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id uuid REFERENCES units(id) ON DELETE RESTRICT,
    name text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 120),
    description text NOT NULL DEFAULT '' CHECK (length(description) <= 2000),
    kind text NOT NULL DEFAULT '' CHECK (length(kind) <= 64),
    ancestors uuid[] NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (parent_id IS DISTINCT FROM id),
    CHECK (cardinality(ancestors) BETWEEN 0 AND 5),
    CHECK ((parent_id IS NULL) = (cardinality(ancestors) = 0))
);
CREATE UNIQUE INDEX units_single_root ON units ((parent_id IS NULL)) WHERE parent_id IS NULL;
CREATE INDEX units_parent_idx ON units(parent_id);
CREATE INDEX units_ancestors_idx ON units USING gin(ancestors);
INSERT INTO units(id,name,kind) VALUES ('00000000-0000-4000-8000-000000000001','Organization','organization');

CREATE TABLE role_permissions (
    role_id uuid NOT NULL REFERENCES roles(id) ON DELETE CASCADE ON UPDATE CASCADE,
    permission text NOT NULL CHECK (permission IN (
        'user:manage','role:manage','audit:read','unit:read','unit:manage','unit:manage_members',
        'connection:read','connection:manage','schema:read','docs:edit','tag:manage','ai:run','data:preview','data:query'
    )),
    PRIMARY KEY (role_id,permission)
);
CREATE TABLE assignments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    role_id uuid NOT NULL REFERENCES roles(id) ON DELETE RESTRICT ON UPDATE CASCADE,
    scope_type text NOT NULL CHECK (scope_type IN ('unit','connection')),
    scope_id uuid NOT NULL CHECK (scope_id <> '00000000-0000-0000-0000-000000000000'),
    note text NOT NULL DEFAULT '' CHECK (length(note) <= 2000),
    created_by uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id,role_id,scope_type,scope_id)
);
CREATE INDEX assignments_scope_idx ON assignments(scope_type,scope_id);
CREATE INDEX assignments_role_idx ON assignments(role_id);

-- Step 1 used users.role_id, not users.is_admin. Preserve every old assignment.
-- Unknown legacy roles remain custom roles with no implicit permissions.
INSERT INTO assignments(user_id,role_id,scope_type,scope_id,note)
SELECT id,role_id,'unit','00000000-0000-4000-8000-000000000001','Migrated from the single-role identity model' FROM users;
ALTER TABLE users DROP COLUMN role_id;
UPDATE roles SET id='00000000-0000-4000-8000-000000000101',is_system=true,description='All application permissions'
WHERE name='Administrator';
-- Name conflicts fail the migration rather than silently promoting custom roles.
INSERT INTO roles(id,name,description,is_system) VALUES
('00000000-0000-4000-8000-000000000102','Manager','Manage a unit subtree and delegate held permissions',true),
('00000000-0000-4000-8000-000000000103','Developer','Work within a unit subtree without managing assignments',true),
('00000000-0000-4000-8000-000000000104','Viewer','Read units, connections, and schemas',true);
INSERT INTO role_permissions(role_id,permission)
SELECT '00000000-0000-4000-8000-000000000101',unnest(ARRAY[
'user:manage','role:manage','audit:read','unit:read','unit:manage','unit:manage_members',
'connection:read','connection:manage','schema:read','docs:edit','tag:manage','ai:run','data:preview','data:query']);
INSERT INTO role_permissions(role_id,permission)
SELECT '00000000-0000-4000-8000-000000000102',permission FROM role_permissions
WHERE role_id='00000000-0000-4000-8000-000000000101' AND permission NOT IN ('user:manage','role:manage','audit:read');
INSERT INTO role_permissions(role_id,permission)
SELECT '00000000-0000-4000-8000-000000000103',permission FROM role_permissions
WHERE role_id='00000000-0000-4000-8000-000000000102' AND permission <> 'unit:manage_members';
INSERT INTO role_permissions(role_id,permission)
SELECT '00000000-0000-4000-8000-000000000104',unnest(ARRAY['unit:read','connection:read','schema:read']);

CREATE TABLE audit_log (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    action text NOT NULL,
    target_type text NOT NULL,
    target_id uuid,
    details jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(details) = 'object'),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_log_created_at_idx ON audit_log(created_at,id);
CREATE INDEX audit_log_actor_idx ON audit_log(actor_user_id);
INSERT INTO audit_log(id,actor_user_id,action,target_type,target_id,details,created_at)
SELECT id,actor_id,action,CASE WHEN target_id IS NULL THEN 'authentication' ELSE 'user' END,target_id,
jsonb_build_object('request_id',request_id,'outcome',outcome,'remote_ip',remote_ip),occurred_at FROM audit_events;
DROP TABLE audit_events;
INSERT INTO audit_log(action,target_type,target_id,details)
VALUES ('unit.bootstrap','unit','00000000-0000-4000-8000-000000000001','{"name":"Organization"}');
INSERT INTO audit_log(action,target_type,target_id,details)
SELECT 'role.seed','role',id,jsonb_build_object('name',name) FROM roles WHERE is_system;
INSERT INTO audit_log(action,target_type,target_id,details)
SELECT 'assignment.migrate','assignment',id,jsonb_build_object('user_id',user_id,'role_id',role_id,'scope_type',scope_type,'scope_id',scope_id) FROM assignments;

-- The root is permanent, and seeded roles and their permission sets are immutable.
-- +goose StatementBegin
CREATE FUNCTION protect_root_unit() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.parent_id IS NULL THEN
        IF TG_OP = 'DELETE' THEN RAISE EXCEPTION 'root unit is permanent' USING ERRCODE='23514'; END IF;
        IF NEW.parent_id IS NOT NULL OR NEW.id <> OLD.id THEN RAISE EXCEPTION 'root unit is permanent' USING ERRCODE='23514'; END IF;
    END IF;
    IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER units_protect_root BEFORE UPDATE OR DELETE ON units FOR EACH ROW EXECUTE FUNCTION protect_root_unit();
CREATE FUNCTION protect_system_role() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.is_system THEN RAISE EXCEPTION 'system role is immutable' USING ERRCODE='23514'; END IF;
    IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
    IF NEW.is_system THEN RAISE EXCEPTION 'custom role cannot become system role' USING ERRCODE='23514'; END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER roles_protect_system BEFORE UPDATE OR DELETE ON roles FOR EACH ROW EXECUTE FUNCTION protect_system_role();
CREATE FUNCTION protect_system_permissions() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP <> 'INSERT' AND EXISTS(SELECT 1 FROM roles WHERE id=OLD.role_id AND is_system) THEN
        RAISE EXCEPTION 'system role permissions are immutable' USING ERRCODE='23514';
    END IF;
    IF TG_OP <> 'DELETE' AND EXISTS(SELECT 1 FROM roles WHERE id=NEW.role_id AND is_system) THEN
        RAISE EXCEPTION 'system role permissions are immutable' USING ERRCODE='23514';
    END IF;
    IF TG_OP <> 'DELETE' AND NEW.permission IN ('user:manage','role:manage','audit:read')
        AND EXISTS(SELECT 1 FROM assignments WHERE role_id=NEW.role_id AND
        (scope_type<>'unit' OR scope_id<>'00000000-0000-4000-8000-000000000001')) THEN
        RAISE EXCEPTION 'system-level role requires root assignments' USING ERRCODE='23514';
    END IF;
    IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER permissions_protect_system BEFORE INSERT OR UPDATE OR DELETE ON role_permissions FOR EACH ROW EXECUTE FUNCTION protect_system_permissions();
CREATE FUNCTION validate_assignment_scope() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.scope_type='unit' AND NOT EXISTS(SELECT 1 FROM units WHERE id=NEW.scope_id) THEN
        RAISE EXCEPTION 'unit scope does not exist' USING ERRCODE='23503';
    END IF;
    IF EXISTS(SELECT 1 FROM role_permissions WHERE role_id=NEW.role_id AND permission IN ('user:manage','role:manage','audit:read'))
        AND (NEW.scope_type <> 'unit' OR NEW.scope_id <> '00000000-0000-4000-8000-000000000001') THEN
        RAISE EXCEPTION 'system-level role requires root scope' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER assignments_validate_scope BEFORE INSERT OR UPDATE ON assignments FOR EACH ROW EXECUTE FUNCTION validate_assignment_scope();
CREATE FUNCTION validate_unit_path() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE expected uuid[];
BEGIN
    IF NEW.parent_id IS NULL THEN
        IF NEW.id <> '00000000-0000-4000-8000-000000000001' THEN
            RAISE EXCEPTION 'only the bootstrap root may be parentless' USING ERRCODE='23514';
        END IF;
    ELSE
        SELECT ancestors || id INTO expected FROM units WHERE id=NEW.parent_id;
        IF NOT FOUND THEN RAISE EXCEPTION 'parent does not exist' USING ERRCODE='23503'; END IF;
        IF NEW.ancestors <> expected OR NEW.id=ANY(NEW.ancestors) THEN
            RAISE EXCEPTION 'invalid ancestor path' USING ERRCODE='23514';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER units_validate_path BEFORE INSERT OR UPDATE ON units FOR EACH ROW EXECUTE FUNCTION validate_unit_path();
CREATE FUNCTION validate_unit_children() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS(SELECT 1 FROM units child JOIN units parent ON parent.id=child.parent_id
        WHERE parent.id=NEW.id AND child.ancestors <> parent.ancestors || parent.id) THEN
        RAISE EXCEPTION 'descendant paths must be rebuilt in the same transaction' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END;
$$;
CREATE CONSTRAINT TRIGGER units_validate_children AFTER UPDATE ON units
DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION validate_unit_children();
CREATE FUNCTION protect_unit_assignments() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS(SELECT 1 FROM assignments WHERE scope_type='unit' AND scope_id=OLD.id) THEN
        RAISE EXCEPTION 'unit still has assignments' USING ERRCODE='23503';
    END IF;
    RETURN OLD;
END;
$$;
CREATE TRIGGER units_protect_assignments BEFORE DELETE ON units FOR EACH ROW EXECUTE FUNCTION protect_unit_assignments();
CREATE FUNCTION protect_last_administrator() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP='UPDATE' AND NEW.user_id=OLD.user_id AND NEW.role_id=OLD.role_id
        AND NEW.scope_type=OLD.scope_type AND NEW.scope_id=OLD.scope_id THEN RETURN NEW; END IF;
    PERFORM pg_advisory_xact_lock(736395103);
    IF OLD.role_id='00000000-0000-4000-8000-000000000101' AND OLD.scope_type='unit'
        AND OLD.scope_id='00000000-0000-4000-8000-000000000001'
        AND EXISTS(SELECT 1 FROM users WHERE id=OLD.user_id AND is_active) THEN
        IF NOT EXISTS(SELECT 1 FROM assignments a JOIN users u ON u.id=a.user_id
            WHERE a.id<>OLD.id AND a.role_id=OLD.role_id AND a.scope_type='unit' AND a.scope_id=OLD.scope_id AND u.is_active) THEN
            RAISE EXCEPTION 'last active root Administrator is protected' USING ERRCODE='23514';
        END IF;
    END IF;
    IF TG_OP='DELETE' THEN RETURN OLD; END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER assignments_protect_last_admin BEFORE DELETE OR UPDATE ON assignments FOR EACH ROW EXECUTE FUNCTION protect_last_administrator();
CREATE FUNCTION protect_last_administrator_user() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.is_active AND NOT NEW.is_active THEN
        PERFORM pg_advisory_xact_lock(736395103);
        IF EXISTS(SELECT 1 FROM assignments WHERE user_id=OLD.id AND role_id='00000000-0000-4000-8000-000000000101'
            AND scope_type='unit' AND scope_id='00000000-0000-4000-8000-000000000001')
            AND NOT EXISTS(SELECT 1 FROM assignments a JOIN users u ON u.id=a.user_id WHERE a.user_id<>OLD.id AND u.is_active
            AND a.role_id='00000000-0000-4000-8000-000000000101' AND a.scope_type='unit' AND a.scope_id='00000000-0000-4000-8000-000000000001') THEN
            RAISE EXCEPTION 'last active root Administrator is protected' USING ERRCODE='23514';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER users_protect_last_admin BEFORE UPDATE OF is_active ON users FOR EACH ROW EXECUTE FUNCTION protect_last_administrator_user();
-- +goose StatementEnd

-- +goose Down
-- Multiple scoped roles and deactivation cannot be represented by Step 1.
-- Refuse an implicit, lossy downgrade; restore a reviewed backup instead.
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Scoped RBAC is forward-only; use a reviewed recovery plan instead of discarding assignments and audit history'; END $$;
-- +goose StatementEnd
