// SPDX-License-Identifier: Apache-2.0

// Generated from backend DTOs and routes. Run make api-generate; do not edit.

export interface paths {
  "/api/v1/access/context": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /** Read the current user's server-evaluated capabilities */
    get: operations["context"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/access/explain": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /** Explain a decision without exposing unreadable assignment origins */
    post: operations["explain"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/assignments": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /** Delegate a held role at a readable scope using CanAssign */
    post: operations["assign"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/assignments/{id}": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    post?: never;
    /** Revoke a manageable assignment, preserving the last active root Administrator */
    delete: operations["revoke"];
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/audit": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /** Read filtered audit history with keyset pagination */
    get: operations["audit"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/auth/csrf": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /** Initialize the existing synchronizer CSRF session */
    get: operations["csrf"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/auth/me": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /** Read the authenticated identity */
    get: operations["identity"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/permissions": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /** List permission areas and plain-language descriptions */
    get: operations["catalog"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/roles": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /** List roles, permissions, and assignment/user counts */
    get: operations["roles"];
    put?: never;
    /** Create a custom role */
    post: operations["createRole"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/roles/{id}": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    post?: never;
    /** Delete an unassigned custom role */
    delete: operations["deleteRole"];
    options?: never;
    head?: never;
    /** Update a custom role without escalating privileges */
    patch: operations["updateRole"];
    trace?: never;
  };
  "/api/v1/roles/{id}/clone": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /** Clone a role into a custom role */
    post: operations["cloneRole"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/units": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /** List only readable units, omitting unreadable ancestors */
    get: operations["units"];
    put?: never;
    /** Create a unit beneath a readable manageable parent */
    post: operations["createUnit"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/units/tree": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /** Read the visible organization forest */
    get: operations["unitTree"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/units/{id}": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /** Read a visible unit */
    get: operations["unit"];
    put?: never;
    post?: never;
    /** Delete an empty non-root unit */
    delete: operations["deleteUnit"];
    options?: never;
    head?: never;
    /** Rename or update a visible unit */
    patch: operations["updateUnit"];
    trace?: never;
  };
  "/api/v1/units/{id}/access": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /** List active users with effective access, including inherited grants */
    get: operations["unitAccess"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/units/{id}/assignable-roles": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /** List only roles the actor can delegate at this unit */
    get: operations["assignableRoles"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/units/{id}/assignments": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /** List direct assignments at a readable unit */
    get: operations["unitAssignments"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/units/{id}/dry-run/delete": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /** Preview delete blockers and access changes */
    post: operations["previewDelete"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/units/{id}/dry-run/move": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /** Preview permission changes before moving a unit */
    post: operations["previewMove"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/units/{id}/members": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /** List direct and inherited members with per-assignment source redaction */
    get: operations["members"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/units/{id}/move": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /** Move a unit after rechecking current authorization */
    post: operations["moveUnit"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/users": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /** Read a minimal active-user directory for authorized assignment management */
    get: operations["users"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/users/{id}/access": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /** Read a user's effective access only within the actor's inspectable units */
    get: operations["userAccess"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/users/{id}/assignments": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /** List assignments visible within the actor's inspectable units */
    get: operations["userAssignments"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
}
export type webhooks = Record<string, never>;
export interface components {
  schemas: {
    AccessChange: {
      gained: (
        | "user:manage"
        | "role:manage"
        | "audit:read"
        | "unit:read"
        | "unit:manage"
        | "unit:manage_members"
        | "connection:read"
        | "connection:manage"
        | "schema:read"
        | "docs:edit"
        | "tag:manage"
        | "ai:run"
        | "data:preview"
        | "data:query"
      )[];
      lost: (
        | "user:manage"
        | "role:manage"
        | "audit:read"
        | "unit:read"
        | "unit:manage"
        | "unit:manage_members"
        | "connection:read"
        | "connection:manage"
        | "schema:read"
        | "docs:edit"
        | "tag:manage"
        | "ai:run"
        | "data:preview"
        | "data:query"
      )[];
      resource: components["schemas"]["Scope"];
      /** Format: uuid */
      user_id: string;
    };
    AccessContext: {
      can_read_units: boolean;
      root_permissions: (
        | "user:manage"
        | "role:manage"
        | "audit:read"
        | "unit:read"
        | "unit:manage"
        | "unit:manage_members"
        | "connection:read"
        | "connection:manage"
        | "schema:read"
        | "docs:edit"
        | "tag:manage"
        | "ai:run"
        | "data:preview"
        | "data:query"
      )[];
      user: components["schemas"]["UserInfo"];
    };
    AssignableRolesResponse: {
      roles: components["schemas"]["Role"][];
    };
    AssignmentRequest: {
      note: string;
      /** Format: uuid */
      role_id: string;
      scope: components["schemas"]["Scope"];
      /** Format: uuid */
      user_id: string;
    };
    AssignmentResponse: {
      assignment: components["schemas"]["AssignmentView"];
    };
    AssignmentView: {
      can_revoke: boolean;
      /** Format: date-time */
      created_at?: string;
      /** Format: uuid */
      id?: string;
      note?: string;
      permissions: (
        | "user:manage"
        | "role:manage"
        | "audit:read"
        | "unit:read"
        | "unit:manage"
        | "unit:manage_members"
        | "connection:read"
        | "connection:manage"
        | "schema:read"
        | "docs:edit"
        | "tag:manage"
        | "ai:run"
        | "data:preview"
        | "data:query"
      )[];
      read_only_reason: string;
      role: components["schemas"]["RoleRef"];
      scope?: components["schemas"]["Scope"];
      /** @enum {string} */
      source: "direct" | "inherited" | "inherited_hidden";
      source_unit?: components["schemas"]["UnitRef"];
    };
    AssignmentsResponse: {
      assignments: components["schemas"]["UserAssignment"][];
    };
    Audit: {
      action: string;
      /** Format: uuid */
      actor_user_id: string;
      /** Format: date-time */
      created_at: string;
      details: {
        [key: string]: unknown;
      };
      /** Format: uuid */
      id: string;
      /** Format: uuid */
      target_id: string;
      target_type: string;
    };
    AuditPage: {
      entries: components["schemas"]["Audit"][];
      has_more: boolean;
    };
    CSRFResponse: {
      csrf_token: string;
    };
    CatalogResponse: {
      permissions: components["schemas"]["PermissionInfo"][];
    };
    CloneRequest: {
      name: string;
    };
    Connection: {
      /** Format: uuid */
      id: string;
      /** Format: uuid */
      unit_id: string;
    };
    CreateUnitRequest: {
      description: string;
      kind: string;
      name: string;
      /** Format: uuid */
      parent_id: string;
    };
    EmptyRequest: Record<string, never>;
    Error: {
      code: string;
      error: string;
    };
    ExplainRequest: {
      /** @enum {string} */
      permission:
        | "user:manage"
        | "role:manage"
        | "audit:read"
        | "unit:read"
        | "unit:manage"
        | "unit:manage_members"
        | "connection:read"
        | "connection:manage"
        | "schema:read"
        | "docs:edit"
        | "tag:manage"
        | "ai:run"
        | "data:preview"
        | "data:query";
      resource: components["schemas"]["Scope"];
      /** Format: uuid */
      user_id: string;
    };
    ExplanationView: {
      allowed: boolean;
      grants: components["schemas"]["AssignmentView"][];
      reason: string;
      reason_text: string;
    };
    IdentityResponse: {
      user: components["schemas"]["User"];
    };
    Impact: {
      blockers: string[];
      changes: components["schemas"]["AccessChange"][];
      connections: components["schemas"]["Connection"][];
      executable: boolean;
    };
    Member: {
      assignments: components["schemas"]["AssignmentView"][];
      permissions: (
        | "user:manage"
        | "role:manage"
        | "audit:read"
        | "unit:read"
        | "unit:manage"
        | "unit:manage_members"
        | "connection:read"
        | "connection:manage"
        | "schema:read"
        | "docs:edit"
        | "tag:manage"
        | "ai:run"
        | "data:preview"
        | "data:query"
      )[];
      user: components["schemas"]["UserInfo"];
    };
    MembersResponse: {
      members: components["schemas"]["Member"][];
    };
    MoveRequest: {
      /** Format: uuid */
      parent_id: string;
    };
    PermissionInfo: {
      area: string;
      can_grant: boolean;
      description: string;
      label: string;
      /** @enum {string} */
      permission:
        | "user:manage"
        | "role:manage"
        | "audit:read"
        | "unit:read"
        | "unit:manage"
        | "unit:manage_members"
        | "connection:read"
        | "connection:manage"
        | "schema:read"
        | "docs:edit"
        | "tag:manage"
        | "ai:run"
        | "data:preview"
        | "data:query";
      system_level: boolean;
    };
    Role: {
      description: string;
      /** Format: uuid */
      id: string;
      is_system: boolean;
      name: string;
      permissions: (
        | "user:manage"
        | "role:manage"
        | "audit:read"
        | "unit:read"
        | "unit:manage"
        | "unit:manage_members"
        | "connection:read"
        | "connection:manage"
        | "schema:read"
        | "docs:edit"
        | "tag:manage"
        | "ai:run"
        | "data:preview"
        | "data:query"
      )[];
    };
    RoleRef: {
      /** Format: uuid */
      id: string;
      name: string;
    };
    RoleRequest: {
      description: string;
      name: string;
      permissions: (
        | "user:manage"
        | "role:manage"
        | "audit:read"
        | "unit:read"
        | "unit:manage"
        | "unit:manage_members"
        | "connection:read"
        | "connection:manage"
        | "schema:read"
        | "docs:edit"
        | "tag:manage"
        | "ai:run"
        | "data:preview"
        | "data:query"
      )[];
    };
    RoleResponse: {
      role: components["schemas"]["Role"];
    };
    RoleView: {
      assignment_count: number;
      can_clone: boolean;
      can_delete: boolean;
      can_edit: boolean;
      description: string;
      /** Format: uuid */
      id: string;
      is_system: boolean;
      member_count: number;
      name: string;
      permissions: (
        | "user:manage"
        | "role:manage"
        | "audit:read"
        | "unit:read"
        | "unit:manage"
        | "unit:manage_members"
        | "connection:read"
        | "connection:manage"
        | "schema:read"
        | "docs:edit"
        | "tag:manage"
        | "ai:run"
        | "data:preview"
        | "data:query"
      )[];
    };
    RolesResponse: {
      roles: components["schemas"]["RoleView"][];
    };
    Scope: {
      /** Format: uuid */
      id: string;
      /** @enum {string} */
      type: "unit" | "connection";
    };
    TreeResponse: {
      roots: components["schemas"]["UnitNode"][];
    };
    UnitAccess: {
      permissions: (
        | "user:manage"
        | "role:manage"
        | "audit:read"
        | "unit:read"
        | "unit:manage"
        | "unit:manage_members"
        | "connection:read"
        | "connection:manage"
        | "schema:read"
        | "docs:edit"
        | "tag:manage"
        | "ai:run"
        | "data:preview"
        | "data:query"
      )[];
      unit: components["schemas"]["UnitView"];
    };
    UnitNode: {
      children: components["schemas"]["UnitNode"][];
      unit: components["schemas"]["UnitView"];
    };
    UnitRef: {
      /** Format: uuid */
      id: string;
      name: string;
    };
    UnitRequest: {
      description: string;
      kind: string;
      name: string;
    };
    UnitResponse: {
      unit: components["schemas"]["UnitView"];
    };
    UnitView: {
      can_move: boolean;
      depth: number;
      description: string;
      /** Format: uuid */
      id: string;
      is_root: boolean;
      kind: string;
      name: string;
      parent_id: string | null;
      permissions: (
        | "user:manage"
        | "role:manage"
        | "audit:read"
        | "unit:read"
        | "unit:manage"
        | "unit:manage_members"
        | "connection:read"
        | "connection:manage"
        | "schema:read"
        | "docs:edit"
        | "tag:manage"
        | "ai:run"
        | "data:preview"
        | "data:query"
      )[];
    };
    UnitsResponse: {
      units: components["schemas"]["UnitView"][];
    };
    User: {
      email: string;
      id: string;
      role: string;
    };
    UserAccess: {
      units: components["schemas"]["UnitAccess"][];
      user: components["schemas"]["UserInfo"];
    };
    UserAssignment: {
      assignment: components["schemas"]["AssignmentView"];
      user: components["schemas"]["UserInfo"];
    };
    UserInfo: {
      email: string;
      /** Format: uuid */
      id: string;
      is_active: boolean;
    };
    UsersResponse: {
      users: components["schemas"]["UserInfo"][];
    };
  };
  responses: never;
  parameters: never;
  requestBodies: never;
  headers: never;
  pathItems: never;
}
export type $defs = Record<string, never>;
export interface operations {
  context: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["AccessContext"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  explain: {
    parameters: {
      query?: never;
      header: {
        "X-CSRF-Token": string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": components["schemas"]["ExplainRequest"];
      };
    };
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["ExplanationView"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  assign: {
    parameters: {
      query?: never;
      header: {
        "X-CSRF-Token": string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": components["schemas"]["AssignmentRequest"];
      };
    };
    responses: {
      /** @description Success */
      201: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["AssignmentResponse"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  revoke: {
    parameters: {
      query?: never;
      header: {
        "X-CSRF-Token": string;
      };
      path: {
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  audit: {
    parameters: {
      query?: {
        /** @description Actor UUID */
        actor_id?: string;
        /** @description Exact action */
        action?: string;
        /** @description Inclusive start time */
        from?: string;
        /** @description Inclusive end time */
        to?: string;
        /** @description Previous page's last created_at */
        before?: string;
        /** @description Previous page's last ID */
        before_id?: string;
        /** @description Page size from 1 to 200; default 50 */
        limit?: number;
      };
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["AuditPage"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  csrf: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["CSRFResponse"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  identity: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["IdentityResponse"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  catalog: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["CatalogResponse"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  roles: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["RolesResponse"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  createRole: {
    parameters: {
      query?: never;
      header: {
        "X-CSRF-Token": string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": components["schemas"]["RoleRequest"];
      };
    };
    responses: {
      /** @description Success */
      201: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["RoleResponse"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  deleteRole: {
    parameters: {
      query?: never;
      header: {
        "X-CSRF-Token": string;
      };
      path: {
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  updateRole: {
    parameters: {
      query?: never;
      header: {
        "X-CSRF-Token": string;
      };
      path: {
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": components["schemas"]["RoleRequest"];
      };
    };
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["RoleResponse"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  cloneRole: {
    parameters: {
      query?: never;
      header: {
        "X-CSRF-Token": string;
      };
      path: {
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": components["schemas"]["CloneRequest"];
      };
    };
    responses: {
      /** @description Success */
      201: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["RoleResponse"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  units: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["UnitsResponse"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  createUnit: {
    parameters: {
      query?: never;
      header: {
        "X-CSRF-Token": string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": components["schemas"]["CreateUnitRequest"];
      };
    };
    responses: {
      /** @description Success */
      201: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["UnitResponse"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  unitTree: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["TreeResponse"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  unit: {
    parameters: {
      query?: never;
      header?: never;
      path: {
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["UnitResponse"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  deleteUnit: {
    parameters: {
      query?: never;
      header: {
        "X-CSRF-Token": string;
      };
      path: {
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  updateUnit: {
    parameters: {
      query?: never;
      header: {
        "X-CSRF-Token": string;
      };
      path: {
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": components["schemas"]["UnitRequest"];
      };
    };
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["UnitResponse"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  unitAccess: {
    parameters: {
      query?: never;
      header?: never;
      path: {
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["MembersResponse"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  assignableRoles: {
    parameters: {
      query?: never;
      header?: never;
      path: {
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["AssignableRolesResponse"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  unitAssignments: {
    parameters: {
      query?: never;
      header?: never;
      path: {
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["MembersResponse"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  previewDelete: {
    parameters: {
      query?: never;
      header: {
        "X-CSRF-Token": string;
      };
      path: {
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": components["schemas"]["EmptyRequest"];
      };
    };
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Impact"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  previewMove: {
    parameters: {
      query?: never;
      header: {
        "X-CSRF-Token": string;
      };
      path: {
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": components["schemas"]["MoveRequest"];
      };
    };
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Impact"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  members: {
    parameters: {
      query?: never;
      header?: never;
      path: {
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["MembersResponse"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  moveUnit: {
    parameters: {
      query?: never;
      header: {
        "X-CSRF-Token": string;
      };
      path: {
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": components["schemas"]["MoveRequest"];
      };
    };
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  users: {
    parameters: {
      query?: {
        /** @description Unit being managed; required without root user:manage */
        unit_id?: string;
        /** @description Email search text */
        q?: string;
      };
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["UsersResponse"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  userAccess: {
    parameters: {
      query?: never;
      header?: never;
      path: {
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["UserAccess"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  userAssignments: {
    parameters: {
      query?: never;
      header?: never;
      path: {
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["AssignmentsResponse"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Request rejected; unreadable and missing units both return 403 */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
}
