// SPDX-License-Identifier: Apache-2.0
"use client";
import Link from "next/link";
import { useCallback, useState } from "react";
import { Check, CircleSlash, ScanSearch } from "lucide-react";
import { api, csrfHeaders, message, result } from "@/lib/admin-api";
import type { components } from "@/lib/api-types";
import type { Permission } from "@/lib/admin-model";
import { useAccess } from "./shell";
import { Grant } from "./unit-detail";
import { Feedback, QueryState, useQuery } from "./shared";

type Explanation = components["schemas"]["ExplanationView"];
type UserAccess = components["schemas"]["UserAccess"];

export function AccessExplorer() {
  const context = useAccess();
  const query = useQuery(
    useCallback(async (signal) => {
      const [units, catalog] = await Promise.all([
        api.GET("/api/v1/units", { signal }),
        api.GET("/api/v1/permissions", { signal }),
      ]);
      return {
        units: result(units).units,
        catalog: result(catalog).permissions,
      };
    }, []),
  );
  const [unit, setUnit] = useState("");
  const [user, setUser] = useState("");
  const [search, setSearch] = useState("");
  const [permission, setPermission] = useState<Permission>("unit:read");
  const [answer, setAnswer] = useState<{
    explanation: Explanation;
    access: UserAccess;
  } | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const selectedUnit = query.data?.units.find((u) => u.id === unit);
  const canInspectOthers =
    context?.root_permissions.includes("user:manage") ||
    selectedUnit?.permissions.includes("unit:manage_members");
  const users = useQuery(
    useCallback(
      async (signal) => {
        if (!context) return [];
        if (!unit || !canInspectOthers) return [context.user];
        return result(
          await api.GET("/api/v1/users", {
            params: { query: { unit_id: unit, q: search } },
            signal,
          }),
        ).users;
      },
      [context, unit, canInspectOthers, search],
    ),
  );
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError("");
    setAnswer(null);
    try {
      const header = await csrfHeaders();
      const [explanation, access] = await Promise.all([
        api.POST("/api/v1/access/explain", {
          params: { header },
          body: {
            user_id: user,
            permission,
            resource: { type: "unit", id: unit },
          },
        }),
        api.GET("/api/v1/users/{id}/access", {
          params: { path: { id: user } },
        }),
      ]);
      setAnswer({ explanation: result(explanation), access: result(access) });
    } catch (error) {
      setError(message(error));
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <div className="page-heading">
        <div>
          <p className="section-label">Authorization</p>
          <h1>Access explorer</h1>
        </div>
      </div>
      <QueryState query={query} />
      {query.data && (
        <form className="explorer-form" onSubmit={submit}>
          <label className="field">
            Resource
            <select
              value={unit}
              required
              disabled={busy}
              onChange={(e) => {
                setUnit(e.target.value);
                setUser("");
                setSearch("");
                setAnswer(null);
              }}
            >
              <option value="" disabled>
                Select a unit
              </option>
              {query.data.units.map((u) => (
                <option key={u.id} value={u.id}>
                  {u.name}
                </option>
              ))}
            </select>
          </label>
          {canInspectOthers && (
            <label className="field">
              Find a user
              <input
                type="search"
                placeholder="Search by email"
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setUser("");
                  setAnswer(null);
                }}
                disabled={busy}
              />
            </label>
          )}
          <label className="field">
            User
            <select
              value={user}
              required
              disabled={busy || users.loading || !unit}
              onChange={(e) => {
                setUser(e.target.value);
                setAnswer(null);
              }}
            >
              <option value="" disabled>
                Select a user
              </option>
              {users.data?.map((u) => (
                <option key={u.id} value={u.id}>
                  {u.email}
                </option>
              ))}
            </select>
          </label>
          <label className="field">
            Permission
            <select
              value={permission}
              required
              disabled={busy}
              onChange={(e) => {
                setPermission(e.target.value as Permission);
                setAnswer(null);
              }}
            >
              {query.data.catalog.map((p) => (
                <option key={p.permission} value={p.permission}>
                  {p.label} ({p.permission})
                </option>
              ))}
            </select>
          </label>
          <p className="muted field-description">
            {
              query.data.catalog.find((p) => p.permission === permission)
                ?.description
            }
          </p>
          <Feedback error={users.error || error} />
          <button
            disabled={busy || !unit || !user || !users.data}
            type="submit"
          >
            <ScanSearch size={16} />
            {busy ? "Checking…" : "Explain access"}
          </button>
        </form>
      )}
      {answer && (
        <div className="explorer-result" aria-live="polite">
          <div className="section-heading">
            <h2 className="result-label">
              {answer.explanation.allowed ? (
                <Check size={20} />
              ) : (
                <CircleSlash size={20} />
              )}
              {answer.explanation.allowed ? "Allowed" : "Denied"}
            </h2>
          </div>
          <p>{answer.explanation.reason_text}</p>
          {answer.explanation.grants.map((grant, i) => (
            <Grant key={grant.id ?? i} assignment={grant} />
          ))}
          <div className="section-heading">
            <h2>Accessible units</h2>
          </div>
          <p className="muted">
            {answer.access.user.email} · Units visible to you
          </p>
          {answer.access.units.length ? (
            <div className="table-scroll">
              <table>
                <thead>
                  <tr>
                    <th>Unit</th>
                    <th>Effective permissions</th>
                  </tr>
                </thead>
                <tbody>
                  {answer.access.units.map((item) => (
                    <tr key={item.unit.id}>
                      <td>
                        <Link href={`/units/${item.unit.id}`}>
                          {item.unit.name}
                        </Link>
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
            <p>No accessible units within your visible scope.</p>
          )}
        </div>
      )}
    </>
  );
}
