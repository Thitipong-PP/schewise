// SPDX-License-Identifier: Apache-2.0

package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"reflect"
	"strconv"
	"time"

	api "github.com/Thitipong-PP/schewise/internal/apicontract"
	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func errorReply(c fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(api.Error{Error: message, Code: code})
}
func statusCode(status int) string {
	switch status {
	case 400:
		return "invalid_request"
	case 401:
		return "authentication_required"
	case 403:
		return "forbidden"
	case 404:
		return "not_found"
	case 405:
		return "method_not_allowed"
	case 409:
		return "conflict"
	case 413:
		return "request_too_large"
	case 429:
		return "rate_limited"
	default:
		return "service_unavailable"
	}
}
func actorUUID(c fiber.Ctx) uuid.UUID { return uuid.MustParse(actorID(c)) }
func pathID(c fiber.Ctx) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, rbac.ErrInvalidInput
	}
	return id, nil
}
func body[T any](c fiber.Ctx) (T, error) {
	var input T
	op, ok := c.Locals("api_operation").(api.Operation)
	if !ok || reflect.TypeOf(input) != reflect.TypeOf(op.Request) {
		return input, errors.New("request contract mismatch")
	}
	media, _, err := mime.ParseMediaType(c.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return input, rbac.ErrInvalidInput
	}
	decoder := json.NewDecoder(bytes.NewReader(c.Body()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return input, rbac.ErrInvalidInput
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return input, rbac.ErrInvalidInput
	}
	return input, nil
}
func respond(c fiber.Ctx, value any) error {
	op, ok := c.Locals("api_operation").(api.Operation)
	if !ok || reflect.TypeOf(value) != reflect.TypeOf(op.Response) {
		return errors.New("response contract mismatch")
	}
	if op.Status == 204 {
		return c.SendStatus(204)
	}
	return c.Status(op.Status).JSON(value)
}
func (s *server) adminError(c fiber.Ctx, err error) error {
	status, code, message := 503, "service_unavailable", "Service temporarily unavailable."
	switch {
	case errors.Is(err, rbac.ErrForbidden):
		status, code, message = 403, "forbidden", "Permission denied."
	case errors.Is(err, rbac.ErrNotFound):
		status, code, message = 404, "not_found", "Resource not found."
	case errors.Is(err, rbac.ErrInvalidInput), errors.Is(err, rbac.ErrInvalidScope):
		status, code, message = 400, "invalid_request", "Check the supplied fields and scope."
	case errors.Is(err, rbac.ErrConflict):
		status, code, message = 409, "conflict", "An equivalent record already exists."
	case errors.Is(err, rbac.ErrLastAdministrator):
		status, code, message = 409, "last_administrator", "The last active root Administrator cannot be revoked."
	case errors.Is(err, rbac.ErrSystemRole):
		status, code, message = 409, "system_role_read_only", "System roles cannot be edited or deleted. Clone the role instead."
	case errors.Is(err, rbac.ErrNotEmpty):
		status, code, message = 409, "not_empty", "The unit or role still has dependent records."
	case errors.Is(err, rbac.ErrSelfLockout):
		status, code, message = 409, "self_lockout", "This change would remove your permission to manage assignments at this unit."
	case errors.Is(err, rbac.ErrRootProtected):
		status, code, message = 409, "root_protected", "The root unit cannot be moved, deleted, or have its inheritance changed."
	case errors.Is(err, rbac.ErrCycle):
		status, code, message = 409, "unit_cycle", "A unit cannot be moved beneath itself or a descendant."
	case errors.Is(err, rbac.ErrDepth):
		status, code, message = 409, "depth_exceeded", "The resulting unit tree would exceed six levels."
	}
	if status == 403 {
		if auditErr := s.Auth.Audit(c.Context(), event(c, "auth.authorization", "denied")); auditErr != nil {
			return s.failure(c, auditErr)
		}
	}
	if status == 503 {
		s.Logger.Error("http.operation_failed", zap.String("request_id", c.GetRespHeader("X-Request-ID")), zap.Int("status", status))
	}
	return errorReply(c, status, code, message)
}
func (s *server) rootPermission(c fiber.Ctx, p rbac.Permission) error {
	allowed, err := s.RBAC.Engine().Can(c.Context(), actorUUID(c), p, rbac.RootScope())
	if err != nil {
		return err
	}
	if !allowed {
		return rbac.ErrForbidden
	}
	return nil
}
func (s *server) visibleID(c fiber.Ctx) (uuid.UUID, error) {
	id, err := pathID(c)
	if err != nil {
		return uuid.Nil, rbac.ErrForbidden
	}
	_, err = s.RBAC.VisibleUnit(c.Context(), actorUUID(c), id)
	return id, err
}
func (s *server) registerAdmin(app *fiber.App) {
	handlers := map[string]fiber.Handler{
		"setInheritance": s.setInheritance, "previewInheritance": s.previewInheritance,
		"context": s.accessContext, "units": s.units, "unitTree": s.unitTree, "createUnit": s.createUnit, "unit": s.unit, "updateUnit": s.updateUnit, "moveUnit": s.moveUnit, "deleteUnit": s.deleteUnit, "previewMove": s.previewMove, "previewDelete": s.previewDelete,
		"roles": s.roles, "createRole": s.createRole, "updateRole": s.updateRole, "cloneRole": s.cloneRole, "deleteRole": s.deleteRole, "catalog": s.catalog, "assignableRoles": s.assignableRoles,
		"members": s.members, "unitAssignments": s.unitAssignments, "unitAccess": s.unitAccess, "users": s.users, "userAssignments": s.userAssignments, "userAccess": s.userAccess, "assign": s.assign, "revoke": s.revoke, "explain": s.explain, "audit": s.audit,
	}
	for _, op := range api.Operations() {
		if op.Existing {
			continue
		}
		handler, ok := handlers[op.ID]
		if !ok {
			panic("missing administration handler: " + op.ID)
		}
		chain := []fiber.Handler{func(c fiber.Ctx) error {
			c.Locals("api_operation", op)
			if s.Auth == nil || s.RBAC == nil {
				return fiber.ErrServiceUnavailable
			}
			return c.Next()
		}, s.authenticated}
		if op.Method != "GET" {
			chain = append(chain, s.mutation)
		}
		chain = append(chain, handler)
		rest := make([]any, 0, len(chain)-1)
		for _, handler := range chain[1:] {
			rest = append(rest, handler)
		}
		app.Add([]string{op.Method}, api.FiberPath(op.Path), chain[0], rest...)
	}
}
func (s *server) accessContext(c fiber.Ctx) error {
	value, err := s.RBAC.Context(c.Context(), actorUUID(c))
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, value)
}
func (s *server) units(c fiber.Ctx) error {
	value, err := s.RBAC.VisibleUnits(c.Context(), actorUUID(c))
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, api.UnitsResponse{Units: value})
}
func (s *server) unitTree(c fiber.Ctx) error {
	units, err := s.RBAC.VisibleUnits(c.Context(), actorUUID(c))
	if err != nil {
		return s.adminError(c, err)
	}
	var children func(*uuid.UUID) []rbac.UnitNode
	children = func(parent *uuid.UUID) []rbac.UnitNode {
		result := []rbac.UnitNode{}
		for _, unit := range units {
			matches := parent == nil && unit.ParentID == nil || parent != nil && unit.ParentID != nil && *parent == *unit.ParentID
			if matches {
				result = append(result, rbac.UnitNode{Unit: unit, Children: children(&unit.ID)})
			}
		}
		return result
	}
	return respond(c, api.TreeResponse{Roots: children(nil)})
}
func (s *server) unit(c fiber.Ctx) error {
	id, err := s.visibleID(c)
	if err != nil {
		return s.adminError(c, err)
	}
	value, err := s.RBAC.VisibleUnit(c.Context(), actorUUID(c), id)
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, api.UnitResponse{Unit: value})
}
func (s *server) createUnit(c fiber.Ctx) error {
	input, err := body[api.CreateUnitRequest](c)
	if err != nil {
		return s.adminError(c, err)
	}
	if _, err := s.RBAC.VisibleUnit(c.Context(), actorUUID(c), input.ParentID); err != nil {
		return s.adminError(c, err)
	}
	unit, err := s.RBAC.CreateUnit(c.Context(), actorUUID(c), input.ParentID, rbac.UnitInput{Name: input.Name, Description: input.Description, Kind: input.Kind})
	if err != nil {
		return s.adminError(c, err)
	}
	value, err := s.RBAC.VisibleUnit(c.Context(), actorUUID(c), unit.ID)
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, api.UnitResponse{Unit: value})
}
func (s *server) updateUnit(c fiber.Ctx) error {
	id, err := s.visibleID(c)
	if err != nil {
		return s.adminError(c, err)
	}
	input, err := body[api.UnitRequest](c)
	if err != nil {
		return s.adminError(c, err)
	}
	if _, err := s.RBAC.UpdateUnit(c.Context(), actorUUID(c), id, rbac.UnitInput{Name: input.Name, Description: input.Description, Kind: input.Kind}); err != nil {
		return s.adminError(c, err)
	}
	value, err := s.RBAC.VisibleUnit(c.Context(), actorUUID(c), id)
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, api.UnitResponse{Unit: value})
}
func (s *server) moveInput(c fiber.Ctx) (uuid.UUID, api.MoveRequest, error) {
	id, err := s.visibleID(c)
	if err != nil {
		return id, api.MoveRequest{}, err
	}
	input, err := body[api.MoveRequest](c)
	if err != nil {
		return id, input, err
	}
	_, err = s.RBAC.VisibleUnit(c.Context(), actorUUID(c), input.ParentID)
	return id, input, err
}
func (s *server) moveUnit(c fiber.Ctx) error {
	id, input, err := s.moveInput(c)
	if err == nil {
		err = s.RBAC.MoveUnit(c.Context(), actorUUID(c), id, input.ParentID)
	}
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, nil)
}
func (s *server) previewMove(c fiber.Ctx) error {
	id, input, err := s.moveInput(c)
	if err != nil {
		return s.adminError(c, err)
	}
	value, err := s.RBAC.DryRunMove(c.Context(), actorUUID(c), id, input.ParentID)
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, value)
}
func (s *server) deleteUnit(c fiber.Ctx) error {
	id, err := s.visibleID(c)
	if err == nil {
		err = s.RBAC.DeleteUnit(c.Context(), actorUUID(c), id)
	}
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, nil)
}
func (s *server) previewDelete(c fiber.Ctx) error {
	id, err := s.visibleID(c)
	if err != nil {
		return s.adminError(c, err)
	}
	if _, err := body[api.EmptyRequest](c); err != nil {
		return s.adminError(c, err)
	}
	value, err := s.RBAC.DryRunDelete(c.Context(), actorUUID(c), id)
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, value)
}
func (s *server) roles(c fiber.Ctx) error {
	value, err := s.RBAC.RolesFor(c.Context(), actorUUID(c))
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, api.RolesResponse{Roles: value})
}
func (s *server) createRole(c fiber.Ctx) error {
	if err := s.rootPermission(c, rbac.RoleManage); err != nil {
		return s.adminError(c, err)
	}
	input, err := body[api.RoleRequest](c)
	if err != nil {
		return s.adminError(c, err)
	}
	role, err := s.RBAC.CreateRole(c.Context(), actorUUID(c), rbac.RoleInput{Name: input.Name, Description: input.Description, Permissions: input.Permissions})
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, api.RoleResponse{Role: role})
}
func (s *server) updateRole(c fiber.Ctx) error {
	if err := s.rootPermission(c, rbac.RoleManage); err != nil {
		return s.adminError(c, err)
	}
	id, err := pathID(c)
	if err != nil {
		return s.adminError(c, err)
	}
	input, err := body[api.RoleRequest](c)
	if err != nil {
		return s.adminError(c, err)
	}
	role, err := s.RBAC.UpdateRole(c.Context(), actorUUID(c), id, rbac.RoleInput{Name: input.Name, Description: input.Description, Permissions: input.Permissions})
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, api.RoleResponse{Role: role})
}
func (s *server) cloneRole(c fiber.Ctx) error {
	if err := s.rootPermission(c, rbac.RoleManage); err != nil {
		return s.adminError(c, err)
	}
	id, err := pathID(c)
	if err != nil {
		return s.adminError(c, err)
	}
	input, err := body[api.CloneRequest](c)
	if err != nil {
		return s.adminError(c, err)
	}
	role, err := s.RBAC.CloneRole(c.Context(), actorUUID(c), id, input.Name)
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, api.RoleResponse{Role: role})
}
func (s *server) deleteRole(c fiber.Ctx) error {
	if err := s.rootPermission(c, rbac.RoleManage); err != nil {
		return s.adminError(c, err)
	}
	id, err := pathID(c)
	if err == nil {
		err = s.RBAC.DeleteRole(c.Context(), actorUUID(c), id)
	}
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, nil)
}
func (s *server) catalog(c fiber.Ctx) error {
	value, err := s.RBAC.PermissionCatalog(c.Context(), actorUUID(c))
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, api.CatalogResponse{Permissions: value})
}
func (s *server) assignableRoles(c fiber.Ctx) error {
	id, err := s.visibleID(c)
	if err != nil {
		return s.adminError(c, err)
	}
	value, err := s.RBAC.AssignableRoles(c.Context(), actorUUID(c), id)
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, api.AssignableRolesResponse{Roles: value})
}
func (s *server) memberList(c fiber.Ctx, direct, access bool) error {
	id, err := s.visibleID(c)
	if err != nil {
		return s.adminError(c, err)
	}
	value, err := s.RBAC.Members(c.Context(), actorUUID(c), id, direct, access)
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, api.MembersResponse{Members: value})
}
func (s *server) members(c fiber.Ctx) error         { return s.memberList(c, false, false) }
func (s *server) unitAssignments(c fiber.Ctx) error { return s.memberList(c, true, false) }
func (s *server) unitAccess(c fiber.Ctx) error      { return s.memberList(c, false, true) }
func (s *server) users(c fiber.Ctx) error {
	var unit *uuid.UUID
	if raw := c.Query("unit_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return s.adminError(c, rbac.ErrForbidden)
		}
		unit = &id
	}
	value, err := s.RBAC.UserDirectory(c.Context(), actorUUID(c), unit, c.Query("q"))
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, api.UsersResponse{Users: value})
}
func (s *server) userAssignments(c fiber.Ctx) error {
	id, err := pathID(c)
	if err != nil {
		return s.adminError(c, err)
	}
	value, err := s.RBAC.UserAssignments(c.Context(), actorUUID(c), id)
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, api.AssignmentsResponse{Assignments: value})
}
func (s *server) userAccess(c fiber.Ctx) error {
	id, err := pathID(c)
	if err != nil {
		return s.adminError(c, err)
	}
	value, err := s.RBAC.AccessForUser(c.Context(), actorUUID(c), id)
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, value)
}
func (s *server) assign(c fiber.Ctx) error {
	input, err := body[api.AssignmentRequest](c)
	if err != nil {
		return s.adminError(c, err)
	}
	if input.Scope.Type != rbac.ScopeUnit {
		return s.adminError(c, rbac.ErrForbidden)
	}
	if _, err := s.RBAC.VisibleUnit(c.Context(), actorUUID(c), input.Scope.ID); err != nil {
		return s.adminError(c, err)
	}
	allowed, err := s.RBAC.Engine().CanAssign(c.Context(), actorUUID(c), input.RoleID, input.Scope)
	if errors.Is(err, rbac.ErrNotFound) {
		err = rbac.ErrForbidden
	}
	if err != nil {
		return s.adminError(c, err)
	}
	if !allowed {
		return s.adminError(c, rbac.ErrForbidden)
	}
	assignment, err := s.RBAC.Assign(c.Context(), actorUUID(c), input.UserID, input.RoleID, input.Scope, input.Note)
	if err != nil {
		return s.adminError(c, err)
	}
	members, err := s.RBAC.Members(c.Context(), actorUUID(c), input.Scope.ID, true, false)
	if err != nil {
		return s.adminError(c, err)
	}
	for _, member := range members {
		for _, item := range member.Assignments {
			if item.ID != nil && *item.ID == assignment.ID {
				return respond(c, api.AssignmentResponse{Assignment: item})
			}
		}
	}
	return s.adminError(c, rbac.ErrForbidden)
}
func (s *server) revoke(c fiber.Ctx) error {
	id, err := pathID(c)
	if err != nil {
		return s.adminError(c, rbac.ErrForbidden)
	}
	if _, err = s.RBAC.AssignmentAtReadableScope(c.Context(), actorUUID(c), id); err == nil {
		err = s.RBAC.Revoke(c.Context(), actorUUID(c), id)
	}
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, nil)
}
func (s *server) explain(c fiber.Ctx) error {
	input, err := body[api.ExplainRequest](c)
	if err != nil {
		return s.adminError(c, err)
	}
	value, err := s.RBAC.ExplainFor(c.Context(), actorUUID(c), input.UserID, input.Permission, input.Resource)
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, value)
}
func parseTime(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	value, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return nil, rbac.ErrInvalidInput
	}
	return &value, nil
}
func parseUUID(raw string) (*uuid.UUID, error) {
	if raw == "" {
		return nil, nil
	}
	value, err := uuid.Parse(raw)
	if err != nil {
		return nil, rbac.ErrInvalidInput
	}
	return &value, nil
}
func (s *server) audit(c fiber.Ctx) error {
	if err := s.rootPermission(c, rbac.AuditRead); err != nil {
		return s.adminError(c, err)
	}
	filter := rbac.AuditFilter{Action: c.Query("action"), Limit: 50}
	var err error
	filter.ActorID, err = parseUUID(c.Query("actor_id"))
	if err == nil {
		filter.From, err = parseTime(c.Query("from"))
	}
	if err == nil {
		filter.To, err = parseTime(c.Query("to"))
	}
	if err == nil {
		filter.Before, err = parseTime(c.Query("before"))
	}
	if err == nil {
		filter.BeforeID, err = parseUUID(c.Query("before_id"))
	}
	if raw := c.Query("limit"); raw != "" && err == nil {
		filter.Limit, err = strconv.Atoi(raw)
		if err != nil {
			err = rbac.ErrInvalidInput
		}
	}
	if err != nil {
		return s.adminError(c, err)
	}
	value, err := s.RBAC.FilteredAudit(c.Context(), actorUUID(c), filter)
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, value)
}

func (s *server) inheritanceInput(c fiber.Ctx) (uuid.UUID, bool, error) {
	id, err := s.visibleID(c)
	if err != nil {
		return id, false, err
	}
	input, err := body[api.InheritRequest](c)
	if err != nil {
		return id, false, err
	}
	if input.InheritAccess == nil {
		return id, false, rbac.ErrInvalidInput
	}
	return id, *input.InheritAccess, nil
}
func (s *server) setInheritance(c fiber.Ctx) error {
	id, value, err := s.inheritanceInput(c)
	if err == nil {
		err = s.RBAC.SetInheritAccess(c.Context(), actorUUID(c), id, value)
	}
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, nil)
}
func (s *server) previewInheritance(c fiber.Ctx) error {
	id, value, err := s.inheritanceInput(c)
	if err != nil {
		return s.adminError(c, err)
	}
	impact, err := s.RBAC.DryRunInheritAccess(c.Context(), actorUUID(c), id, value)
	if err != nil {
		return s.adminError(c, err)
	}
	return respond(c, impact)
}
