// SPDX-License-Identifier: Apache-2.0
"use client";
import Link from "next/link";
import { useCallback, useState } from "react";
import {
  Database,
  Plus,
  Pencil,
  KeyRound,
  PlugZap,
  RefreshCw,
  Trash2,
  Search,
  ArrowLeft,
} from "lucide-react";
import { api, csrfHeaders, message, result } from "@/lib/admin-api";
import type { components } from "@/lib/api-types";
import { orderedUnits } from "@/lib/admin-model";
import {
  Actions,
  Feedback,
  Modal,
  QueryState,
  Select,
  useQuery,
} from "./shared";

type Connection = components["schemas"]["View"];
type Input = components["schemas"]["Input"];
export function timeLabel(value: string | null | undefined) {
  return value ? new Date(value).toLocaleString() : "Never";
}
export function Connections({ unitID }: { unitID?: string }) {
  const query = useQuery(
    useCallback(
      async (signal) => {
        const [connections, units] = await Promise.all([
          api
            .GET("/api/v1/connections", {
              params: { query: { unit_id: unitID } },
              signal,
            })
            .then(result),
          api
            .GET("/api/v1/units", { signal })
            .then((r) => (r.response.ok ? (r.data?.units ?? []) : [])),
        ]);
        return { ...connections, units };
      },
      [unitID],
    ),
    true,
  );
  const [search, setSearch] = useState("");
  const [form, setForm] = useState<Connection | "new" | null>(null);
  const [password, setPassword] = useState<Connection | null>(null);
  const [deleting, setDeleting] = useState<Connection | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const [notice, setNotice] = useState<{ text: string; error: boolean } | null>(
    null,
  );
  const run = async (c: Connection, sync: boolean) => {
    setBusy(c.id);
    setNotice(null);
    try {
      const response = sync
        ? await api.POST("/api/v1/connections/{id}/sync", {
            params: { path: { id: c.id }, header: await csrfHeaders() },
            body: {},
          })
        : await api.POST("/api/v1/connections/{id}/test", {
            params: { path: { id: c.id }, header: await csrfHeaders() },
            body: {},
          });
      const value = result(response);
      setNotice({ text: `${c.name}: ${value.message}`, error: !value.success });
      query.reload();
    } catch (e) {
      setNotice({ text: message(e), error: true });
    } finally {
      setBusy(null);
    }
  };
  const units = query.data?.units ?? [];
  const manageable = units.filter(
    (u) =>
      u.permissions.includes("connection:manage") &&
      (!unitID || u.id === unitID),
  );
  const owner = units.find((u) => u.id === unitID);
  const close = () => {
    setForm(null);
    setPassword(null);
    setDeleting(null);
    query.reload();
  };
  const rows = (query.data?.connections ?? []).filter((c) =>
    `${c.name} ${c.unit?.name ?? ""}`
      .toLowerCase()
      .includes(search.toLowerCase()),
  );
  return (
    <>
      {unitID && (
        <Link href={`/units/${unitID}`} className="back-link">
          <ArrowLeft size={15} />
          {owner?.name ?? "Unit"}
        </Link>
      )}
      <div className="page-heading">
        <div>
          <p className="section-label">PostgreSQL</p>
          <h1>{owner ? `${owner.name} connections` : "Connections"}</h1>
        </div>
        {manageable.length > 0 && (
          <button onClick={() => setForm("new")}>
            <Plus size={16} />
            Add connection
          </button>
        )}
      </div>
      <label className="connection-search">
        <Search size={16} aria-hidden="true" />
        <input
          aria-label="Search connections"
          type="search"
          placeholder="Search connections"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </label>
      {notice &&
        (notice.error ? (
          <Feedback error={notice.text} />
        ) : (
          <p role="status" className="notice">
            {notice.text}
          </p>
        ))}
      <QueryState query={query} />
      {query.data &&
        (rows.length ? (
          <div className="table-scroll">
            <table>
              <thead>
                <tr>
                  <th>Connection</th>
                  <th>Status</th>
                  <th>Last synced</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((c) => (
                  <tr key={c.id}>
                    <td>
                      <div className="name-with-badge">
                        <Database size={16} />
                        {c.can_read_schema ? (
                          <Link
                            className="text-link"
                            href={`/connections/${c.id}`}
                          >
                            {c.name}
                          </Link>
                        ) : (
                          <strong>{c.name}</strong>
                        )}
                      </div>
                      <span className="row-description">
                        {c.unit?.name ?? "Restricted unit"}
                      </span>
                      {c.details && (
                        <span className="row-description">
                          {c.details.host}:{c.details.port} /{" "}
                          {c.details.database}
                        </span>
                      )}
                      {c.last_error && (
                        <span className="row-description">{c.last_error}</span>
                      )}
                    </td>
                    <td>
                      <span className="badge">
                        {c.status === "ok"
                          ? "Connected"
                          : c.status === "error"
                            ? "Error"
                            : "Not tested"}
                      </span>
                      {busy === c.id && (
                        <span role="status" className="row-description">
                          Working…
                        </span>
                      )}
                    </td>
                    <td>{timeLabel(c.last_synced_at)}</td>
                    <td>
                      <div className="row-actions">
                        {c.can_operate && (
                          <>
                            <button
                              className="secondary"
                              disabled={busy !== null}
                              onClick={() => run(c, false)}
                            >
                              <PlugZap size={15} />
                              Test
                            </button>
                            <button
                              className="secondary"
                              disabled={busy !== null}
                              onClick={() => run(c, true)}
                            >
                              <RefreshCw size={15} />
                              Sync
                            </button>
                          </>
                        )}
                        {c.can_manage && (
                          <>
                            <button
                              className="icon-button"
                              title="Edit connection"
                              aria-label={`Edit ${c.name}`}
                              disabled={busy !== null}
                              onClick={() => setForm(c)}
                            >
                              <Pencil size={16} />
                            </button>
                            <button
                              className="icon-button"
                              title="Change database password"
                              aria-label={`Change password for ${c.name}`}
                              disabled={busy !== null}
                              onClick={() => setPassword(c)}
                            >
                              <KeyRound size={16} />
                            </button>
                            <button
                              className="icon-button"
                              title="Delete connection"
                              aria-label={`Delete ${c.name}`}
                              disabled={busy !== null}
                              onClick={() => setDeleting(c)}
                            >
                              <Trash2 size={16} />
                            </button>
                          </>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <p className="empty-state">
            {search ? "No matching connections." : "No connections available."}
          </p>
        ))}
      {form && (
        <ConnectionForm
          connection={form === "new" ? undefined : form}
          units={manageable}
          unitID={unitID}
          close={close}
        />
      )}
      {password && <PasswordDialog connection={password} close={close} />}
      {deleting && <DeleteConnection connection={deleting} close={close} />}
    </>
  );
}
function ConnectionForm({
  connection,
  units,
  unitID,
  close,
}: {
  connection?: Connection;
  units: components["schemas"]["UnitView"][];
  unitID?: string;
  close: () => void;
}) {
  const [input, setInput] = useState<Input>({
    unit_id: connection?.unit?.id ?? unitID ?? units[0]?.id ?? "",
    name: connection?.name ?? "",
    engine: "postgres",
    host: connection?.details?.host ?? "",
    port: connection?.details?.port ?? 5432,
    database: connection?.details?.database ?? "",
    username: connection?.details?.username ?? "",
    ssl_mode: connection?.details?.ssl_mode ?? "require",
  });
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string>();
  const field = (key: keyof Input, value: string | number) =>
    setInput((i) => ({ ...i, [key]: value }));
  return (
    <Modal
      title={connection ? "Edit connection" : "Add connection"}
      close={close}
      busy={busy}
    >
      <form
        className="admin-form connection-form"
        onSubmit={async (e) => {
          e.preventDefault();
          setBusy(true);
          setError(undefined);
          try {
            if (connection)
              result(
                await api.PATCH("/api/v1/connections/{id}", {
                  params: {
                    path: { id: connection.id },
                    header: await csrfHeaders(),
                  },
                  body: input,
                }),
              );
            else
              result(
                await api.POST("/api/v1/connections", {
                  params: { header: await csrfHeaders() },
                  body: { ...input, password },
                }),
              );
            setPassword("");
            close();
          } catch (err) {
            setError(message(err));
          } finally {
            setBusy(false);
          }
        }}
      >
        <label className="field">
          Unit
          <Select
            value={input.unit_id}
            onChange={(e) => field("unit_id", e.target.value)}
            disabled={!!connection || busy}
            required
          >
            {orderedUnits(units).map(({ unit, level }) => (
              <option key={unit.id} value={unit.id}>
                {"— ".repeat(level)}
                {unit.name}
              </option>
            ))}
          </Select>
        </label>
        <label className="field">
          Name
          <input
            required
            maxLength={120}
            value={input.name}
            onChange={(e) => field("name", e.target.value)}
            disabled={busy}
          />
        </label>
        <div className="connection-fields">
          <label className="field">
            Host
            <input
              required
              autoCapitalize="none"
              autoCorrect="off"
              spellCheck={false}
              value={input.host}
              onChange={(e) => field("host", e.target.value)}
              disabled={busy}
            />
          </label>
          <label className="field">
            Port
            <input
              type="number"
              required
              min={1}
              max={65535}
              value={input.port || ""}
              onChange={(e) => field("port", Number(e.target.value))}
              disabled={busy}
            />
          </label>
        </div>
        <label className="field">
          Database
          <input
            required
            value={input.database}
            onChange={(e) => field("database", e.target.value)}
            disabled={busy}
          />
        </label>
        <label className="field">
          Username
          <input
            required
            autoComplete="off"
            value={input.username}
            onChange={(e) => field("username", e.target.value)}
            disabled={busy}
          />
        </label>
        {!connection && (
          <label className="field">
            Database password
            <input
              type="password"
              autoComplete="new-password"
              maxLength={4096}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={busy}
            />
          </label>
        )}
        <label className="field">
          SSL mode
          <Select
            value={input.ssl_mode}
            onChange={(e) => field("ssl_mode", e.target.value)}
            disabled={busy}
          >
            {[
              "verify-full",
              "verify-ca",
              "require",
              "prefer",
              "allow",
              "disable",
            ].map((mode) => (
              <option key={mode}>{mode}</option>
            ))}
          </Select>
        </label>
        <Feedback error={error} />
        <Actions
          busy={busy}
          close={close}
          label={connection ? "Save changes" : "Add connection"}
          icon={<Database size={16} />}
        />
      </form>
    </Modal>
  );
}
function PasswordDialog({
  connection,
  close,
}: {
  connection: Connection;
  close: () => void;
}) {
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string>();
  return (
    <Modal title="Change database password" close={close} busy={busy}>
      <form
        className="admin-form"
        onSubmit={async (e) => {
          e.preventDefault();
          setBusy(true);
          setError(undefined);
          try {
            result(
              await api.POST("/api/v1/connections/{id}/password", {
                params: {
                  path: { id: connection.id },
                  header: await csrfHeaders(),
                },
                body: { password },
              }),
            );
            setPassword("");
            close();
          } catch (err) {
            setError(message(err));
          } finally {
            setBusy(false);
          }
        }}
      >
        <p>{connection.name}</p>
        <label className="field">
          New database password
          <input
            type="password"
            required
            autoComplete="new-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            maxLength={4096}
            disabled={busy}
          />
        </label>
        <Feedback error={error} />
        <Actions
          busy={busy}
          close={close}
          label="Change password"
          icon={<KeyRound size={16} />}
        />
      </form>
    </Modal>
  );
}
function DeleteConnection({
  connection,
  close,
}: {
  connection: Connection;
  close: () => void;
}) {
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string>();
  const query = useQuery(
    useCallback(
      async (signal) =>
        result(
          await api.POST("/api/v1/connections/{id}/dry-run/delete", {
            params: {
              path: { id: connection.id },
              header: await csrfHeaders(),
            },
            body: {},

            signal,
          }),
        ),
      [connection.id],
    ),
  );
  return (
    <Modal title={`Delete ${connection.name}`} close={close} busy={busy}>
      <QueryState query={query} />
      {query.data && (
        <form
          className="admin-form"
          onSubmit={async (e) => {
            e.preventDefault();
            setBusy(true);
            setError(undefined);
            try {
              result(
                await api.DELETE("/api/v1/connections/{id}", {
                  params: {
                    path: { id: connection.id },
                    header: await csrfHeaders(),
                  },
                  body: { name },
                }),
              );
              close();
            } catch (err) {
              setError(message(err));
            } finally {
              setBusy(false);
            }
          }}
        >
          <p>
            This permanently removes {query.data.snapshot_count} schema
            {query.data.snapshot_count === 1
              ? " snapshot"
              : " snapshots"} and {query.data.assignment_count} connection{" "}
            {query.data.assignment_count === 1 ? "assignment" : "assignments"}.
            The target database will remain unchanged.
          </p>
          {query.data.users.length > 0 && (
            <div>
              <h3>Users losing connection assignments</h3>
              <ul>
                {query.data.users.map((u) => (
                  <li key={u.id}>{u.email}</li>
                ))}
              </ul>
            </div>
          )}
          <label className="field">
            Type {connection.name} to confirm
            <input
              required
              autoComplete="off"
              value={name}
              onChange={(e) => setName(e.target.value)}
              disabled={busy}
            />
          </label>
          <Feedback error={error} />
          <Actions
            busy={busy}
            close={close}
            label="Delete connection"
            disabled={name !== connection.name}
            icon={<Trash2 size={16} />}
          />
        </form>
      )}
    </Modal>
  );
}
