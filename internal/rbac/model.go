// SPDX-License-Identifier: Apache-2.0

// Package rbac implements hierarchical, scoped, additive authorization.
package rbac

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
)

type Permission string

const (
	UserManage        Permission = "user:manage"
	RoleManage        Permission = "role:manage"
	AuditRead         Permission = "audit:read"
	UnitRead          Permission = "unit:read"
	UnitManage        Permission = "unit:manage"
	UnitManageMembers Permission = "unit:manage_members"
	ConnectionRead    Permission = "connection:read"
	ConnectionManage  Permission = "connection:manage"
	SchemaRead        Permission = "schema:read"
	DocsEdit          Permission = "docs:edit"
	TagManage         Permission = "tag:manage"
	AIRun             Permission = "ai:run"
	DataPreview       Permission = "data:preview"
	DataQuery         Permission = "data:query"

	RootUnitID          = "00000000-0000-4000-8000-000000000001"
	AdministratorRoleID = "00000000-0000-4000-8000-000000000101"
	ManagerRoleID       = "00000000-0000-4000-8000-000000000102"
	DeveloperRoleID     = "00000000-0000-4000-8000-000000000103"
	ViewerRoleID        = "00000000-0000-4000-8000-000000000104"
	MaxDepth            = 6 // The root is depth one.
)

var permissions = []Permission{UserManage, RoleManage, AuditRead, UnitRead, UnitManage, UnitManageMembers, ConnectionRead, ConnectionManage, SchemaRead, DocsEdit, TagManage, AIRun, DataPreview, DataQuery}

func Permissions() []Permission        { return slices.Clone(permissions) }
func (p Permission) Valid() bool       { return slices.Contains(permissions, p) }
func (p Permission) SystemLevel() bool { return p == UserManage || p == RoleManage || p == AuditRead }

var (
	ErrForbidden         = errors.New("permission denied")
	ErrNotFound          = errors.New("resource not found")
	ErrInvalidInput      = errors.New("invalid RBAC input")
	ErrInvalidScope      = errors.New("invalid assignment scope")
	ErrUnavailable       = errors.New("scope resolver unavailable")
	ErrConflict          = errors.New("RBAC record already exists")
	ErrRootProtected     = errors.New("root unit cannot be moved or deleted")
	ErrSystemRole        = errors.New("system role cannot be changed")
	ErrLastAdministrator = errors.New("last active root Administrator must be preserved")
	ErrNotEmpty          = errors.New("resource is not empty")
	ErrCycle             = errors.New("unit move would create a cycle")
	ErrSelfLockout       = errors.New("inheritance change would remove your membership-management permission")
	ErrDepth             = errors.New("unit tree exceeds maximum depth")
)

type ScopeType string

const (
	ScopeUnit       ScopeType = "unit"
	ScopeConnection ScopeType = "connection"
)

type Scope struct {
	Type ScopeType `json:"type"`
	ID   uuid.UUID `json:"id"`
}

func RootScope() Scope             { return Scope{Type: ScopeUnit, ID: uuid.MustParse(RootUnitID)} }
func UnitScope(id uuid.UUID) Scope { return Scope{Type: ScopeUnit, ID: id} }

type Unit struct {
	InheritAccess bool        `json:"inherit_access"`
	ID            uuid.UUID   `json:"id"`
	ParentID      uuid.UUID   `json:"parent_id"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	Kind          string      `json:"kind"`
	Ancestors     []uuid.UUID `json:"ancestors"`
	CreatedAt     time.Time   `json:"created_at"`
}

type Role struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	IsSystem    bool         `json:"is_system"`
	Permissions []Permission `json:"permissions"`
}

type Assignment struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	RoleID    uuid.UUID `json:"role_id"`
	Scope     Scope     `json:"scope"`
	Note      string    `json:"note"`
	CreatedBy uuid.UUID `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type RoleGrant struct {
	Assignment Assignment
	Role       Role
}

type Grant struct {
	AssignmentID uuid.UUID `json:"assignment_id"`
	RoleID       uuid.UUID `json:"role_id"`
	Role         string    `json:"role"`
	Scope        Scope     `json:"scope"`
	Inherited    bool      `json:"inherited"`
}

type BlockedGrant struct {
	Grant
	BlockedBy uuid.UUID `json:"blocked_by"`
}

type Explanation struct {
	Blocked []BlockedGrant `json:"blocked"`
	Allowed bool           `json:"allowed"`
	Grants  []Grant        `json:"grants"`
	Reason  string         `json:"reason"`
}

// Reader supplies trusted persisted identities, hierarchy, roles, and grants.
// Resources from callers contain only identifiers, never trusted ancestor paths.
type Reader interface {
	ActiveRootAdministrators(context.Context) (int, error)
	ActiveUser(context.Context, uuid.UUID) (bool, error)
	Unit(context.Context, uuid.UUID) (Unit, error)
	Role(context.Context, uuid.UUID) (Role, error)
	Grants(context.Context, uuid.UUID, []Scope) ([]RoleGrant, error)
	ConnectionUnit(context.Context, uuid.UUID) (uuid.UUID, error)
}
