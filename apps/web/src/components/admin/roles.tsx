// SPDX-License-Identifier: Apache-2.0
"use client";
import { useRouter } from "next/navigation";
import { useCallback, useState } from "react";
import {
  Copy,
  Eye,
  LockKeyhole,
  Pencil,
  Plus,
  Save,
  Trash2,
} from "lucide-react";
import { api, csrfHeaders, result, message } from "@/lib/admin-api";
import type { components } from "@/lib/api-types";
import type { Permission, Role } from "@/lib/admin-model";
import { useAccess } from "./shell";
import { Actions, Feedback, Modal, QueryState, useQuery } from "./shared";

type Editor = {
  mode: "create" | "edit" | "view" | "clone" | "delete";
  role?: Role;
};
type Catalog = components["schemas"]["PermissionInfo"][];

export function Roles() {
  const router = useRouter();
  const context = useAccess();
  const allowed = context?.root_permissions.includes("role:manage");
  const query = useQuery(
    useCallback(
      async (signal) => {
        if (!allowed) return null;
        const [roles, catalog] = await Promise.all([
          api.GET("/api/v1/roles", { signal }),
          api.GET("/api/v1/permissions", { signal }),
        ]);
        return {
          roles: result(roles).roles,
          catalog: result(catalog).permissions,
        };
      },
      [allowed],
    ),
  );
  const [editor, setEditor] = useState<Editor | null>(null);
  if (!allowed)
    return (
      <>
        <h1>Roles</h1>
        <p className="muted">You do not have permission to manage roles.</p>
      </>
    );
  return (
    <>
      <div className="page-heading">
        <div>
          <p className="section-label">Workspace administration</p>
          <h1>Roles</h1>
        </div>
        <button onClick={() => setEditor({ mode: "create" })}>
          <Plus size={16} />
          Create role
        </button>
      </div>
      <QueryState query={query} />
      {query.data && (
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>Role</th>
                <th>Permissions</th>
                <th>Members</th>
                <th>Assignments</th>
                <th>
                  <span className="sr-only">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {query.data.roles.map((role) => (
                <tr key={role.id}>
                  <td>
                    <div className="name-with-badge">
                      <strong>{role.name}</strong>
                      {role.is_system && (
                        <span className="badge">
                          <LockKeyhole size={12} />
                          System
                        </span>
                      )}
                    </div>
                    {role.description && (
                      <span className="row-description">
                        {role.description}
                      </span>
                    )}
                  </td>
                  <td>{role.permissions.length}</td>
                  <td>{role.member_count}</td>
                  <td>{role.assignment_count}</td>
                  <td>
                    <div className="row-actions">
                      <button
                        className="icon-button"
                        title={
                          role.can_edit
                            ? `Edit ${role.name}`
                            : `View ${role.name}`
                        }
                        aria-label={
                          role.can_edit
                            ? `Edit ${role.name}`
                            : `View ${role.name}`
                        }
                        onClick={() =>
                          setEditor({
                            mode: role.can_edit ? "edit" : "view",
                            role,
                          })
                        }
                      >
                        {role.can_edit ? (
                          <Pencil size={16} />
                        ) : (
                          <Eye size={16} />
                        )}
                      </button>
                      <button
                        className="icon-button"
                        title={
                          role.can_clone
                            ? `Clone ${role.name}`
                            : "You cannot grant all permissions in this role"
                        }
                        aria-label={`Clone ${role.name}`}
                        disabled={!role.can_clone}
                        onClick={() => setEditor({ mode: "clone", role })}
                      >
                        <Copy size={16} />
                      </button>
                      {!role.is_system && (
                        <button
                          className="icon-button"
                          title={
                            role.can_delete
                              ? `Delete ${role.name}`
                              : "Revoke all assignments before deleting this role"
                          }
                          aria-label={`Delete ${role.name}`}
                          disabled={!role.can_delete}
                          onClick={() => setEditor({ mode: "delete", role })}
                        >
                          <Trash2 size={16} />
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {editor && query.data && (
        <RoleDialog
          key={`${editor.mode}:${editor.role?.id}`}
          editor={editor}
          catalog={query.data.catalog}
          close={() => setEditor(null)}
          saved={() => {
            setEditor(null);
            query.reload();
            router.refresh();
          }}
        />
      )}
    </>
  );
}

function RoleDialog({
  editor: { mode, role },
  catalog,
  close,
  saved,
}: {
  editor: Editor;
  catalog: Catalog;
  close: () => void;
  saved: () => void;
}) {
  const [name, setName] = useState(
    mode === "clone" ? `${role?.name ?? "Role"} copy` : (role?.name ?? ""),
  );
  const [description, setDescription] = useState(role?.description ?? "");
  const [permissions, setPermissions] = useState<Permission[]>(
    role?.permissions ?? [],
  );
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const readOnly = mode === "view" || mode === "clone";
  const title =
    mode === "create"
      ? "Create custom role"
      : `${{ edit: "Edit", view: "View", clone: "Clone", delete: "Delete", create: "Create" }[mode]} ${role?.name}`;
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (mode === "view") return;
    setBusy(true);
    setError("");
    try {
      const header = await csrfHeaders();
      if (mode === "create")
        result(
          await api.POST("/api/v1/roles", {
            params: { header },
            body: { name, description, permissions },
          }),
        );
      if (mode === "edit" && role)
        result(
          await api.PATCH("/api/v1/roles/{id}", {
            params: { path: { id: role.id }, header },
            body: { name, description, permissions },
          }),
        );
      if (mode === "clone" && role)
        result(
          await api.POST("/api/v1/roles/{id}/clone", {
            params: { path: { id: role.id }, header },
            body: { name },
          }),
        );
      if (mode === "delete" && role)
        result(
          await api.DELETE("/api/v1/roles/{id}", {
            params: { path: { id: role.id }, header },
          }),
        );
      saved();
    } catch (error) {
      setError(message(error));
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal title={title} close={close} busy={busy}>
      <form onSubmit={submit}>
        {mode === "delete" ? (
          <p>
            Delete <strong>{role?.name}</strong>? This custom role has no
            assignments.
          </p>
        ) : (
          <>
            <label className="field">
              Name
              <input
                value={name}
                required
                maxLength={120}
                onChange={(e) => setName(e.target.value)}
                disabled={busy || mode === "view"}
              />
            </label>
            <label className="field">
              Description
              <textarea
                value={description}
                maxLength={2000}
                rows={2}
                onChange={(e) => setDescription(e.target.value)}
                disabled={busy || readOnly}
              />
            </label>
            <div className="permission-editor">
              {Array.from(new Set(catalog.map((p) => p.area))).map((area) => (
                <fieldset key={area}>
                  <legend>{area}</legend>
                  {catalog
                    .filter((p) => p.area === area)
                    .map((item) => (
                      <label
                        className="permission-option"
                        key={item.permission}
                      >
                        <input
                          type="checkbox"
                          checked={permissions.includes(item.permission)}
                          disabled={
                            busy ||
                            readOnly ||
                            (!item.can_grant &&
                              !permissions.includes(item.permission))
                          }
                          onChange={(e) =>
                            setPermissions(
                              e.target.checked
                                ? [...permissions, item.permission]
                                : permissions.filter(
                                    (p) => p !== item.permission,
                                  ),
                            )
                          }
                        />
                        <span>
                          <strong>{item.label}</strong>
                          <small>{item.description}</small>
                          {!item.can_grant && !readOnly && (
                            <small>You cannot grant this permission.</small>
                          )}
                        </span>
                      </label>
                    ))}
                </fieldset>
              ))}
            </div>
          </>
        )}
        <Feedback error={error} />
        {mode === "view" ? (
          <div className="dialog-actions">
            <button type="button" onClick={close}>
              Close
            </button>
          </div>
        ) : (
          <Actions
            busy={busy}
            close={close}
            label={
              mode === "delete"
                ? "Delete role"
                : mode === "clone"
                  ? "Clone role"
                  : "Save role"
            }
            icon={
              mode === "delete" ? (
                <Trash2 size={16} />
              ) : mode === "clone" ? (
                <Copy size={16} />
              ) : (
                <Save size={16} />
              )
            }
          />
        )}
      </form>
    </Modal>
  );
}
