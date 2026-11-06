// SPDX-License-Identifier: Apache-2.0
"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import {
  Check,
  ChevronLeft,
  ChevronRight,
  Copy,
  KeyRound,
  Search,
  UserCheck,
  UserX,
} from "lucide-react";
import { api, csrfHeaders, message, result } from "@/lib/admin-api";
import type { components } from "@/lib/api-types";
import { useAccess } from "./shell";
import { Actions, Feedback, Modal, QueryState, useQuery } from "./shared";

type User = components["schemas"]["ManagedUser"];
type ResetLink = components["schemas"]["PasswordResetLink"];
type Selection = { user: User; action: "deactivate" | "reactivate" | "reset" };

export function Users() {
  const context = useAccess();
  const allowed = !!context?.root_permissions.includes("user:manage");
  const [text, setText] = useState("");
  const [search, setSearch] = useState("");
  const [before, setBefore] = useState("");
  const [history, setHistory] = useState<string[]>([]);
  useEffect(() => {
    const timer = setTimeout(() => setSearch(text), 250);
    return () => clearTimeout(timer);
  }, [text]);
  const query = useQuery(
    useCallback(
      async (signal) => {
        if (!allowed) return null;
        return result(
          await api.GET("/api/v1/admin/users", {
            params: { query: { q: search, before: before || undefined } },
            signal,
          }),
        );
      },
      [allowed, search, before],
    ),
    true,
  );
  const pending = query.loading || search !== text;
  const [selection, setSelection] = useState<Selection | null>(null);
  const [link, setLink] = useState<ResetLink | null>(null);
  if (!allowed)
    return (
      <>
        <h1>Users</h1>
        <p className="muted">You do not have permission to manage users.</p>
      </>
    );
  return (
    <>
      <div className="page-heading">
        <div>
          <p className="section-label">Workspace administration</p>
          <h1>Users</h1>
        </div>
      </div>
      <div className="users-toolbar">
        <label className="field">
          Search users
          <span className="users-search">
            <Search size={16} aria-hidden="true" />
            <input
              type="search"
              placeholder="Email or display name"
              maxLength={254}
              value={text}
              onChange={(e) => {
                setText(e.target.value);
                setBefore("");
                setHistory([]);
              }}
            />
          </span>
        </label>
        <span className="muted" role="status">
          {pending
            ? "Searching…"
            : query.data
              ? `${query.data.users.length} ${query.data.users.length === 1 ? "user" : "users"}`
              : ""}
        </span>
      </div>
      {!query.data && <QueryState query={query} />}
      {query.data && (
        <>
          {query.data.users.length ? (
            <div className="table-scroll" aria-busy={pending}>
              <table>
                <thead>
                  <tr>
                    <th>User</th>
                    <th>Status</th>
                    <th>Roles</th>
                    <th>Last sign-in</th>
                    <th>
                      <span className="sr-only">Actions</span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {query.data.users.map((user) => (
                    <tr key={user.id}>
                      <td>
                        <strong>{user.display_name || user.email}</strong>
                        {user.display_name && (
                          <span className="row-description">{user.email}</span>
                        )}
                        {user.id === context?.user.id && (
                          <span className="row-description">You</span>
                        )}
                      </td>
                      <td>
                        <span className="badge">
                          {user.is_active ? "Active" : "Deactivated"}
                        </span>
                        {user.deactivate_reason && (
                          <span className="row-description">
                            {user.deactivate_reason}
                          </span>
                        )}
                      </td>
                      <td>{user.role || "No assignments"}</td>
                      <td>
                        {user.last_login_at
                          ? new Date(user.last_login_at).toLocaleString()
                          : "Not recorded"}
                      </td>
                      <td>
                        <div className="row-actions">
                          {user.is_active ? (
                            <button
                              className="icon-button"
                              aria-label={`Deactivate ${user.email}`}
                              title={
                                user.deactivate_reason ||
                                `Deactivate ${user.email}`
                              }
                              disabled={pending || !user.can_deactivate}
                              onClick={() =>
                                setSelection({ user, action: "deactivate" })
                              }
                            >
                              <UserX size={17} />
                            </button>
                          ) : (
                            <button
                              className="icon-button"
                              aria-label={`Reactivate ${user.email}`}
                              title={`Reactivate ${user.email}`}
                              disabled={pending}
                              onClick={() =>
                                setSelection({ user, action: "reactivate" })
                              }
                            >
                              <UserCheck size={17} />
                            </button>
                          )}
                          <button
                            className="icon-button"
                            aria-label={`Create password reset link for ${user.email}`}
                            title={
                              user.is_active
                                ? "Create password reset link"
                                : "Reactivate this account first"
                            }
                            disabled={pending || !user.is_active}
                            onClick={() =>
                              setSelection({ user, action: "reset" })
                            }
                          >
                            <KeyRound size={17} />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="muted">No users match your search.</p>
          )}
          {query.error && <QueryState query={query} />}
          <div className="invite-pagination">
            <button
              className="secondary"
              disabled={pending || !history.length}
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
              disabled={pending || !query.data.next_before}
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
      {selection && (
        <UserAction
          selection={selection}
          self={selection.user.id === context?.user.id}
          close={() => setSelection(null)}
          saved={(value) => {
            setSelection(null);
            if (value) setLink(value);
            query.reload();
          }}
        />
      )}
      {link && <ResetLinkDialog value={link} close={() => setLink(null)} />}
    </>
  );
}

function UserAction({
  selection: { user, action },
  self,
  close,
  saved,
}: {
  selection: Selection;
  self: boolean;
  close: () => void;
  saved: (v?: ResetLink) => void;
}) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      const params = { path: { id: user.id }, header: await csrfHeaders() };
      if (action === "reset")
        saved(
          result(
            await api.POST("/api/v1/admin/users/{id}/password-reset", {
              params,
              body: {},
            }),
          ),
        );
      else {
        if (action === "deactivate")
          result(
            await api.POST("/api/v1/admin/users/{id}/deactivate", {
              params,
              body: {},
            }),
          );
        else
          result(
            await api.POST("/api/v1/admin/users/{id}/reactivate", {
              params,
              body: {},
            }),
          );
        if (self && action === "deactivate") {
          router.replace("/login");
          router.refresh();
        } else {
          saved();
          router.refresh();
        }
      }
    } catch (cause) {
      setError(message(cause));
    } finally {
      setBusy(false);
    }
  }
  const title =
    action === "reset"
      ? "Create password reset link"
      : action === "deactivate"
        ? "Deactivate user"
        : "Reactivate user";
  return (
    <Modal title={title} close={close} busy={busy}>
      <form onSubmit={submit}>
        <p>
          <strong>{user.email}</strong>
        </p>
        <p>
          {action === "reset"
            ? "A new link will revoke all previous pending reset links. The link will be shown once; share it privately with this person."
            : action === "deactivate"
              ? self
                ? "Your account will be deactivated and all your sessions signed out. Another account manager must reactivate you."
                : "All sessions will be signed out and sign-in blocked. The account and role assignments will be preserved."
              : "The user can sign in again with their existing password. No sessions or old reset links will be restored."}
        </p>
        <Feedback error={error} />
        <Actions
          busy={busy}
          close={close}
          label={title}
          icon={
            action === "reset" ? (
              <KeyRound size={16} />
            ) : action === "deactivate" ? (
              <UserX size={16} />
            ) : (
              <UserCheck size={16} />
            )
          }
        />
      </form>
    </Modal>
  );
}

function ResetLinkDialog({
  value,
  close,
}: {
  value: ResetLink;
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
    <Modal title="Password reset link created" close={close}>
      <p>{value.email}</p>
      <p className="notice">
        Copy this link now. It cannot be retrieved after closing. Share it
        privately with the account owner.
      </p>
      <label className="field">
        Password reset link
        <textarea
          readOnly
          rows={3}
          value={value.link}
          onFocus={(e) => e.target.select()}
        />
      </label>
      <p className="muted">
        Expires {new Date(value.expires_at).toLocaleString()}
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
