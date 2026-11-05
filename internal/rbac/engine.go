// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
	"fmt"
	"slices"
	"sort"

	"github.com/google/uuid"
)

type Engine struct{ reader Reader }

func NewEngine(reader Reader) *Engine { return &Engine{reader: reader} }

type scopeResolver func(context.Context, Reader, uuid.UUID) (Unit, error)

// This registry is deliberately closed to the two implemented scope kinds.
// A connection adapter supplies ownership later; absent adapters fail closed.
var scopeRegistry = map[ScopeType]scopeResolver{
	ScopeUnit: func(ctx context.Context, r Reader, id uuid.UUID) (Unit, error) { return r.Unit(ctx, id) },
	ScopeConnection: func(ctx context.Context, r Reader, id uuid.UUID) (Unit, error) {
		owner, err := r.ConnectionUnit(ctx, id)
		if err != nil {
			return Unit{}, err
		}
		return r.Unit(ctx, owner)
	},
}

func (e *Engine) resolve(ctx context.Context, scope Scope) ([]Scope, error) {
	resolve, ok := scopeRegistry[scope.Type]
	if !ok || scope.ID == uuid.Nil {
		return nil, ErrInvalidScope
	}
	unit, err := resolve(ctx, e.reader, scope.ID)
	if err != nil {
		return nil, err
	}
	ids := append(slices.Clone(unit.Ancestors), unit.ID)
	if len(ids) == 0 || len(ids) > MaxDepth || ids[0] != RootScope().ID {
		return nil, ErrInvalidScope
	}
	seen := make(map[uuid.UUID]bool, len(ids))
	chain := make([]Scope, 0, len(ids)+1)
	for _, id := range ids {
		if id == uuid.Nil || seen[id] {
			return nil, ErrInvalidScope
		}
		seen[id] = true
		chain = append(chain, UnitScope(id))
	}
	if scope.Type == ScopeConnection {
		chain = append(chain, scope)
	}
	return chain, nil
}

// Can is the authorization gate. Errors always accompany a denial.
func (e *Engine) Can(ctx context.Context, user uuid.UUID, permission Permission, resource Scope) (bool, error) {
	result, err := e.Explain(ctx, user, permission, resource)
	return result.Allowed, err
}

type snapshotReader interface {
	snapshot(context.Context, func(Reader) error) error
}

func (e *Engine) withSnapshot(ctx context.Context, fn func(*Engine) error) error {
	if reader, ok := e.reader.(snapshotReader); ok {
		return reader.snapshot(ctx, func(view Reader) error { return fn(NewEngine(view)) })
	}
	return fn(e)
}

func (e *Engine) Explain(ctx context.Context, user uuid.UUID, permission Permission, resource Scope) (Explanation, error) {
	result := Explanation{Grants: []Grant{}, Reason: "no_matching_assignment"}
	err := e.withSnapshot(ctx, func(view *Engine) error {
		var err error
		result, err = view.explain(ctx, user, permission, resource)
		return err
	})
	if err != nil {
		return Explanation{Grants: []Grant{}, Reason: "evaluation_failed"}, err
	}
	return result, nil
}

func (e *Engine) explain(ctx context.Context, user uuid.UUID, permission Permission, resource Scope) (Explanation, error) {
	denied := Explanation{Grants: []Grant{}, Reason: "no_matching_assignment"}
	if err := ctx.Err(); err != nil {
		return denied, err
	}
	if !permission.Valid() {
		denied.Reason = "unknown_permission"
		return denied, nil
	}
	if user == uuid.Nil {
		denied.Reason = "inactive_or_unknown_user"
		return denied, nil
	}
	active, err := e.reader.ActiveUser(ctx, user)
	if err != nil {
		return denied, err
	}
	if !active {
		denied.Reason = "inactive_or_unknown_user"
		return denied, nil
	}
	accepted, blocked, boundary, err := e.applicableGrants(ctx, user, resource)
	if err != nil {
		return denied, err
	}
	matching := func(candidate RoleGrant) bool {
		return (!permission.SystemLevel() || candidate.Assignment.Scope == RootScope()) && slices.Contains(candidate.Role.Permissions, permission)
	}
	grant := func(candidate RoleGrant) Grant {
		a := candidate.Assignment
		return Grant{AssignmentID: a.ID, RoleID: candidate.Role.ID, Role: candidate.Role.Name, Scope: a.Scope, Inherited: a.Scope != resource}
	}
	denied.Blocked = []BlockedGrant{}
	for _, candidate := range accepted {
		if matching(candidate) {
			denied.Grants = append(denied.Grants, grant(candidate))
		}
	}
	for _, candidate := range blocked {
		if matching(candidate) {
			denied.Blocked = append(denied.Blocked, BlockedGrant{Grant: grant(candidate), BlockedBy: boundary.ID})
		}
	}
	if len(denied.Blocked) > 0 {
		denied.Reason = "blocked_by_inherit_access"
	}
	sort.Slice(denied.Grants, func(i, j int) bool {
		return denied.Grants[i].AssignmentID.String() < denied.Grants[j].AssignmentID.String()
	})
	denied.Allowed = len(denied.Grants) > 0
	if denied.Allowed {
		denied.Reason = "permission_granted"
	}
	return denied, nil
}

func (e *Engine) CanAssign(ctx context.Context, actor, roleID uuid.UUID, scope Scope) (bool, error) {
	allowed := false
	err := e.withSnapshot(ctx, func(view *Engine) error {
		var err error
		allowed, err = view.canAssign(ctx, actor, roleID, scope)
		return err
	})
	if err != nil {
		return false, err
	}
	return allowed, nil
}

func (e *Engine) canAssign(ctx context.Context, actor, roleID uuid.UUID, scope Scope) (bool, error) {
	role, err := e.reader.Role(ctx, roleID)
	if err != nil {
		return false, err
	}
	for _, permission := range role.Permissions {
		if !permission.Valid() {
			return false, fmt.Errorf("%w: unknown permission", ErrInvalidInput)
		}
		if permission.SystemLevel() && scope != RootScope() {
			return false, nil
		}
	}
	allowed, err := e.Can(ctx, actor, UnitManageMembers, scope)
	if err != nil || !allowed {
		return false, err
	}
	for _, permission := range role.Permissions {
		allowed, err = e.Can(ctx, actor, permission, scope)
		if err != nil || !allowed {
			return false, err
		}
	}
	return true, nil
}

// applicableGrants keeps physical ancestry separate from the inheritance walk.
// A boundary includes its own assignments; only the built-in root Administrator
// bypasses it. Blocked candidates are retained solely for Explain provenance.
func (e *Engine) applicableGrants(ctx context.Context, user uuid.UUID, resource Scope) (accepted, blocked []RoleGrant, boundary *Unit, err error) {
	chain, err := e.resolve(ctx, resource)
	if err != nil {
		return nil, nil, nil, err
	}
	first := 0
	for i := len(chain) - 1; i >= 0; i-- {
		if chain[i].Type != ScopeUnit {
			continue
		}
		unit, loadErr := e.reader.Unit(ctx, chain[i].ID)
		if loadErr != nil {
			return nil, nil, nil, loadErr
		}
		if !unit.InheritAccess {
			first = i
			boundary = &unit
			break
		}
	}
	candidates, err := e.reader.Grants(ctx, user, chain)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, candidate := range candidates {
		a := candidate.Assignment
		if a.UserID != user || a.RoleID != candidate.Role.ID || !slices.Contains(chain, a.Scope) {
			continue
		}
		if slices.Contains(chain[first:], a.Scope) || rootAdministratorGrant(candidate) {
			accepted = append(accepted, candidate)
		} else {
			blocked = append(blocked, candidate)
		}
	}
	return accepted, blocked, boundary, nil
}
func rootAdministratorGrant(grant RoleGrant) bool {
	return grant.Assignment.Scope == RootScope() && grant.Assignment.RoleID == uuid.MustParse(AdministratorRoleID) && grant.Role.ID == grant.Assignment.RoleID && grant.Role.IsSystem
}
func (e *Engine) rootAdministrator(ctx context.Context, user uuid.UUID) (bool, error) {
	active, err := e.reader.ActiveUser(ctx, user)
	if err != nil || !active {
		return false, err
	}
	grants, err := e.reader.Grants(ctx, user, []Scope{RootScope()})
	if err != nil {
		return false, err
	}
	for _, grant := range grants {
		if grant.Assignment.UserID == user && rootAdministratorGrant(grant) {
			return true, nil
		}
	}
	return false, nil
}

// CanRevoke owns all revocation authority, including the last-admin invariant.
// Callers supply a persisted assignment; repository writes recheck under Lock.
func (e *Engine) CanRevoke(ctx context.Context, actor uuid.UUID, assignment Assignment) (bool, error) {
	allowed := false
	err := e.withSnapshot(ctx, func(view *Engine) error {
		var err error
		allowed, err = view.canRevoke(ctx, actor, assignment)
		return err
	})
	if err != nil {
		return false, err
	}
	return allowed, nil
}
func (e *Engine) canRevoke(ctx context.Context, actor uuid.UUID, assignment Assignment) (bool, error) {
	allowed, err := e.canAssign(ctx, actor, assignment.RoleID, assignment.Scope)
	if err != nil || !allowed {
		return false, err
	}
	grants, _, _, err := e.applicableGrants(ctx, actor, assignment.Scope)
	if err != nil {
		return false, err
	}
	within := false
	for _, grant := range grants {
		if grant.Assignment.Scope.Type == ScopeUnit && slices.Contains(grant.Role.Permissions, UnitManageMembers) {
			within = true
			break
		}
	}
	if !within {
		return false, nil
	}
	if assignment.RoleID == uuid.MustParse(AdministratorRoleID) && assignment.Scope == RootScope() {
		active, err := e.reader.ActiveUser(ctx, assignment.UserID)
		if err != nil {
			return false, err
		}
		if active {
			count, err := e.reader.ActiveRootAdministrators(ctx)
			if err != nil {
				return false, err
			}
			if count <= 1 {
				return false, ErrLastAdministrator
			}
		}
	}
	return true, nil
}
