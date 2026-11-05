// SPDX-License-Identifier: Apache-2.0
"use client";
import { useCallback, useState } from "react";
import { ShieldCheck } from "lucide-react";
import { api, csrfHeaders, message, result } from "@/lib/admin-api";
import type { Unit } from "@/lib/admin-model";
import { Actions, Feedback, Modal, QueryState, useQuery } from "./shared";
import { ImpactReport } from "./impact-report";

export function InheritanceDialog({
  unit,
  units,
  close,
  saved,
}: {
  unit: Unit;
  units: Unit[];
  close: () => void;
  saved: () => void;
}) {
  const [value, setValue] = useState(!unit.inherit_access);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const query = useQuery(
    useCallback(
      async (signal) =>
        result(
          await api.POST("/api/v1/units/{id}/dry-run/inherit-access", {
            params: { path: { id: unit.id }, header: await csrfHeaders() },
            body: { inherit_access: value },
            signal,
          }),
        ),
      [unit.id, value],
    ),
  );
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (!query.data?.executable || query.loading) return;
    setBusy(true);
    setError("");
    try {
      result(
        await api.PATCH("/api/v1/units/{id}/inherit-access", {
          params: { path: { id: unit.id }, header: await csrfHeaders() },
          body: { inherit_access: value },
        }),
      );
      saved();
    } catch (error) {
      setError(message(error));
      query.reload();
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal title={`Parent access for ${unit.name}`} close={close} busy={busy}>
      <form onSubmit={submit}>
        <label className="inherit-switch inheritance-setting">
          <input
            type="checkbox"
            role="switch"
            checked={value}
            disabled={busy}
            onChange={(event) => setValue(event.target.checked)}
          />
          <span>Inherit access from parent units</span>
        </label>
        <p className="muted">
          When disabled, assignments above this unit stop here. Assignments at
          this unit still apply below it. Root Administrators always retain
          access.
        </p>
        <QueryState query={query} />
        {query.data && <ImpactReport impact={query.data} units={units} />}
        <Feedback error={error} />
        <Actions
          busy={busy}
          close={close}
          label="Confirm access change"
          disabled={
            query.loading ||
            !query.data?.executable ||
            value === unit.inherit_access
          }
          icon={<ShieldCheck size={16} />}
        />
      </form>
    </Modal>
  );
}
