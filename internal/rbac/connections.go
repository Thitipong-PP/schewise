// SPDX-License-Identifier: Apache-2.0
package rbac

import (
	"context"

	"github.com/google/uuid"
)

// SQLConnectionResolver uses application metadata, never a target database.
type SQLConnectionResolver struct{}

func (SQLConnectionResolver) Owner(ctx context.Context, q Queryer, id uuid.UUID) (uuid.UUID, error) {
	var owner uuid.UUID
	err := q.QueryRowContext(ctx, `SELECT unit_id FROM connections WHERE id=$1`, id).Scan(&owner)
	return owner, storeError(err)
}
func (SQLConnectionResolver) InUnits(ctx context.Context, q Queryer, units []uuid.UUID) ([]Connection, error) {
	rows, err := q.QueryContext(ctx, `SELECT id,unit_id FROM connections WHERE unit_id=ANY($1::uuid[]) ORDER BY id`, uuidArray(units))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Connection{}
	for rows.Next() {
		var c Connection
		if err := rows.Scan(&c.ID, &c.UnitID); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}
