// SPDX-License-Identifier: Apache-2.0
"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useState } from "react";
import {
  Eye,
  EyeOff,
  KeyRound,
  LogOut,
  Monitor,
  Save,
  ShieldCheck,
  UserRound,
} from "lucide-react";
import { api, csrfHeaders, message, result } from "@/lib/admin-api";
import type { components } from "@/lib/api-types";
import {
  Actions,
  Feedback,
  Modal,
  QueryState,
  useQuery,
} from "@/components/admin/shared";
import { Grant } from "@/components/admin/unit-detail";

export type AccountTab = "profile" | "security" | "sessions" | "access";
const tabs = [
  { id: "profile", label: "Profile", icon: UserRound },
  { id: "security", label: "Security", icon: KeyRound },
  { id: "sessions", label: "Sessions", icon: Monitor },
  { id: "access", label: "My access", icon: ShieldCheck },
] as const;

export function Account({ tab }: { tab: AccountTab }) {
  return (
    <>
      <div className="page-heading">
        <div>
          <p className="section-label">Your account</p>
          <h1>Account</h1>
        </div>
      </div>
      <nav className="account-tabs" aria-label="Account settings">
        {tabs.map(({ id, label, icon: Icon }) => (
          <Link
            key={id}
            href={`/account/${id}`}
            aria-current={tab === id ? "page" : undefined}
          >
            <Icon size={16} />
            {label}
          </Link>
        ))}
      </nav>
      {tab === "profile" ? (
        <Profile />
      ) : tab === "security" ? (
        <Security />
      ) : tab === "sessions" ? (
        <Sessions />
      ) : (
        <MyAccess />
      )}
    </>
  );
}

function Profile() {
  const query = useQuery(
    useCallback(
      async (signal) =>
        result(await api.GET("/api/v1/me/profile", { signal })).user,
      [],
    ),
  );
  return (
    <>
      <QueryState query={query} />
      {query.data && <ProfileForm user={query.data} />}
    </>
  );
}
function ProfileForm({ user }: { user: components["schemas"]["User"] }) {
  const router = useRouter();
  const [name, setName] = useState(user.display_name);
  const [baseline, setBaseline] = useState(user.display_name);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (busy) return;
    setBusy(true);
    setError("");
    setSaved(false);
    try {
      const updated = result(
        await api.PATCH("/api/v1/me/profile", {
          params: { header: await csrfHeaders() },
          body: { display_name: name },
        }),
      ).user;
      setName(updated.display_name);
      setBaseline(updated.display_name);
      setSaved(true);
      router.refresh();
    } catch (cause) {
      setError(message(cause));
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="account-section">
      <h2>Profile</h2>
      <form className="account-form" onSubmit={submit} aria-busy={busy}>
        <label className="field">
          Email
          <input
            type="email"
            autoComplete="username"
            readOnly
            value={user.email}
          />
        </label>
        <label className="field">
          Display name
          <input
            autoComplete="name"
            required
            maxLength={120}
            value={name}
            disabled={busy}
            onChange={(e) => {
              setName(e.target.value);
              setSaved(false);
            }}
          />
        </label>
        <Feedback error={error} />
        {saved && (
          <p role="status" className="muted">
            Profile saved.
          </p>
        )}
        <button
          type="submit"
          disabled={busy || name.trim() === baseline || !name.trim()}
        >
          <Save size={16} />
          {busy ? "Saving…" : "Save changes"}
        </button>
      </form>
    </section>
  );
}

function PasswordInput({
  label,
  name,
  current,
  busy,
}: {
  label: string;
  name: string;
  current?: boolean;
  busy: boolean;
}) {
  const [visible, setVisible] = useState(false);
  return (
    <div className="field">
      <label htmlFor={name}>{label}</label>
      <span className="account-password-control">
        <input
          id={name}
          aria-describedby={current ? undefined : "account-password-policy"}
          name={name}
          type={visible ? "text" : "password"}
          autoComplete={current ? "current-password" : "new-password"}
          required
          minLength={current ? undefined : 12}
          maxLength={current ? 512 : 128}
          disabled={busy}
        />
        <button
          className="icon-button"
          aria-controls={name}
          type="button"
          aria-label={`${visible ? "Hide" : "Show"} ${label.toLowerCase()}`}
          title={`${visible ? "Hide" : "Show"} password`}
          disabled={busy}
          onClick={() => setVisible((v) => !v)}
        >
          {visible ? <EyeOff size={17} /> : <Eye size={17} />}
        </button>
      </span>
    </div>
  );
}

function Security() {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);
  const [version, setVersion] = useState(0);
  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;
    const form = event.currentTarget;
    const data = new FormData(form);
    setBusy(true);
    setError("");
    setSaved(false);
    try {
      result(
        await api.POST("/api/v1/me/password", {
          params: { header: await csrfHeaders() },
          body: {
            current_password: String(data.get("current_password") ?? ""),
            password: String(data.get("password") ?? ""),
          },
        }),
      );
      form.reset();
      setVersion((v) => v + 1);
      setSaved(true);
    } catch (cause) {
      setError(message(cause));
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="account-section">
      <h2>Change password</h2>
      <p className="muted">
        Your other sessions will be signed out. This session stays signed in.
      </p>
      <form className="account-form" onSubmit={submit} aria-busy={busy}>
        <PasswordInput
          key={`old:${version}`}
          label="Current password"
          name="current_password"
          current
          busy={busy}
        />
        <PasswordInput
          key={`new:${version}`}
          label="New password"
          name="password"
          busy={busy}
        />
        <p className="muted field-description" id="account-password-policy">
          12–128 characters. Avoid common passwords.
        </p>
        <Feedback error={error} />
        {saved && (
          <p role="status" className="muted">
            Password updated. Your other sessions are signed out.
          </p>
        )}
        <button type="submit" disabled={busy}>
          <KeyRound size={16} />
          {busy ? "Updating…" : "Update password"}
        </button>
      </form>
    </section>
  );
}

type Session = components["schemas"]["SessionInfo"];
function Sessions() {
  const router = useRouter();
  const query = useQuery(
    useCallback(
      async (signal) =>
        result(await api.GET("/api/v1/me/sessions", { signal })).sessions,
      [],
    ),
    true,
  );
  const [selection, setSelection] = useState<Session | "others" | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function revoke(event: React.FormEvent) {
    event.preventDefault();
    if (!selection || busy) return;
    setBusy(true);
    setError("");
    try {
      const header = await csrfHeaders();
      if (selection === "others")
        result(
          await api.POST("/api/v1/me/sessions/revoke-others", {
            params: { header },
            body: {},
          }),
        );
      else
        result(
          await api.DELETE("/api/v1/me/sessions/{id}", {
            params: { path: { id: selection.id }, header },
          }),
        );
      if (selection !== "others" && selection.current) {
        router.replace("/login");
        router.refresh();
      } else {
        setSelection(null);
        query.reload();
      }
    } catch (cause) {
      setError(message(cause));
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="account-section">
      <div className="section-heading">
        <h2>Active sessions</h2>
        <button
          className="secondary"
          disabled={query.loading || !query.data?.some((s) => !s.current)}
          onClick={() => {
            setSelection("others");
            setError("");
          }}
        >
          <LogOut size={16} />
          Sign out of other devices
        </button>
      </div>
      {!query.data && <QueryState query={query} />}
      {query.data && (
        <div className="table-scroll" aria-busy={query.loading}>
          <table>
            <thead>
              <tr>
                <th>Device</th>
                <th>IP</th>
                <th>Created</th>
                <th>Last seen</th>
                <th>
                  <span className="sr-only">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {query.data.map((session) => (
                <tr key={session.id}>
                  <td>
                    <strong>{session.user_agent}</strong>
                    {session.current && (
                      <span className="row-description">
                        <span className="badge">Current session</span>
                      </span>
                    )}
                  </td>
                  <td>{session.ip ?? "Not available"}</td>
                  <td>{new Date(session.created_at).toLocaleString()}</td>
                  <td>{new Date(session.last_seen_at).toLocaleString()}</td>
                  <td>
                    <button
                      className="secondary"
                      disabled={query.loading}
                      onClick={() => {
                        setSelection(session);
                        setError("");
                      }}
                    >
                      <LogOut size={16} />
                      {session.current ? "Sign out" : "Revoke"}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {query.data && query.error && <QueryState query={query} />}
      {selection && (
        <Modal
          title={
            selection === "others"
              ? "Sign out of other devices"
              : selection.current
                ? "Sign out of this session"
                : "Revoke session"
          }
          close={() => setSelection(null)}
          busy={busy}
        >
          <form onSubmit={revoke}>
            <p>
              {selection === "others"
                ? "All other sessions will be signed out. This session will stay signed in."
                : selection.current
                  ? "You will be signed out of this session."
                  : `Sign out ${selection.user_agent}?`}
            </p>
            <Feedback error={error} />
            <Actions
              busy={busy}
              close={() => setSelection(null)}
              label={
                selection === "others" ? "Sign out other devices" : "Sign out"
              }
              icon={<LogOut size={16} />}
            />
          </form>
        </Modal>
      )}
    </section>
  );
}

function MyAccess() {
  const query = useQuery(
    useCallback(
      async (signal) => result(await api.GET("/api/v1/me/access", { signal })),
      [],
    ),
  );
  return (
    <section className="account-section">
      <h2>My access</h2>
      <QueryState query={query} />
      {query.data &&
        (query.data.units.length ? (
          <div className="table-scroll">
            <table>
              <thead>
                <tr>
                  <th>Unit</th>
                  <th>Roles / source</th>
                  <th>Effective permissions</th>
                </tr>
              </thead>
              <tbody>
                {query.data.units.map((item) => (
                  <tr key={item.unit.id}>
                    <td>
                      <Link href={`/units/${item.unit.id}`}>
                        {item.unit.name}
                      </Link>
                    </td>
                    <td>
                      {item.assignments.map((assignment, i) => (
                        <Grant
                          key={assignment.id ?? i}
                          assignment={assignment}
                        />
                      ))}
                    </td>
                    <td className="permission-list">
                      {item.permissions.join(", ")}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <p className="muted">
            No readable units are assigned to your account.
          </p>
        ))}
    </section>
  );
}
