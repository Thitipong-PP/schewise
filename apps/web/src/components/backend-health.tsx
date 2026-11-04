"use client";

import { useState } from "react";

type Status = "idle" | "checking" | "connected" | "unavailable";

const messages: Record<Status, string> = {
  idle: "Ready to check the Go API connection.",
  checking: "Connecting to the Go API...",
  connected: "Connected to the Go API.",
  unavailable:
    "Could not reach the Go API. Check that it is running and retry.",
};

export function BackendHealth() {
  const [status, setStatus] = useState<Status>("idle");

  async function checkConnection() {
    setStatus("checking");
    try {
      const response = await fetch("/api/v1/health", {
        cache: "no-store",
        signal: AbortSignal.timeout(5000),
      });
      if (!response.ok) throw new Error("API health request failed");
      const body: unknown = await response.json();
      if (
        typeof body !== "object" ||
        body === null ||
        !("status" in body) ||
        body.status !== "ok" ||
        !("service" in body) ||
        body.service !== "schewise-api"
      ) {
        throw new Error("Unexpected API health response");
      }
      setStatus("connected");
    } catch {
      setStatus("unavailable");
    }
  }

  return (
    <section className="connection" aria-labelledby="connection-heading">
      <h2 id="connection-heading">API connection</h2>
      <p role="status" aria-live="polite" data-status={status}>
        {messages[status]}
      </p>
      <button
        type="button"
        onClick={checkConnection}
        disabled={status === "checking"}
      >
        {status === "checking" ? "Checking..." : "Check API connection"}
      </button>
    </section>
  );
}
