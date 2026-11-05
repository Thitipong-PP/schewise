// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"

	"github.com/google/uuid"
)

type SelfUnitAccess struct {
	UnitAccess
	Assignments []AssignmentView `json:"assignments"`
}
type SelfAccess struct {
	User  UserInfo         `json:"user"`
	Units []SelfUnitAccess `json:"units"`
}

// MyAccess reuses the existing effective-access and per-assignment projections
// within one snapshot. Active accounts without readable units get an empty list.
func (r *Repository) MyAccess(ctx context.Context, actor uuid.UUID) (SelfAccess, error) {
	result := SelfAccess{Units: []SelfUnitAccess{}}
	err := r.read(ctx, actor, func(v *view) error {
		var err error
		result.User, err = userInfo(ctx, v.q, actor)
		if err != nil {
			return err
		}
		units, err := v.units(ctx)
		if err != nil {
			return err
		}
		for _, unit := range units {
			scope := UnitScope(unit.ID)
			if err := v.inspect(ctx, actor, scope); err != nil {
				return err
			}
			permissions, err := effective(ctx, v.engine, actor, scope)
			if err != nil {
				return err
			}
			item := SelfUnitAccess{UnitAccess: UnitAccess{Unit: unit, Permissions: permissions}, Assignments: []AssignmentView{}}
			grants, _, _, err := v.engine.applicableGrants(ctx, actor, scope)
			if err != nil {
				return err
			}
			for _, grant := range grants {
				assignment, err := v.assignmentView(ctx, grant.Assignment, grant.Role, scope)
				if err != nil {
					return err
				}
				assignment.CanRevoke = false
				if assignment.Source != SourceHidden {
					assignment.ReadOnlyReason = "Account access is read-only. Manage assignments from the unit."
				}
				item.Assignments = append(item.Assignments, assignment)
			}
			result.Units = append(result.Units, item)
		}
		return nil
	})
	return result, err
}
