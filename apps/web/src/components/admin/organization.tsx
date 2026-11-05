// SPDX-License-Identifier: Apache-2.0
"use client";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useState } from "react";
import {
  ArrowRight,
  CornerDownRight,
  Folder,
  FolderPlus,
  Move,
  Pencil,
  Plus,
  Save,
  Trash2,
} from "lucide-react";
import { api, csrfHeaders, message, result } from "@/lib/admin-api";
import {
  moveDestinations,
  orderedUnits,
  type Impact,
  type Unit,
} from "@/lib/admin-model";
import { useAccess } from "./shell";
import { Actions, Feedback, Modal, QueryState, useQuery } from "./shared";

type Edit = { mode: "create" | "edit" | "move" | "delete"; unit: Unit };

export function Organization() {
  const router = useRouter();
  const context = useAccess();
  const query = useQuery(
    useCallback(
      async (signal) =>
        result(await api.GET("/api/v1/units", { signal })).units,
      [],
    ),
  );
  const [edit, setEdit] = useState<Edit | null>(null);
  if (!context?.can_read_units)
    return (
      <>
        <h1>Organization</h1>
        <p className="muted">You do not have permission to view units.</p>
      </>
    );
  const units = query.data ?? [];
  const firstParent = units.find(
    (u) => u.permissions.includes("unit:manage") && u.depth < 6,
  );
  return (
    <>
      <div className="page-heading">
        <div>
          <p className="section-label">Workspace administration</p>
          <h1>Organization</h1>
        </div>
        {firstParent && (
          <button
            onClick={() => setEdit({ mode: "create", unit: firstParent })}
          >
            <Plus size={16} />
            Create unit
          </button>
        )}
      </div>
      <QueryState query={query} />
      {query.data && (
        <div className="table-scroll">
          <table className="units-table">
            <thead>
              <tr>
                <th scope="col">Unit</th>
                <th scope="col">Kind</th>
                <th scope="col">Access</th>
                <th scope="col">
                  <span className="sr-only">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {orderedUnits(units).map(({ unit, level }) => (
                <tr key={unit.id}>
                  <td>
                    <div
                      className="unit-name"
                      style={{ paddingInlineStart: level * 20 }}
                    >
                      {level > 0 ? (
                        <CornerDownRight
                          size={16}
                          className="muted"
                          aria-hidden="true"
                        />
                      ) : (
                        <Folder
                          size={16}
                          className="muted"
                          aria-hidden="true"
                        />
                      )}
                      <div>
                        <Link href={`/units/${unit.id}`}>
                          {unit.name}
                          <ArrowRight size={13} aria-hidden="true" />
                        </Link>
                        {unit.description && (
                          <span className="row-description">
                            {unit.description}
                          </span>
                        )}
                      </div>
                      {unit.is_root && <span className="badge">Root</span>}
                    </div>
                  </td>
                  <td className="muted">{unit.kind || "—"}</td>
                  <td className="muted">
                    {unit.permissions.includes("unit:manage")
                      ? "Manage"
                      : "Read"}
                  </td>
                  <td>
                    <div className="row-actions">
                      {unit.permissions.includes("unit:manage") && (
                        <>
                          {unit.depth < 6 && (
                            <button
                              className="icon-button"
                              title={`Create a unit in ${unit.name}`}
                              aria-label={`Create a unit in ${unit.name}`}
                              onClick={() => setEdit({ mode: "create", unit })}
                            >
                              <FolderPlus size={16} />
                            </button>
                          )}
                          <button
                            className="icon-button"
                            title={`Edit ${unit.name}`}
                            aria-label={`Edit ${unit.name}`}
                            onClick={() => setEdit({ mode: "edit", unit })}
                          >
                            <Pencil size={15} />
                          </button>
                          {!unit.is_root && (
                            <>
                              <button
                                className="icon-button"
                                title={
                                  unit.can_move
                                    ? `Move ${unit.name}`
                                    : "Moving requires management access to the current parent"
                                }
                                aria-label={`Move ${unit.name}`}
                                disabled={!unit.can_move}
                                onClick={() => setEdit({ mode: "move", unit })}
                              >
                                <Move size={16} />
                              </button>
                              <button
                                className="icon-button"
                                title={`Delete ${unit.name}`}
                                aria-label={`Delete ${unit.name}`}
                                onClick={() =>
                                  setEdit({ mode: "delete", unit })
                                }
                              >
                                <Trash2 size={15} />
                              </button>
                            </>
                          )}
                        </>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {edit && (
        <UnitDialog
          key={`${edit.mode}:${edit.unit.id}`}
          edit={edit}
          units={units}
          close={() => setEdit(null)}
          saved={() => {
            setEdit(null);
            query.reload();
            router.refresh();
          }}
        />
      )}
    </>
  );
}

function UnitDialog({
  edit: { mode, unit },
  units,
  close,
  saved,
}: {
  edit: Edit;
  units: Unit[];
  close: () => void;
  saved: () => void;
}) {
  const [name, setName] = useState(mode === "edit" ? unit.name : "");
  const [description, setDescription] = useState(
    mode === "edit" ? unit.description : "",
  );
  const [kind, setKind] = useState(mode === "edit" ? unit.kind : "");
  const [parent, setParent] = useState(mode === "create" ? unit.id : "");
  const [impact, setImpact] = useState<Impact | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const destinations =
    mode === "move"
      ? moveDestinations(units, unit)
      : units.filter(
          (u) => u.permissions.includes("unit:manage") && u.depth < 6,
        );
  const title = {
    create: "Create unit",
    edit: `Edit ${unit.name}`,
    move: `Move ${unit.name}`,
    delete: `Delete ${unit.name}`,
  }[mode];
  const preview = mode === "move" || mode === "delete";
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const header = await csrfHeaders();
      const params = { path: { id: unit.id }, header };
      if (mode === "create")
        result(
          await api.POST("/api/v1/units", {
            params: { header },
            body: { name, description, kind, parent_id: parent },
          }),
        );
      if (mode === "edit")
        result(
          await api.PATCH("/api/v1/units/{id}", {
            params,
            body: { name, description, kind },
          }),
        );
      if (mode === "move") {
        if (!impact) {
          setImpact(
            result(
              await api.POST("/api/v1/units/{id}/dry-run/move", {
                params,
                body: { parent_id: parent },
              }),
            ),
          );
          return;
        }
        result(
          await api.POST("/api/v1/units/{id}/move", {
            params,
            body: { parent_id: parent },
          }),
        );
      }
      if (mode === "delete") {
        if (!impact) {
          setImpact(
            result(
              await api.POST("/api/v1/units/{id}/dry-run/delete", {
                params,
                body: {},
              }),
            ),
          );
          return;
        }
        result(await api.DELETE("/api/v1/units/{id}", { params }));
      }
      saved();
    } catch (error) {
      setError(message(error));
      if (preview) setImpact(null);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal title={title} close={close} busy={busy}>
      <form onSubmit={submit}>
        {mode === "create" || mode === "edit" ? (
          <>
            <label className="field">
              Name
              <input
                required
                maxLength={120}
                value={name}
                onChange={(e) => setName(e.target.value)}
                disabled={busy}
              />
            </label>
            <label className="field">
              Description
              <textarea
                maxLength={2000}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                disabled={busy}
                rows={3}
              />
            </label>
            <label className="field">
              Kind <span className="muted">(optional)</span>
              <input
                maxLength={64}
                value={kind}
                onChange={(e) => setKind(e.target.value)}
                disabled={busy}
              />
            </label>
          </>
        ) : null}
        {(mode === "move" || mode === "create") && (
          <label className="field">
            Parent unit
            <select
              required
              value={parent}
              disabled={busy}
              onChange={(e) => {
                setParent(e.target.value);
                setImpact(null);
              }}
            >
              <option value="" disabled>
                Select a unit
              </option>
              {destinations.map((u) => (
                <option key={u.id} value={u.id}>
                  {u.name}
                </option>
              ))}
            </select>
          </label>
        )}
        {mode === "move" && destinations.length === 0 && (
          <p className="muted">
            No other manageable parent units are available.
          </p>
        )}
        {mode === "delete" && (
          <p>
            Delete <strong>{unit.name}</strong>? Units with direct assignments,
            child units, or connections cannot be deleted.
          </p>
        )}
        {impact && <ImpactReport impact={impact} units={units} />}
        <Feedback error={error} />
        <Actions
          busy={busy}
          close={close}
          disabled={
            (mode === "move" && !parent) ||
            (impact !== null && !impact.executable)
          }
          label={
            preview && !impact
              ? "Review impact"
              : mode === "delete"
                ? "Confirm deletion"
                : mode === "move"
                  ? "Confirm move"
                  : "Save unit"
          }
          icon={
            mode === "delete" ? (
              <Trash2 size={16} />
            ) : mode === "move" ? (
              <Move size={16} />
            ) : (
              <Save size={16} />
            )
          }
        />
      </form>
    </Modal>
  );
}

export function ImpactReport({
  impact,
  units,
}: {
  impact: Impact;
  units: Unit[];
}) {
  return (
    <section className="impact-report" aria-label="Impact preview">
      <h3>Impact preview</h3>
      {impact.blockers.map((blocker) => (
        <p className="notice" key={blocker}>
          {{
            child_units: "This unit contains child units.",
            direct_assignments:
              "Revoke direct assignments before deleting this unit.",
            connections: "This unit contains connections.",
          }[blocker] ?? blocker}
        </p>
      ))}
      <p className="muted">
        {impact.changes.length} user/resource access changes ·{" "}
        {impact.connections.length} connections
      </p>
      {impact.changes.length > 0 ? (
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>User / resource</th>
                <th>Gained</th>
                <th>Lost</th>
              </tr>
            </thead>
            <tbody>
              {impact.changes.map((change) => (
                <tr
                  key={`${change.user_id}:${change.resource.type}:${change.resource.id}`}
                >
                  <td>
                    <code title={change.user_id}>{change.user_id}</code>
                    <span className="row-description">
                      {units.find((u) => u.id === change.resource.id)?.name ??
                        change.resource.id}
                    </span>
                  </td>
                  <td>{change.gained.join(", ") || "—"}</td>
                  <td>{change.lost.join(", ") || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <p>No effective permissions change.</p>
      )}
    </section>
  );
}
