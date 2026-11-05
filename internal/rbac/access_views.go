// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

const HiddenSourceMessage = "Inherited from a higher-level unit you don't have access to"

func (v *view) inspect(ctx context.Context, target uuid.UUID, scope Scope) error {
	if err := v.requireRead(ctx, scope); err != nil {
		return err
	}
	if target == v.actor {
		return nil
	}
	global, err := v.can(ctx, UserManage, RootScope())
	if err != nil {
		return err
	}
	if global {
		return nil
	}
	return require(ctx, v.engine, v.actor, UnitManageMembers, scope)
}
func (v *view) assignmentView(ctx context.Context, a Assignment, role Role, target Scope) (AssignmentView, error) {
	result := AssignmentView{Role: RoleRef{ID: role.ID, Name: role.Name}, Permissions: slices.Clone(role.Permissions), Source: SourceDirect}
	if a.Scope != target {
		result.Source = SourceInherited
	}
	readable, err := v.readable(ctx, a.Scope)
	if err != nil {
		return result, err
	}
	if !readable {
		// Do not serialize the assignment ID, scope, creator, note, time, or source
		// unit when its origin is hidden. Notes can themselves contain unit names.
		result.Source = SourceHidden
		result.ReadOnlyReason = HiddenSourceMessage
		return result, nil
	}
	source, err := v.reader.Unit(ctx, a.Scope.ID)
	if err != nil {
		return result, err
	}
	result.ID = &a.ID
	result.Scope = &a.Scope
	result.SourceUnit = &UnitRef{ID: source.ID, Name: source.Name}
	result.Note = &a.Note
	result.CreatedAt = &a.CreatedAt
	result.CanRevoke, err = v.engine.CanAssign(ctx, v.actor, a.RoleID, a.Scope)
	if err != nil {
		return result, err
	}
	if !result.CanRevoke {
		result.ReadOnlyReason = "This assignment is outside your manageable subtree or grants permissions you cannot delegate."
		return result, nil
	}
	active, err := v.reader.ActiveUser(ctx, a.UserID)
	if err != nil {
		return result, err
	}
	if active && a.RoleID == uuid.MustParse(AdministratorRoleID) && a.Scope == RootScope() {
		if v.lastAdmins == nil {
			count, err := remainingAdministrators(ctx, v.q, uuid.Nil, uuid.Nil)
			if err != nil {
				return result, err
			}
			v.lastAdmins = &count
		}
		if *v.lastAdmins <= 1 {
			result.CanRevoke = false
			result.ReadOnlyReason = ErrLastAdministrator.Error()
		}
	}
	return result, nil
}
func (r *Repository) Members(ctx context.Context, actor, unitID uuid.UUID, directOnly, accessOnly bool) ([]Member, error) {
	result := []Member{}
	err := r.read(ctx, actor, func(v *view) error {
		scope := UnitScope(unitID)
		if err := v.requireRead(ctx, scope); err != nil {
			return err
		}
		chain, err := v.engine.resolve(ctx, scope)
		if err != nil {
			return err
		}
		ids := []uuid.UUID{}
		for _, s := range chain {
			ids = append(ids, s.ID)
		}
		if directOnly {
			ids = []uuid.UUID{unitID}
		}
		rows, err := v.q.QueryContext(ctx, `SELECT DISTINCT u.id,u.email,u.is_active FROM users u JOIN assignments a ON a.user_id=u.id WHERE a.scope_type='unit' AND a.scope_id=ANY($1::uuid[]) ORDER BY u.email,u.id`, uuidArray(ids))
		if err != nil {
			return err
		}
		users := []UserInfo{}
		for rows.Next() {
			var user UserInfo
			if err := rows.Scan(&user.ID, &user.Email, &user.IsActive); err != nil {
				_ = rows.Close()
				return err
			}
			users = append(users, user)
		}
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return err
		}
		for _, user := range users {
			member := Member{User: user, Assignments: []AssignmentView{}}
			member.Permissions, err = effective(ctx, v.engine, user.ID, scope)
			if err != nil {
				return err
			}
			if accessOnly && len(member.Permissions) == 0 {
				continue
			}
			grants, err := v.reader.Grants(ctx, user.ID, chain)
			if err != nil {
				return err
			}
			for _, grant := range grants {
				if directOnly && grant.Assignment.Scope != scope {
					continue
				}
				item, err := v.assignmentView(ctx, grant.Assignment, grant.Role, scope)
				if err != nil {
					return err
				}
				member.Assignments = append(member.Assignments, item)
			}
			result = append(result, member)
		}
		return nil
	})
	return result, err
}
func (r *Repository) UserAssignments(ctx context.Context, actor, target uuid.UUID) ([]UserAssignment, error) {
	result := []UserAssignment{}
	err := r.read(ctx, actor, func(v *view) error {
		units, err := v.units(ctx)
		if err != nil {
			return err
		}
		global, err := v.can(ctx, UserManage, RootScope())
		if err != nil {
			return err
		}
		authorized := actor == target && len(units) > 0 || global
		if !authorized {
			for _, unit := range units {
				manage, err := v.can(ctx, UnitManageMembers, UnitScope(unit.ID))
				if err != nil {
					return err
				}
				authorized = authorized || manage
			}
		}
		if !authorized {
			return ErrForbidden
		}
		user, err := userInfo(ctx, v.q, target)
		if err != nil {
			return err
		}
		// Include applicable inherited assignments per readable/manageable unit,
		// redacting each origin independently. Deduplicate hidden/visible rows by ID.
		seen := map[uuid.UUID]bool{}
		for _, unit := range units {
			scope := UnitScope(unit.ID)
			if err := v.inspect(ctx, target, scope); errors.Is(err, ErrForbidden) {
				continue
			} else if err != nil {
				return err
			}
			chain, err := v.engine.resolve(ctx, scope)
			if err != nil {
				return err
			}
			grants, err := v.reader.Grants(ctx, target, chain)
			if err != nil {
				return err
			}
			for _, grant := range grants {
				if seen[grant.Assignment.ID] {
					continue
				}
				seen[grant.Assignment.ID] = true
				item, err := v.assignmentView(ctx, grant.Assignment, grant.Role, scope)
				if err != nil {
					return err
				}
				result = append(result, UserAssignment{User: user, Assignment: item})
			}
		}
		return nil
	})
	return result, err
}
func (r *Repository) AccessForUser(ctx context.Context, actor, target uuid.UUID) (UserAccess, error) {
	result := UserAccess{Units: []UnitAccess{}}
	err := r.read(ctx, actor, func(v *view) error {
		units, err := v.units(ctx)
		if err != nil {
			return err
		}
		global, err := v.can(ctx, UserManage, RootScope())
		if err != nil {
			return err
		}
		authorized := actor == target && len(units) > 0 || global
		for _, unit := range units {
			scope := UnitScope(unit.ID)
			if err := v.inspect(ctx, target, scope); errors.Is(err, ErrForbidden) {
				continue
			} else if err != nil {
				return err
			}
			authorized = true
			held, err := effective(ctx, v.engine, target, scope)
			if err != nil {
				return err
			}
			if len(held) > 0 {
				result.Units = append(result.Units, UnitAccess{Unit: unit, Permissions: held})
			}
		}
		if !authorized {
			return ErrForbidden
		}
		result.User, err = userInfo(ctx, v.q, target)
		return err
	})
	return result, err
}
func (r *Repository) ExplainFor(ctx context.Context, actor, target uuid.UUID, permission Permission, resource Scope) (ExplanationView, error) {
	result := ExplanationView{Grants: []AssignmentView{}}
	err := r.read(ctx, actor, func(v *view) error {
		if resource.Type != ScopeUnit {
			return ErrForbidden
		}
		if err := v.inspect(ctx, target, resource); err != nil {
			return err
		}
		if !permission.Valid() {
			return ErrInvalidInput
		}
		if _, err := userInfo(ctx, v.q, target); err != nil {
			return err
		}
		explanation, err := v.engine.Explain(ctx, target, permission, resource)
		if err != nil {
			return err
		}
		result.Allowed = explanation.Allowed
		result.Reason = explanation.Reason
		result.ReasonText = "No matching assignment grants this permission."
		if explanation.Reason == "inactive_or_unknown_user" {
			result.ReasonText = "The user is inactive and has no effective access."
		}
		if explanation.Allowed {
			result.ReasonText = "Granted by the matching assignments shown below."
		}
		chain, err := v.engine.resolve(ctx, resource)
		if err != nil {
			return err
		}
		grants, err := v.reader.Grants(ctx, target, chain)
		if err != nil {
			return err
		}
		for _, provenance := range explanation.Grants {
			for _, grant := range grants {
				if grant.Assignment.ID == provenance.AssignmentID {
					item, err := v.assignmentView(ctx, grant.Assignment, grant.Role, resource)
					if err != nil {
						return err
					}
					result.Grants = append(result.Grants, item)
					break
				}
			}
		}
		return nil
	})
	return result, err
}
func (r *Repository) UserDirectory(ctx context.Context, actor uuid.UUID, unit *uuid.UUID, query string) ([]UserInfo, error) {
	result := []UserInfo{}
	err := r.read(ctx, actor, func(v *view) error {
		global, err := v.can(ctx, UserManage, RootScope())
		if err != nil {
			return err
		}
		if unit != nil {
			scope := UnitScope(*unit)
			if err := v.requireRead(ctx, scope); err != nil {
				return err
			}
			if !global {
				if err := require(ctx, v.engine, actor, UnitManageMembers, scope); err != nil {
					return err
				}
			}
		} else if !global {
			return ErrForbidden
		}
		if !validText(query, 254, false) {
			return ErrInvalidInput
		}
		pattern := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.ToLower(query)) + "%"
		rows, err := v.q.QueryContext(ctx, `SELECT id,email,is_active FROM users WHERE is_active AND email LIKE $1 ORDER BY email,id LIMIT 200`, pattern)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var user UserInfo
			if err := rows.Scan(&user.ID, &user.Email, &user.IsActive); err != nil {
				return err
			}
			result = append(result, user)
		}
		return rows.Err()
	})
	return result, err
}
func (r *Repository) AssignmentAtReadableScope(ctx context.Context, actor, id uuid.UUID) (Assignment, error) {
	var result Assignment
	err := r.read(ctx, actor, func(v *view) error {
		var err error
		result, err = scanAssignment(v.q.QueryRowContext(ctx, `SELECT `+assignmentColumns+` FROM assignments WHERE id=$1`, id))
		if errors.Is(err, ErrNotFound) {
			return ErrForbidden
		}
		if err != nil {
			return err
		}
		if err := v.requireRead(ctx, result.Scope); err != nil {
			return err
		}
		allowed, err := v.engine.CanAssign(ctx, actor, result.RoleID, result.Scope)
		if err != nil {
			return err
		}
		if !allowed {
			return ErrForbidden
		}
		return nil
	})
	return result, err
}

type AuditFilter struct {
	ActorID  *uuid.UUID
	Action   string
	From     *time.Time
	To       *time.Time
	Before   *time.Time
	BeforeID *uuid.UUID
	Limit    int
}
type AuditPage struct {
	Entries []Audit `json:"entries"`
	HasMore bool    `json:"has_more"`
}

func (r *Repository) FilteredAudit(ctx context.Context, actor uuid.UUID, filter AuditFilter) (AuditPage, error) {
	result := AuditPage{Entries: []Audit{}}
	err := r.read(ctx, actor, func(v *view) error {
		if err := require(ctx, v.engine, actor, AuditRead, RootScope()); err != nil {
			return err
		}
		if filter.Limit < 1 || filter.Limit > 200 || !validText(filter.Action, 120, false) {
			return ErrInvalidInput
		}
		if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
			return ErrInvalidInput
		}
		if (filter.Before == nil) != (filter.BeforeID == nil) {
			return ErrInvalidInput
		}
		rows, err := v.q.QueryContext(ctx, `SELECT id,actor_user_id,action,target_type,target_id,details,created_at FROM audit_log
   WHERE ($1::uuid IS NULL OR actor_user_id=$1) AND ($2='' OR action=$2)
   AND ($3::timestamptz IS NULL OR created_at>=$3) AND ($4::timestamptz IS NULL OR created_at<=$4)
   AND ($5::timestamptz IS NULL OR (created_at,id)<($5,$6::uuid)) ORDER BY created_at DESC,id DESC LIMIT $7`, filter.ActorID, filter.Action, filter.From, filter.To, filter.Before, filter.BeforeID, filter.Limit+1)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var row Audit
			var user, target sql.NullString
			var details []byte
			if err := rows.Scan(&row.ID, &user, &row.Action, &row.TargetType, &target, &details, &row.CreatedAt); err != nil {
				return err
			}
			if user.Valid {
				row.ActorUserID, err = uuid.Parse(user.String)
				if err != nil {
					return err
				}
			}
			if target.Valid {
				row.TargetID, err = uuid.Parse(target.String)
				if err != nil {
					return err
				}
			}
			if err := json.Unmarshal(details, &row.Details); err != nil {
				return err
			}
			result.Entries = append(result.Entries, row)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		if len(result.Entries) > filter.Limit {
			result.HasMore = true
			result.Entries = result.Entries[:filter.Limit]
		}
		return nil
	})
	return result, err
}
