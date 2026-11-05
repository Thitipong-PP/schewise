// SPDX-License-Identifier: Apache-2.0
"use client";

import { useCallback, useState } from "react";
import {
  ChevronLeft,
  ChevronRight,
  Copy,
  Check,
  Link2,
  Plus,
  RotateCw,
  X,
} from "lucide-react";
import { api, csrfHeaders, message, result } from "@/lib/admin-api";
import type { components } from "@/lib/api-types";
import { orderedUnits, type Unit } from "@/lib/admin-model";
import { useAccess } from "./shell";
import {
  Actions,
  Feedback,
  Modal,
  QueryState,
  Select,
  useQuery,
} from "./shared";

type Invite = components["schemas"]["Invite"];
type InviteLink = components["schemas"]["InviteLink"];
type Change = { invite: Invite; regenerate: boolean };

export function Invites() {
  const context = useAccess();
  const allowed = !!context?.root_permissions.includes("user:manage");
  const readable = !!context?.can_read_units;
  const [before, setBefore] = useState("");
  const [history, setHistory] = useState<string[]>([]);
  const query = useQuery(
    useCallback(
      async (signal) => {
        if (!allowed) return null;
        return result(
          await api.GET("/api/v1/invites", {
            params: { query: { before: before || undefined } },
            signal,
          }),
        );
      },
      [allowed, before],
    ),
  );
  const units = useQuery(
    useCallback(
      async (signal) => {
        if (!allowed || !readable) return [];
        return result(await api.GET("/api/v1/units", { signal })).units;
      },
      [allowed, readable],
    ),
  );
  const [creating, setCreating] = useState(false);
  const [change, setChange] = useState<Change | null>(null);
  const [link, setLink] = useState<InviteLink | null>(null);
  function reload() {
    query.reload();
  }
  function created(value: InviteLink) {
    setCreating(false);
    setChange(null);
    setLink(value);
    setBefore("");
    setHistory([]);
    reload();
  }
  if (!allowed)
    return (
      <>
        <h1>Invites</h1>
        <p className="muted">
          You do not have permission to manage invitations.
        </p>
      </>
    );
  return (
    <>
      <div className="page-heading">
        <div>
          <p className="section-label">Workspace administration</p>
          <h1>Invites</h1>
        </div>
        <button
          disabled={
            !units.data?.some((u) =>
              u.permissions.includes("unit:manage_members"),
            )
          }
          onClick={() => setCreating(true)}
        >
          <Plus size={16} />
          Create invite
        </button>
      </div>
      <QueryState query={query} />
      {units.error && <Feedback error={units.error} />}
      {query.data && (
        <>
          {query.data.invites.length ? (
            <div className="table-scroll">
              <table>
                <thead>
                  <tr>
                    <th>Email</th>
                    <th>Role / scope</th>
                    <th>Status</th>
                    <th>Expires</th>
                    <th>
                      <span className="sr-only">Actions</span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {query.data.invites.map((invite) => (
                    <tr key={invite.id}>
                      <td>
                        <strong>{invite.email}</strong>
                        {invite.note && (
                          <span className="row-description">{invite.note}</span>
                        )}
                      </td>
                      <td>
                        {invite.role_name}
                        <span className="row-description">
                          {invite.scope_name}
                        </span>
                      </td>
                      <td>
                        <span className="badge">{invite.status}</span>
                      </td>
                      <td>{new Date(invite.expires_at).toLocaleString()}</td>
                      <td>
                        <div className="row-actions">
                          {!invite.used_at && !invite.revoked_at && (
                            <>
                              <button
                                className="icon-button"
                                title={`Regenerate invitation for ${invite.email}`}
                                aria-label={`Regenerate invitation for ${invite.email}`}
                                onClick={() =>
                                  setChange({ invite, regenerate: true })
                                }
                              >
                                <RotateCw size={16} />
                              </button>
                              <button
                                className="icon-button"
                                title={`Revoke invitation for ${invite.email}`}
                                aria-label={`Revoke invitation for ${invite.email}`}
                                onClick={() =>
                                  setChange({ invite, regenerate: false })
                                }
                              >
                                <X size={16} />
                              </button>
                            </>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="muted">No invitations.</p>
          )}
          <div className="invite-pagination">
            <button
              className="secondary"
              disabled={!history.length}
              onClick={() => {
                setBefore(history.at(-1) ?? "");
                setHistory((v) => v.slice(0, -1));
              }}
            >
              <ChevronLeft size={16} />
              Previous
            </button>
            <button
              className="secondary"
              disabled={!query.data.next_before}
              onClick={() => {
                setHistory((v) => [...v, before]);
                setBefore(query.data?.next_before ?? "");
              }}
            >
              Next
              <ChevronRight size={16} />
            </button>
          </div>
        </>
      )}
      {creating && (
        <CreateInvite
          units={units.data ?? []}
          close={() => setCreating(false)}
          saved={created}
        />
      )}
      {change && (
        <ChangeInvite
          change={change}
          close={() => setChange(null)}
          saved={(value) => {
            if (value) created(value);
            else {
              setChange(null);
              reload();
            }
          }}
        />
      )}
      {link && <OneTimeLink value={link} close={() => setLink(null)} />}
    </>
  );
}

function CreateInvite({
  units,
  close,
  saved,
}: {
  units: Unit[];
  close: () => void;
  saved: (v: InviteLink) => void;
}) {
  const [unit, setUnit] = useState("");
  const [role, setRole] = useState("");
  const [email, setEmail] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const roles = useQuery(
    useCallback(
      async (signal) => {
        if (!unit) return [];
        return result(
          await api.GET("/api/v1/units/{id}/assignable-roles", {
            params: { path: { id: unit } },
            signal,
          }),
        ).roles;
      },
      [unit],
    ),
  );
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      saved(
        result(
          await api.POST("/api/v1/invites", {
            params: { header: await csrfHeaders() },
            body: {
              email,
              role_id: role,
              scope: { type: "unit", id: unit },
              note,
            },
          }),
        ),
      );
    } catch (cause) {
      setError(message(cause));
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal title="Create invite" close={close} busy={busy}>
      <form onSubmit={submit}>
        <label className="field">
          Email
          <input
            type="email"
            autoComplete="email"
            required
            maxLength={254}
            value={email}
            disabled={busy}
            onChange={(e) => setEmail(e.target.value)}
          />
        </label>
        <label className="field">
          Scope
          <Select
            required
            value={unit}
            disabled={busy}
            onChange={(e) => {
              setUnit(e.target.value);
              setRole("");
            }}
          >
            <option value="" disabled>
              Select a unit
            </option>
            {orderedUnits(units)
              .filter(({ unit }) =>
                unit.permissions.includes("unit:manage_members"),
              )
              .map(({ unit, level }) => (
                <option key={unit.id} value={unit.id}>
                  {"— ".repeat(level)}
                  {unit.name}
                </option>
              ))}
          </Select>
        </label>
        <label className="field">
          Role
          <Select
            required
            value={role}
            disabled={busy || roles.loading || !unit}
            onChange={(e) => setRole(e.target.value)}
          >
            <option value="" disabled>
              {roles.loading ? "Loading roles…" : "Select a role"}
            </option>
            {roles.data?.map((role) => (
              <option key={role.id} value={role.id}>
                {role.name}
              </option>
            ))}
          </Select>
        </label>
        {unit && !roles.loading && !roles.error && !roles.data?.length && (
          <p className="muted">No roles can be delegated at this scope.</p>
        )}
        {roles.error && <QueryState query={roles} />}
        <label className="field">
          Note (optional)
          <textarea
            rows={2}
            maxLength={2000}
            value={note}
            disabled={busy}
            onChange={(e) => setNote(e.target.value)}
          />
        </label>
        <Feedback error={error} />
        <Actions
          busy={busy}
          close={close}
          label="Create invite"
          disabled={!unit || !role || roles.loading || !!roles.error}
          icon={<Link2 size={16} />}
        />
      </form>
    </Modal>
  );
}

function ChangeInvite({
  change: { invite, regenerate },
  close,
  saved,
}: {
  change: Change;
  close: () => void;
  saved: (v?: InviteLink) => void;
}) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      const params = { path: { id: invite.id }, header: await csrfHeaders() };
      if (regenerate)
        saved(
          result(
            await api.POST("/api/v1/invites/{id}/regenerate", {
              params,
              body: {},
            }),
          ),
        );
      else {
        result(
          await api.POST("/api/v1/invites/{id}/revoke", { params, body: {} }),
        );
        saved();
      }
    } catch (cause) {
      setError(message(cause));
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal
      title={regenerate ? "Regenerate invitation" : "Revoke invitation"}
      close={close}
      busy={busy}
    >
      <form onSubmit={submit}>
        <p>{invite.email}</p>
        <p>
          {regenerate
            ? "The old link will stop working. A replacement link will be shown once with a new expiry."
            : "This link will stop working. The person will no longer be able to accept this invitation."}
        </p>
        <Feedback error={error} />
        <Actions
          busy={busy}
          close={close}
          label={regenerate ? "Regenerate link" : "Revoke invite"}
          icon={regenerate ? <RotateCw size={16} /> : <X size={16} />}
        />
      </form>
    </Modal>
  );
}

function OneTimeLink({
  value,
  close,
}: {
  value: InviteLink;
  close: () => void;
}) {
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState("");
  async function copy() {
    try {
      await navigator.clipboard.writeText(value.link);
      setCopied(true);
      setError("");
    } catch {
      setError("Copy failed. Select and copy the link manually.");
    }
  }
  return (
    <Modal title="Invitation created" close={close}>
      <p>
        {value.invite.email} · {value.invite.role_name} ·{" "}
        {value.invite.scope_name}
      </p>
      <p className="notice">
        Copy this link now. It is shown only once and cannot be retrieved after
        closing. Share it privately with the invited person.
      </p>
      <label className="field">
        Invitation link
        <textarea
          className="invite-link"
          rows={3}
          readOnly
          value={value.link}
          onFocus={(e) => e.target.select()}
        />
      </label>
      <p className="muted">
        Expires {new Date(value.invite.expires_at).toLocaleString()}
      </p>
      <Feedback error={error} />
      <div className="dialog-actions">
        <button className="secondary" onClick={close}>
          Close
        </button>
        <button onClick={copy}>
          {copied ? <Check size={16} /> : <Copy size={16} />}
          <span role="status">{copied ? "Copied" : "Copy link"}</span>
        </button>
      </div>
    </Modal>
  );
}
