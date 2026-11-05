// SPDX-License-Identifier: Apache-2.0

package httpapi

import (
	"errors"
	"strconv"
	"time"

	api "github.com/Thitipong-PP/schewise/internal/apicontract"
	"github.com/Thitipong-PP/schewise/internal/auth"
	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/google/uuid"
)

func (s *server) passwordUserLimit() fiber.Handler {
	return limiter.New(limiter.Config{Max: 5, Expiration: time.Minute, KeyGenerator: actorID, LimitReached: func(c fiber.Ctx) error {
		if err := s.Auth.Audit(c.Context(), event(c, "user.password_rate_limited", "denied")); err != nil {
			return s.failure(c, err)
		}
		return errorReply(c, 429, "rate_limited", "Too many attempts. Try again later.")
	}})
}
func (s *server) accountError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, auth.ErrResetInvalid):
		return errorReply(c, 400, "reset_unavailable", "This password reset link is unavailable. Ask your administrator for a new link.")
	case errors.Is(err, auth.ErrCurrentPassword):
		return errorReply(c, 400, "invalid_current_password", "The current password is incorrect.")
	case errors.Is(err, auth.ErrProfileInput):
		return errorReply(c, 400, "invalid_profile", "Enter a display name of 1–120 characters.")
	case errors.Is(err, auth.ErrAccountInactive):
		return errorReply(c, 409, "account_inactive", "Reactivate the account before creating a password reset link.")
	case errors.Is(err, auth.ErrNotFound):
		return errorReply(c, 404, "not_found", "Resource not found.")
	case errors.Is(err, auth.ErrUnauthorized), errors.Is(err, auth.ErrForbidden), errors.Is(err, auth.ErrInvalidInput), errors.Is(err, auth.ErrBusy):
		return s.failure(c, err)
	default:
		return s.adminError(c, err)
	}
}
func (s *server) profile(c fiber.Ctx) error {
	session, _ := s.session(c)
	user, err := s.Auth.Profile(c.Context(), session)
	if err != nil {
		return s.accountError(c, err)
	}
	return respond(c, api.IdentityResponse{User: *user})
}
func (s *server) updateProfile(c fiber.Ctx) error {
	input, err := body[api.ProfileRequest](c)
	if err != nil {
		return s.adminError(c, err)
	}
	session, _ := s.session(c)
	user, err := s.Auth.UpdateProfile(c.Context(), session, input.DisplayName, event(c, "user.profile_updated", "attempt"))
	if err != nil {
		return s.accountError(c, err)
	}
	return respond(c, api.IdentityResponse{User: *user})
}
func (s *server) changePassword(c fiber.Ctx) error {
	input, err := body[api.PasswordChangeRequest](c)
	if err != nil {
		return s.adminError(c, err)
	}
	session, _ := s.session(c)
	if err := s.Auth.ChangePassword(c.Context(), session, input.CurrentPassword, input.Password, event(c, "user.password_changed", "attempt")); err != nil {
		return s.accountError(c, err)
	}
	return respond(c, nil)
}
func (s *server) ownSessions(c fiber.Ctx) error {
	session, _ := s.session(c)
	list, err := s.Auth.Sessions(c.Context(), session)
	if err != nil {
		return s.accountError(c, err)
	}
	return respond(c, list)
}
func (s *server) revokeOwnSession(c fiber.Ctx) error {
	id, err := pathID(c)
	if err != nil {
		return s.adminError(c, err)
	}
	session, _ := s.session(c)
	self, err := s.Auth.RevokeOwnSessions(c.Context(), session, &id, event(c, "session.revoke_own", "attempt"))
	if err != nil {
		return s.accountError(c, err)
	}
	if self {
		clearCookie(c)
	}
	return respond(c, nil)
}
func (s *server) revokeOtherSessions(c fiber.Ctx) error {
	if _, err := body[api.EmptyRequest](c); err != nil {
		return s.adminError(c, err)
	}
	session, _ := s.session(c)
	if _, err := s.Auth.RevokeOwnSessions(c.Context(), session, nil, event(c, "session.revoke_others", "attempt")); err != nil {
		return s.accountError(c, err)
	}
	return respond(c, nil)
}
func (s *server) myAccess(c fiber.Ctx) error {
	access, err := s.RBAC.MyAccess(c.Context(), actorUUID(c))
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, access)
}
func (s *server) managedUsers(c fiber.Ctx) error {
	if err := s.rootPermission(c, rbac.UserManage); err != nil {
		return s.adminError(c, err)
	}
	before, err := parseUUID(c.Query("before"))
	if err != nil {
		return s.adminError(c, err)
	}
	cursor := uuid.Nil
	if before != nil {
		cursor = *before
	}
	limit := 50
	if raw := c.Query("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil {
			return s.adminError(c, rbac.ErrInvalidInput)
		}
	}
	session, _ := s.session(c)
	users, err := s.Auth.ManagedUsers(c.Context(), session.User, c.Query("q"), cursor, limit)
	if err != nil {
		return s.accountError(c, err)
	}
	return respond(c, users)
}
func (s *server) setUserActive(c fiber.Ctx, active bool) error {
	if err := s.rootPermission(c, rbac.UserManage); err != nil {
		return s.adminError(c, err)
	}
	id, err := pathID(c)
	if err != nil {
		return s.adminError(c, err)
	}
	if _, err := body[api.EmptyRequest](c); err != nil {
		return s.adminError(c, err)
	}
	if err := s.RBAC.SetUserActive(c.Context(), actorUUID(c), id, active); err != nil {
		return s.adminError(c, err)
	}
	if !active && id == actorUUID(c) {
		clearCookie(c)
	}
	return respond(c, nil)
}
func (s *server) deactivateUser(c fiber.Ctx) error { return s.setUserActive(c, false) }
func (s *server) reactivateUser(c fiber.Ctx) error { return s.setUserActive(c, true) }
func (s *server) createPasswordReset(c fiber.Ctx) error {
	if err := s.rootPermission(c, rbac.UserManage); err != nil {
		return s.adminError(c, err)
	}
	id, err := pathID(c)
	if err != nil {
		return s.adminError(c, err)
	}
	if _, err := body[api.EmptyRequest](c); err != nil {
		return s.adminError(c, err)
	}
	session, _ := s.session(c)
	value, err := s.Auth.CreatePasswordReset(c.Context(), session.User, id)
	if err != nil {
		return s.accountError(c, err)
	}
	return respond(c, value)
}
func (s *server) inspectPasswordReset(c fiber.Ctx) error {
	c.Set("Referrer-Policy", "no-referrer")
	value, err := s.Auth.InspectPasswordReset(c.Context(), c.Params("token"))
	if err != nil {
		return s.accountError(c, err)
	}
	return respond(c, value)
}
func (s *server) resetPassword(c fiber.Ctx) error {
	c.Set("Referrer-Policy", "no-referrer")
	input, err := body[api.PasswordResetRequest](c)
	if err != nil {
		return s.adminError(c, err)
	}
	id, err := s.Auth.ResetPassword(c.Context(), c.Params("token"), input.Password, event(c, "password_reset.used", "attempt"))
	if err != nil {
		return s.accountError(c, err)
	}
	session, _ := s.session(c)
	if session != nil && session.User != nil && session.User.ID == id.String() {
		clearCookie(c)
	}
	return respond(c, nil)
}
