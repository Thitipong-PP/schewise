// SPDX-License-Identifier: Apache-2.0

package rbac

type PermissionInfo struct {
	Permission  Permission `json:"permission"`
	Area        string     `json:"area"`
	Label       string     `json:"label"`
	Description string     `json:"description"`
	SystemLevel bool       `json:"system_level"`
	CanGrant    bool       `json:"can_grant"`
}

func Catalog() []PermissionInfo {
	return []PermissionInfo{
		{Permission: UserManage, Area: "Administration", Label: "Manage users", Description: "Manage user accounts and revoke sessions. Root scope only.", SystemLevel: true},
		{Permission: RoleManage, Area: "Administration", Label: "Manage roles", Description: "Create and maintain custom roles. Root scope only.", SystemLevel: true},
		{Permission: AuditRead, Area: "Administration", Label: "Read audit log", Description: "Read installation audit history. Root scope only.", SystemLevel: true},
		{Permission: UnitRead, Area: "Organization", Label: "Read units", Description: "View units and their access information."},
		{Permission: UnitManage, Area: "Organization", Label: "Manage units", Description: "Create, rename, move, and delete units within the assigned scope."},
		{Permission: UnitManageMembers, Area: "Organization", Label: "Manage assignments", Description: "Grant or revoke only the permissions you hold within the assigned scope."},
		{Permission: ConnectionRead, Area: "Connections", Label: "Read connections", Description: "View connection metadata when connection management is available."},
		{Permission: ConnectionManage, Area: "Connections", Label: "Manage connections", Description: "Manage connection metadata when connection management is available."},
		{Permission: SchemaRead, Area: "Schema", Label: "Read schemas", Description: "Inspect schemas when schema features are available."},
		{Permission: DocsEdit, Area: "Documentation", Label: "Edit documentation", Description: "Maintain database documentation when documentation features are available."},
		{Permission: TagManage, Area: "Tags", Label: "Manage tags", Description: "Maintain tags when tagging is available."},
		{Permission: AIRun, Area: "AI", Label: "Run AI workflows", Description: "Use explicitly configured AI workflows when available."},
		{Permission: DataPreview, Area: "Data", Label: "Preview data", Description: "Preview records when data features are available."},
		{Permission: DataQuery, Area: "Data", Label: "Query data", Description: "Run queries when data features are available."},
	}
}
