// SPDX-License-Identifier: Apache-2.0
"use client";
import Link from "next/link";
import { useCallback, useState } from "react";
import { useRouter } from "next/navigation";
import { CircleAlert, Eye, EyeOff, LoaderCircle, UserPlus } from "lucide-react";
import { api, csrfHeaders, message, result } from "@/lib/admin-api";
import { useQuery } from "@/components/admin/shared";

export function InviteForm({ token }: { token: string }) {
  const router = useRouter();
  const query = useQuery(
    useCallback(
      async (signal) =>
        result(
          await api.GET("/api/v1/invites/{token}", {
            params: { path: { token } },
            signal,
            cache: "no-store",
          }),
        ),
      [token],
    ),
  );
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [visible, setVisible] = useState(false);
  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;
    const fields = new FormData(event.currentTarget);
    setBusy(true);
    setVisible(false);
    setError("");
    try {
      result(
        await api.POST("/api/v1/invites/{token}/accept", {
          params: { path: { token }, header: await csrfHeaders() },
          body: {
            display_name: String(fields.get("display_name") ?? ""),
            password: String(fields.get("password") ?? ""),
          },
        }),
      );
      router.replace("/");
      router.refresh();
    } catch (cause) {
      setError(message(cause));
      setBusy(false);
    }
  }
  if (query.loading) return <p role="status">Loading invitation…</p>;
  if (query.error)
    return (
      <>
        <p className="login-error" role="alert">
          <CircleAlert size={16} />
          {query.error}
        </p>
        <Link href="/login" prefetch={false}>
          Back to sign in
        </Link>
      </>
    );
  if (!query.data) return null;
  return (
    <>
      <p className="invite-summary">
        {query.data.role_name} · {query.data.scope_name}
      </p>
      <form
        className="login-form"
        aria-labelledby="invite-title"
        aria-busy={busy}
        onSubmit={submit}
      >
        <div className="login-field">
          <label htmlFor="invite-email">Email</label>
          <input
            id="invite-email"
            type="email"
            autoComplete="username"
            readOnly
            value={query.data.email}
          />
        </div>
        <div className="login-field">
          <label htmlFor="display-name">Display name</label>
          <input
            id="display-name"
            name="display_name"
            autoComplete="name"
            required
            maxLength={120}
            disabled={busy}
          />
        </div>
        <div className="login-field">
          <label htmlFor="invite-password">Password</label>
          <div className="login-password-field">
            <input
              id="invite-password"
              name="password"
              type={visible ? "text" : "password"}
              autoComplete="new-password"
              required
              minLength={12}
              maxLength={128}
              aria-describedby="invite-password-policy"
              disabled={busy}
            />
            <button
              className="login-password-toggle"
              type="button"
              disabled={busy}
              aria-controls="invite-password"
              aria-label={visible ? "Hide password" : "Show password"}
              title={visible ? "Hide password" : "Show password"}
              onClick={() => setVisible((v) => !v)}
            >
              {visible ? <EyeOff size={18} /> : <Eye size={18} />}
            </button>
          </div>
          <small id="invite-password-policy">
            12–128 characters. Avoid common passwords.
          </small>
        </div>
        {error && (
          <p className="login-error" role="alert">
            <CircleAlert size={16} />
            {error}
          </p>
        )}
        <button className="login-submit" disabled={busy} type="submit">
          {busy ? (
            <LoaderCircle size={17} className="login-spinner" />
          ) : (
            <UserPlus size={17} />
          )}
          <span>{busy ? "Joining…" : "Create account and join"}</span>
        </button>
      </form>
    </>
  );
}
