// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

func (r *Repository) Assign(ctx context.Context, actor, user, roleID uuid.UUID, scope Scope, note string) (Assignment, error) {
	assignment := Assignment{ID: uuid.New()}
	err := r.mutate(ctx, actor, "assignment.create", "assignment", assignment.ID, func(tx *sql.Tx, e *Engine) (map[string]any, error) {
		if !validText(note, 2000, false) {
			return nil, ErrInvalidInput
		}
		role, err := e.reader.Role(ctx, roleID)
		if err != nil {
			return nil, err
		}
		for _, p := range role.Permissions {
			if p.SystemLevel() && scope != RootScope() {
				return nil, ErrInvalidScope
			}
		}
		allowed, err := e.CanAssign(ctx, actor, roleID, scope)
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, ErrForbidden
		}
		active, err := e.reader.ActiveUser(ctx, user)
		if err != nil {
			return nil, err
		}
		if !active {
			return nil, ErrNotFound
		}
		assignment, err = scanAssignment(tx.QueryRowContext(ctx, `INSERT INTO assignments(id,user_id,role_id,scope_type,scope_id,note,created_by) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING `+assignmentColumns, assignment.ID, user, roleID, scope.Type, scope.ID, note, actor))
		return map[string]any{"assignment": assignment}, err
	})
	return assignment, err
}
func remainingAdministrators(ctx context.Context, tx *sql.Tx, excludedAssignment, excludedUser uuid.UUID) (int, error) {
	var count int
	err := tx.QueryRowContext(ctx, `SELECT count(*) FROM assignments a JOIN users u ON u.id=a.user_id
 WHERE a.role_id=$1 AND a.scope_type='unit' AND a.scope_id=$2 AND u.is_active AND a.id<>$3 AND a.user_id<>$4`, uuid.MustParse(AdministratorRoleID), RootScope().ID, excludedAssignment, excludedUser).Scan(&count)
	return count, err
}
func (r *Repository) Revoke(ctx context.Context, actor, id uuid.UUID) error {
	return r.mutate(ctx, actor, "assignment.revoke", "assignment", id, func(tx *sql.Tx, e *Engine) (map[string]any, error) {
		assignment, err := scanAssignment(tx.QueryRowContext(ctx, `SELECT `+assignmentColumns+` FROM assignments WHERE id=$1`, id))
		if err != nil {
			return nil, err
		}
		allowed, err := e.CanRevoke(ctx, actor, assignment)
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, ErrForbidden
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM assignments WHERE id=$1`, id)
		return map[string]any{"assignment": assignment}, err
	})
}
func (r *Repository) SetUserActive(ctx context.Context, actor, user uuid.UUID, active bool) error {
	return r.mutate(ctx, actor, "user.set_active", "user", user, func(tx *sql.Tx, e *Engine) (map[string]any, error) {
		if err := require(ctx, e, actor, UserManage, RootScope()); err != nil {
			return nil, err
		}
		var before bool
		if err := tx.QueryRowContext(ctx, `SELECT is_active FROM users WHERE id=$1 FOR UPDATE`, user).Scan(&before); err != nil {
			return nil, err
		}
		if before && !active {
			var administrator bool
			if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM assignments WHERE user_id=$1 AND role_id=$2 AND scope_type='unit' AND scope_id=$3)`, user, uuid.MustParse(AdministratorRoleID), RootScope().ID).Scan(&administrator); err != nil {
				return nil, err
			}
			if administrator {
				count, err := remainingAdministrators(ctx, tx, uuid.Nil, user)
				if err != nil {
					return nil, err
				}
				if count == 0 {
					return nil, ErrLastAdministrator
				}
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE users SET is_active=$2 WHERE id=$1`, user, active); err != nil {
			return nil, err
		}
		if !active {
			if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id=$1`, user); err != nil {
				return nil, err
			}
		}
		return map[string]any{"before": before, "after": active}, nil
	})
}
