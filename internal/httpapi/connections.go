// SPDX-License-Identifier: Apache-2.0
package httpapi

import (
	api "github.com/Thitipong-PP/schewise/internal/apicontract"
	"strconv"

	"github.com/Thitipong-PP/schewise/internal/connections"
	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (s *server) connectionService(c fiber.Ctx) bool {
	if s.Connections == nil {
		_ = s.adminError(c, fiber.ErrServiceUnavailable)
		return false
	}
	return true
}
func (s *server) connections(c fiber.Ctx) error {
	if !s.connectionService(c) {
		return nil
	}
	var unit uuid.UUID
	var err error
	if c.Params("id") != "" {
		unit, err = pathID(c)
	} else if c.Query("unit_id") != "" {
		unit, err = uuid.Parse(c.Query("unit_id"))
	}
	if err != nil {
		return s.adminError(c, rbac.ErrForbidden)
	}
	value, err := s.Connections.List(c.Context(), actorUUID(c), unit)
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, value)
}
func (s *server) connection(c fiber.Ctx) error {
	if !s.connectionService(c) {
		return nil
	}
	id, err := pathID(c)
	if err != nil {
		return s.adminError(c, rbac.ErrForbidden)
	}
	value, err := s.Connections.Get(c.Context(), actorUUID(c), id)
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, value)
}
func (s *server) createConnection(c fiber.Ctx) error {
	if !s.connectionService(c) {
		return nil
	}
	in, err := body[connections.CreateInput](c)
	if err != nil {
		return s.adminError(c, err)
	}
	value, err := s.Connections.Create(c.Context(), actorUUID(c), in)
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, value)
}
func (s *server) updateConnection(c fiber.Ctx) error {
	if !s.connectionService(c) {
		return nil
	}
	id, err := pathID(c)
	if err != nil {
		return s.adminError(c, rbac.ErrForbidden)
	}
	in, err := body[connections.Input](c)
	if err != nil {
		return s.adminError(c, err)
	}
	value, err := s.Connections.Update(c.Context(), actorUUID(c), id, in)
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, value)
}
func (s *server) connectionPassword(c fiber.Ctx) error {
	if !s.connectionService(c) {
		return nil
	}
	id, err := pathID(c)
	if err != nil {
		return s.adminError(c, rbac.ErrForbidden)
	}
	in, err := body[connections.PasswordInput](c)
	if err != nil {
		return s.adminError(c, err)
	}
	value, err := s.Connections.ChangePassword(c.Context(), actorUUID(c), id, in.Password)
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, value)
}
func (s *server) previewConnectionDelete(c fiber.Ctx) error {
	if !s.connectionService(c) {
		return nil
	}
	id, err := pathID(c)
	if err != nil {
		return s.adminError(c, rbac.ErrForbidden)
	}
	if _, err := body[api.EmptyRequest](c); err != nil {
		return s.adminError(c, err)
	}
	value, err := s.Connections.Delete(c.Context(), actorUUID(c), id, nil)
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, value)
}
func (s *server) deleteConnection(c fiber.Ctx) error {
	if !s.connectionService(c) {
		return nil
	}
	id, err := pathID(c)
	if err != nil {
		return s.adminError(c, rbac.ErrForbidden)
	}
	in, err := body[connections.DeleteInput](c)
	if err != nil {
		return s.adminError(c, err)
	}
	_, err = s.Connections.Delete(c.Context(), actorUUID(c), id, &in.Name)
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, nil)
}
func (s *server) runConnection(c fiber.Ctx, syncSchema bool) error {
	if !s.connectionService(c) {
		return nil
	}
	id, err := pathID(c)
	if err != nil {
		return s.adminError(c, rbac.ErrForbidden)
	}
	if _, err := body[api.EmptyRequest](c); err != nil {
		return s.adminError(c, err)
	}
	value, err := s.Connections.Run(c.Context(), actorUUID(c), id, syncSchema)
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, value)
}
func (s *server) testConnection(c fiber.Ctx) error { return s.runConnection(c, false) }
func (s *server) syncConnection(c fiber.Ctx) error { return s.runConnection(c, true) }
func (s *server) snapshots(c fiber.Ctx) error {
	if !s.connectionService(c) {
		return nil
	}
	id, err := pathID(c)
	if err != nil {
		return s.adminError(c, rbac.ErrForbidden)
	}
	var before uuid.UUID
	if c.Query("before") != "" {
		before, err = uuid.Parse(c.Query("before"))
		if err != nil {
			return s.adminError(c, rbac.ErrInvalidInput)
		}
	}
	limit := 0
	if c.Query("limit") != "" {
		limit, err = strconv.Atoi(c.Query("limit"))
		if err != nil {
			return s.adminError(c, rbac.ErrInvalidInput)
		}
	}
	value, err := s.Connections.Snapshots(c.Context(), actorUUID(c), id, before, limit)
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, value)
}
func (s *server) readSnapshot(c fiber.Ctx, specific bool) error {
	if !s.connectionService(c) {
		return nil
	}
	id, err := pathID(c)
	if err != nil {
		return s.adminError(c, rbac.ErrForbidden)
	}
	var sid uuid.UUID
	if specific {
		sid, err = uuid.Parse(c.Params("sid"))
		if err != nil {
			return s.adminError(c, rbac.ErrInvalidInput)
		}
	}
	value, err := s.Connections.Snapshot(c.Context(), actorUUID(c), id, sid)
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, value)
}
func (s *server) schema(c fiber.Ctx) error   { return s.readSnapshot(c, false) }
func (s *server) snapshot(c fiber.Ctx) error { return s.readSnapshot(c, true) }
