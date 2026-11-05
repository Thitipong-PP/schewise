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
	chain, err := e.resolve(ctx, resource)
	if err != nil {
		return denied, err
	}
	candidates, err := e.reader.Grants(ctx, user, chain)
	if err != nil {
		return denied, err
	}
	for _, candidate := range candidates {
		assignment := candidate.Assignment
		// Defend even against a faulty reader returning unrelated assignments.
		if assignment.UserID != user || assignment.RoleID != candidate.Role.ID || !slices.Contains(chain, assignment.Scope) {
			continue
		}
		if permission.SystemLevel() && assignment.Scope != RootScope() {
			continue
		}
		if !slices.Contains(candidate.Role.Permissions, permission) {
			continue
		}
		denied.Grants = append(denied.Grants, Grant{AssignmentID: assignment.ID, RoleID: candidate.Role.ID, Role: candidate.Role.Name, Scope: assignment.Scope, Inherited: assignment.Scope != resource})
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
