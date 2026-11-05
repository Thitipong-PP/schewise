// SPDX-License-Identifier: Apache-2.0
"use client";
import Link from "next/link";
import { useCallback, useState } from "react";
import { Check, CircleSlash, RotateCcw, ScanSearch } from "lucide-react";
import { api, csrfHeaders, message, result } from "@/lib/admin-api";
import type { components } from "@/lib/api-types";
import {
  accessSearchMode,
  blockingLabel,
  filterAccessMembers,
  type AccessFilters,
  type Permission,
  type UserInfo,
} from "@/lib/admin-model";
import { useAccess } from "./shell";
import { Grant } from "./unit-detail";
import { UserPicker } from "./user-picker";
import { Feedback, QueryState, Select, useQuery } from "./shared";

type Explanation = components["schemas"]["ExplanationView"];
type UserAccess = components["schemas"]["UserAccess"];
type Member = components["schemas"]["Member"];
type SearchResult =
  | {
      mode: "members";
      members: Member[];
      unitName: string;
      user: UserInfo | null;
      permissionLabel: string;
    }
  | { mode: "user"; access: UserAccess }
  | { mode: "explain"; explanation: Explanation; access: UserAccess };

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
  const [permission, setPermission] = useState<Permission | "">("");
  const [answer, setAnswer] = useState<SearchResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [resetVersion, setResetVersion] = useState(0);
  const selectedUnit = query.data?.units.find((u) => u.id === unit);
  const globalDirectory = !!context?.root_permissions.includes("user:manage");
  // This fallback only initializes the existing scoped directory. Every API
  // still authorizes the requested user/resource and filters visible results.
  const directoryUnit =
    unit ||
    query.data?.units.find((u) => u.permissions.includes("unit:manage_members"))
      ?.id ||
    "";
  const canInspectOthers =
    globalDirectory ||
    (unit
      ? selectedUnit?.permissions.includes("unit:manage_members")
      : !!directoryUnit);
  const filters: AccessFilters = {
    unitID: unit,
    userID: user?.id ?? "",
    permission,
  };
  const mode = accessSearchMode(filters);
  function reset() {
    setUnit("");
    setUser(null);
    setPermission("");
    setAnswer(null);
    setError("");
    setResetVersion((v) => v + 1);
  }
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (!mode || busy) return;
    setBusy(true);
    setError("");
    setAnswer(null);
    try {
      if (mode === "members") {
        const members = result(
          await api.GET("/api/v1/units/{id}/access", {
            params: { path: { id: unit } },
          }),
        ).members;
        setAnswer({
          mode,
          members: filterAccessMembers(members, filters),
          unitName: selectedUnit?.name ?? "Unit",
          user,
          permissionLabel:
            query.data?.catalog.find((p) => p.permission === permission)
              ?.label ?? "",
        });
      } else if (mode === "user" && user) {
        setAnswer({
          mode,
          access: result(
            await api.GET("/api/v1/users/{id}/access", {
              params: { path: { id: user.id } },
            }),
          ),
        });
      } else if (mode === "explain" && user && permission) {
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
        setAnswer({
          mode,
          explanation: result(explanation),
          access: result(access),
        });
      }
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
            Resource <span className="muted">(optional)</span>
            <Select
              value={unit}
              disabled={busy}
              onChange={(e) => {
                const nextUnit = query.data?.units.find(
                  (u) => u.id === e.target.value,
                );
                setUnit(e.target.value);
                if (!e.target.value) setPermission("");
                if (
                  nextUnit &&
                  !globalDirectory &&
                  !nextUnit.permissions.includes("unit:manage_members") &&
                  user?.id !== context?.user.id
                )
                  setUser(null);
                setAnswer(null);
                setError("");
              }}
            >
              <option value="">Any resource</option>
              {query.data.units.map((u) => (
                <option key={u.id} value={u.id}>
                  {u.name}
                </option>
              ))}
            </Select>
          </label>
          <UserPicker
            key={`${directoryUnit}:${globalDirectory}:${resetVersion}`}
            label="User"
            unitID={globalDirectory && !unit ? "" : directoryUnit}
            allowGlobal={globalDirectory && !unit}
            required={false}
            value={user}
            onChange={(value) => {
              setUser(value);
              setAnswer(null);
              setError("");
            }}
            disabled={busy}
            selfOnly={!canInspectOthers ? context?.user : undefined}
          />
          <label className="field">
            Permission <span className="muted">(optional)</span>
            <Select
              value={permission}
              disabled={busy || !unit}
              title={!unit ? "Select a resource first" : undefined}
              onChange={(e) => {
                setPermission(e.target.value as Permission | "");
                setAnswer(null);
                setError("");
              }}
            >
              <option value="">Any permission</option>
              {query.data.catalog.map((p) => (
                <option key={p.permission} value={p.permission}>
                  {p.label} ({p.permission})
                </option>
              ))}
            </Select>
          </label>
          {permission && (
            <p className="muted field-description">
              {
                query.data.catalog.find((p) => p.permission === permission)
                  ?.description
              }
            </p>
          )}
          <Feedback error={error} />
          <div className="explorer-actions">
            <button disabled={busy || !mode} type="submit">
              <ScanSearch size={16} />
              {busy ? "Searching…" : "Search access"}
            </button>
            <button
              type="button"
              className="secondary"
              disabled={busy || (!unit && !user && !permission)}
              onClick={reset}
            >
              <RotateCcw size={15} />
              Reset
            </button>
          </div>
        </form>
      )}
      {answer && (
        <div className="explorer-result" aria-live="polite">
          {answer.mode === "members" ? (
            <MemberResults result={answer} />
          ) : (
            <>
              {answer.mode === "explain" && (
                <>
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
                  {answer.explanation.blocked.map((block, i) => (
                    <div className="blocked-grant" key={i}>
                      <p className="notice">{blockingLabel(block)}</p>
                      <Grant assignment={block.assignment} />
                    </div>
                  ))}
                </>
              )}
              <UserAccessResults access={answer.access} />
            </>
          )}
        </div>
      )}
    </>
  );
}

function MemberResults({
  result,
}: {
  result: Extract<SearchResult, { mode: "members" }>;
}) {
  return (
    <>
      <div className="section-heading">
        <h2>
          {result.user
            ? `Access at ${result.unitName}`
            : `Users with access to ${result.unitName}`}
        </h2>
        <span className="muted">
          {result.members.length}{" "}
          {result.members.length === 1 ? "user" : "users"}
        </span>
      </div>
      {result.user && <p className="muted">{result.user.email}</p>}
      {result.permissionLabel && (
        <p className="muted">{result.permissionLabel}</p>
      )}
      {result.members.length ? (
        <div className="table-scroll">
          <table className="explorer-members">
            <thead>
              <tr>
                <th>User</th>
                <th>Roles / source</th>
                <th>Effective permissions</th>
              </tr>
            </thead>
            <tbody>
              {result.members.map((member) => (
                <tr key={member.user.id}>
                  <td>
                    <strong>{member.user.email}</strong>
                  </td>
                  <td>
                    {member.assignments.map((assignment, i) => (
                      <Grant key={assignment.id ?? i} assignment={assignment} />
                    ))}
                  </td>
                  <td className="permission-list">
                    {member.permissions.join(", ")}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <p>
          {result.user
            ? "This user has no effective access matching these filters at this unit."
            : "No users have effective access matching these filters."}
        </p>
      )}
    </>
  );
}

function UserAccessResults({ access }: { access: UserAccess }) {
  return (
    <>
      <div className="section-heading">
        <h2>Accessible units</h2>
        <span className="muted">
          {access.units.length} {access.units.length === 1 ? "unit" : "units"}
        </span>
      </div>
      <p className="muted">{access.user.email} · Units visible to you</p>
      {access.units.length ? (
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>Unit</th>
                <th>Effective permissions</th>
              </tr>
            </thead>
            <tbody>
              {access.units.map((item) => (
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
    </>
  );
}
