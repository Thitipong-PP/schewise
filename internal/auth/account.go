// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Thitipong-PP/schewise/internal/onetimetoken"
	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/google/uuid"
)

var (
	ErrCurrentPassword = errors.New("current password is incorrect")
	ErrAccountInactive = errors.New("reactivate the account first")
	ErrProfileInput    = errors.New("display name must contain 1–120 characters")
)

type AccountConfig struct {
	DisableSessionIP bool
	ResetTTL         time.Duration
}

// ConfigureAccount is startup-only; configure before sharing the service.
func (s *Service) ConfigureAccount(config AccountConfig) error {
	if config.ResetTTL == 0 {
		config.ResetTTL = 24 * time.Hour
	}
	if config.ResetTTL < time.Minute || config.ResetTTL > 7*24*time.Hour {
		return errors.New("password reset lifetime must be between 1m and 168h")
	}
	s.account = config
	return nil
}

// SessionIP preserves the supplied peer IP or omits it when privacy is enabled.
// HTTP/audit logging uses the same choice so disabling storage is meaningful.
func (s *Service) SessionIP(ip string) string {
	if s.account.DisableSessionIP {
		return ""
	}
	return ip
}

func truncatedAgent(agent string) string {
	runes := []rune(strings.ToValidUTF8(agent, ""))
	if len(runes) > 256 {
		runes = runes[:256]
	}
	return string(runes)
}

func coarseAgent(agent string) string {
	browser := "Unknown browser"
	switch {
	case strings.Contains(agent, "Edg/"):
		browser = "Edge"
	case strings.Contains(agent, "Firefox/") || strings.Contains(agent, "FxiOS/"):
		browser = "Firefox"
	case strings.Contains(agent, "Chrome/") || strings.Contains(agent, "CriOS/"):
		browser = "Chrome"
	case strings.Contains(agent, "Safari/"):
		browser = "Safari"
	}
	platform := "Unknown device"
	switch {
	case strings.Contains(agent, "Android"):
		platform = "Android"
	case strings.Contains(agent, "iPhone") || strings.Contains(agent, "iPad"):
		platform = "iOS"
	case strings.Contains(agent, "Windows"):
		platform = "Windows"
	case strings.Contains(agent, "Macintosh") || strings.Contains(agent, "Mac OS X"):
		platform = "macOS"
	case strings.Contains(agent, "Linux"):
		platform = "Linux"
	}
	return browser + " · " + platform
}

func (s *Service) insertSession(ctx context.Context, tx *sql.Tx, session *Session, event Event) error {
	var ip any
	if value := s.SessionIP(event.RemoteIP); value != "" {
		ip = value
	}
	return tx.QueryRowContext(ctx, `INSERT INTO sessions(token_hash,user_id,csrf_token,expires_at,user_agent,ip) VALUES($1,$2,$3,$4,$5,$6) RETURNING id`, tokenHash(session.Token), session.User.ID, session.CSRF, session.ExpiresAt, truncatedAgent(event.UserAgent), ip).Scan(&session.ID)
}

// sessionUser verifies live ownership independently of any caller-supplied label
// or ID. Mutations take the shared lock before checking and changing credentials.
func sessionUser(ctx context.Context, db rbac.Queryer, session *Session) (*User, error) {
	if session == nil || session.User == nil || !validToken(session.Token) {
		return nil, ErrUnauthorized
	}
	user := &User{}
	err := db.QueryRowContext(ctx, `SELECT u.id,u.email,u.display_name,`+roleDisplay+` FROM users u JOIN sessions s ON s.user_id=u.id WHERE s.token_hash=$1 AND u.id=$2 AND u.is_active AND s.expires_at>clock_timestamp()`, tokenHash(session.Token), session.User.ID).Scan(&user.ID, &user.Email, &user.DisplayName, &user.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUnauthorized
	}
	return user, err
}

func (s *Service) Profile(ctx context.Context, current *Session) (*User, error) {
	return sessionUser(ctx, s.db, current)
}

func (s *Service) UpdateProfile(ctx context.Context, current *Session, name string, event Event) (*User, error) {
	name = strings.TrimSpace(name)
	if !utf8.ValidString(name) || name == "" || utf8.RuneCountInString(name) > 120 {
		return nil, ErrProfileInput
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := rbac.Lock(ctx, tx); err != nil {
		return nil, err
	}
	user, err := sessionUser(ctx, tx, current)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE users SET display_name=$2 WHERE id=$1`, user.ID, name); err != nil {
		return nil, err
	}
	event.ActorID, event.TargetID, event.Action, event.Outcome = user.ID, user.ID, "user.profile_updated", "success"
	if err := writeEvent(ctx, tx, event); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	s.LogEvent(event)
	user.DisplayName = name
	return user, nil
}

func (s *Service) ChangePassword(ctx context.Context, current *Session, oldPassword, newPassword string, event Event) error {
	if !ValidPassword(newPassword) {
		return ErrInvalidInput
	}
	user, err := s.Profile(ctx, current)
	if err != nil {
		return err
	}
	if !s.acquire() {
		return ErrBusy
	}
	defer func() { <-s.hashSlots }()
	var before string
	if err := s.db.QueryRowContext(ctx, `SELECT password_hash FROM users WHERE id=$1`, user.ID).Scan(&before); err != nil {
		return err
	}
	if !VerifyPassword(oldPassword, before) {
		event.ActorID, event.TargetID, event.Action, event.Outcome = user.ID, user.ID, "user.password_change_failed", "invalid_current_password"
		if err := s.Audit(ctx, event); err != nil {
			return err
		}
		return ErrCurrentPassword
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := rbac.Lock(ctx, tx); err != nil {
		return err
	}
	if _, err := sessionUser(ctx, tx, current); err != nil {
		return err
	}
	var live string
	if err := tx.QueryRowContext(ctx, `SELECT password_hash FROM users WHERE id=$1 FOR UPDATE`, user.ID).Scan(&live); err != nil {
		return err
	}
	if live != before {
		return ErrCurrentPassword
	}
	if _, err := tx.ExecContext(ctx, `UPDATE users SET password_hash=$2 WHERE id=$1`, user.ID, hash); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE password_resets SET revoked_at=clock_timestamp() WHERE user_id=$1 AND used_at IS NULL AND revoked_at IS NULL`, user.ID); err != nil {
		return err
	}
	deleted, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id=$1 AND token_hash<>$2`, user.ID, tokenHash(current.Token))
	if err != nil {
		return err
	}
	count, err := deleted.RowsAffected()
	if err != nil {
		return err
	}
	if err := sessionsAudit(ctx, tx, user.ID, "session.revoke_others", count); err != nil {
		return err
	}
	event.ActorID, event.TargetID, event.Action, event.Outcome = user.ID, user.ID, "user.password_changed", "success"
	if err := writeEvent(ctx, tx, event); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.LogEvent(event)
	return nil
}

type SessionInfo struct {
	ID         uuid.UUID `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	UserAgent  string    `json:"user_agent"`
	IP         *string   `json:"ip"`
	Current    bool      `json:"current"`
}
type SessionList struct {
	Sessions []SessionInfo `json:"sessions"`
}

func (s *Service) Sessions(ctx context.Context, current *Session) (SessionList, error) {
	list := SessionList{Sessions: []SessionInfo{}}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return list, err
	}
	defer func() { _ = tx.Rollback() }()
	user, err := sessionUser(ctx, tx, current)
	if err != nil {
		return list, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,created_at,last_seen_at,expires_at,user_agent,ip,token_hash=$2 FROM sessions WHERE user_id=$1 AND expires_at>clock_timestamp() ORDER BY created_at DESC,id DESC`, user.ID, tokenHash(current.Token))
	if err != nil {
		return list, err
	}
	defer rows.Close()
	for rows.Next() {
		var item SessionInfo
		var agent string
		if err := rows.Scan(&item.ID, &item.CreatedAt, &item.LastSeenAt, &item.ExpiresAt, &agent, &item.IP, &item.Current); err != nil {
			return list, err
		}
		item.UserAgent = coarseAgent(agent)
		if s.account.DisableSessionIP {
			item.IP = nil
		}
		list.Sessions = append(list.Sessions, item)
	}
	if err := rows.Err(); err != nil {
		return list, err
	}
	return list, tx.Commit()
}

func sessionsAudit(ctx context.Context, tx *sql.Tx, userID, action string, count int64) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return rbac.WriteAudit(ctx, tx, rbac.Audit{ActorUserID: id, Action: action, TargetType: "user", TargetID: id, Details: map[string]any{"count": count}})
}

func (s *Service) RevokeOwnSessions(ctx context.Context, current *Session, id *uuid.UUID, event Event) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := rbac.Lock(ctx, tx); err != nil {
		return false, err
	}
	user, err := sessionUser(ctx, tx, current)
	if err != nil {
		return false, err
	}
	var result sql.Result
	self := false
	if id != nil {
		var digest []byte
		err := tx.QueryRowContext(ctx, `SELECT token_hash FROM sessions WHERE id=$1 AND user_id=$2`, id, user.ID).Scan(&digest)
		if errors.Is(err, sql.ErrNoRows) {
			return false, ErrNotFound
		}
		if err != nil {
			return false, err
		}
		self = onetimetoken.Matches(current.Token, digest)
		result, err = tx.ExecContext(ctx, `DELETE FROM sessions WHERE id=$1 AND user_id=$2`, id, user.ID)
	} else {
		result, err = tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id=$1 AND token_hash<>$2`, user.ID, tokenHash(current.Token))
	}
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	action := "session.revoke_others"
	if id != nil {
		action = "session.revoke_own"
	}
	if err := sessionsAudit(ctx, tx, user.ID, action, count); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	event.ActorID, event.TargetID, event.Action, event.Outcome = user.ID, user.ID, action, "success"
	s.LogEvent(event)
	return self, nil
}
