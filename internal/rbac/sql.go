// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type Queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type Connection struct {
	ID     uuid.UUID `json:"id"`
	UnitID uuid.UUID `json:"unit_id"`
}

// ConnectionResolver must resolve one owner per connection and enumerate all
// connections in the supplied units using the same transaction. Future ownership
// mutations must participate in Lock, just like unit/assignment mutations.
// No implementation or target database connection is installed in Step 2A.
type ConnectionResolver interface {
	Owner(context.Context, Queryer, uuid.UUID) (uuid.UUID, error)
	InUnits(context.Context, Queryer, []uuid.UUID) ([]Connection, error)
}

type SQLReader struct {
	db          Queryer
	connections ConnectionResolver
}

func NewSQLReader(db Queryer, connections ConnectionResolver) *SQLReader {
	return &SQLReader{db: db, connections: connections}
}

// Lock serializes hierarchy, role, assignment, and active-user mutations. Use
// READ COMMITTED: take fresh reads after acquiring the lock, never an old snapshot.
func Lock(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(736395103)`)
	return err
}
func (r *SQLReader) ActiveUser(ctx context.Context, id uuid.UUID) (bool, error) {
	var active bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND is_active)`, id).Scan(&active)
	return active, err
}
func (r *SQLReader) ConnectionUnit(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	if r.connections == nil {
		return uuid.Nil, ErrUnavailable
	}
	return r.connections.Owner(ctx, r.db, id)
}

type scanner interface{ Scan(...any) error }

func scanUnit(row scanner) (Unit, error) {
	var unit Unit
	var parent sql.NullString
	var ancestors []byte
	err := row.Scan(&unit.ID, &parent, &unit.Name, &unit.Description, &unit.Kind, &ancestors, &unit.CreatedAt, &unit.InheritAccess)
	if err != nil {
		return unit, storeError(err)
	}
	if parent.Valid {
		unit.ParentID, err = uuid.Parse(parent.String)
		if err != nil {
			return unit, err
		}
	}
	err = json.Unmarshal(ancestors, &unit.Ancestors)
	return unit, err
}

const unitColumns = `id,parent_id,name,description,kind,to_json(ancestors),created_at,inherit_access`

func (r *SQLReader) Unit(ctx context.Context, id uuid.UUID) (Unit, error) {
	return scanUnit(r.db.QueryRowContext(ctx, `SELECT `+unitColumns+` FROM units WHERE id=$1`, id))
}

const roleColumns = `r.id,r.name,r.description,r.is_system,COALESCE((SELECT json_agg(p.permission ORDER BY p.permission) FROM role_permissions p WHERE p.role_id=r.id),'[]'::json)`

func scanRole(row scanner) (Role, error) {
	var role Role
	var permissions []byte
	err := row.Scan(&role.ID, &role.Name, &role.Description, &role.IsSystem, &permissions)
	if err != nil {
		return role, storeError(err)
	}
	err = json.Unmarshal(permissions, &role.Permissions)
	return role, err
}
func (r *SQLReader) Role(ctx context.Context, id uuid.UUID) (Role, error) {
	return scanRole(r.db.QueryRowContext(ctx, `SELECT `+roleColumns+` FROM roles r WHERE r.id=$1`, id))
}
func (r *SQLReader) Grants(ctx context.Context, user uuid.UUID, scopes []Scope) ([]RoleGrant, error) {
	var units, connections []uuid.UUID
	for _, scope := range scopes {
		if scope.Type == ScopeUnit {
			units = append(units, scope.ID)
		} else if scope.Type == ScopeConnection {
			connections = append(connections, scope.ID)
		}
	}
	rows, err := r.db.QueryContext(ctx, `SELECT a.id,a.user_id,a.role_id,a.scope_type,a.scope_id,a.note,a.created_by,a.created_at,`+roleColumns+`
 FROM assignments a JOIN roles r ON r.id=a.role_id WHERE a.user_id=$1 AND
 ((a.scope_type='unit' AND a.scope_id=ANY($2::uuid[])) OR (a.scope_type='connection' AND a.scope_id=ANY($3::uuid[]))) ORDER BY a.id`, user, uuidArray(units), uuidArray(connections))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []RoleGrant{}
	for rows.Next() {
		var grant RoleGrant
		var creator sql.NullString
		var permissions []byte
		a, role := &grant.Assignment, &grant.Role
		if err := rows.Scan(&a.ID, &a.UserID, &a.RoleID, &a.Scope.Type, &a.Scope.ID, &a.Note, &creator, &a.CreatedAt, &role.ID, &role.Name, &role.Description, &role.IsSystem, &permissions); err != nil {
			return nil, err
		}
		if creator.Valid {
			a.CreatedBy, err = uuid.Parse(creator.String)
			if err != nil {
				return nil, err
			}
		}
		if err := json.Unmarshal(permissions, &role.Permissions); err != nil {
			return nil, err
		}
		result = append(result, grant)
	}
	return result, rows.Err()
}
func uuidArray(ids []uuid.UUID) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = id.String()
	}
	return "{" + strings.Join(parts, ",") + "}"
}
func storeError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrConflict
		case "23503":
			return ErrNotEmpty
		case "23514":
			return ErrInvalidInput
		}
	}
	return err
}

// A decision must not combine an old ancestry path with a newly committed
// assignment. Reads outside an existing transaction receive one MVCC snapshot.
// Repository mutations and dry runs already supply their transaction handle.
func (r *SQLReader) snapshot(ctx context.Context, fn func(Reader) error) error {
	db, ok := r.db.(*sql.DB)
	if !ok {
		return fn(cacheReader(r))
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(cacheReader(NewSQLReader(tx, r.connections))); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SQLReader) ActiveRootAdministrators(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM assignments a JOIN users u ON u.id=a.user_id WHERE a.role_id=$1 AND a.scope_type='unit' AND a.scope_id=$2 AND u.is_active`, uuid.MustParse(AdministratorRoleID), RootScope().ID).Scan(&count)
	return count, err
}
