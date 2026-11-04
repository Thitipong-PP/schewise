"use client";

import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { authMutation } from "@/lib/auth-client";

export function LoginForm() {
  const router = useRouter();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError("");
    const data = new FormData(event.currentTarget);
    try {
      await authMutation("login", {
        email: String(data.get("email") ?? ""),
        password: String(data.get("password") ?? ""),
      });
      const next = new URLSearchParams(window.location.search).get("next");
      // Resolve and compare origins to reject protocol-relative and backslash
      // redirects, while retaining a protected local destination.
      const destination = new URL(next ?? "/", window.location.origin);
      router.replace(
        destination.origin === window.location.origin &&
          destination.pathname !== "/login"
          ? destination.pathname + destination.search
          : "/",
      );
      router.refresh();
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "Sign in failed. Please retry.",
      );
      setPending(false);
    }
  }

  return (
    <form
      className="login-form"
      onSubmit={submit}
      aria-labelledby="login-form-title"
      aria-busy={pending}
    >
      <div className="login-form-heading">
        <span className="login-form-kicker">Workspace access</span>
        <h2 id="login-form-title">Sign in</h2>
      </div>
      <div className="login-field">
        <label htmlFor="email">Email</label>
        <input
          id="email"
          name="email"
          type="email"
          autoComplete="username"
          required
          maxLength={254}
          disabled={pending}
        />
      </div>
      <div className="login-field">
        <label htmlFor="password">Password</label>
        <input
          id="password"
          name="password"
          type="password"
          autoComplete="current-password"
          required
          maxLength={256}
          disabled={pending}
        />
      </div>
      {error && (
        <p className="login-error" role="alert">
          {error}
        </p>
      )}
      <button type="submit" disabled={pending}>
        <span>{pending ? "Signing in..." : "Sign in"}</span>
        <svg
          width="20"
          height="20"
          viewBox="0 0 24 24"
          fill="none"
          aria-hidden="true"
        >
          <path
            d="M5 12h14m-6-6 6 6-6 6"
            stroke="currentColor"
            strokeWidth="1.7"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </svg>
      </button>
    </form>
  );
}
