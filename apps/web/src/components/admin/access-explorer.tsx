// SPDX-License-Identifier: Apache-2.0
"use client";
import Link from "next/link";
import { useCallback, useState } from "react";
import { Check, CircleSlash, ScanSearch } from "lucide-react";
import { api, csrfHeaders, message, result } from "@/lib/admin-api";
import type { components } from "@/lib/api-types";
import {
  blockingLabel,
  type Permission,
  type UserInfo,
} from "@/lib/admin-model";
import { useAccess } from "./shell";
import { Grant } from "./unit-detail";
import { UserPicker } from "./user-picker";
import { Feedback, QueryState, Select, useQuery } from "./shared";

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
  const [user, setUser] = useState<UserInfo | null>(null);
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
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (!user || busy) return;
    setBusy(true);
    setError("");
    setAnswer(null);
    try {
      const header = await csrfHeaders();
      const [explanation, access] = await Promise.all([
        api.POST("/api/v1/access/explain", {
          params: { header },
          body: {
            user_id: user.id,
            permission,
            resource: { type: "unit", id: unit },
          },
        }),
        api.GET("/api/v1/users/{id}/access", {
          params: { path: { id: user.id } },
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
            <Select
              value={unit}
              required
              disabled={busy}
              onChange={(e) => {
                setUnit(e.target.value);
                setUser(null);
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
            </Select>
          </label>
          <UserPicker
            key={unit}
            label="User"
            unitID={unit}
            value={user}
            onChange={(value) => {
              setUser(value);
              setAnswer(null);
            }}
            disabled={busy || !unit}
            selfOnly={!canInspectOthers ? context?.user : undefined}
          />
          <label className="field">
            Permission
            <Select
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
            </Select>
          </label>
          <p className="muted field-description">
            {
              query.data.catalog.find((p) => p.permission === permission)
                ?.description
            }
          </p>
          <Feedback error={error} />
          <button disabled={busy || !unit || !user} type="submit">
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
          {answer.explanation.blocked.map((block, index) => (
            <div className="blocked-grant" key={index}>
              <p className="notice">{blockingLabel(block)}</p>
              <Grant assignment={block.assignment} />
            </div>
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
