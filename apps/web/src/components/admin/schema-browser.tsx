// SPDX-License-Identifier: Apache-2.0
"use client";
import Link from "next/link";
import { useCallback, useState } from "react";
import {
  ArrowLeft,
  Table2,
  Search,
  RefreshCw,
  KeyRound,
  Link2,
} from "lucide-react";
import { api, result, csrfHeaders, message } from "@/lib/admin-api";
import type { components } from "@/lib/api-types";
import { Feedback, QueryState, Select, useQuery } from "./shared";
import {
  matchingTables,
  hasReferencedTable,
  estimateLabel,
} from "@/lib/schema-model";
import { timeLabel } from "./connections";

type Snapshot = components["schemas"]["Snapshot"];
type Table = components["schemas"]["Table"];
export function SchemaBrowser({ id }: { id: string }) {
  const [snapshotID, setSnapshotID] = useState("");
  const [older, setOlder] = useState<components["schemas"]["SnapshotInfo"][]>(
    [],
  );
  const [olderCursor, setOlderCursor] = useState<string | null | undefined>();
  const [loadingOlder, setLoadingOlder] = useState(false);
  const [search, setSearch] = useState("");
  const [selected, setSelected] = useState<{
    schema: string;
    table: string;
  } | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string>();
  const query = useQuery(
    useCallback(
      async (signal) => {
        const [profile, list] = await Promise.all([
          api.GET("/api/v1/connections/{id}", {
            params: { path: { id } },
            signal,
          }),
          api
            .GET("/api/v1/connections/{id}/snapshots", {
              params: { path: { id }, query: { limit: 200 } },
              signal,
            })
            .then(result),
        ]);
        return {
          connection: profile.response.ok
            ? profile.data?.connection
            : undefined,
          ...list,
        };
      },
      [id],
    ),
    true,
  );
  const schema = useQuery(
    useCallback(
      async (signal) => {
        const response = snapshotID
          ? await api.GET("/api/v1/connections/{id}/snapshots/{sid}", {
              params: { path: { id, sid: snapshotID } },
              signal,
            })
          : await api.GET("/api/v1/connections/{id}/schema", {
              params: { path: { id } },
              signal,
            });
        if (response.response.status === 404) return null;
        return result(response);
      },
      [id, snapshotID],
    ),
    true,
  );
  const content = schema.data?.content;
  const selectedSchema =
    content?.schemas.find(
      (s) =>
        s.name === selected?.schema &&
        s.tables.some((t) => t.name === selected.table),
    ) ?? content?.schemas.find((s) => s.tables.length > 0);
  const table =
    selectedSchema?.tables.find((t) => t.name === selected?.table) ??
    selectedSchema?.tables[0];
  const sync = async () => {
    setBusy(true);
    setError(undefined);
    try {
      const value = result(
        await api.POST("/api/v1/connections/{id}/sync", {
          params: { path: { id }, header: await csrfHeaders() },
          body: {},
        }),
      );
      if (!value.success) throw new Error(value.message);
      setSnapshotID("");
      setOlder([]);
      setOlderCursor(undefined);
      query.reload();
      schema.reload();
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  };
  return (
    <>
      <Link
        className="back-link"
        href={
          query.data?.connection?.unit
            ? `/units/${query.data.connection.unit.id}/connections`
            : "/connections"
        }
      >
        <ArrowLeft size={15} />
        Connections
      </Link>
      <div className="page-heading">
        <div>
          <p className="section-label">Schema browser</p>
          <h1>{query.data?.connection?.name ?? "Database schema"}</h1>
          <p className="muted">
            Last synced:{" "}
            {timeLabel(
              query.data?.connection?.last_synced_at ??
                schema.data?.stats.observed_at,
            )}
          </p>
        </div>
        {query.data?.connection?.can_operate && (
          <button disabled={busy} onClick={sync}>
            <RefreshCw size={16} />
            {busy ? "Syncing…" : "Sync schema"}
          </button>
        )}
      </div>
      <Feedback error={error} />
      <QueryState query={query} />
      <QueryState query={schema} />
      {query.data && (
        <label className="snapshot-picker">
          Snapshot
          <Select
            aria-label="Snapshot"
            value={snapshotID}
            onChange={(e) => setSnapshotID(e.target.value)}
          >
            <option value="">Latest snapshot</option>
            {[...query.data.snapshots, ...older].map((s) => (
              <option key={s.id} value={s.id}>
                {timeLabel(s.taken_at)} · {s.content_hash.slice(0, 8)}
              </option>
            ))}
          </Select>
        </label>
      )}
      {(olderCursor === undefined ? query.data?.next_before : olderCursor) && (
        <button
          className="secondary"
          disabled={loadingOlder}
          onClick={async () => {
            const before =
              olderCursor === undefined ? query.data?.next_before : olderCursor;
            if (!before) return;
            setLoadingOlder(true);
            setError(undefined);
            try {
              const page = result(
                await api.GET("/api/v1/connections/{id}/snapshots", {
                  params: { path: { id }, query: { before, limit: 200 } },
                }),
              );
              setOlder((current) => [...current, ...page.snapshots]);
              setOlderCursor(page.next_before ?? null);
            } catch (e) {
              setError(message(e));
            } finally {
              setLoadingOlder(false);
            }
          }}
        >
          Load older snapshots
        </button>
      )}
      {schema.data === null && !schema.loading && (
        <p className="empty-state">No schema snapshot yet.</p>
      )}
      {content && (
        <div className="schema-workbench" aria-busy={schema.loading}>
          <aside className="schema-navigation" aria-label="Database objects">
            <label className="connection-search">
              <Search size={16} aria-hidden="true" />
              <input
                aria-label="Search schemas, tables, or columns"
                type="search"
                placeholder="Search schema"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </label>
            {content.schemas.map((s) => {
              const tables = matchingTables(s, search);
              return tables.length ? (
                <div className="schema-group" key={s.name}>
                  <h2>{s.name}</h2>
                  {tables.map((t) => (
                    <button
                      className="schema-object"
                      key={t.name}
                      title={`${s.name}.${t.name}`}
                      aria-pressed={
                        selectedSchema?.name === s.name &&
                        table?.name === t.name
                      }
                      onClick={() =>
                        setSelected({ schema: s.name, table: t.name })
                      }
                    >
                      <Table2 size={15} aria-hidden="true" />
                      <span>
                        {t.name}
                        <small>{t.kind.replaceAll("_", " ")}</small>
                      </span>
                    </button>
                  ))}
                </div>
              ) : null;
            })}
            {!content.schemas.some(
              (s) => matchingTables(s, search).length > 0,
            ) && <p className="muted">No matching objects.</p>}
          </aside>
          <section
            className="schema-detail"
            aria-label="Selected database object"
          >
            {table && selectedSchema && schema.data ? (
              <TableDetail
                table={table}
                schemaName={selectedSchema.name}
                snapshot={schema.data}
                select={(value) => {
                  setSearch("");
                  setSelected(value);
                }}
              />
            ) : (
              <p className="empty-state">
                No tables or views in this snapshot.
              </p>
            )}
          </section>
        </div>
      )}
    </>
  );
}
function TableDetail({
  table,
  schemaName,
  snapshot,
  select,
}: {
  table: Table;
  schemaName: string;
  snapshot: Snapshot;
  select: (v: { schema: string; table: string }) => void;
}) {
  const estimate = snapshot.stats.estimates.find(
    (e) => e.schema === schemaName && e.table === table.name,
  )?.rows;
  return (
    <>
      <div className="schema-table-heading">
        <span className="section-label">{schemaName}</span>
        <h2>{table.name}</h2>
        <span className="muted">
          {table.kind.replaceAll("_", " ")} · Estimated rows:{" "}
          {estimateLabel(estimate)}
        </span>
        {table.comment && <p className="schema-comment">{table.comment}</p>}
      </div>
      <h3>
        Columns <span className="muted">({table.columns.length})</span>
      </h3>
      <div className="table-scroll">
        <table className="schema-columns">
          <thead>
            <tr>
              <th>Column</th>
              <th>Type</th>
              <th>Nullable</th>
              <th>Default / generated</th>
              <th>Comment</th>
            </tr>
          </thead>
          <tbody>
            {table.columns.map((c) => {
              const primary = table.constraints.some(
                (k) => k.kind === "primary_key" && k.columns.includes(c.name),
              );
              const foreign = table.constraints.some(
                (k) => k.kind === "foreign_key" && k.columns.includes(c.name),
              );
              return (
                <tr key={c.name}>
                  <td>
                    <div className="name-with-badge">
                      <code>{c.name}</code>
                      {primary && (
                        <span className="badge" title="Primary key">
                          <KeyRound size={12} />
                          PK
                        </span>
                      )}
                      {foreign && (
                        <span className="badge" title="Declared foreign key">
                          <Link2 size={12} />
                          FK
                        </span>
                      )}
                    </div>
                  </td>
                  <td>
                    <code>{c.data_type}</code>
                  </td>
                  <td>{c.nullable ? "Yes" : "No"}</td>
                  <td>
                    <code>
                      {c.identity
                        ? `Identity (${c.identity.replaceAll("_", " ")})`
                        : c.generated
                          ? `${c.generated_expression} (${c.generated})`
                          : (c.default_expression ?? "—")}
                    </code>
                  </td>
                  <td>{c.comment ?? "—"}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      <h3>
        Constraints <span className="muted">({table.constraints.length})</span>
      </h3>
      {table.constraints.length ? (
        <ul className="schema-facts">
          {table.constraints.map((c) => (
            <li key={c.name}>
              <strong>{c.name}</strong>
              <span className="badge">{c.kind.replaceAll("_", " ")}</span>
              <code>{c.definition}</code>
              {c.reference && (
                <div>
                  {hasReferencedTable(snapshot.content.schemas, c.reference) ? (
                    <button
                      className="text-link"
                      onClick={() =>
                        select({
                          schema: c.reference!.schema,
                          table: c.reference!.table,
                        })
                      }
                    >
                      {c.reference.schema}.{c.reference.table}
                    </button>
                  ) : (
                    <span>
                      {c.reference.schema}.{c.reference.table}
                    </span>
                  )}
                  <span className="row-description">
                    Update: {c.on_update.replaceAll("_", " ")} · Delete:{" "}
                    {c.on_delete.replaceAll("_", " ")}
                  </span>
                </div>
              )}
              {c.comment && <p className="schema-comment">{c.comment}</p>}
            </li>
          ))}
        </ul>
      ) : (
        <p className="muted">None declared.</p>
      )}
      <h3>
        Indexes <span className="muted">({table.indexes.length})</span>
      </h3>
      {table.indexes.length ? (
        <ul className="schema-facts">
          {table.indexes.map((i) => (
            <li key={i.name}>
              <strong>{i.name}</strong>
              <span className="badge">
                {i.method}
                {i.unique ? " · unique" : ""}
                {!i.valid ? " · invalid" : ""}
              </span>
              <code>{i.definition}</code>
              {i.comment && <p className="schema-comment">{i.comment}</p>}
            </li>
          ))}
        </ul>
      ) : (
        <p className="muted">None.</p>
      )}
    </>
  );
}
