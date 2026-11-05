// SPDX-License-Identifier: Apache-2.0

package apicontract

import "github.com/Thitipong-PP/schewise/internal/rbac"

type Query struct {
	Name        string
	Type        string
	Description string
}
type Operation struct {
	Existing                  bool
	Public                    bool
	ID, Method, Path, Summary string
	Request, Response         any
	Status                    int
	Query                     []Query
}

func Operations() []Operation {
	return []Operation{
		{ID: "csrf", Method: "GET", Path: "/api/v1/auth/csrf", Summary: "Initialize the existing synchronizer CSRF session", Response: CSRFResponse{}, Status: 200, Existing: true, Public: true},
		{ID: "identity", Method: "GET", Path: "/api/v1/auth/me", Summary: "Read the authenticated identity", Response: IdentityResponse{}, Status: 200, Existing: true},
		{ID: "context", Method: "GET", Path: "/api/v1/access/context", Summary: "Read the current user's server-evaluated capabilities", Response: rbac.AccessContext{}, Status: 200},
		{ID: "units", Method: "GET", Path: "/api/v1/units", Summary: "List only readable units, omitting unreadable ancestors", Response: UnitsResponse{}, Status: 200},
		{ID: "unitTree", Method: "GET", Path: "/api/v1/units/tree", Summary: "Read the visible organization forest", Response: TreeResponse{}, Status: 200},
		{ID: "createUnit", Method: "POST", Path: "/api/v1/units", Summary: "Create a unit beneath a readable manageable parent", Request: CreateUnitRequest{}, Response: UnitResponse{}, Status: 201},
		{ID: "unit", Method: "GET", Path: "/api/v1/units/{id}", Summary: "Read a visible unit", Response: UnitResponse{}, Status: 200},
		{ID: "updateUnit", Method: "PATCH", Path: "/api/v1/units/{id}", Summary: "Rename or update a visible unit", Request: UnitRequest{}, Response: UnitResponse{}, Status: 200},
		{ID: "moveUnit", Method: "POST", Path: "/api/v1/units/{id}/move", Summary: "Move a unit after rechecking current authorization", Request: MoveRequest{}, Status: 204},
		{ID: "deleteUnit", Method: "DELETE", Path: "/api/v1/units/{id}", Summary: "Delete an empty non-root unit", Status: 204},
		{ID: "previewMove", Method: "POST", Path: "/api/v1/units/{id}/dry-run/move", Summary: "Preview permission changes before moving a unit", Request: MoveRequest{}, Response: rbac.Impact{}, Status: 200},
		{ID: "previewDelete", Method: "POST", Path: "/api/v1/units/{id}/dry-run/delete", Summary: "Preview delete blockers and access changes", Request: EmptyRequest{}, Response: rbac.Impact{}, Status: 200},
		{ID: "setInheritance", Method: "PATCH", Path: "/api/v1/units/{id}/inherit-access", Summary: "Set an inheritance boundary, protecting against self-lockout", Request: InheritRequest{}, Status: 204},
		{ID: "previewInheritance", Method: "POST", Path: "/api/v1/units/{id}/dry-run/inherit-access", Summary: "Preview inheritance changes, roles, and affected connections", Request: InheritRequest{}, Response: rbac.Impact{}, Status: 200},
		{ID: "roles", Method: "GET", Path: "/api/v1/roles", Summary: "List roles, permissions, and assignment/user counts", Response: RolesResponse{}, Status: 200},
		{ID: "createRole", Method: "POST", Path: "/api/v1/roles", Summary: "Create a custom role", Request: RoleRequest{}, Response: RoleResponse{}, Status: 201},
		{ID: "updateRole", Method: "PATCH", Path: "/api/v1/roles/{id}", Summary: "Update a custom role without escalating privileges", Request: RoleRequest{}, Response: RoleResponse{}, Status: 200},
		{ID: "cloneRole", Method: "POST", Path: "/api/v1/roles/{id}/clone", Summary: "Clone a role into a custom role", Request: CloneRequest{}, Response: RoleResponse{}, Status: 201},
		{ID: "deleteRole", Method: "DELETE", Path: "/api/v1/roles/{id}", Summary: "Delete an unassigned custom role", Status: 204},
		{ID: "catalog", Method: "GET", Path: "/api/v1/permissions", Summary: "List permission areas and plain-language descriptions", Response: CatalogResponse{}, Status: 200},
		{ID: "assignableRoles", Method: "GET", Path: "/api/v1/units/{id}/assignable-roles", Summary: "List only roles the actor can delegate at this unit", Response: AssignableRolesResponse{}, Status: 200},
		{ID: "members", Method: "GET", Path: "/api/v1/units/{id}/members", Summary: "List direct and inherited members with per-assignment source redaction", Response: MembersResponse{}, Status: 200},
		{ID: "unitAssignments", Method: "GET", Path: "/api/v1/units/{id}/assignments", Summary: "List direct assignments at a readable unit", Response: MembersResponse{}, Status: 200},
		{ID: "unitAccess", Method: "GET", Path: "/api/v1/units/{id}/access", Summary: "List active users with effective access, including inherited grants", Response: MembersResponse{}, Status: 200},
		{ID: "users", Method: "GET", Path: "/api/v1/users", Summary: "Read a minimal active-user directory for authorized assignment management", Response: UsersResponse{}, Status: 200, Query: []Query{{"unit_id", "uuid", "Unit being managed; required without root user:manage"}, {"q", "string", "Email search text"}}},
		{ID: "userAssignments", Method: "GET", Path: "/api/v1/users/{id}/assignments", Summary: "List assignments visible within the actor's inspectable units", Response: AssignmentsResponse{}, Status: 200},
		{ID: "userAccess", Method: "GET", Path: "/api/v1/users/{id}/access", Summary: "Read a user's effective access only within the actor's inspectable units", Response: rbac.UserAccess{}, Status: 200},
		{ID: "assign", Method: "POST", Path: "/api/v1/assignments", Summary: "Delegate a held role at a readable scope using CanAssign", Request: AssignmentRequest{}, Response: AssignmentResponse{}, Status: 201},
		{ID: "revoke", Method: "DELETE", Path: "/api/v1/assignments/{id}", Summary: "Revoke a manageable assignment, preserving the last active root Administrator", Status: 204},
		{ID: "explain", Method: "POST", Path: "/api/v1/access/explain", Summary: "Explain a decision without exposing unreadable assignment origins", Request: ExplainRequest{}, Response: rbac.ExplanationView{}, Status: 200},
		{ID: "audit", Method: "GET", Path: "/api/v1/audit", Summary: "Read filtered audit history with keyset pagination", Response: rbac.AuditPage{}, Status: 200, Query: []Query{{"actor_id", "uuid", "Actor UUID"}, {"action", "string", "Exact action"}, {"from", "date-time", "Inclusive start time"}, {"to", "date-time", "Inclusive end time"}, {"before", "date-time", "Previous page's last created_at"}, {"before_id", "uuid", "Previous page's last ID"}, {"limit", "integer", "Page size from 1 to 200; default 50"}}},
	}
}
