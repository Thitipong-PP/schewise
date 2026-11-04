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
    <form className="connection auth-form" onSubmit={submit}>
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
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      <button type="submit" disabled={pending}>
        {pending ? "Signing in..." : "Sign in"}
      </button>
    </form>
  );
}
