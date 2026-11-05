// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Repository struct {
	db          *sql.DB
	logger      *zap.Logger
	connections ConnectionResolver
}

func NewRepository(db *sql.DB, logger *zap.Logger, connections ConnectionResolver) *Repository {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Repository{db: db, logger: logger, connections: connections}
}
func (r *Repository) Engine() *Engine { return NewEngine(NewSQLReader(r.db, r.connections)) }

type Audit struct {
	ID          uuid.UUID      `json:"id"`
	ActorUserID uuid.UUID      `json:"actor_user_id"`
	Action      string         `json:"action"`
	TargetType  string         `json:"target_type"`
	TargetID    uuid.UUID      `json:"target_id"`
	Details     map[string]any `json:"details"`
	CreatedAt   time.Time      `json:"created_at"`
}

func nullableID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}

// WriteAudit lets authentication share its transaction with the RBAC audit log.
// Callers log the committed event only after their transaction commits.
func WriteAudit(ctx context.Context, db Queryer, event Audit) error {
	if event.Details == nil {
		event.Details = map[string]any{}
	}
	details, err := json.Marshal(event.Details)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO audit_log(actor_user_id,action,target_type,target_id,details) VALUES($1,$2,$3,$4,$5::jsonb)`, nullableID(event.ActorUserID), event.Action, event.TargetType, nullableID(event.TargetID), string(details))
	return err
}
func require(ctx context.Context, e *Engine, actor uuid.UUID, p Permission, scope Scope) error {
	allowed, err := e.Can(ctx, actor, p, scope)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}
func (r *Repository) mutate(ctx context.Context, actor uuid.UUID, action, targetType string, targetID uuid.UUID, fn func(*sql.Tx, *Engine) (map[string]any, error)) (err error) {
	defer func() {
		if err != nil {
			r.logger.Warn("rbac.mutation_failed", zap.String("actor_id", actor.String()), zap.String("action", action), zap.String("target_type", targetType), zap.String("target_id", targetID.String()))
		}
	}()
	if actor == uuid.Nil {
		return ErrForbidden
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err = Lock(ctx, tx); err != nil {
		return err
	}
	details, err := fn(tx, NewEngine(NewSQLReader(tx, r.connections)))
	if err != nil {
		return storeError(err)
	}
	if err = WriteAudit(ctx, tx, Audit{ActorUserID: actor, Action: action, TargetType: targetType, TargetID: targetID, Details: details}); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	r.logger.Info("audit", zap.String("actor_id", actor.String()), zap.String("action", action), zap.String("target_type", targetType), zap.String("target_id", targetID.String()), zap.String("outcome", "success"))
	return nil
}
func validText(s string, max int, required bool) bool {
	return utf8.ValidString(s) && utf8.RuneCountInString(s) <= max && (!required || strings.TrimSpace(s) != "")
}
func (r *Repository) Unit(ctx context.Context, id uuid.UUID) (Unit, error) {
	return NewSQLReader(r.db, r.connections).Unit(ctx, id)
}
func (r *Repository) GetChain(ctx context.Context, id uuid.UUID) ([]Unit, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	reader := NewSQLReader(tx, r.connections)
	unit, err := reader.Unit(ctx, id)
	if err != nil {
		return nil, err
	}
	chain := make([]Unit, 0, len(unit.Ancestors)+1)
	for _, ancestor := range unit.Ancestors {
		item, err := reader.Unit(ctx, ancestor)
		if err != nil {
			return nil, err
		}
		chain = append(chain, item)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return append(chain, unit), nil
}

func (r *Repository) Role(ctx context.Context, id uuid.UUID) (Role, error) {
	return NewSQLReader(r.db, r.connections).Role(ctx, id)
}
func (r *Repository) Roles(ctx context.Context) ([]Role, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+roleColumns+` FROM roles r ORDER BY r.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Role{}
	for rows.Next() {
		role, err := scanRole(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, role)
	}
	return result, rows.Err()
}
func scanAssignment(row scanner) (Assignment, error) {
	var a Assignment
	var creator sql.NullString
	err := row.Scan(&a.ID, &a.UserID, &a.RoleID, &a.Scope.Type, &a.Scope.ID, &a.Note, &creator, &a.CreatedAt)
	if err != nil {
		return a, storeError(err)
	}
	if creator.Valid {
		a.CreatedBy, err = uuid.Parse(creator.String)
	}
	return a, err
}

const assignmentColumns = `id,user_id,role_id,scope_type,scope_id,note,created_by,created_at`

func (r *Repository) Assignments(ctx context.Context, user uuid.UUID) ([]Assignment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+assignmentColumns+` FROM assignments WHERE user_id=$1 ORDER BY id`, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Assignment{}
	for rows.Next() {
		a, err := scanAssignment(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
func (r *Repository) AuditLog(ctx context.Context, actor uuid.UUID, limit int) ([]Audit, error) {
	if limit < 1 || limit > 200 {
		return nil, ErrInvalidInput
	}
	if err := require(ctx, r.Engine(), actor, AuditRead, RootScope()); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,actor_user_id,action,target_type,target_id,details,created_at FROM audit_log ORDER BY created_at DESC,id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Audit{}
	for rows.Next() {
		var a Audit
		var actorID, targetID sql.NullString
		var details []byte
		if err := rows.Scan(&a.ID, &actorID, &a.Action, &a.TargetType, &targetID, &details, &a.CreatedAt); err != nil {
			return nil, err
		}
		if actorID.Valid {
			a.ActorUserID, err = uuid.Parse(actorID.String)
			if err != nil {
				return nil, err
			}
		}
		if targetID.Valid {
			a.TargetID, err = uuid.Parse(targetID.String)
			if err != nil {
				return nil, err
			}
		}
		if err := json.Unmarshal(details, &a.Details); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
