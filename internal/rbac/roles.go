// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
	"database/sql"
	"slices"
	"strings"

	"github.com/google/uuid"
)

type RoleInput struct {
	Name        string
	Description string
	Permissions []Permission
}

func validateRole(input RoleInput) error {
	if !validText(input.Name, 120, true) || !validText(input.Description, 2000, false) {
		return ErrInvalidInput
	}
	seen := map[Permission]bool{}
	for _, p := range input.Permissions {
		if !p.Valid() || seen[p] {
			return ErrInvalidInput
		}
		seen[p] = true
	}
	return nil
}
func writePermissions(ctx context.Context, tx *sql.Tx, id uuid.UUID, permissions []Permission) error {
	for _, p := range permissions {
		if _, err := tx.ExecContext(ctx, `INSERT INTO role_permissions(role_id,permission) VALUES($1,$2)`, id, p); err != nil {
			return err
		}
	}
	return nil
}
func authorizeRole(ctx context.Context, e *Engine, actor uuid.UUID, permissions []Permission) error {
	if err := require(ctx, e, actor, RoleManage, RootScope()); err != nil {
		return err
	}
	// Editing a role already assigned to oneself must not become an escalation path.
	for _, p := range permissions {
		if err := require(ctx, e, actor, p, RootScope()); err != nil {
			return err
		}
	}
	return nil
}
func (r *Repository) CreateRole(ctx context.Context, actor uuid.UUID, input RoleInput) (Role, error) {
	return r.createRole(ctx, actor, input, uuid.Nil)
}
func (r *Repository) CloneRole(ctx context.Context, actor, source uuid.UUID, name string) (Role, error) {
	return r.createRole(ctx, actor, RoleInput{Name: name}, source)
}
func (r *Repository) createRole(ctx context.Context, actor uuid.UUID, input RoleInput, source uuid.UUID) (Role, error) {
	role := Role{ID: uuid.New()}
	err := r.mutate(ctx, actor, "role.create", "role", role.ID, func(tx *sql.Tx, e *Engine) (map[string]any, error) {
		if source != uuid.Nil {
			original, err := e.reader.Role(ctx, source)
			if err != nil {
				return nil, err
			}
			input.Description = original.Description
			input.Permissions = slices.Clone(original.Permissions)
		}
		input.Name = strings.TrimSpace(input.Name)
		if err := validateRole(input); err != nil {
			return nil, err
		}
		if err := authorizeRole(ctx, e, actor, input.Permissions); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO roles(id,name,description) VALUES($1,$2,$3)`, role.ID, input.Name, input.Description); err != nil {
			return nil, err
		}
		if err := writePermissions(ctx, tx, role.ID, input.Permissions); err != nil {
			return nil, err
		}
		var err error
		role, err = e.reader.Role(ctx, role.ID)
		return map[string]any{"role": role, "cloned_from": nullableID(source)}, err
	})
	return role, err
}
func (r *Repository) UpdateRole(ctx context.Context, actor, id uuid.UUID, input RoleInput) (Role, error) {
	var result Role
	err := r.mutate(ctx, actor, "role.update", "role", id, func(tx *sql.Tx, e *Engine) (map[string]any, error) {
		if err := require(ctx, e, actor, RoleManage, RootScope()); err != nil {
			return nil, err
		}
		before, err := e.reader.Role(ctx, id)
		if err != nil {
			return nil, err
		}
		if before.IsSystem {
			return nil, ErrSystemRole
		}
		input.Name = strings.TrimSpace(input.Name)
		if err := validateRole(input); err != nil {
			return nil, err
		}
		if err := authorizeRole(ctx, e, actor, input.Permissions); err != nil {
			return nil, err
		}
		for _, p := range input.Permissions {
			if p.SystemLevel() {
				var invalid bool
				err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM assignments WHERE role_id=$1 AND (scope_type<>'unit' OR scope_id<>$2))`, id, RootScope().ID).Scan(&invalid)
				if err != nil {
					return nil, err
				}
				if invalid {
					return nil, ErrInvalidScope
				}
				break
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE roles SET name=$2,description=$3 WHERE id=$1`, id, input.Name, input.Description); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM role_permissions WHERE role_id=$1`, id); err != nil {
			return nil, err
		}
		if err := writePermissions(ctx, tx, id, input.Permissions); err != nil {
			return nil, err
		}
		result, err = e.reader.Role(ctx, id)
		return map[string]any{"before": before, "after": result}, err
	})
	return result, err
}
func (r *Repository) DeleteRole(ctx context.Context, actor, id uuid.UUID) error {
	return r.mutate(ctx, actor, "role.delete", "role", id, func(tx *sql.Tx, e *Engine) (map[string]any, error) {
		if err := require(ctx, e, actor, RoleManage, RootScope()); err != nil {
			return nil, err
		}
		role, err := e.reader.Role(ctx, id)
		if err != nil {
			return nil, err
		}
		if role.IsSystem {
			return nil, ErrSystemRole
		}
		var assigned bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM assignments WHERE role_id=$1)`, id).Scan(&assigned); err != nil {
			return nil, err
		}
		if assigned {
			return nil, ErrNotEmpty
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM roles WHERE id=$1`, id)
		return map[string]any{"role": role}, err
	})
}
