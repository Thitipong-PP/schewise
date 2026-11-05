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

func (s *server) inviteGlobalLimit() fiber.Handler {
	return limiter.New(limiter.Config{Max: 200, Expiration: time.Minute, KeyGenerator: func(fiber.Ctx) string { return "invite-attempts" }, LimitReached: func(c fiber.Ctx) error {
		if err := s.Auth.Audit(c.Context(), event(c, "invite.rate_limited", "denied")); err != nil {
			return s.failure(c, err)
		}
		return errorReply(c, 429, "rate_limited", "Too many requests. Try again later.")
	}})
}
func (s *server) inviteError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, auth.ErrInviteInvalid):
		return errorReply(c, 400, "invitation_unavailable", "This invitation is unavailable. Ask your administrator for a new link.")
	case errors.Is(err, auth.ErrInviteConflict):
		return errorReply(c, 409, "invite_pending", "A pending invitation already exists. Revoke or regenerate it first.")
	case errors.Is(err, auth.ErrInviteAccount):
		return errorReply(c, 409, "account_exists", "An account with this email already exists. Assign the role directly instead.")
	case errors.Is(err, auth.ErrInvalidInput), errors.Is(err, auth.ErrForbidden), errors.Is(err, auth.ErrUnauthorized), errors.Is(err, auth.ErrBusy), errors.Is(err, auth.ErrNotFound):
		return s.failure(c, err)
	default:
		return s.adminError(c, err)
	}
}
func (s *server) createInvite(c fiber.Ctx) error {
	if err := s.rootPermission(c, rbac.UserManage); err != nil {
		return s.adminError(c, err)
	}
	input, err := body[auth.InviteInput](c)
	if err != nil {
		return s.adminError(c, err)
	}
	session, _ := s.session(c)
	value, err := s.Auth.CreateInvite(c.Context(), session.User, input)
	if err != nil {
		return s.inviteError(c, err)
	}
	return respond(c, value)
}
func (s *server) invites(c fiber.Ctx) error {
	if err := s.rootPermission(c, rbac.UserManage); err != nil {
		return s.adminError(c, err)
	}
	before, err := parseUUID(c.Query("before"))
	if err != nil {
		return s.adminError(c, err)
	}
	limit := 50
	if raw := c.Query("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil {
			return s.adminError(c, rbac.ErrInvalidInput)
		}
	}
	session, _ := s.session(c)
	cursor := uuid.Nil
	if before != nil {
		cursor = *before
	}
	value, err := s.Auth.ListInvites(c.Context(), session.User, cursor, limit)
	if err != nil {
		return s.inviteError(c, err)
	}
	return respond(c, value)
}
func (s *server) changeInvite(c fiber.Ctx, regenerate bool) error {
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
	if regenerate {
		value, err := s.Auth.RegenerateInvite(c.Context(), session.User, id)
		if err != nil {
			return s.inviteError(c, err)
		}
		return respond(c, value)
	}
	if err := s.Auth.RevokeInvite(c.Context(), session.User, id); err != nil {
		return s.inviteError(c, err)
	}
	return respond(c, nil)
}
func (s *server) revokeInvite(c fiber.Ctx) error     { return s.changeInvite(c, false) }
func (s *server) regenerateInvite(c fiber.Ctx) error { return s.changeInvite(c, true) }
func (s *server) inspectInvite(c fiber.Ctx) error {
	c.Set("Referrer-Policy", "no-referrer")
	value, err := s.Auth.InspectInvite(c.Context(), c.Params("token"))
	if err != nil {
		return s.inviteError(c, err)
	}
	return respond(c, value)
}
func (s *server) acceptInvite(c fiber.Ctx) error {
	c.Set("Referrer-Policy", "no-referrer")
	input, err := body[api.AcceptInviteRequest](c)
	if err != nil {
		return s.adminError(c, err)
	}
	previous, _ := s.session(c)
	session, err := s.Auth.AcceptInvite(c.Context(), c.Params("token"), input.DisplayName, input.Password, previous, event(c, "invite.accepted", "attempt"))
	if err != nil {
		return s.inviteError(c, err)
	}
	c.Locals("session", session)
	setCookie(c, session)
	return respond(c, api.IdentityResponse{User: *session.User})
}
