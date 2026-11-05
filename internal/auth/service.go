// Package auth implements server-side identity, sessions, and authorization.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

var (
	ErrUnauthorized = errors.New("authentication required")
	ErrCredentials  = errors.New("invalid email or password")
	ErrForbidden    = errors.New("required permission missing")
	ErrInvalidInput = errors.New("invalid input")
	ErrConflict     = errors.New("account already exists")
	ErrNotFound     = errors.New("user not found")
	ErrBusy         = errors.New("authentication capacity exceeded")
)

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type Session struct {
	Token     string    `json:"-"`
	CSRF      string    `json:"-"`
	User      *User     `json:"-"`
	ExpiresAt time.Time `json:"-"`
}

type Event struct {
	RequestID string
	ActorID   string
	TargetID  string
	Action    string
	Outcome   string
	RemoteIP  string
}

type Service struct {
	db        *sql.DB
	logger    *zap.Logger
	ttl       time.Duration
	dummyHash string
	hashSlots chan struct{}
}

func New(db *sql.DB, logger *zap.Logger, ttl time.Duration) (*Service, error) {
	// Unknown accounts perform the same password work as known accounts.
	dummy, err := HashPassword("Synthetic-dummy-password-42!")
	if err != nil {
		return nil, err
	}
	return &Service{db: db, logger: logger, ttl: ttl, dummyHash: dummy, hashSlots: make(chan struct{}, 4)}, nil
}

func normalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || len(email) > 254 {
		return "", ErrInvalidInput
	}
	return email, nil
}

func RandomToken() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func tokenHash(token string) []byte {
	hash := sha256.Sum256([]byte(token))
	return hash[:]
}

func validToken(token string) bool {
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(token)
	return err == nil && len(decoded) == 32
}

func (s *Service) Lookup(ctx context.Context, token string) (*Session, error) {
	if !validToken(token) {
		return nil, ErrUnauthorized
	}
	var userID, email, role sql.NullString
	result := &Session{Token: token}
	err := s.db.QueryRowContext(ctx, `SELECT s.csrf_token,s.expires_at,u.id,u.email,`+roleDisplay+`
 FROM sessions s LEFT JOIN users u ON u.id=s.user_id
 WHERE s.token_hash=$1 AND s.expires_at>now() AND (s.user_id IS NULL OR u.is_active)`, tokenHash(token)).Scan(&result.CSRF, &result.ExpiresAt, &userID, &email, &role)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	if userID.Valid {
		result.User = &User{ID: userID.String, Email: email.String, Role: role.String}
	}
	return result, nil
}

func (s *Service) Anonymous(ctx context.Context) (*Session, error) {
	token, err := RandomToken()
	if err != nil {
		return nil, err
	}
	csrf, err := RandomToken()
	if err != nil {
		return nil, err
	}
	result := &Session{Token: token, CSRF: csrf, ExpiresAt: time.Now().Add(15 * time.Minute)}
	_, err = s.db.ExecContext(ctx, `INSERT INTO sessions(token_hash,csrf_token,expires_at) VALUES($1,$2,$3)`, tokenHash(token), csrf, result.ExpiresAt)
	return result, err
}

func VerifyCSRF(session *Session, provided string) bool {
	return session != nil && validToken(provided) && subtle.ConstantTimeCompare([]byte(session.CSRF), []byte(provided)) == 1
}

// Role is retained as a display-only compatibility label, never an authority.
const roleDisplay = `COALESCE((SELECT string_agg(DISTINCT r.name, ', ' ORDER BY r.name) FROM assignments a JOIN roles r ON r.id=a.role_id WHERE a.user_id=u.id),'')`

type executor = rbac.Queryer

func (s *Service) Can(ctx context.Context, user *User, permission rbac.Permission, resource rbac.Scope) (bool, error) {
	if user == nil {
		return false, nil
	}
	id, err := uuid.Parse(user.ID)
	if err != nil {
		return false, nil
	}
	return rbac.NewEngine(rbac.NewSQLReader(s.db, nil)).Can(ctx, id, permission, resource)
}

func authorizeManagement(ctx context.Context, db rbac.Queryer, actor *User, grantsAdministrator bool) error {
	if actor == nil {
		return ErrUnauthorized
	}
	id, err := uuid.Parse(actor.ID)
	if err != nil {
		return ErrUnauthorized
	}
	engine := rbac.NewEngine(rbac.NewSQLReader(db, nil))
	allowed, err := engine.Can(ctx, id, rbac.UserManage, rbac.RootScope())
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	if grantsAdministrator {
		allowed, err = engine.CanAssign(ctx, id, uuid.MustParse(rbac.AdministratorRoleID), rbac.RootScope())
		if err != nil {
			return err
		}
		if !allowed {
			return ErrForbidden
		}
	}
	return nil
}

func grantAdministrator(ctx context.Context, tx *sql.Tx, userID, actorID string) error {
	user, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	actor, err := uuid.Parse(actorID)
	if err != nil {
		return err
	}
	var id uuid.UUID
	err = tx.QueryRowContext(ctx, `INSERT INTO assignments(user_id,role_id,scope_type,scope_id,created_by) VALUES($1,$2,'unit',$3,$4) RETURNING id`, user, uuid.MustParse(rbac.AdministratorRoleID), rbac.RootScope().ID, actor).Scan(&id)
	if err != nil {
		return err
	}
	return rbac.WriteAudit(ctx, tx, rbac.Audit{ActorUserID: actor, Action: "assignment.create", TargetType: "assignment", TargetID: id, Details: map[string]any{"user_id": user, "role_id": rbac.AdministratorRoleID, "scope": rbac.RootScope()}})
}

func writeEvent(ctx context.Context, db executor, event Event) error {
	actor, target := uuid.Nil, uuid.Nil
	var err error
	if event.ActorID != "" {
		actor, err = uuid.Parse(event.ActorID)
		if err != nil {
			return err
		}
	}
	if event.TargetID != "" {
		target, err = uuid.Parse(event.TargetID)
		if err != nil {
			return err
		}
	}
	targetType := "authentication"
	if target != uuid.Nil {
		targetType = "user"
	}
	return rbac.WriteAudit(ctx, db, rbac.Audit{ActorUserID: actor, Action: event.Action, TargetType: targetType, TargetID: target, Details: map[string]any{"request_id": event.RequestID, "outcome": event.Outcome, "remote_ip": event.RemoteIP}})
}

func (s *Service) LogEvent(event Event) {
	s.logger.Info("audit", zap.String("request_id", event.RequestID), zap.String("actor_id", event.ActorID),
		zap.String("target_id", event.TargetID), zap.String("action", event.Action), zap.String("outcome", event.Outcome), zap.String("remote_ip", event.RemoteIP))
}

func (s *Service) Audit(ctx context.Context, event Event) error {
	if err := writeEvent(ctx, s.db, event); err != nil {
		return err
	}
	s.LogEvent(event)
	return nil
}

func (s *Service) acquire() bool {
	select {
	case s.hashSlots <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s *Service) Bootstrap(ctx context.Context, email, password string) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := rbac.Lock(ctx, tx); err != nil {
		return false, err
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users)`).Scan(&exists); err != nil {
		return false, err
	}
	if exists {
		return false, tx.Commit()
	}
	email, err = normalizeEmail(email)
	if err != nil || !ValidPassword(password) {
		return false, errors.New("first installation requires a valid SCHEWISE_ADMIN_EMAIL and SCHEWISE_ADMIN_PASSWORD (12+ characters, at most 256 bytes)")
	}
	hash, err := HashPassword(password)
	if err != nil {
		return false, err
	}
	var userID string
	err = tx.QueryRowContext(ctx, `INSERT INTO users(email,password_hash) VALUES($1,$2) RETURNING id`, email, hash).Scan(&userID)
	if err != nil {
		return false, err
	}
	if err := grantAdministrator(ctx, tx, userID, userID); err != nil {
		return false, err
	}
	event := Event{RequestID: "installation", ActorID: userID, TargetID: userID, Action: "user.bootstrap", Outcome: "success"}
	if err := writeEvent(ctx, tx, event); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	s.LogEvent(event)
	return true, nil
}

func (s *Service) Login(ctx context.Context, previous *Session, email, password string, event Event) (*Session, error) {
	if previous == nil {
		return nil, ErrUnauthorized
	}
	if !s.acquire() {
		return nil, ErrBusy
	}
	defer func() { <-s.hashSlots }()
	email, normalizeErr := normalizeEmail(email)
	var user User
	var hash string
	err := s.db.QueryRowContext(ctx, `SELECT u.id,u.email,`+roleDisplay+`,u.password_hash FROM users u WHERE u.email=$1 AND u.is_active`, email).Scan(&user.ID, &user.Email, &user.Role, &hash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	known := err == nil && normalizeErr == nil
	if !known {
		hash = s.dummyHash
	}
	matched := VerifyPassword(password, hash)
	if !known || !matched {
		event.Action = "auth.login"
		event.Outcome = "failure"
		if err := s.Audit(ctx, event); err != nil {
			return nil, err
		}
		return nil, ErrCredentials
	}
	token, err := RandomToken()
	if err != nil {
		return nil, err
	}
	csrf, err := RandomToken()
	if err != nil {
		return nil, err
	}
	result := &Session{Token: token, CSRF: csrf, User: &user, ExpiresAt: time.Now().Add(s.ttl)}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	// Serialize with deactivation so a concurrent login cannot recreate sessions
	// after the user's sessions have been revoked.
	var active bool
	if err := tx.QueryRowContext(ctx, `SELECT is_active FROM users WHERE id=$1 FOR UPDATE`, user.ID).Scan(&active); err != nil {
		return nil, err
	}
	if !active {
		event.Action = "auth.login"
		event.Outcome = "failure"
		if err := writeEvent(ctx, tx, event); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		s.LogEvent(event)
		return nil, ErrCredentials
	}
	// Consume the pre-login session once. Concurrent login replays cannot reuse it.
	res, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash=$1 AND expires_at>now()`, tokenHash(previous.Token))
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n != 1 {
		return nil, ErrUnauthorized
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO sessions(token_hash,user_id,csrf_token,expires_at) VALUES($1,$2,$3,$4)`, tokenHash(token), user.ID, csrf, result.ExpiresAt)
	if err != nil {
		return nil, err
	}
	event.ActorID = user.ID
	event.TargetID = user.ID
	event.Action = "auth.login"
	event.Outcome = "success"
	if err := writeEvent(ctx, tx, event); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	s.LogEvent(event)
	return result, nil
}

func (s *Service) Register(ctx context.Context, actor *User, email, password string, event Event) (*User, error) {
	if err := authorizeManagement(ctx, s.db, actor, true); err != nil {
		return nil, err
	}
	email, err := normalizeEmail(email)
	if err != nil || !ValidPassword(password) {
		return nil, ErrInvalidInput
	}
	if !s.acquire() {
		return nil, ErrBusy
	}
	defer func() { <-s.hashSlots }()
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := rbac.Lock(ctx, tx); err != nil {
		return nil, err
	}
	if err := authorizeManagement(ctx, tx, actor, true); err != nil {
		return nil, err
	}
	user := &User{Email: email, Role: "Administrator"}
	err = tx.QueryRowContext(ctx, `INSERT INTO users(email,password_hash) VALUES($1,$2) RETURNING id`, email, hash).Scan(&user.ID)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return nil, ErrConflict
	}
	if err != nil {
		return nil, err
	}
	event.ActorID = actor.ID
	event.TargetID = user.ID
	if err := grantAdministrator(ctx, tx, user.ID, actor.ID); err != nil {
		return nil, err
	}
	event.Action = "user.register"
	event.Outcome = "success"
	if err := writeEvent(ctx, tx, event); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	s.LogEvent(event)
	return user, nil
}

func (s *Service) Logout(ctx context.Context, session *Session, event Event) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash=$1`, tokenHash(session.Token)); err != nil {
		return err
	}
	if session.User != nil {
		event.ActorID = session.User.ID
	}
	event.Action = "auth.logout"
	event.Outcome = "success"
	if err := writeEvent(ctx, tx, event); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.LogEvent(event)
	return nil
}

func (s *Service) RevokeUserSessions(ctx context.Context, actor *User, userID string, event Event) error {
	if actor == nil {
		return ErrUnauthorized
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := rbac.Lock(ctx, tx); err != nil {
		return err
	}
	if err := authorizeManagement(ctx, tx, actor, false); err != nil {
		return err
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1::uuid)`, userID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id=$1::uuid`, userID); err != nil {
		return err
	}
	event.ActorID = actor.ID
	event.TargetID = userID
	event.Action = "session.revoke_user"
	event.Outcome = "success"
	if err := writeEvent(ctx, tx, event); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.LogEvent(event)
	return nil
}

func (s *Service) Cleanup(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at<=now()`)
	return err
}
