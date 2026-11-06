// SPDX-License-Identifier: Apache-2.0
package connections

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/google/uuid"
)

// A transaction-scoped advisory lock spans target I/O. The hierarchy lock is
// acquired only afterwards, so slow targets cannot stall unrelated RBAC writes.
func (s *Service) Run(ctx context.Context, actor, id uuid.UUID, syncSchema bool) (OperationResult, error) {
	result := OperationResult{}
	if err := require(ctx, s.db, actor, rbac.ConnectionManage, rbac.Scope{Type: rbac.ScopeConnection, ID: id}); err != nil {
		return result, err
	}
	tx, err := s.locked(ctx, id)
	if err != nil {
		return result, err
	}
	defer func() { _ = tx.Rollback() }()
	scope := rbac.Scope{Type: rbac.ScopeConnection, ID: id}
	if err := require(ctx, tx, actor, rbac.ConnectionManage, scope); err != nil {
		return result, err
	}
	r, err := scanRecord(tx.QueryRowContext(ctx, `SELECT `+recordColumns+` FROM connections WHERE id=$1`, id))
	if err != nil {
		return result, err
	}
	password, err := s.vault.decrypt(id, r.encrypted, r.nonce)
	if err != nil {
		return result, err
	}
	remote, cancel := context.WithTimeout(ctx, s.options.OperationTimeout)
	defer cancel()
	conn, operationErr := s.policy.connect(remote, r.Input, password, s.options.StatementTimeout)
	var content Content
	var stats Stats
	var encoded []byte
	var hash string
	if operationErr == nil {
		if syncSchema {
			content, stats, operationErr = introspect(remote, conn)
			if operationErr == nil {
				encoded, hash, operationErr = encodeContent(&content)
				if operationErr == nil && containsSecret(content, password) {
					operationErr = &TargetError{"metadata_contains_secret", "Schema metadata contains a connection credential. Remove it before syncing."}
				}
				if len(encoded) > 32*1024*1024 {
					operationErr = &TargetError{"schema_too_large", "The schema exceeds the current snapshot size limit."}
				}
			}
		} else {
			operationErr = conn.Ping(remote)
		}
		cleanup, done := context.WithTimeout(context.Background(), 2*time.Second)
		_ = conn.Close(cleanup)
		done()
	}
	if err := rbac.Lock(ctx, tx); err != nil {
		return result, err
	}
	if err := require(ctx, tx, actor, rbac.ConnectionManage, scope); err != nil {
		return result, err
	}
	action := "connection.tested"
	if syncSchema {
		action = "connection.synced"
	}
	details := map[string]any{"success": operationErr == nil}
	r.status = "ok"
	r.lastError = nil
	result.Success = operationErr == nil
	result.Code = "ok"
	result.Message = "Connection successful."
	if operationErr != nil {
		safe := targetError(operationErr)
		r.status = "error"
		r.lastError = &safe.Message
		result.Code = safe.Code
		result.Message = safe.Message
		details["category"] = safe.Code
	}
	if syncSchema && operationErr == nil {
		var previousID uuid.UUID
		var previousHash string
		err = tx.QueryRowContext(ctx, `SELECT id,content_hash FROM schema_snapshots WHERE connection_id=$1 ORDER BY taken_at DESC,id DESC LIMIT 1`, id).Scan(&previousID, &previousHash)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return result, err
		}
		statsJSON, err := json.Marshal(stats)
		if err != nil {
			return result, err
		}
		if previousID != uuid.Nil && previousHash == hash {
			_, err = tx.ExecContext(ctx, `UPDATE schema_snapshots SET stats=$2::jsonb WHERE id=$1`, previousID, string(statsJSON))
		} else {
			_, err = tx.ExecContext(ctx, `INSERT INTO schema_snapshots(connection_id,content_hash,content,stats) VALUES($1,$2,$3::jsonb,$4::jsonb)`, id, hash, string(encoded), string(statsJSON))
			result.CreatedSnapshot = true
		}
		if err != nil {
			return result, err
		}
		now := time.Now().UTC()
		r.lastSynced = &now
		details["new_snapshot"] = result.CreatedSnapshot
		result.Message = "Schema synced."
	}
	_, err = tx.ExecContext(ctx, `UPDATE connections SET status=$2,last_error=$3,last_synced_at=$4,updated_at=clock_timestamp() WHERE id=$1`, id, r.status, r.lastError, r.lastSynced)
	if err != nil {
		return result, err
	}
	if err := s.audit(ctx, tx, actor, id, action, details); err != nil {
		return result, err
	}
	result.Connection, err = s.view(ctx, tx, actor, r)
	if err != nil {
		return result, err
	}
	return result, s.commit(tx, actor, id, action)
}

// Fail closed rather than silently corrupting catalog facts by redacting them.
// Check values, never JSON keys: schemas and comments are untrusted input.
func containsSecret(content Content, password string) bool {
	if password == "" {
		return false
	}
	data, _ := json.Marshal(content)
	var value any
	_ = json.Unmarshal(data, &value)
	var visit func(any) bool
	visit = func(v any) bool {
		switch item := v.(type) {
		case string:
			return strings.Contains(item, password)
		case []any:
			for _, child := range item {
				if visit(child) {
					return true
				}
			}
		case map[string]any:
			for _, child := range item {
				if visit(child) {
					return true
				}
			}
		}
		return false
	}
	return visit(value)
}
func (s *Service) read(ctx context.Context, actor, id uuid.UUID, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := require(ctx, tx, actor, rbac.SchemaRead, rbac.Scope{Type: rbac.ScopeConnection, ID: id}); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

const snapshotColumns = `id,connection_id,taken_at,content_hash,stats`

func scanSnapshot(row interface{ Scan(...any) error }, content bool) (Snapshot, error) {
	var s Snapshot
	var stats, data []byte
	args := []any{&s.ID, &s.ConnectionID, &s.TakenAt, &s.ContentHash, &stats}
	if content {
		args = append(args, &data)
	}
	if err := row.Scan(args...); err != nil {
		return s, dbError(err)
	}
	if err := json.Unmarshal(stats, &s.Stats); err != nil {
		return s, err
	}
	if content {
		if err := json.Unmarshal(data, &s.Content); err != nil {
			return s, err
		}
	}
	return s, nil
}
func (s *Service) Snapshot(ctx context.Context, actor, id, snapshot uuid.UUID) (Snapshot, error) {
	var result Snapshot
	err := s.read(ctx, actor, id, func(tx *sql.Tx) error {
		var err error
		result, err = scanSnapshot(tx.QueryRowContext(ctx, `SELECT `+snapshotColumns+`,content FROM schema_snapshots WHERE connection_id=$1 AND ($2::uuid IS NULL OR id=$2) ORDER BY taken_at DESC,id DESC LIMIT 1`, id, nullable(snapshot)), true)
		return err
	})
	return result, err
}
func (s *Service) Snapshots(ctx context.Context, actor, id, before uuid.UUID, limit int) (Snapshots, error) {
	result := Snapshots{Snapshots: []SnapshotInfo{}}
	if limit == 0 {
		limit = 50
	}
	if limit < 1 || limit > 200 {
		return result, rbac.ErrInvalidInput
	}
	err := s.read(ctx, actor, id, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `SELECT `+snapshotColumns+` FROM schema_snapshots WHERE connection_id=$1 AND ($2::uuid IS NULL OR (taken_at,id)<(SELECT taken_at,id FROM schema_snapshots WHERE id=$2 AND connection_id=$1)) ORDER BY taken_at DESC,id DESC LIMIT $3`, id, nullable(before), limit+1)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			item, err := scanSnapshot(rows, false)
			if err != nil {
				return err
			}
			if len(result.Snapshots) == limit {
				cursor := result.Snapshots[limit-1].ID
				result.NextBefore = &cursor
				break
			}
			result.Snapshots = append(result.Snapshots, item.SnapshotInfo)
		}
		return rows.Err()
	})
	return result, err
}
