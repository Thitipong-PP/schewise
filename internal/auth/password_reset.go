// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Thitipong-PP/schewise/internal/onetimetoken"
	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var ErrResetInvalid = errors.New("password reset unavailable")

type PasswordResetLink struct {
	Link      string    `json:"link"`
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
}
type PasswordResetDisplay struct {
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
}
type passwordReset struct {
	id, userID, createdBy uuid.UUID
	email                 string
	expiresAt             time.Time
}

func (s *Service) CreatePasswordReset(ctx context.Context, actor *User, userID uuid.UUID) (PasswordResetLink, error) {
	var link PasswordResetLink
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return link, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := rbac.Lock(ctx, tx); err != nil {
		return link, err
	}
	if err := authorizeManagement(ctx, tx, actor, false); err != nil {
		return link, err
	}
	var active bool
	err = tx.QueryRowContext(ctx, `SELECT email,is_active FROM users WHERE id=$1 FOR UPDATE`, userID).Scan(&link.Email, &active)
	if errors.Is(err, sql.ErrNoRows) {
		return link, ErrNotFound
	}
	if err != nil {
		return link, err
	}
	if !active {
		return link, ErrAccountInactive
	}
	if _, err := tx.ExecContext(ctx, `UPDATE password_resets SET revoked_at=clock_timestamp() WHERE user_id=$1 AND used_at IS NULL AND revoked_at IS NULL`, userID); err != nil {
		return link, err
	}
	token, err := onetimetoken.Generate()
	if err != nil {
		return link, err
	}
	id := uuid.New()
	actorID := uuid.MustParse(actor.ID)
	link.ExpiresAt = time.Now().Add(s.account.ResetTTL)
	if _, err := tx.ExecContext(ctx, `INSERT INTO password_resets(id,user_id,token_hash,created_by,expires_at) VALUES($1,$2,$3,$4,$5)`, id, userID, onetimetoken.Hash(token), actorID, link.ExpiresAt); err != nil {
		return PasswordResetLink{}, err
	}
	if err := rbac.WriteAudit(ctx, tx, rbac.Audit{ActorUserID: actorID, Action: "password_reset.created", TargetType: "user", TargetID: userID, Details: map[string]any{"reset_id": id, "expires_at": link.ExpiresAt}}); err != nil {
		return PasswordResetLink{}, err
	}
	if err := tx.Commit(); err != nil {
		return PasswordResetLink{}, err
	}
	s.logger.Info("audit", zap.String("actor_id", actor.ID), zap.String("target_id", userID.String()), zap.String("action", "password_reset.created"))
	link.Link = s.invites.PublicBaseURL + "/reset-password/" + token
	return link, nil
}

func loadPasswordReset(ctx context.Context, tx *sql.Tx, token string) (passwordReset, error) {
	var value passwordReset
	if !onetimetoken.Valid(token) {
		return value, ErrResetInvalid
	}
	var digest []byte
	err := tx.QueryRowContext(ctx, `SELECT p.id,p.user_id,p.created_by,u.email,p.expires_at,p.token_hash FROM password_resets p JOIN users u ON u.id=p.user_id WHERE p.token_hash=$1 AND p.used_at IS NULL AND p.revoked_at IS NULL AND p.expires_at>clock_timestamp() AND u.is_active`, onetimetoken.Hash(token)).Scan(&value.id, &value.userID, &value.createdBy, &value.email, &value.expiresAt, &digest)
	if errors.Is(err, sql.ErrNoRows) {
		return value, ErrResetInvalid
	}
	if err != nil {
		return value, err
	}
	if !onetimetoken.Matches(token, digest) {
		return value, ErrResetInvalid
	}
	// An old link must not outlive the issuer's account-management authority.
	if err := authorizeManagement(ctx, tx, &User{ID: value.createdBy.String()}, false); err != nil {
		if errors.Is(err, ErrForbidden) || errors.Is(err, ErrUnauthorized) {
			return value, ErrResetInvalid
		}
		return value, err
	}
	return value, nil
}

func (s *Service) InspectPasswordReset(ctx context.Context, token string) (PasswordResetDisplay, error) {
	var display PasswordResetDisplay
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return display, err
	}
	defer func() { _ = tx.Rollback() }()
	value, err := loadPasswordReset(ctx, tx, token)
	if err != nil {
		return display, err
	}
	display = PasswordResetDisplay{Email: value.email, ExpiresAt: value.expiresAt}
	return display, tx.Commit()
}

func (s *Service) ResetPassword(ctx context.Context, token, password string, event Event) (uuid.UUID, error) {
	if _, err := s.InspectPasswordReset(ctx, token); err != nil {
		return uuid.Nil, err
	}
	if !ValidPassword(password) {
		return uuid.Nil, ErrInvalidInput
	}
	if !s.acquire() {
		return uuid.Nil, ErrBusy
	}
	defer func() { <-s.hashSlots }()
	hash, err := HashPassword(password)
	if err != nil {
		return uuid.Nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := rbac.Lock(ctx, tx); err != nil {
		return uuid.Nil, err
	}
	value, err := loadPasswordReset(ctx, tx, token)
	if err != nil {
		return uuid.Nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE users SET password_hash=$2 WHERE id=$1`, value.userID, hash); err != nil {
		return uuid.Nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE password_resets SET used_at=clock_timestamp() WHERE id=$1`, value.id); err != nil {
		return uuid.Nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE password_resets SET revoked_at=clock_timestamp() WHERE user_id=$1 AND id<>$2 AND used_at IS NULL AND revoked_at IS NULL`, value.userID, value.id); err != nil {
		return uuid.Nil, err
	}
	deleted, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id=$1`, value.userID)
	if err != nil {
		return uuid.Nil, err
	}
	count, err := deleted.RowsAffected()
	if err != nil {
		return uuid.Nil, err
	}
	if err := sessionsAudit(ctx, tx, value.userID.String(), "session.revoke_all", count); err != nil {
		return uuid.Nil, err
	}
	if err := rbac.WriteAudit(ctx, tx, rbac.Audit{ActorUserID: value.userID, Action: "password_reset.used", TargetType: "user", TargetID: value.userID, Details: map[string]any{"reset_id": value.id, "created_by": value.createdBy}}); err != nil {
		return uuid.Nil, err
	}
	event.ActorID, event.TargetID, event.Action, event.Outcome = value.userID.String(), value.userID.String(), "user.password_changed", "success"
	if err := writeEvent(ctx, tx, event); err != nil {
		return uuid.Nil, err
	}
	if err := tx.Commit(); err != nil {
		return uuid.Nil, err
	}
	s.LogEvent(event)
	s.logger.Info("audit", zap.String("actor_id", value.userID.String()), zap.String("action", "password_reset.used"))
	return value.userID, nil
}
