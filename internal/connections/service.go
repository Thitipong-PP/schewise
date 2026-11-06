// SPDX-License-Identifier: Apache-2.0
package connections

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

var ErrBusy = errors.New("a connection operation is already running")
var ErrConfirmation = errors.New("the confirmation name does not match")

type Options struct {
	SecretKey                          string
	StatementTimeout, OperationTimeout time.Duration
	HostAllow, HostDeny                []string
}
type Service struct {
	db      *sql.DB
	logger  *zap.Logger
	vault   vault
	policy  hostPolicy
	options Options
}
type record struct {
	Input
	ID               uuid.UUID
	encrypted, nonce []byte
	status           string
	lastError        *string
	lastSynced       *time.Time
}

const recordColumns = `id,unit_id,name,engine,host,port,database,username,ssl_mode,encrypted_password,nonce,status,last_error,last_synced_at`

func scanRecord(row interface{ Scan(...any) error }) (record, error) {
	var r record
	err := row.Scan(&r.ID, &r.UnitID, &r.Name, &r.Engine, &r.Host, &r.Port, &r.Database, &r.Username, &r.SSLMode, &r.encrypted, &r.nonce, &r.status, &r.lastError, &r.lastSynced)
	return r, dbError(err)
}
func dbError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return rbac.ErrNotFound
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505":
			return rbac.ErrConflict
		case "23503", "23514":
			return rbac.ErrInvalidInput
		}
	}
	return err
}
func New(ctx context.Context, db *sql.DB, logger *zap.Logger, options Options) (*Service, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	if options.StatementTimeout == 0 {
		options.StatementTimeout = 30 * time.Second
	}
	if options.OperationTimeout == 0 {
		options.OperationTimeout = 45 * time.Second
	}
	if options.StatementTimeout < time.Second || options.StatementTimeout > 5*time.Minute || options.OperationTimeout < time.Second || options.OperationTimeout > 10*time.Minute {
		return nil, errors.New("connection timeouts are outside supported limits")
	}
	v, err := newVault(options.SecretKey)
	if err != nil {
		return nil, err
	}
	policy, err := newHostPolicy(options.HostAllow, options.HostDeny)
	if err != nil {
		return nil, err
	}
	s := &Service{db: db, logger: logger, vault: v, policy: policy, options: options}
	rows, err := db.QueryContext(ctx, `SELECT id,encrypted_password,nonce FROM connections`)
	if err != nil {
		return nil, errors.New("connection metadata initialization failed")
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var encrypted, nonce []byte
		if err := rows.Scan(&id, &encrypted, &nonce); err != nil {
			return nil, errors.New("connection metadata initialization failed")
		}
		if _, err := v.decrypt(id, encrypted, nonce); err != nil {
			return nil, ErrSecretKey
		}
	}
	return s, rows.Err()
}
func (s *Service) OperationTimeout() time.Duration { return s.options.OperationTimeout }
func validate(in *Input) error {
	in.Name = strings.TrimSpace(in.Name)
	in.Host = strings.TrimSpace(in.Host)
	if in.UnitID == uuid.Nil || in.Name == "" || utf8.RuneCountInString(in.Name) > 120 || in.Engine != "postgres" || !validHost(in.Host) || in.Port < 1 || in.Port > 65535 || in.Database == "" || in.Username == "" {
		return rbac.ErrInvalidInput
	}
	for _, v := range []string{in.Name, in.Host, in.Database, in.Username} {
		if !utf8.ValidString(v) || strings.ContainsRune(v, 0) || len(v) > 512 {
			return rbac.ErrInvalidInput
		}
	}
	switch in.SSLMode {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
	default:
		return rbac.ErrInvalidInput
	}
	return nil
}
func engine(q rbac.Queryer) *rbac.Engine { return rbac.NewEngine(rbac.NewSQLReader(q, nil)) }
func require(ctx context.Context, q rbac.Queryer, actor uuid.UUID, p rbac.Permission, scope rbac.Scope) error {
	ok, err := engine(q).Can(ctx, actor, p, scope)
	if errors.Is(err, rbac.ErrNotFound) || errors.Is(err, rbac.ErrInvalidScope) {
		return rbac.ErrForbidden
	}
	if err != nil {
		return err
	}
	if !ok {
		return rbac.ErrForbidden
	}
	return nil
}
func (s *Service) locked(ctx context.Context, id uuid.UUID) (*sql.Tx, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	var ok bool
	err = tx.QueryRowContext(ctx, `SELECT pg_try_advisory_xact_lock(hashtextextended('schewise.connection.' || $1::text,0))`, id).Scan(&ok)
	if err != nil || !ok {
		_ = tx.Rollback()
		if err == nil {
			err = ErrBusy
		}
		return nil, err
	}
	return tx, nil
}

// Check authority before exposing the busy state of a known or unknown profile.
// Mutations repeat this check inside the metadata lock before committing.
func (s *Service) manageOwner(ctx context.Context, actor, id uuid.UUID) error {
	var owner uuid.UUID
	err := s.db.QueryRowContext(ctx, `SELECT unit_id FROM connections WHERE id=$1`, id).Scan(&owner)
	if errors.Is(err, sql.ErrNoRows) {
		return rbac.ErrForbidden
	}
	if err != nil {
		return err
	}
	return require(ctx, s.db, actor, rbac.ConnectionManage, rbac.UnitScope(owner))
}
func (s *Service) audit(ctx context.Context, tx *sql.Tx, actor, id uuid.UUID, action string, details map[string]any) error {
	return rbac.WriteAudit(ctx, tx, rbac.Audit{ActorUserID: actor, TargetID: id, TargetType: "connection", Action: action, Details: details})
}
func (s *Service) commit(tx *sql.Tx, actor, id uuid.UUID, action string) error {
	if err := tx.Commit(); err != nil {
		return err
	}
	s.logger.Info(action, zap.String("actor_id", actor.String()), zap.String("connection_id", id.String()))
	return nil
}
func (s *Service) view(ctx context.Context, q rbac.Queryer, actor uuid.UUID, r record) (View, error) {
	v := View{ID: r.ID, Name: r.Name, Engine: r.Engine, Status: r.status, LastError: r.lastError, LastSyncedAt: r.lastSynced}
	e := engine(q)
	scope := rbac.Scope{Type: rbac.ScopeConnection, ID: r.ID}
	var err error
	if v.CanManage, err = e.Can(ctx, actor, rbac.ConnectionManage, rbac.UnitScope(r.UnitID)); err != nil {
		return v, err
	}
	if v.CanOperate, err = e.Can(ctx, actor, rbac.ConnectionManage, scope); err != nil {
		return v, err
	}
	if v.CanReadSchema, err = e.Can(ctx, actor, rbac.SchemaRead, scope); err != nil {
		return v, err
	}
	if v.CanOperate {
		v.Details = &Details{Host: r.Host, Port: r.Port, Database: r.Database, Username: r.Username, SSLMode: r.SSLMode}
	}
	readable, err := e.Can(ctx, actor, rbac.UnitRead, rbac.UnitScope(r.UnitID))
	if err != nil {
		return v, err
	}
	if readable {
		var name string
		if err := q.QueryRowContext(ctx, `SELECT name FROM units WHERE id=$1`, r.UnitID).Scan(&name); err != nil {
			return v, err
		}
		v.Unit = &rbac.UnitRef{ID: r.UnitID, Name: name}
	}
	return v, nil
}
func (s *Service) List(ctx context.Context, actor, unit uuid.UUID) (List, error) {
	result := List{Connections: []View{}}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return result, err
	}
	defer func() { _ = tx.Rollback() }()
	if unit != uuid.Nil {
		if err := require(ctx, tx, actor, rbac.UnitRead, rbac.UnitScope(unit)); err != nil {
			return result, err
		}
	}
	rows, err := tx.QueryContext(ctx, `SELECT `+recordColumns+` FROM connections WHERE ($1::uuid IS NULL OR unit_id=$1) ORDER BY name,id`, nullable(unit))
	if err != nil {
		return result, err
	}
	records := []record{}
	for rows.Next() {
		r, err := scanRecord(rows)
		if err != nil {
			rows.Close()
			return result, err
		}
		records = append(records, r)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	for _, r := range records {
		ok, err := engine(tx).Can(ctx, actor, rbac.ConnectionRead, rbac.Scope{Type: rbac.ScopeConnection, ID: r.ID})
		if err != nil {
			return result, err
		}
		if !ok {
			continue
		}
		v, err := s.view(ctx, tx, actor, r)
		if err != nil {
			return result, err
		}
		result.Connections = append(result.Connections, v)
	}
	if len(result.Connections) == 0 {
		rows, err := tx.QueryContext(ctx, `SELECT id FROM units`)
		if err != nil {
			return result, err
		}
		ids := []uuid.UUID{}
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return result, err
			}
			ids = append(ids, id)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return result, err
		}
		allowed := false
		for _, id := range ids {
			ok, err := engine(tx).Can(ctx, actor, rbac.ConnectionRead, rbac.UnitScope(id))
			if err != nil {
				return result, err
			}
			if !ok {
				ok, err = engine(tx).Can(ctx, actor, rbac.ConnectionManage, rbac.UnitScope(id))
				if err != nil {
					return result, err
				}
			}
			if ok {
				allowed = true
				break
			}
		}
		if !allowed {
			return result, rbac.ErrForbidden
		}
	}
	return result, tx.Commit()
}
func nullable(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}
func (s *Service) Get(ctx context.Context, actor, id uuid.UUID) (Result, error) {
	result := Result{}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return result, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := require(ctx, tx, actor, rbac.ConnectionRead, rbac.Scope{Type: rbac.ScopeConnection, ID: id}); err != nil {
		return result, err
	}
	r, err := scanRecord(tx.QueryRowContext(ctx, `SELECT `+recordColumns+` FROM connections WHERE id=$1`, id))
	if err != nil {
		return result, err
	}
	result.Connection, err = s.view(ctx, tx, actor, r)
	if err != nil {
		return result, err
	}
	return result, tx.Commit()
}
func (s *Service) Create(ctx context.Context, actor uuid.UUID, in CreateInput) (Result, error) {
	if err := require(ctx, s.db, actor, rbac.ConnectionManage, rbac.UnitScope(in.UnitID)); err != nil {
		return Result{}, err
	}
	if err := validate(&in.Input); err != nil {
		return Result{}, err
	}
	if len(in.Password) > 4096 {
		return Result{}, rbac.ErrInvalidInput
	}
	id := uuid.New()
	tx, err := s.locked(ctx, id)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := rbac.Lock(ctx, tx); err != nil {
		return Result{}, err
	}
	if err := require(ctx, tx, actor, rbac.ConnectionManage, rbac.UnitScope(in.UnitID)); err != nil {
		return Result{}, err
	}
	encrypted, nonce, err := s.vault.encrypt(id, in.Password)
	if err != nil {
		return Result{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO connections(id,unit_id,name,engine,host,port,database,username,ssl_mode,encrypted_password,nonce,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, id, in.UnitID, in.Name, in.Engine, in.Host, in.Port, in.Database, in.Username, in.SSLMode, encrypted, nonce, actor)
	if err != nil {
		return Result{}, dbError(err)
	}
	if err := s.audit(ctx, tx, actor, id, "connection.created", nil); err != nil {
		return Result{}, err
	}
	r := record{Input: in.Input, ID: id, status: "unknown"}
	v, err := s.view(ctx, tx, actor, r)
	if err != nil {
		return Result{}, err
	}
	return Result{Connection: v}, s.commit(tx, actor, id, "connection.created")
}

// Ownership changes use unit moves, whose existing dry-run reports access impact.
func (s *Service) Update(ctx context.Context, actor, id uuid.UUID, in Input) (Result, error) {
	return s.update(ctx, actor, id, &in, nil)
}
func (s *Service) ChangePassword(ctx context.Context, actor, id uuid.UUID, password string) (Result, error) {
	return s.update(ctx, actor, id, nil, &password)
}
func (s *Service) update(ctx context.Context, actor, id uuid.UUID, in *Input, password *string) (Result, error) {
	if err := s.manageOwner(ctx, actor, id); err != nil {
		return Result{}, err
	}
	tx, err := s.locked(ctx, id)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := rbac.Lock(ctx, tx); err != nil {
		return Result{}, err
	}
	r, err := scanRecord(tx.QueryRowContext(ctx, `SELECT `+recordColumns+` FROM connections WHERE id=$1`, id))
	if errors.Is(err, rbac.ErrNotFound) {
		return Result{}, rbac.ErrForbidden
	}
	if err != nil {
		return Result{}, err
	}
	if err := require(ctx, tx, actor, rbac.ConnectionManage, rbac.UnitScope(r.UnitID)); err != nil {
		return Result{}, err
	}
	action := "connection.updated"
	if in != nil {
		if err := validate(in); err != nil {
			return Result{}, err
		}
		if in.UnitID != r.UnitID {
			return Result{}, rbac.ErrInvalidInput
		}
		_, err = tx.ExecContext(ctx, `UPDATE connections SET name=$2,engine=$3,host=$4,port=$5,database=$6,username=$7,ssl_mode=$8,status='unknown',last_error=NULL,updated_at=clock_timestamp() WHERE id=$1`, id, in.Name, in.Engine, in.Host, in.Port, in.Database, in.Username, in.SSLMode)
		r.Input = *in
	} else {
		if len(*password) > 4096 {
			return Result{}, rbac.ErrInvalidInput
		}
		encrypted, nonce, e := s.vault.encrypt(id, *password)
		if e != nil {
			return Result{}, e
		}
		_, err = tx.ExecContext(ctx, `UPDATE connections SET encrypted_password=$2,nonce=$3,status='unknown',last_error=NULL,updated_at=clock_timestamp() WHERE id=$1`, id, encrypted, nonce)
		action = "connection.password_changed"
	}
	if err != nil {
		return Result{}, dbError(err)
	}
	if err := s.audit(ctx, tx, actor, id, action, nil); err != nil {
		return Result{}, err
	}
	r.status = "unknown"
	r.lastError = nil
	v, err := s.view(ctx, tx, actor, r)
	if err != nil {
		return Result{}, err
	}
	return Result{Connection: v}, s.commit(tx, actor, id, action)
}
func deletionImpact(ctx context.Context, q rbac.Queryer, r record) (DeleteImpact, error) {
	impact := DeleteImpact{ConnectionID: r.ID, Name: r.Name, Users: []UserRef{}}
	if err := q.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM schema_snapshots WHERE connection_id=$1),(SELECT count(*) FROM assignments WHERE scope_type='connection' AND scope_id=$1)`, r.ID).Scan(&impact.SnapshotCount, &impact.AssignmentCount); err != nil {
		return impact, err
	}
	rows, err := q.QueryContext(ctx, `SELECT DISTINCT u.id,u.email FROM assignments a JOIN users u ON u.id=a.user_id WHERE a.scope_type='connection' AND a.scope_id=$1 ORDER BY u.email,u.id`, r.ID)
	if err != nil {
		return impact, err
	}
	defer rows.Close()
	for rows.Next() {
		var u UserRef
		if err := rows.Scan(&u.ID, &u.Email); err != nil {
			return impact, err
		}
		impact.Users = append(impact.Users, u)
	}
	return impact, rows.Err()
}
func (s *Service) Delete(ctx context.Context, actor, id uuid.UUID, name *string) (DeleteImpact, error) {
	if err := s.manageOwner(ctx, actor, id); err != nil {
		return DeleteImpact{}, err
	}
	tx, err := s.locked(ctx, id)
	if err != nil {
		return DeleteImpact{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := rbac.Lock(ctx, tx); err != nil {
		return DeleteImpact{}, err
	}
	r, err := scanRecord(tx.QueryRowContext(ctx, `SELECT `+recordColumns+` FROM connections WHERE id=$1`, id))
	if errors.Is(err, rbac.ErrNotFound) {
		return DeleteImpact{}, rbac.ErrForbidden
	}
	if err != nil {
		return DeleteImpact{}, err
	}
	if err := require(ctx, tx, actor, rbac.ConnectionManage, rbac.UnitScope(r.UnitID)); err != nil {
		return DeleteImpact{}, err
	}
	impact, err := deletionImpact(ctx, tx, r)
	if err != nil {
		return impact, err
	}
	if name == nil {
		return impact, tx.Commit()
	}
	if *name != r.Name {
		return impact, ErrConfirmation
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM connections WHERE id=$1`, id); err != nil {
		return impact, err
	}
	if err := s.audit(ctx, tx, actor, id, "connection.deleted", map[string]any{"name": r.Name, "snapshots_removed": impact.SnapshotCount, "assignments_removed": impact.AssignmentCount}); err != nil {
		return impact, err
	}
	return impact, s.commit(tx, actor, id, "connection.deleted")
}
