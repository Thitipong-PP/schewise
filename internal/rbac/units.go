// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
	"database/sql"
	"slices"
	"strings"

	"github.com/google/uuid"
)

type UnitInput struct {
	Name        string
	Description string
	Kind        string
}

func validateUnit(input UnitInput) error {
	if !validText(input.Name, 120, true) || !validText(input.Description, 2000, false) || !validText(input.Kind, 64, false) {
		return ErrInvalidInput
	}
	return nil
}
func (r *Repository) CreateUnit(ctx context.Context, actor, parent uuid.UUID, input UnitInput) (Unit, error) {
	result := Unit{ID: uuid.New()}
	err := r.mutate(ctx, actor, "unit.create", "unit", result.ID, func(tx *sql.Tx, e *Engine) (map[string]any, error) {
		input.Name = strings.TrimSpace(input.Name)
		if err := validateUnit(input); err != nil {
			return nil, err
		}
		if err := require(ctx, e, actor, UnitManage, UnitScope(parent)); err != nil {
			return nil, err
		}
		parentUnit, err := e.reader.Unit(ctx, parent)
		if err != nil {
			return nil, err
		}
		ancestors := append(slices.Clone(parentUnit.Ancestors), parent)
		if len(ancestors) >= MaxDepth {
			return nil, ErrDepth
		}
		result, err = scanUnit(tx.QueryRowContext(ctx, `INSERT INTO units(id,parent_id,name,description,kind,ancestors) VALUES($1,$2,$3,$4,$5,$6::uuid[]) RETURNING `+unitColumns, result.ID, parent, input.Name, input.Description, input.Kind, uuidArray(ancestors)))
		return map[string]any{"unit": result}, err
	})
	return result, err
}
func (r *Repository) UpdateUnit(ctx context.Context, actor, id uuid.UUID, input UnitInput) (Unit, error) {
	var result Unit
	err := r.mutate(ctx, actor, "unit.update", "unit", id, func(tx *sql.Tx, e *Engine) (map[string]any, error) {
		if err := require(ctx, e, actor, UnitManage, UnitScope(id)); err != nil {
			return nil, err
		}
		input.Name = strings.TrimSpace(input.Name)
		if err := validateUnit(input); err != nil {
			return nil, err
		}
		before, err := e.reader.Unit(ctx, id)
		if err != nil {
			return nil, err
		}
		result, err = scanUnit(tx.QueryRowContext(ctx, `UPDATE units SET name=$2,description=$3,kind=$4 WHERE id=$1 RETURNING `+unitColumns, id, input.Name, input.Description, input.Kind))
		return map[string]any{"before": before, "after": result}, err
	})
	return result, err
}
func subtree(ctx context.Context, q Queryer, id uuid.UUID) ([]Unit, error) {
	rows, err := q.QueryContext(ctx, `SELECT `+unitColumns+` FROM units WHERE id=$1 OR $1=ANY(ancestors) ORDER BY cardinality(ancestors),id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Unit{}
	for rows.Next() {
		unit, err := scanUnit(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, unit)
	}
	return result, rows.Err()
}
func movePlan(ctx context.Context, q Queryer, e *Engine, actor, id, parent uuid.UUID) ([]Unit, []Unit, error) {
	source, err := e.reader.Unit(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if err := require(ctx, e, actor, UnitManage, UnitScope(id)); err != nil {
		return nil, nil, err
	}
	if id == RootScope().ID {
		return nil, nil, ErrRootProtected
	}
	// Managing a child must not allow escaping or importing an unmanaged subtree.
	for _, scope := range []Scope{UnitScope(source.ParentID), UnitScope(parent)} {
		if err := require(ctx, e, actor, UnitManage, scope); err != nil {
			return nil, nil, err
		}
	}
	destination, err := e.reader.Unit(ctx, parent)
	if err != nil {
		return nil, nil, err
	}
	if parent == id || slices.Contains(destination.Ancestors, id) {
		return nil, nil, ErrCycle
	}
	before, err := subtree(ctx, q, id)
	if err != nil {
		return nil, nil, err
	}
	prefix := append(slices.Clone(destination.Ancestors), parent)
	after := make([]Unit, 0, len(before))
	for _, unit := range before {
		updated := unit
		updated.Ancestors = slices.Clone(prefix)
		if unit.ID == id {
			updated.ParentID = parent
		} else {
			updated.Ancestors = append(updated.Ancestors, id)
			updated.Ancestors = append(updated.Ancestors, unit.Ancestors[len(source.Ancestors)+1:]...)
		}
		if len(updated.Ancestors) >= MaxDepth {
			return nil, nil, ErrDepth
		}
		after = append(after, updated)
	}
	return before, after, nil
}
func (r *Repository) MoveUnit(ctx context.Context, actor, id, parent uuid.UUID) error {
	return r.mutate(ctx, actor, "unit.move", "unit", id, func(tx *sql.Tx, e *Engine) (map[string]any, error) {
		before, after, err := movePlan(ctx, tx, e, actor, id, parent)
		if err != nil {
			return nil, err
		}
		for _, unit := range after {
			if _, err := tx.ExecContext(ctx, `UPDATE units SET parent_id=$2,ancestors=$3::uuid[] WHERE id=$1`, unit.ID, unit.ParentID, uuidArray(unit.Ancestors)); err != nil {
				return nil, err
			}
		}
		return map[string]any{"before": before, "after": after}, nil
	})
}
func (r *Repository) connectionsIn(ctx context.Context, q Queryer, units []uuid.UUID) ([]Connection, error) {
	if r.connections == nil {
		return []Connection{}, nil
	}
	result, err := r.connections.InUnits(ctx, q, units)
	if err != nil {
		return nil, err
	}
	seen := make(map[uuid.UUID]bool, len(result))
	for _, connection := range result {
		if connection.ID == uuid.Nil || !slices.Contains(units, connection.UnitID) || seen[connection.ID] {
			return nil, ErrInvalidScope
		}
		seen[connection.ID] = true
		owner, err := r.connections.Owner(ctx, q, connection.ID)
		if err != nil {
			return nil, err
		}
		if owner != connection.UnitID {
			return nil, ErrInvalidScope
		}
	}
	slices.SortFunc(result, func(a, b Connection) int { return strings.Compare(a.ID.String(), b.ID.String()) })
	return result, nil
}
func (r *Repository) deletePlan(ctx context.Context, q Queryer, e *Engine, actor, id uuid.UUID) (Unit, []string, []Connection, error) {
	if err := require(ctx, e, actor, UnitManage, UnitScope(id)); err != nil {
		return Unit{}, nil, nil, err
	}
	if id == RootScope().ID {
		return Unit{}, nil, nil, ErrRootProtected
	}
	unit, err := e.reader.Unit(ctx, id)
	if err != nil {
		return unit, nil, nil, err
	}
	var children, assignments bool
	if err := q.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM units WHERE parent_id=$1),EXISTS(SELECT 1 FROM assignments WHERE scope_type='unit' AND scope_id=$1)`, id).Scan(&children, &assignments); err != nil {
		return unit, nil, nil, err
	}
	descendants, err := subtree(ctx, q, id)
	if err != nil {
		return unit, nil, nil, err
	}
	ids := make([]uuid.UUID, 0, len(descendants))
	for _, child := range descendants {
		ids = append(ids, child.ID)
	}
	connections, err := r.connectionsIn(ctx, q, ids)
	if err != nil {
		return unit, nil, nil, err
	}
	blockers := []string{}
	if children {
		blockers = append(blockers, "child_units")
	}
	if assignments {
		blockers = append(blockers, "direct_assignments")
	}
	if len(connections) > 0 {
		blockers = append(blockers, "connections")
	}
	return unit, blockers, connections, nil
}
func (r *Repository) DeleteUnit(ctx context.Context, actor, id uuid.UUID) error {
	return r.mutate(ctx, actor, "unit.delete", "unit", id, func(tx *sql.Tx, e *Engine) (map[string]any, error) {
		unit, blockers, _, err := r.deletePlan(ctx, tx, e, actor, id)
		if err != nil {
			return nil, err
		}
		if len(blockers) > 0 {
			return nil, ErrNotEmpty
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM units WHERE id=$1`, id)
		return map[string]any{"unit": unit}, err
	})
}
