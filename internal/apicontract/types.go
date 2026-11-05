// SPDX-License-Identifier: Apache-2.0

// Package apicontract owns the HTTP DTOs and endpoint definitions used to
// generate OpenAPI and frontend types. It contains no authorization decisions.
package apicontract

import (
	"github.com/Thitipong-PP/schewise/internal/auth"
	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/google/uuid"
)

type Error struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}
type UnitRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
}
type CreateUnitRequest struct {
	UnitRequest
	ParentID uuid.UUID `json:"parent_id"`
}
type MoveRequest struct {
	ParentID uuid.UUID `json:"parent_id"`
}
type InheritRequest struct {
	InheritAccess *bool `json:"inherit_access"`
}
type EmptyRequest struct{}
type RoleRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Permissions []rbac.Permission `json:"permissions"`
}
type CloneRequest struct {
	Name string `json:"name"`
}
type AssignmentRequest struct {
	UserID uuid.UUID  `json:"user_id"`
	RoleID uuid.UUID  `json:"role_id"`
	Scope  rbac.Scope `json:"scope"`
	Note   string     `json:"note"`
}
type ExplainRequest struct {
	UserID     uuid.UUID       `json:"user_id"`
	Permission rbac.Permission `json:"permission"`
	Resource   rbac.Scope      `json:"resource"`
}
type UnitsResponse struct {
	Units []rbac.UnitView `json:"units"`
}
type TreeResponse struct {
	Roots []rbac.UnitNode `json:"roots"`
}
type UnitResponse struct {
	Unit rbac.UnitView `json:"unit"`
}
type RolesResponse struct {
	Roles []rbac.RoleView `json:"roles"`
}
type RoleResponse struct {
	Role rbac.Role `json:"role"`
}
type AssignableRolesResponse struct {
	Roles []rbac.Role `json:"roles"`
}
type CatalogResponse struct {
	Permissions []rbac.PermissionInfo `json:"permissions"`
}
type MembersResponse struct {
	Members []rbac.Member `json:"members"`
}
type AssignmentsResponse struct {
	Assignments []rbac.UserAssignment `json:"assignments"`
}
type AssignmentResponse struct {
	Assignment rbac.AssignmentView `json:"assignment"`
}
type UsersResponse struct {
	Users []rbac.UserInfo `json:"users"`
}

type CSRFResponse struct {
	Token string `json:"csrf_token"`
}
type IdentityResponse struct {
	User auth.User `json:"user"`
}
