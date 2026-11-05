// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Thitipong-PP/schewise/internal/onetimetoken"
	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

var (
	ErrInviteInvalid  = errors.New("invitation unavailable")
	ErrInviteConflict = errors.New("pending invitation already exists")
	ErrInviteAccount  = errors.New("account exists; assign the role directly")
)

// Mailer is optional. No email transport is installed; links are returned once
// to the creating administrator. Implementations must never log the link.
type Mailer interface {
	SendInvite(context.Context, InviteMessage) error
}
type InviteMessage struct {
	Email, Link string
	ExpiresAt   time.Time
}
type NoopMailer struct{}

func (NoopMailer) SendInvite(context.Context, InviteMessage) error { return nil }

type InviteConfig struct {
	PublicBaseURL string
	TTL           time.Duration
	Mailer        Mailer
}

func (s *Service) configureInvites(config InviteConfig) error {
	if config.PublicBaseURL == "" {
		config.PublicBaseURL = "https://localhost:8443"
	}
	parsed, err := url.Parse(config.PublicBaseURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("PUBLIC_BASE_URL must be an HTTPS origin without credentials, path, query, or fragment")
	}
	if config.TTL == 0 {
		config.TTL = 7 * 24 * time.Hour
	}
	if config.TTL < time.Minute || config.TTL > 30*24*time.Hour {
		return errors.New("invite lifetime must be between 1m and 720h")
	}
	if config.Mailer == nil {
		config.Mailer = NoopMailer{}
	}
	s.invites = config
	s.invites.PublicBaseURL = strings.TrimRight(config.PublicBaseURL, "/")
	return nil
}

type Invite struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	RoleID       uuid.UUID  `json:"role_id"`
	RoleName     string     `json:"role_name"`
	Scope        rbac.Scope `json:"scope"`
	ScopeName    string     `json:"scope_name"`
	Note         string     `json:"note"`
	CreatedBy    uuid.UUID  `json:"created_by"`
	CreatedAt    time.Time  `json:"created_at"`
	ExpiresAt    time.Time  `json:"expires_at"`
	UsedAt       *time.Time `json:"used_at"`
	RevokedAt    *time.Time `json:"revoked_at"`
	UsedByUserID *uuid.UUID `json:"used_by_user_id"`
	Status       string     `json:"status"`
}
type InviteInput struct {
	Email  string     `json:"email"`
	RoleID uuid.UUID  `json:"role_id"`
	Scope  rbac.Scope `json:"scope"`
	Note   string     `json:"note"`
}
type InviteLink struct {
	Invite Invite `json:"invite"`
	Link   string `json:"link"`
}
type InvitePage struct {
	Invites    []Invite   `json:"invites"`
	NextBefore *uuid.UUID `json:"next_before,omitempty"`
}
type InviteDisplay struct {
	Email     string    `json:"email"`
	RoleName  string    `json:"role_name"`
	ScopeName string    `json:"scope_name"`
	ExpiresAt time.Time `json:"expires_at"`
}

const inviteColumns = `i.id,i.email,i.role_id,COALESCE(r.name,'Deleted role'),i.scope_type,COALESCE(i.scope_id,'00000000-0000-4000-8000-000000000001'::uuid),COALESCE(u.name,'Unavailable scope'),i.note,i.created_by,i.created_at,i.expires_at,i.used_at,i.revoked_at,i.used_by_user_id,CASE WHEN i.used_at IS NOT NULL THEN 'used' WHEN i.revoked_at IS NOT NULL THEN 'revoked' WHEN i.expires_at<=clock_timestamp() THEN 'expired' ELSE 'pending' END`
const inviteFrom = ` FROM invites i LEFT JOIN roles r ON r.id=i.role_id LEFT JOIN units u ON i.scope_type='unit' AND u.id=COALESCE(i.scope_id,'00000000-0000-4000-8000-000000000001'::uuid)`

type inviteScanner interface{ Scan(...any) error }

func scanInvite(row inviteScanner) (Invite, error) {
	var value Invite
	err := row.Scan(&value.ID, &value.Email, &value.RoleID, &value.RoleName, &value.Scope.Type, &value.Scope.ID, &value.ScopeName, &value.Note, &value.CreatedBy, &value.CreatedAt, &value.ExpiresAt, &value.UsedAt, &value.RevokedAt, &value.UsedByUserID, &value.Status)
	return value, err
}
func inviteScopeID(scope rbac.Scope) any {
	if scope == rbac.RootScope() {
		return nil
	}
	return scope.ID
}

func inviteAudit(ctx context.Context, tx *sql.Tx, actor uuid.UUID, action string, invite Invite) error {
	return rbac.WriteAudit(ctx, tx, rbac.Audit{ActorUserID: actor, Action: action, TargetType: "invite", TargetID: invite.ID, Details: map[string]any{"email": invite.Email, "role_id": invite.RoleID, "scope": invite.Scope}})
}
func (s *Service) logInvite(actor uuid.UUID, action string, invite Invite) {
	s.logger.Info("audit", zap.String("actor_id", actor.String()), zap.String("action", action), zap.String("target_type", "invite"), zap.String("target_id", invite.ID.String()), zap.String("outcome", "success"))
}
func (s *Service) deliverInvite(ctx context.Context, value InviteLink) {
	if err := s.invites.Mailer.SendInvite(ctx, InviteMessage{Email: value.Invite.Email, Link: value.Link, ExpiresAt: value.Invite.ExpiresAt}); err != nil {
		// Transport errors may include the token. The committed invite and its
		// one-time response remain valid even if an optional mailer fails.
		s.logger.Warn("invite.delivery_failed", zap.String("invite_id", value.Invite.ID.String()))
	}
}

func validateInviteGrant(ctx context.Context, tx *sql.Tx, actor uuid.UUID, roleID uuid.UUID, scope rbac.Scope) error {
	reader := rbac.NewSQLReader(tx, nil)
	role, err := reader.Role(ctx, roleID)
	if err != nil {
		return err
	}
	if scope.Type != rbac.ScopeUnit || scope.ID == uuid.Nil {
		return rbac.ErrInvalidScope
	}
	if _, err := reader.Unit(ctx, scope.ID); err != nil {
		return err
	}
	for _, permission := range role.Permissions {
		if permission.SystemLevel() && scope != rbac.RootScope() {
			return rbac.ErrInvalidScope
		}
	}
	allowed, err := rbac.NewEngine(reader).CanAssign(ctx, actor, roleID, scope)
	if err != nil {
		return err
	}
	if !allowed {
		return rbac.ErrForbidden
	}
	return nil
}

func (s *Service) insertInvite(ctx context.Context, tx *sql.Tx, actor uuid.UUID, input InviteInput) (InviteLink, error) {
	var value InviteLink
	email, err := normalizeEmail(input.Email)
	if err != nil || !utf8.ValidString(input.Note) || utf8.RuneCountInString(input.Note) > 2000 {
		return value, ErrInvalidInput
	}
	if err := validateInviteGrant(ctx, tx, actor, input.RoleID, input.Scope); err != nil {
		return value, err
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)`, email).Scan(&exists); err != nil {
		return value, err
	}
	if exists {
		return value, ErrInviteAccount
	}
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM invites WHERE email=$1 AND used_at IS NULL AND revoked_at IS NULL AND expires_at>clock_timestamp())`, email).Scan(&exists); err != nil {
		return value, err
	}
	if exists {
		return value, ErrInviteConflict
	}
	token, err := onetimetoken.Generate()
	if err != nil {
		return value, err
	}
	id := uuid.New()
	_, err = tx.ExecContext(ctx, `INSERT INTO invites(id,email,role_id,scope_type,scope_id,token_hash,note,created_by,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, id, email, input.RoleID, input.Scope.Type, inviteScopeID(input.Scope), onetimetoken.Hash(token), input.Note, actor, time.Now().Add(s.invites.TTL))
	if err != nil {
		return value, err
	}
	value.Invite, err = scanInvite(tx.QueryRowContext(ctx, `SELECT `+inviteColumns+inviteFrom+` WHERE i.id=$1`, id))
	value.Link = s.invites.PublicBaseURL + "/invite/" + token
	return value, err
}

func (s *Service) CreateInvite(ctx context.Context, actor *User, input InviteInput) (InviteLink, error) {
	var value InviteLink
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return value, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := rbac.Lock(ctx, tx); err != nil {
		return value, err
	}
	if err := authorizeManagement(ctx, tx, actor, false); err != nil {
		return value, err
	}
	id := uuid.MustParse(actor.ID)
	value, err = s.insertInvite(ctx, tx, id, input)
	if err != nil {
		return InviteLink{}, err
	}
	if err := inviteAudit(ctx, tx, id, "invite.created", value.Invite); err != nil {
		return InviteLink{}, err
	}
	if err := tx.Commit(); err != nil {
		return InviteLink{}, err
	}
	s.logInvite(id, "invite.created", value.Invite)
	s.deliverInvite(ctx, value)
	return value, nil
}

func (s *Service) ListInvites(ctx context.Context, actor *User, before uuid.UUID, limit int) (InvitePage, error) {
	page := InvitePage{Invites: []Invite{}}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return page, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := authorizeManagement(ctx, tx, actor, false); err != nil {
		return page, err
	}
	if limit < 1 || limit > 200 {
		return page, ErrInvalidInput
	}
	rows, err := tx.QueryContext(ctx, `SELECT `+inviteColumns+inviteFrom+` WHERE ($1::uuid='00000000-0000-0000-0000-000000000000' OR (i.created_at,i.id)<(SELECT created_at,id FROM invites WHERE id=$1)) ORDER BY i.created_at DESC,i.id DESC LIMIT $2`, before, limit+1)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		value, err := scanInvite(rows)
		if err != nil {
			return page, err
		}
		page.Invites = append(page.Invites, value)
	}
	if err := rows.Err(); err != nil {
		return page, err
	}
	if len(page.Invites) > limit {
		page.Invites = page.Invites[:limit]
		id := page.Invites[limit-1].ID
		page.NextBefore = &id
	}
	return page, tx.Commit()
}

func (s *Service) changeInvite(ctx context.Context, actor *User, id uuid.UUID, regenerate bool) (InviteLink, error) {
	var value InviteLink
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return value, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := rbac.Lock(ctx, tx); err != nil {
		return value, err
	}
	if err := authorizeManagement(ctx, tx, actor, false); err != nil {
		return value, err
	}
	old, err := scanInvite(tx.QueryRowContext(ctx, `SELECT `+inviteColumns+inviteFrom+` WHERE i.id=$1 FOR UPDATE OF i`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return value, ErrNotFound
	}
	if err != nil {
		return value, err
	}
	if old.UsedAt != nil || old.RevokedAt != nil {
		return value, ErrInviteInvalid
	}
	if _, err := tx.ExecContext(ctx, `UPDATE invites SET revoked_at=clock_timestamp() WHERE id=$1`, id); err != nil {
		return value, err
	}
	actorID := uuid.MustParse(actor.ID)
	if err := inviteAudit(ctx, tx, actorID, "invite.revoked", old); err != nil {
		return value, err
	}
	if regenerate {
		value, err = s.insertInvite(ctx, tx, actorID, InviteInput{Email: old.Email, RoleID: old.RoleID, Scope: old.Scope, Note: old.Note})
		if err != nil {
			return InviteLink{}, err
		}
		if err := inviteAudit(ctx, tx, actorID, "invite.regenerated", value.Invite); err != nil {
			return InviteLink{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return InviteLink{}, err
	}
	s.logInvite(actorID, "invite.revoked", old)
	if regenerate {
		s.logInvite(actorID, "invite.regenerated", value.Invite)
		s.deliverInvite(ctx, value)
	}
	return value, nil
}
func (s *Service) RevokeInvite(ctx context.Context, actor *User, id uuid.UUID) error {
	_, err := s.changeInvite(ctx, actor, id, false)
	return err
}
func (s *Service) RegenerateInvite(ctx context.Context, actor *User, id uuid.UUID) (InviteLink, error) {
	return s.changeInvite(ctx, actor, id, true)
}

func loadValidInvite(ctx context.Context, tx *sql.Tx, token string) (Invite, error) {
	var empty Invite
	if !onetimetoken.Valid(token) {
		return empty, ErrInviteInvalid
	}
	var digest []byte
	if err := tx.QueryRowContext(ctx, `SELECT token_hash FROM invites WHERE token_hash=$1 AND used_at IS NULL AND revoked_at IS NULL AND expires_at>clock_timestamp()`, onetimetoken.Hash(token)).Scan(&digest); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return empty, ErrInviteInvalid
		}
		return empty, err
	}
	if !onetimetoken.Matches(token, digest) {
		return empty, ErrInviteInvalid
	}
	value, err := scanInvite(tx.QueryRowContext(ctx, `SELECT `+inviteColumns+inviteFrom+` WHERE i.token_hash=$1`, digest))
	return value, err
}
func revalidateInvite(ctx context.Context, tx *sql.Tx, invite Invite) error {
	// The inviter must still be active and able to grant the current role. This
	// also prevents role edits from turning an old link into an escalation.
	if err := authorizeManagement(ctx, tx, &User{ID: invite.CreatedBy.String()}, false); err != nil {
		if errors.Is(err, ErrForbidden) {
			return ErrInviteInvalid
		}
		return err
	}
	if err := validateInviteGrant(ctx, tx, invite.CreatedBy, invite.RoleID, invite.Scope); err != nil {
		if errors.Is(err, rbac.ErrForbidden) || errors.Is(err, rbac.ErrNotFound) || errors.Is(err, rbac.ErrInvalidScope) || errors.Is(err, rbac.ErrUnavailable) {
			return ErrInviteInvalid
		}
		return err
	}
	return nil
}

func (s *Service) InspectInvite(ctx context.Context, token string) (InviteDisplay, error) {
	var display InviteDisplay
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return display, err
	}
	defer func() { _ = tx.Rollback() }()
	invite, err := loadValidInvite(ctx, tx, token)
	if err != nil {
		return display, err
	}
	if err := revalidateInvite(ctx, tx, invite); err != nil {
		return display, err
	}
	display = InviteDisplay{Email: invite.Email, RoleName: invite.RoleName, ScopeName: invite.ScopeName, ExpiresAt: invite.ExpiresAt}
	return display, tx.Commit()
}

func (s *Service) AcceptInvite(ctx context.Context, token, name, password string, previous *Session, event Event) (*Session, error) {
	if previous == nil {
		return nil, ErrUnauthorized
	}
	if _, err := s.InspectInvite(ctx, token); err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" || !utf8.ValidString(name) || utf8.RuneCountInString(name) > 120 || !ValidPassword(password) {
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
	invite, err := loadValidInvite(ctx, tx, token)
	if err != nil {
		return nil, err
	}
	if err := revalidateInvite(ctx, tx, invite); err != nil {
		return nil, err
	}
	user := &User{Email: invite.Email, DisplayName: name, Role: invite.RoleName}
	err = tx.QueryRowContext(ctx, `INSERT INTO users(email,password_hash,display_name) VALUES($1,$2,$3) RETURNING id`, invite.Email, hash, name).Scan(&user.ID)
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			return nil, ErrInviteInvalid
		}
		return nil, err
	}
	assignmentID := uuid.New()
	if _, err := tx.ExecContext(ctx, `INSERT INTO assignments(id,user_id,role_id,scope_type,scope_id,note,created_by) VALUES($1,$2,$3,$4,$5,$6,$7)`, assignmentID, user.ID, invite.RoleID, invite.Scope.Type, invite.Scope.ID, invite.Note, invite.CreatedBy); err != nil {
		return nil, err
	}
	if err := rbac.WriteAudit(ctx, tx, rbac.Audit{ActorUserID: invite.CreatedBy, Action: "assignment.create", TargetType: "assignment", TargetID: assignmentID, Details: map[string]any{"user_id": user.ID, "role_id": invite.RoleID, "scope": invite.Scope, "invite_id": invite.ID}}); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE invites SET used_at=clock_timestamp(),used_by_user_id=$2 WHERE id=$1`, invite.ID, user.ID); err != nil {
		return nil, err
	}
	nextToken, err := RandomToken()
	if err != nil {
		return nil, err
	}
	csrf, err := RandomToken()
	if err != nil {
		return nil, err
	}
	next := &Session{Token: nextToken, CSRF: csrf, User: user, ExpiresAt: time.Now().Add(s.ttl)}
	res, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash=$1 AND expires_at>clock_timestamp()`, tokenHash(previous.Token))
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
	if _, err := tx.ExecContext(ctx, `INSERT INTO sessions(token_hash,user_id,csrf_token,expires_at) VALUES($1,$2,$3,$4)`, tokenHash(next.Token), user.ID, csrf, next.ExpiresAt); err != nil {
		return nil, err
	}
	userID := uuid.MustParse(user.ID)
	if err := inviteAudit(ctx, tx, userID, "invite.accepted", invite); err != nil {
		return nil, err
	}
	event.ActorID, event.TargetID, event.Action, event.Outcome = user.ID, user.ID, "auth.login", "success"
	if err := writeEvent(ctx, tx, event); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	s.logInvite(userID, "invite.accepted", invite)
	s.LogEvent(event)
	return next, nil
}
