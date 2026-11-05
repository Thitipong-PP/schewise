// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

type UserInfo struct {
	ID       uuid.UUID `json:"id"`
	Email    string    `json:"email"`
	IsActive bool      `json:"is_active"`
}
type UnitView struct {
	InheritAccess bool         `json:"inherit_access"`
	ID            uuid.UUID    `json:"id"`
	ParentID      *uuid.UUID   `json:"parent_id"`
	Name          string       `json:"name"`
	Description   string       `json:"description"`
	Kind          string       `json:"kind"`
	Depth         int          `json:"depth"`
	IsRoot        bool         `json:"is_root"`
	CanMove       bool         `json:"can_move"`
	Permissions   []Permission `json:"permissions"`
}
type UnitNode struct {
	Unit     UnitView   `json:"unit"`
	Children []UnitNode `json:"children"`
}
type RoleView struct {
	Role
	AssignmentCount int  `json:"assignment_count"`
	MemberCount     int  `json:"member_count"`
	CanEdit         bool `json:"can_edit"`
	CanClone        bool `json:"can_clone"`
	CanDelete       bool `json:"can_delete"`
}
type RoleRef struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}
type UnitRef struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}
type Source string

const (
	SourceDirect    Source = "direct"
	SourceInherited Source = "inherited"
	SourceHidden    Source = "inherited_hidden"
)

type AssignmentView struct {
	ID             *uuid.UUID   `json:"id,omitempty"`
	Role           RoleRef      `json:"role"`
	Permissions    []Permission `json:"permissions"`
	Source         Source       `json:"source"`
	SourceUnit     *UnitRef     `json:"source_unit,omitempty"`
	Scope          *Scope       `json:"scope,omitempty"`
	Note           *string      `json:"note,omitempty"`
	CreatedAt      *time.Time   `json:"created_at,omitempty"`
	CanRevoke      bool         `json:"can_revoke"`
	ReadOnlyReason string       `json:"read_only_reason"`
}
type Member struct {
	User        UserInfo         `json:"user"`
	Assignments []AssignmentView `json:"assignments"`
	Permissions []Permission     `json:"permissions"`
}
type UserAssignment struct {
	User       UserInfo       `json:"user"`
	Assignment AssignmentView `json:"assignment"`
}
type UnitAccess struct {
	Unit        UnitView     `json:"unit"`
	Permissions []Permission `json:"permissions"`
}
type UserAccess struct {
	User  UserInfo     `json:"user"`
	Units []UnitAccess `json:"units"`
}
type BlockView struct {
	Assignment AssignmentView `json:"assignment"`
	Source     string         `json:"source"`
	Unit       *UnitRef       `json:"unit,omitempty"`
	ReasonText string         `json:"reason_text"`
}

type ExplanationView struct {
	Blocked    []BlockView      `json:"blocked"`
	Allowed    bool             `json:"allowed"`
	Grants     []AssignmentView `json:"grants"`
	Reason     string           `json:"reason"`
	ReasonText string           `json:"reason_text"`
}
type AccessContext struct {
	User            UserInfo     `json:"user"`
	RootPermissions []Permission `json:"root_permissions"`
	CanReadUnits    bool         `json:"can_read_units"`
}

type cachedReader struct {
	administrators *int
	Reader
	users  map[uuid.UUID]bool
	units  map[uuid.UUID]Unit
	roles  map[uuid.UUID]Role
	grants map[string][]RoleGrant
}

func (r *cachedReader) ActiveUser(ctx context.Context, id uuid.UUID) (bool, error) {
	if value, ok := r.users[id]; ok {
		return value, nil
	}
	value, err := r.Reader.ActiveUser(ctx, id)
	if err == nil {
		r.users[id] = value
	}
	return value, err
}
func (r *cachedReader) Unit(ctx context.Context, id uuid.UUID) (Unit, error) {
	if value, ok := r.units[id]; ok {
		return value, nil
	}
	value, err := r.Reader.Unit(ctx, id)
	if err == nil {
		r.units[id] = value
	}
	return value, err
}
func (r *cachedReader) Role(ctx context.Context, id uuid.UUID) (Role, error) {
	if value, ok := r.roles[id]; ok {
		return value, nil
	}
	value, err := r.Reader.Role(ctx, id)
	if err == nil {
		r.roles[id] = value
	}
	return value, err
}
func (r *cachedReader) Grants(ctx context.Context, user uuid.UUID, scopes []Scope) ([]RoleGrant, error) {
	key := user.String()
	for _, scope := range scopes {
		key += "/" + string(scope.Type) + ":" + scope.ID.String()
	}
	if value, ok := r.grants[key]; ok {
		return value, nil
	}
	value, err := r.Reader.Grants(ctx, user, scopes)
	if err == nil {
		r.grants[key] = value
	}
	return value, err
}

type view struct {
	q      *sql.Tx
	reader *cachedReader
	engine *Engine
	actor  uuid.UUID
}

func (r *Repository) read(ctx context.Context, actor uuid.UUID, fn func(*view) error) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	reader := cacheReader(NewSQLReader(tx, r.connections))
	v := &view{q: tx, reader: reader, engine: NewEngine(reader), actor: actor}
	if err := fn(v); err != nil {
		return err
	}
	return tx.Commit()
}
func (v *view) can(ctx context.Context, p Permission, scope Scope) (bool, error) {
	return v.engine.Can(ctx, v.actor, p, scope)
}
func (v *view) readable(ctx context.Context, scope Scope) (bool, error) {
	allowed, err := v.can(ctx, UnitRead, scope)
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrInvalidScope) || errors.Is(err, ErrUnavailable) {
		return false, nil
	}
	return allowed, err
}
func (v *view) requireRead(ctx context.Context, scope Scope) error {
	allowed, err := v.readable(ctx, scope)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}
func effective(ctx context.Context, e *Engine, user uuid.UUID, scope Scope) ([]Permission, error) {
	result := []Permission{}
	for _, p := range permissions {
		allowed, err := e.Can(ctx, user, p, scope)
		if err != nil {
			return nil, err
		}
		if allowed {
			result = append(result, p)
		}
	}
	return result, nil
}
func userInfo(ctx context.Context, q Queryer, id uuid.UUID) (UserInfo, error) {
	var user UserInfo
	err := q.QueryRowContext(ctx, `SELECT id,email,is_active FROM users WHERE id=$1`, id).Scan(&user.ID, &user.Email, &user.IsActive)
	return user, storeError(err)
}
func (v *view) unitView(ctx context.Context, unit Unit) (UnitView, error) {
	result := UnitView{InheritAccess: unit.InheritAccess, ID: unit.ID, Name: unit.Name, Description: unit.Description, Kind: unit.Kind, Depth: len(unit.Ancestors) + 1, IsRoot: unit.ID == RootScope().ID}
	var err error
	result.Permissions, err = effective(ctx, v.engine, v.actor, UnitScope(unit.ID))
	if err != nil {
		return result, err
	}
	if unit.ParentID != uuid.Nil {
		readable, err := v.readable(ctx, UnitScope(unit.ParentID))
		if err != nil {
			return result, err
		}
		if readable {
			parent := unit.ParentID
			result.ParentID = &parent
		}
		manageParent, err := v.can(ctx, UnitManage, UnitScope(unit.ParentID))
		if err != nil {
			return result, err
		}
		result.CanMove = manageParent && slices.Contains(result.Permissions, UnitManage)
	}
	return result, nil
}
func (v *view) units(ctx context.Context) ([]UnitView, error) {
	all, err := subtree(ctx, v.q, RootScope().ID)
	if err != nil {
		return nil, err
	}
	for _, unit := range all {
		v.reader.units[unit.ID] = unit
	}
	result := []UnitView{}
	for _, unit := range all {
		allowed, err := v.readable(ctx, UnitScope(unit.ID))
		if err != nil {
			return nil, err
		}
		if allowed {
			item, err := v.unitView(ctx, unit)
			if err != nil {
				return nil, err
			}
			result = append(result, item)
		}
	}
	slices.SortFunc(result, func(a, b UnitView) int {
		if a.Depth != b.Depth {
			return a.Depth - b.Depth
		}
		return strings.Compare(a.Name, b.Name)
	})
	return result, nil
}
func (r *Repository) VisibleUnits(ctx context.Context, actor uuid.UUID) ([]UnitView, error) {
	var result []UnitView
	err := r.read(ctx, actor, func(v *view) error {
		var err error
		result, err = v.units(ctx)
		if err != nil {
			return err
		}
		if len(result) == 0 {
			return ErrForbidden
		}
		return nil
	})
	return result, err
}
func (r *Repository) VisibleUnit(ctx context.Context, actor, id uuid.UUID) (UnitView, error) {
	var result UnitView
	err := r.read(ctx, actor, func(v *view) error {
		if err := v.requireRead(ctx, UnitScope(id)); err != nil {
			return err
		}
		unit, err := v.reader.Unit(ctx, id)
		if err != nil {
			return err
		}
		result, err = v.unitView(ctx, unit)
		return err
	})
	return result, err
}
func (r *Repository) Context(ctx context.Context, actor uuid.UUID) (AccessContext, error) {
	var result AccessContext
	err := r.read(ctx, actor, func(v *view) error {
		var err error
		result.RootPermissions, err = effective(ctx, v.engine, actor, RootScope())
		if err != nil {
			return err
		}
		units, err := v.units(ctx)
		if err != nil {
			return err
		}
		result.CanReadUnits = len(units) > 0
		if !result.CanReadUnits && len(result.RootPermissions) == 0 {
			return ErrForbidden
		}
		result.User, err = userInfo(ctx, v.q, actor)
		return err
	})
	return result, err
}
func (r *Repository) PermissionCatalog(ctx context.Context, actor uuid.UUID) ([]PermissionInfo, error) {
	var result []PermissionInfo
	err := r.read(ctx, actor, func(v *view) error {
		roleManager, err := v.can(ctx, RoleManage, RootScope())
		if err != nil {
			return err
		}
		if !roleManager {
			units, err := v.units(ctx)
			if err != nil {
				return err
			}
			if len(units) == 0 {
				return ErrForbidden
			}
		}
		result = Catalog()
		for i := range result {
			result[i].CanGrant, err = v.can(ctx, result[i].Permission, RootScope())
			if err != nil {
				return err
			}
		}
		return nil
	})
	return result, err
}
func (r *Repository) RolesFor(ctx context.Context, actor uuid.UUID) ([]RoleView, error) {
	result := []RoleView{}
	err := r.read(ctx, actor, func(v *view) error {
		if err := require(ctx, v.engine, actor, RoleManage, RootScope()); err != nil {
			return err
		}
		rows, err := v.q.QueryContext(ctx, `SELECT `+roleColumns+` FROM roles r ORDER BY r.name`)
		if err != nil {
			return err
		}
		roles := []Role{}
		for rows.Next() {
			role, err := scanRole(rows)
			if err != nil {
				_ = rows.Close()
				return err
			}
			roles = append(roles, role)
		}
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return err
		}
		for _, role := range roles {
			item := RoleView{Role: role, CanEdit: !role.IsSystem, CanClone: true}
			if err := v.q.QueryRowContext(ctx, `SELECT count(*),count(DISTINCT user_id) FROM assignments WHERE role_id=$1`, role.ID).Scan(&item.AssignmentCount, &item.MemberCount); err != nil {
				return err
			}
			item.CanDelete = !role.IsSystem && item.AssignmentCount == 0
			for _, p := range role.Permissions {
				held, err := v.can(ctx, p, RootScope())
				if err != nil {
					return err
				}
				if !held {
					item.CanClone = false
				}
			}
			result = append(result, item)
		}
		return nil
	})
	return result, err
}
func (r *Repository) AssignableRoles(ctx context.Context, actor, unitID uuid.UUID) ([]Role, error) {
	result := []Role{}
	err := r.read(ctx, actor, func(v *view) error {
		scope := UnitScope(unitID)
		if err := v.requireRead(ctx, scope); err != nil {
			return err
		}
		if err := require(ctx, v.engine, actor, UnitManageMembers, scope); err != nil {
			return err
		}
		rows, err := v.q.QueryContext(ctx, `SELECT id FROM roles ORDER BY name`)
		if err != nil {
			return err
		}
		ids := []uuid.UUID{}
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				_ = rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return err
		}
		for _, id := range ids {
			allowed, err := v.engine.CanAssign(ctx, actor, id, scope)
			if err != nil {
				return err
			}
			if allowed {
				role, err := v.reader.Role(ctx, id)
				if err != nil {
					return err
				}
				result = append(result, role)
			}
		}
		return nil
	})
	return result, err
}

func cacheReader(reader Reader) *cachedReader {
	return &cachedReader{Reader: reader, users: map[uuid.UUID]bool{}, units: map[uuid.UUID]Unit{}, roles: map[uuid.UUID]Role{}, grants: map[string][]RoleGrant{}}
}
func (r *cachedReader) ActiveRootAdministrators(ctx context.Context) (int, error) {
	if r.administrators != nil {
		return *r.administrators, nil
	}
	count, err := r.Reader.ActiveRootAdministrators(ctx)
	if err == nil {
		r.administrators = &count
	}
	return count, err
}
