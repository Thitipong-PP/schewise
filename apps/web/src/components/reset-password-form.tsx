// SPDX-License-Identifier: Apache-2.0
"use client";
import Link from "next/link";
import { useCallback, useState } from "react";
import {
  Check,
  CircleAlert,
  Eye,
  EyeOff,
  KeyRound,
  LoaderCircle,
} from "lucide-react";
import { api, csrfHeaders, message, result } from "@/lib/admin-api";
import { useQuery } from "@/components/admin/shared";

export function ResetPasswordForm({ token }: { token: string }) {
  const query = useQuery(
    useCallback(
      async (signal) =>
        result(
          await api.GET("/api/v1/reset-password/{token}", {
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
  const [done, setDone] = useState(false);
  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;
    const data = new FormData(event.currentTarget);
    setBusy(true);
    setVisible(false);
    setError("");
    try {
      result(
        await api.POST("/api/v1/reset-password/{token}", {
          params: { path: { token }, header: await csrfHeaders() },
          body: { password: String(data.get("password") ?? "") },
        }),
      );
      setDone(true);
    } catch (cause) {
      setError(message(cause));
      setBusy(false);
    }
  }
  if (done)
    return (
      <>
        <p role="status" className="login-reset-success">
          <Check size={18} />
          Password updated. All previous sessions are signed out.
        </p>
        <Link href="/login" prefetch={false}>
          Sign in with your new password
        </Link>
      </>
    );
  if (query.loading) return <p role="status">Checking reset link…</p>;
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
    <form
      className="login-form"
      aria-labelledby="reset-title"
      aria-busy={busy}
      onSubmit={submit}
    >
      <div className="login-field">
        <label htmlFor="reset-email">Email</label>
        <input
          id="reset-email"
          type="email"
          autoComplete="username"
          readOnly
          value={query.data.email}
        />
      </div>
      <div className="login-field">
        <label htmlFor="reset-password">New password</label>
        <div className="login-password-field">
          <input
            id="reset-password"
            name="password"
            type={visible ? "text" : "password"}
            autoComplete="new-password"
            required
            minLength={12}
            maxLength={128}
            disabled={busy}
            aria-describedby="reset-policy"
          />
          <button
            className="login-password-toggle"
            type="button"
            disabled={busy}
            aria-label={visible ? "Hide password" : "Show password"}
            title={visible ? "Hide password" : "Show password"}
            aria-controls="reset-password"
            onClick={() => setVisible((v) => !v)}
          >
            {visible ? <EyeOff size={18} /> : <Eye size={18} />}
          </button>
        </div>
        <small id="reset-policy">
          12–128 characters. Avoid common passwords.
        </small>
      </div>
      <p className="login-reset-note">
        Updating your password signs out all existing sessions.
      </p>
      {error && (
        <p className="login-error" role="alert">
          <CircleAlert size={16} />
          {error}
        </p>
      )}
      <button className="login-submit" type="submit" disabled={busy}>
        {busy ? (
          <LoaderCircle className="login-spinner" size={17} />
        ) : (
          <KeyRound size={17} />
        )}
        <span>{busy ? "Updating…" : "Set new password"}</span>
      </button>
    </form>
  );
}
