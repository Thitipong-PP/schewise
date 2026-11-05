// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
	"database/sql"
	"slices"

	"github.com/google/uuid"
)

type AccessChange struct {
	UserID   uuid.UUID    `json:"user_id"`
	Resource Scope        `json:"resource"`
	Gained   []Permission `json:"gained"`
	Lost     []Permission `json:"lost"`
}
type Impact struct {
	Executable  bool           `json:"executable"`
	Blockers    []string       `json:"blockers"`
	Changes     []AccessChange `json:"changes"`
	Connections []Connection   `json:"connections"`
}

type projectedReader struct {
	Reader
	units   map[uuid.UUID]Unit
	deleted uuid.UUID
}

func (r projectedReader) Unit(ctx context.Context, id uuid.UUID) (Unit, error) {
	if id == r.deleted {
		return Unit{}, ErrNotFound
	}
	if unit, ok := r.units[id]; ok {
		return unit, nil
	}
	return r.Reader.Unit(ctx, id)
}

// Dry runs use a read-only consistent snapshot. They are reports, not approval
// tokens: a subsequent mutation rechecks authorization and tree constraints.
func (r *Repository) DryRunMove(ctx context.Context, actor, id, parent uuid.UUID) (Impact, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return Impact{}, err
	}
	defer func() { _ = tx.Rollback() }()
	reader := NewSQLReader(tx, r.connections)
	_, after, err := movePlan(ctx, tx, NewEngine(reader), actor, id, parent)
	if err != nil {
		return Impact{}, err
	}
	projected := projectedReader{Reader: reader, units: map[uuid.UUID]Unit{}}
	ids := make([]uuid.UUID, 0, len(after))
	resources := make([]Scope, 0, len(after))
	for _, unit := range after {
		projected.units[unit.ID] = unit
		ids = append(ids, unit.ID)
		resources = append(resources, UnitScope(unit.ID))
	}
	connections, err := r.connectionsIn(ctx, tx, ids)
	if err != nil {
		return Impact{}, err
	}
	for _, connection := range connections {
		resources = append(resources, Scope{Type: ScopeConnection, ID: connection.ID})
	}
	changes, err := accessChanges(ctx, tx, NewEngine(reader), NewEngine(projected), resources, false)
	if err != nil {
		return Impact{}, err
	}
	return Impact{Executable: true, Blockers: []string{}, Changes: changes, Connections: connections}, tx.Commit()
}
func (r *Repository) DryRunDelete(ctx context.Context, actor, id uuid.UUID) (Impact, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return Impact{}, err
	}
	defer func() { _ = tx.Rollback() }()
	reader := NewSQLReader(tx, r.connections)
	_, blockers, connections, err := r.deletePlan(ctx, tx, NewEngine(reader), actor, id)
	if err != nil {
		return Impact{}, err
	}
	result := Impact{Executable: len(blockers) == 0, Blockers: blockers, Connections: connections, Changes: []AccessChange{}}
	// A blocked delete has no executable access changes, and never cascades.
	if result.Executable {
		result.Changes, err = accessChanges(ctx, tx, NewEngine(reader), NewEngine(projectedReader{Reader: reader, deleted: id}), []Scope{UnitScope(id)}, true)
		if err != nil {
			return Impact{}, err
		}
	}
	return result, tx.Commit()
}
func accessChanges(ctx context.Context, q Queryer, before, after *Engine, resources []Scope, deleted bool) ([]AccessChange, error) {
	rows, err := q.QueryContext(ctx, `SELECT id FROM users WHERE is_active ORDER BY id`)
	if err != nil {
		return nil, err
	}
	users := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return nil, err
		}
		users = append(users, id)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return nil, err
	}
	result := []AccessChange{}
	for _, user := range users {
		for _, resource := range resources {
			change := AccessChange{UserID: user, Resource: resource, Gained: []Permission{}, Lost: []Permission{}}
			for _, permission := range permissions {
				old, err := before.Can(ctx, user, permission, resource)
				if err != nil {
					return nil, err
				}
				next := false
				if !deleted {
					next, err = after.Can(ctx, user, permission, resource)
					if err != nil {
						return nil, err
					}
				}
				if old && !next {
					change.Lost = append(change.Lost, permission)
				}
				if !old && next {
					change.Gained = append(change.Gained, permission)
				}
			}
			if len(change.Gained) > 0 || len(change.Lost) > 0 {
				result = append(result, change)
			}
		}
	}
	// Resource order is stable even if an adapter enumerates connections differently.
	slices.SortFunc(result, func(a, b AccessChange) int {
		if c := a.UserID.String(); c != b.UserID.String() {
			if c < b.UserID.String() {
				return -1
			}
			return 1
		}
		if a.Resource.Type != b.Resource.Type {
			if a.Resource.Type < b.Resource.Type {
				return -1
			}
			return 1
		}
		if a.Resource.ID.String() < b.Resource.ID.String() {
			return -1
		}
		if a.Resource.ID == b.Resource.ID {
			return 0
		}
		return 1
	})
	return result, nil
}
