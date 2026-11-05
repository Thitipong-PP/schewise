"use client";

import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { CircleAlert, Eye, EyeOff, LoaderCircle, LogIn } from "lucide-react";
import { authMutation } from "@/lib/auth-client";

export function LoginForm() {
  const router = useRouter();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const [passwordVisible, setPasswordVisible] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setPasswordVisible(false);
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
      aria-labelledby="login-title"
      aria-busy={pending}
    >
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
        <div className="login-password-field">
          <input
            id="password"
            name="password"
            type={passwordVisible ? "text" : "password"}
            autoComplete="current-password"
            required
            maxLength={512}
            disabled={pending}
          />
          <button
            className="login-password-toggle"
            type="button"
            aria-label={passwordVisible ? "Hide password" : "Show password"}
            title={passwordVisible ? "Hide password" : "Show password"}
            aria-controls="password"
            disabled={pending}
            onClick={() => setPasswordVisible((visible) => !visible)}
          >
            {passwordVisible ? (
              <EyeOff size={18} aria-hidden="true" />
            ) : (
              <Eye size={18} aria-hidden="true" />
            )}
          </button>
        </div>
      </div>
      {error && (
        <p className="login-error" role="alert">
          <CircleAlert size={16} aria-hidden="true" />
          <span>{error}</span>
        </p>
      )}
      <button className="login-submit" type="submit" disabled={pending}>
        {pending ? (
          <LoaderCircle
            className="login-spinner"
            size={17}
            aria-hidden="true"
          />
        ) : (
          <LogIn size={17} aria-hidden="true" />
        )}
        <span>{pending ? "Signing in..." : "Sign in"}</span>
      </button>
    </form>
  );
}
