// SPDX-License-Identifier: Apache-2.0
package rbac

import (
	"context"
	"database/sql"
	"github.com/google/uuid"
)

func inheritancePlan(ctx context.Context, e *Engine, actor, id uuid.UUID, value bool) (Unit, bool, error) {
	if err := require(ctx, e, actor, UnitManageMembers, UnitScope(id)); err != nil {
		return Unit{}, false, err
	}
	if id == RootScope().ID {
		return Unit{}, false, ErrRootProtected
	}
	unit, err := e.reader.Unit(ctx, id)
	if err != nil {
		return Unit{}, false, err
	}
	proposed := unit
	proposed.InheritAccess = value
	projected := NewEngine(projectedReader{Reader: e.reader, units: map[uuid.UUID]Unit{id: proposed}})
	retained, err := projected.Can(ctx, actor, UnitManageMembers, UnitScope(id))
	if err != nil {
		return Unit{}, false, err
	}
	if !retained {
		admin, err := e.rootAdministrator(ctx, actor)
		if err != nil {
			return Unit{}, false, err
		}
		if !admin {
			return unit, false, nil
		}
	}
	return unit, true, nil
}

func (r *Repository) SetInheritAccess(ctx context.Context, actor, id uuid.UUID, value bool) error {
	return r.mutate(ctx, actor, "unit.set_inherit_access", "unit", id, func(tx *sql.Tx, e *Engine) (map[string]any, error) {
		before, executable, err := inheritancePlan(ctx, e, actor, id, value)
		if err != nil {
			return nil, err
		}
		if !executable {
			return nil, ErrSelfLockout
		}
		if _, err := tx.ExecContext(ctx, `UPDATE units SET inherit_access=$2 WHERE id=$1`, id, value); err != nil {
			return nil, err
		}
		return map[string]any{"before": before.InheritAccess, "after": value}, nil
	})
}

func (r *Repository) DryRunInheritAccess(ctx context.Context, actor, id uuid.UUID, value bool) (Impact, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return Impact{}, err
	}
	defer func() { _ = tx.Rollback() }()
	reader := cacheReader(NewSQLReader(tx, r.connections))
	before := NewEngine(reader)
	unit, executable, err := inheritancePlan(ctx, before, actor, id, value)
	if err != nil {
		return Impact{}, err
	}
	unit.InheritAccess = value
	after := NewEngine(cacheReader(projectedReader{Reader: reader, units: map[uuid.UUID]Unit{id: unit}}))
	units, err := subtree(ctx, tx, id)
	if err != nil {
		return Impact{}, err
	}
	ids := []uuid.UUID{}
	resources := []Scope{}
	for _, u := range units {
		ids = append(ids, u.ID)
		resources = append(resources, UnitScope(u.ID))
	}
	connections, err := r.connectionsIn(ctx, tx, ids)
	if err != nil {
		return Impact{}, err
	}
	for _, c := range connections {
		resources = append(resources, Scope{Type: ScopeConnection, ID: c.ID})
	}
	changes, err := accessChanges(ctx, tx, before, after, resources, false)
	if err != nil {
		return Impact{}, err
	}
	blockers := []string{}
	if !executable {
		blockers = append(blockers, "self_lockout")
	}
	return Impact{Executable: executable, Blockers: blockers, Changes: changes, Connections: connections}, tx.Commit()
}
