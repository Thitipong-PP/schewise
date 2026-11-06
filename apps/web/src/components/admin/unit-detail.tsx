// SPDX-License-Identifier: Apache-2.0
"use client";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useState } from "react";
import { ArrowLeft, UserPlus, UserMinus, Shield } from "lucide-react";
import { api, csrfHeaders, message, result } from "@/lib/admin-api";
import {
  sourceLabel,
  type Assignment,
  type Unit,
  type UserInfo,
} from "@/lib/admin-model";
import { UserPicker } from "./user-picker";
import {
  Actions,
  Feedback,
  Modal,
  QueryState,
  Select,
  useQuery,
} from "./shared";

export function Grant({ assignment }: { assignment: Assignment }) {
  return (
    <div className="grant">
      <div className="name-with-badge">
        <strong>{assignment.role.name}</strong>
        {!assignment.can_revoke && <span className="badge">Read-only</span>}
      </div>
      <span className="row-description">{sourceLabel(assignment)}</span>
      {assignment.source !== "inherited_hidden" &&
        assignment.source === "inherited" &&
        assignment.source_unit && (
          <Link
            className="text-link"
            href={`/units/${assignment.source_unit.id}`}
          >
            View source unit
          </Link>
        )}
      {assignment.note && (
        <span className="assignment-note">{assignment.note}</span>
      )}
      <details className="permission-details">
        <summary>{assignment.permissions.length} permissions</summary>
        <ul>
          {assignment.permissions.map((p) => (
            <li key={p}>
              <code>{p}</code>
            </li>
          ))}
        </ul>
      </details>
      {!assignment.can_revoke &&
        assignment.read_only_reason &&
        assignment.source !== "inherited_hidden" && (
          <span className="row-description">{assignment.read_only_reason}</span>
        )}
    </div>
  );
}

export function UnitDetail({ id }: { id: string }) {
  const router = useRouter();
  const query = useQuery(
    useCallback(
      async (signal) => {
        const unit = result(
          await api.GET("/api/v1/units/{id}", {
            params: { path: { id } },
            signal,
          }),
        ).unit;
        const members = result(
          await api.GET("/api/v1/units/{id}/members", {
            params: { path: { id } },
            signal,
          }),
        ).members;
        return { unit, members };
      },
      [id],
    ),
  );
  const [assigning, setAssigning] = useState(false);
  const [revoke, setRevoke] = useState<{
    assignment: Assignment;
    email: string;
  } | null>(null);
  return (
    <>
      <Link className="back-link" href="/">
        <ArrowLeft size={15} />
        Organization
      </Link>
      <QueryState query={query} />
      {query.data && (
        <>
          <div className="page-heading">
            <div>
              <p className="section-label">
                {query.data.unit.is_root
                  ? "Root unit"
                  : query.data.unit.kind || "Unit"}
              </p>
              <h1>{query.data.unit.name}</h1>
              {query.data.unit.description && (
                <p className="muted">{query.data.unit.description}</p>
              )}
            </div>
            {query.data.unit.permissions.includes("connection:read") && (
              <Link
                className="secondary connection-unit-link"
                href={`/units/${id}/connections`}
              >
                Connections
              </Link>
            )}
            {query.data.unit.permissions.includes("unit:manage_members") && (
              <button onClick={() => setAssigning(true)}>
                <UserPlus size={16} />
                Assign role
              </button>
            )}
          </div>
          <div className="section-heading">
            <h2>Members &amp; access</h2>
            <span className="muted">{query.data.members.length} members</span>
          </div>
          {query.data.members.length ? (
            <div className="table-scroll">
              <table>
                <thead>
                  <tr>
                    <th>Member</th>
                    <th>Role / source</th>
                    <th>
                      <span className="sr-only">Actions</span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {query.data.members.flatMap((member) =>
                    member.assignments.map((assignment, index) => (
                      <tr key={`${member.user.id}:${assignment.id ?? index}`}>
                        <td>
                          {index === 0 && (
                            <>
                              <strong>{member.user.email}</strong>
                              {!member.user.is_active && (
                                <span className="badge">Inactive</span>
                              )}
                              <span className="row-description">
                                {member.permissions.length} effective
                                permissions
                              </span>
                            </>
                          )}
                        </td>
                        <td>
                          <Grant assignment={assignment} />
                        </td>
                        <td>
                          {assignment.can_revoke && assignment.id && (
                            <button
                              className="secondary"
                              onClick={() =>
                                setRevoke({
                                  assignment,
                                  email: member.user.email,
                                })
                              }
                            >
                              <UserMinus size={15} />
                              Revoke
                            </button>
                          )}
                        </td>
                      </tr>
                    )),
                  )}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="empty-state">
              No members have assignments at this unit or its ancestors.
            </p>
          )}
          {assigning && (
            <AssignDialog
              unit={query.data.unit}
              close={() => setAssigning(false)}
              saved={() => {
                setAssigning(false);
                query.reload();
                router.refresh();
              }}
            />
          )}
        </>
      )}
      {revoke && (
        <RevokeDialog
          {...revoke}
          close={() => setRevoke(null)}
          saved={() => {
            setRevoke(null);
            query.reload();
            router.refresh();
          }}
        />
      )}
    </>
  );
}

function AssignDialog({
  unit,
  close,
  saved,
}: {
  unit: Unit;
  close: () => void;
  saved: () => void;
}) {
  const query = useQuery(
    useCallback(
      async (signal) =>
        result(
          await api.GET("/api/v1/units/{id}/assignable-roles", {
            params: { path: { id: unit.id } },
            signal,
          }),
        ).roles,
      [unit.id],
    ),
  );
  const [user, setUser] = useState<UserInfo | null>(null);
  const [role, setRole] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (!user || busy) return;
    setBusy(true);
    setError("");
    try {
      result(
        await api.POST("/api/v1/assignments", {
          params: { header: await csrfHeaders() },
          body: {
            user_id: user.id,
            role_id: role,
            scope: { type: "unit", id: unit.id },
            note,
          },
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
    <Modal title={`Assign a role at ${unit.name}`} close={close} busy={busy}>
      <form onSubmit={submit}>
        <UserPicker
          label="Member"
          unitID={unit.id}
          value={user}
          onChange={setUser}
          disabled={busy}
        />
        <QueryState query={query} />
        <label className="field">
          Role
          <Select
            required
            value={role}
            onChange={(e) => setRole(e.target.value)}
            disabled={busy || query.loading}
          >
            <option value="" disabled>
              Select a role
            </option>
            {query.data?.map((r) => (
              <option key={r.id} value={r.id}>
                {r.name}
              </option>
            ))}
          </Select>
        </label>
        {query.data?.length === 0 && (
          <p className="muted">
            No roles can be delegated with your current permissions.
          </p>
        )}
        <label className="field">
          Note <span className="muted">(optional)</span>
          <textarea
            rows={3}
            maxLength={2000}
            value={note}
            onChange={(e) => setNote(e.target.value)}
            disabled={busy}
          />
        </label>
        <Feedback error={error} />
        <Actions
          busy={busy}
          close={close}
          label="Assign role"
          disabled={!user || !role || !query.data}
          icon={<Shield size={16} />}
        />
      </form>
    </Modal>
  );
}

function RevokeDialog({
  assignment,
  email,
  close,
  saved,
}: {
  assignment: Assignment;
  email: string;
  close: () => void;
  saved: () => void;
}) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (!assignment.id) return;
    setBusy(true);
    setError("");
    try {
      result(
        await api.DELETE("/api/v1/assignments/{id}", {
          params: { path: { id: assignment.id }, header: await csrfHeaders() },
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
    <Modal title="Revoke role assignment" close={close} busy={busy}>
      <form onSubmit={submit}>
        <p>
          Revoke{" "}
          <div className="name-with-badge">
            <strong>{assignment.role.name}</strong>
            {!assignment.can_revoke && <span className="badge">Read-only</span>}
          </div>{" "}
          for <strong>{email}</strong> at{" "}
          <strong>{assignment.source_unit?.name}</strong>? Access inherited by
          descendant units will also change.
        </p>
        <Feedback error={error} />
        <Actions
          busy={busy}
          close={close}
          label="Revoke assignment"
          icon={<UserMinus size={16} />}
        />
      </form>
    </Modal>
  );
}
