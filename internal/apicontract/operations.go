// SPDX-License-Identifier: Apache-2.0

package apicontract

import (
	"github.com/Thitipong-PP/schewise/internal/auth"
	"github.com/Thitipong-PP/schewise/internal/connections"
	"github.com/Thitipong-PP/schewise/internal/rbac"
)

type Query struct {
	Name        string
	Type        string
	Description string
}
type Operation struct {
	Existing                  bool
	Public                    bool
	Self                      bool
	ID, Method, Path, Summary string
	Request, Response         any
	Status                    int
	Query                     []Query
}

func Operations() []Operation {
	return []Operation{
		{ID: "connections", Method: "GET", Path: "/api/v1/connections", Summary: "List readable connections; redact destination details without manage permission", Status: 200, Response: connections.List{}, Query: []Query{{"unit_id", "uuid", "Optional owning unit"}}},
		{ID: "unitConnections", Method: "GET", Path: "/api/v1/units/{id}/connections", Summary: "List readable connections owned by a readable unit", Status: 200, Response: connections.List{}},
		{ID: "createConnection", Method: "POST", Path: "/api/v1/connections", Summary: "Create an encrypted PostgreSQL profile in a manageable unit", Status: 201, Request: connections.CreateInput{}, Response: connections.Result{}},
		{ID: "connection", Method: "GET", Path: "/api/v1/connections/{id}", Summary: "Read permitted connection metadata", Status: 200, Response: connections.Result{}},
		{ID: "updateConnection", Method: "PATCH", Path: "/api/v1/connections/{id}", Summary: "Update a connection without changing its owner or password", Status: 200, Request: connections.Input{}, Response: connections.Result{}},
		{ID: "connectionPassword", Method: "POST", Path: "/api/v1/connections/{id}/password", Summary: "Replace the encrypted target password separately", Status: 200, Request: connections.PasswordInput{}, Response: connections.Result{}},
		{ID: "previewConnectionDelete", Method: "POST", Path: "/api/v1/connections/{id}/dry-run/delete", Summary: "Preview snapshots and direct access removed by deletion", Status: 200, Request: EmptyRequest{}, Response: connections.DeleteImpact{}},
		{ID: "deleteConnection", Method: "DELETE", Path: "/api/v1/connections/{id}", Summary: "Delete confirmed connection metadata atomically without touching the target", Status: 204, Request: connections.DeleteInput{}},
		{ID: "testConnection", Method: "POST", Path: "/api/v1/connections/{id}/test", Summary: "Test read-only connectivity and return a sanitized result", Status: 200, Request: EmptyRequest{}, Response: connections.OperationResult{}},
		{ID: "syncConnection", Method: "POST", Path: "/api/v1/connections/{id}/sync", Summary: "Introspect catalogs and refresh structural history and volatile stats", Status: 200, Request: EmptyRequest{}, Response: connections.OperationResult{}},
		{ID: "snapshots", Method: "GET", Path: "/api/v1/connections/{id}/snapshots", Summary: "List authorized structural snapshots with keyset pagination", Status: 200, Response: connections.Snapshots{}, Query: []Query{{"before", "uuid", "Previous page cursor"}, {"limit", "integer", "Page size 1–200, default 50"}}},
		{ID: "schema", Method: "GET", Path: "/api/v1/connections/{id}/schema", Summary: "Read the latest normalized structure and refreshed statistics", Status: 200, Response: connections.Snapshot{}},
		{ID: "snapshot", Method: "GET", Path: "/api/v1/connections/{id}/snapshots/{sid}", Summary: "Read an authorized snapshot from this connection", Status: 200, Response: connections.Snapshot{}},
		{ID: "csrf", Method: "GET", Path: "/api/v1/auth/csrf", Summary: "Initialize the existing synchronizer CSRF session", Response: CSRFResponse{}, Status: 200, Existing: true, Public: true},
		{ID: "identity", Method: "GET", Path: "/api/v1/auth/me", Summary: "Read the authenticated identity", Response: IdentityResponse{}, Status: 200, Existing: true},
		{ID: "createInvite", Method: "POST", Path: "/api/v1/invites", Summary: "Create an authorized invitation and return its link once", Request: auth.InviteInput{}, Response: auth.InviteLink{}, Status: 201},
		{ID: "invites", Method: "GET", Path: "/api/v1/invites", Summary: "List invitations without tokens", Response: auth.InvitePage{}, Status: 200, Query: []Query{{"before", "uuid", "Previous page cursor"}, {"limit", "integer", "Page size 1–200, default 50"}}},
		{ID: "revokeInvite", Method: "POST", Path: "/api/v1/invites/{id}/revoke", Summary: "Revoke an unused invitation", Request: EmptyRequest{}, Status: 204},
		{ID: "regenerateInvite", Method: "POST", Path: "/api/v1/invites/{id}/regenerate", Summary: "Revoke and replace an unused invitation, returning the new link once", Request: EmptyRequest{}, Response: auth.InviteLink{}, Status: 201},
		{ID: "inspectInvite", Method: "GET", Path: "/api/v1/invites/{token}", Summary: "Inspect a valid invitation; all unavailable tokens return the same error", Response: auth.InviteDisplay{}, Status: 200, Public: true},
		{ID: "acceptInvite", Method: "POST", Path: "/api/v1/invites/{token}/accept", Summary: "Accept an invitation atomically and rotate into an authenticated session", Request: AcceptInviteRequest{}, Response: IdentityResponse{}, Status: 201, Public: true},
		{ID: "profile", Method: "GET", Path: "/api/v1/me/profile", Summary: "Read the current account's profile", Response: IdentityResponse{}, Status: 200, Self: true},
		{ID: "updateProfile", Method: "PATCH", Path: "/api/v1/me/profile", Summary: "Update only the current account's display name", Request: ProfileRequest{}, Response: IdentityResponse{}, Status: 200, Self: true},
		{ID: "changePassword", Method: "POST", Path: "/api/v1/me/password", Summary: "Verify the current password, replace it, and revoke other sessions", Request: PasswordChangeRequest{}, Status: 204, Self: true},
		{ID: "ownSessions", Method: "GET", Path: "/api/v1/me/sessions", Summary: "List the current account's active sessions without bearer tokens", Response: auth.SessionList{}, Status: 200, Self: true},
		{ID: "revokeOtherSessions", Method: "POST", Path: "/api/v1/me/sessions/revoke-others", Summary: "Revoke all other sessions for the current account", Request: EmptyRequest{}, Status: 204, Self: true},
		{ID: "revokeOwnSession", Method: "DELETE", Path: "/api/v1/me/sessions/{id}", Summary: "Revoke a session owned by the current account", Status: 204, Self: true},
		{ID: "myAccess", Method: "GET", Path: "/api/v1/me/access", Summary: "Read only the current account's access with existing origin redaction", Response: rbac.SelfAccess{}, Status: 200, Self: true},
		{ID: "managedUsers", Method: "GET", Path: "/api/v1/admin/users", Summary: "List active and deactivated accounts without credentials", Response: auth.ManagedUsers{}, Status: 200, Query: []Query{{"q", "string", "Search email or display name"}, {"before", "uuid", "Previous page cursor"}, {"limit", "integer", "Page size 1–200; default 50"}}},
		{ID: "deactivateUser", Method: "POST", Path: "/api/v1/admin/users/{id}/deactivate", Summary: "Deactivate an account and revoke all its sessions, preserving the last root Administrator", Request: EmptyRequest{}, Status: 204},
		{ID: "reactivateUser", Method: "POST", Path: "/api/v1/admin/users/{id}/reactivate", Summary: "Reactivate an account without changing its password", Request: EmptyRequest{}, Status: 204},
		{ID: "createPasswordReset", Method: "POST", Path: "/api/v1/admin/users/{id}/password-reset", Summary: "Revoke older pending resets and return a replacement link once", Request: EmptyRequest{}, Response: auth.PasswordResetLink{}, Status: 201},
		{ID: "inspectPasswordReset", Method: "GET", Path: "/api/v1/reset-password/{token}", Summary: "Inspect a usable password reset with generic rejection for unavailable links", Response: auth.PasswordResetDisplay{}, Status: 200, Public: true},
		{ID: "resetPassword", Method: "POST", Path: "/api/v1/reset-password/{token}", Summary: "Consume a reset link and revoke all sessions without signing in", Request: PasswordResetRequest{}, Status: 204, Public: true},
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
