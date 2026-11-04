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

	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

var (
	ErrUnauthorized = errors.New("authentication required")
	ErrCredentials  = errors.New("invalid email or password")
	ErrForbidden    = errors.New("administrator permission required")
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
	err := s.db.QueryRowContext(ctx, `SELECT s.csrf_token,s.expires_at,u.id,u.email,r.name
		FROM sessions s LEFT JOIN users u ON u.id=s.user_id LEFT JOIN roles r ON r.id=u.role_id
		WHERE s.token_hash=$1 AND s.expires_at>now()`, tokenHash(token)).Scan(&result.CSRF, &result.ExpiresAt, &userID, &email, &role)
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

type executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func writeEvent(ctx context.Context, db executor, event Event) error {
	_, err := db.ExecContext(ctx, `INSERT INTO audit_events(request_id,actor_id,target_id,action,outcome,remote_ip)
		VALUES($1,NULLIF($2,'')::uuid,NULLIF($3,'')::uuid,$4,$5,$6)`, event.RequestID, event.ActorID, event.TargetID, event.Action, event.Outcome, event.RemoteIP)
	return err
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
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(736395102)`); err != nil {
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
	err = tx.QueryRowContext(ctx, `INSERT INTO users(email,password_hash,role_id)
		VALUES($1,$2,(SELECT id FROM roles WHERE name='Administrator')) RETURNING id`, email, hash).Scan(&userID)
	if err != nil {
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
	err := s.db.QueryRowContext(ctx, `SELECT u.id,u.email,r.name,u.password_hash FROM users u JOIN roles r ON r.id=u.role_id WHERE u.email=$1`, email).Scan(&user.ID, &user.Email, &user.Role, &hash)
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
	if actor == nil {
		return nil, ErrUnauthorized
	}
	if actor.Role != "Administrator" {
		return nil, ErrForbidden
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
	user := &User{Email: email, Role: "Administrator"}
	err = tx.QueryRowContext(ctx, `INSERT INTO users(email,password_hash,role_id) VALUES($1,$2,(SELECT id FROM roles WHERE name='Administrator')) RETURNING id`, email, hash).Scan(&user.ID)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return nil, ErrConflict
	}
	if err != nil {
		return nil, err
	}
	event.ActorID = actor.ID
	event.TargetID = user.ID
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
	if actor.Role != "Administrator" {
		return ErrForbidden
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
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
