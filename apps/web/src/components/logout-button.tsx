"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { authMutation } from "@/lib/auth-client";

export function LogoutButton() {
  const router = useRouter();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  async function logout() {
    setPending(true);
    setError("");
    try {
      await authMutation("logout");
      router.replace("/login");
      router.refresh();
    } catch {
      setError("Could not sign out. Please retry.");
      setPending(false);
    }
  }
  return (
    <div>
      <button type="button" disabled={pending} onClick={logout}>
        {pending ? "Signing out..." : "Sign out"}
      </button>
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
    </div>
  );
}
