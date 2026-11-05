// SPDX-License-Identifier: Apache-2.0
"use client";
import { useCallback, useState } from "react";
import { ArrowLeft, ArrowRight, Filter } from "lucide-react";
import { api, result } from "@/lib/admin-api";
import { useAccess } from "./shell";
import { QueryState, useQuery } from "./shared";

type Filters = {
  actor_id?: string;
  action?: string;
  from?: string;
  to?: string;
  before?: string;
  before_id?: string;
};

export function AuditLog() {
  const allowed = useAccess()?.root_permissions.includes("audit:read");
  const [filters, setFilters] = useState<Filters>({});
  const [history, setHistory] = useState<Filters[]>([]);
  const [actor, setActor] = useState("");
  const [action, setAction] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const query = useQuery(
    useCallback(
      async (signal) => {
        if (!allowed) return null;
        return result(
          await api.GET("/api/v1/audit", {
            params: { query: { ...filters, limit: 50 } },
            signal,
          }),
        );
      },
      [allowed, filters],
    ),
  );
  if (!allowed)
    return (
      <>
        <h1>Audit log</h1>
        <p className="muted">
          You do not have permission to read the audit log.
        </p>
      </>
    );
  return (
    <>
      <div className="page-heading">
        <div>
          <p className="section-label">Workspace administration</p>
          <h1>Audit log</h1>
        </div>
      </div>
      <form
        className="audit-filters"
        onSubmit={(event) => {
          event.preventDefault();
          setHistory([]);
          setFilters({
            actor_id: actor || undefined,
            action: action || undefined,
            from: from ? new Date(from).toISOString() : undefined,
            to: to ? new Date(to).toISOString() : undefined,
          });
        }}
      >
        <label className="field">
          Actor ID
          <input
            value={actor}
            placeholder="Any actor"
            pattern="[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}"
            title="A user UUID"
            onChange={(e) => setActor(e.target.value)}
          />
        </label>
        <label className="field">
          Action
          <input
            value={action}
            maxLength={120}
            placeholder="Any action"
            onChange={(e) => setAction(e.target.value)}
          />
        </label>
        <label className="field">
          From
          <input
            type="datetime-local"
            value={from}
            max={to || undefined}
            onChange={(e) => setFrom(e.target.value)}
          />
        </label>
        <label className="field">
          To
          <input
            type="datetime-local"
            value={to}
            min={from || undefined}
            onChange={(e) => setTo(e.target.value)}
          />
        </label>
        <button type="submit">
          <Filter size={15} />
          Filter
        </button>
      </form>
      <QueryState query={query} />
      {query.data && (
        <>
          <div className="table-scroll">
            <table className="audit-table">
              <thead>
                <tr>
                  <th>Time</th>
                  <th>Actor / action</th>
                  <th>Target</th>
                  <th>Details</th>
                </tr>
              </thead>
              <tbody>
                {query.data.entries.map((entry) => (
                  <tr key={entry.id}>
                    <td>
                      <time dateTime={entry.created_at}>
                        {new Date(entry.created_at).toLocaleString("en-GB")}
                      </time>
                    </td>
                    <td>
                      <strong>{entry.action}</strong>
                      <code className="row-description">
                        {entry.actor_user_id ===
                        "00000000-0000-0000-0000-000000000000"
                          ? "System / anonymous"
                          : entry.actor_user_id}
                      </code>
                    </td>
                    <td>
                      {entry.target_type || "—"}
                      <code className="row-description">
                        {entry.target_id ===
                        "00000000-0000-0000-0000-000000000000"
                          ? "—"
                          : entry.target_id}
                      </code>
                    </td>
                    <td>
                      <details>
                        <summary>View details</summary>
                        <pre>{JSON.stringify(entry.details, null, 2)}</pre>
                      </details>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {query.data.entries.length === 0 && (
            <p className="empty-state">No audit entries match these filters.</p>
          )}
          <div className="pagination">
            <button
              className="secondary"
              disabled={history.length === 0 || query.loading}
              onClick={() => {
                const previous = history.at(-1);
                if (previous) {
                  setFilters(previous);
                  setHistory(history.slice(0, -1));
                }
              }}
            >
              <ArrowLeft size={15} />
              Newer
            </button>
            <button
              className="secondary"
              disabled={!query.data.has_more || query.loading}
              onClick={() => {
                const last = query.data?.entries.at(-1);
                if (last) {
                  setHistory([...history, filters]);
                  setFilters({
                    ...filters,
                    before: last.created_at,
                    before_id: last.id,
                  });
                }
              }}
            >
              Older
              <ArrowRight size={15} />
            </button>
          </div>
        </>
      )}
    </>
  );
}
