// SPDX-License-Identifier: Apache-2.0
"use client";

import { useCallback, useEffect, useId, useRef, useState } from "react";
import {
  AlertCircle,
  ChevronDown,
  LoaderCircle,
  RefreshCw,
  X,
} from "lucide-react";
import { message } from "@/lib/admin-api";

export function useQuery<T>(
  load: (signal: AbortSignal) => Promise<T>,
  keepPrevious = false,
) {
  const [state, setState] = useState<{
    data?: T;
    error?: string;
    loader?: typeof load;
    revision?: number;
  }>({});
  const [revision, setRevision] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    load(controller.signal)
      .then((data) => {
        if (!controller.signal.aborted)
          setState({ data, loader: load, revision });
      })
      .catch((error) => {
        if (!controller.signal.aborted)
          setState({ error: message(error), loader: load, revision });
      });
    return () => controller.abort();
  }, [load, revision]);
  const reload = useCallback(() => setRevision((value) => value + 1), []);
  const current = state.loader === load && state.revision === revision;
  return {
    data: current || keepPrevious ? state.data : undefined,
    error: current ? state.error : undefined,
    loading: !current,
    reload,
  };
}

export function Select(props: React.ComponentProps<"select">) {
  return (
    <span className="select-control">
      <select {...props} />
      <ChevronDown size={16} aria-hidden="true" />
    </span>
  );
}

export function Feedback({ error }: { error?: string }) {
  return error ? (
    <p className="notice error" role="alert">
      <AlertCircle size={16} aria-hidden="true" />
      {error}
    </p>
  ) : null;
}

export function Loading() {
  return (
    <p className="muted loading" role="status">
      <LoaderCircle size={16} className="spin" aria-hidden="true" />
      Loading…
    </p>
  );
}

export function QueryState({
  query,
}: {
  query: { loading: boolean; error?: string; reload: () => void };
}) {
  if (query.loading) return <Loading />;
  if (query.error)
    return (
      <>
        <Feedback error={query.error} />
        <button className="secondary" onClick={query.reload}>
          <RefreshCw size={15} />
          Retry
        </button>
      </>
    );
  return null;
}

export function Modal({
  title,
  children,
  close,
  busy = false,
}: {
  title: string;
  children: React.ReactNode;
  close: () => void;
  busy?: boolean;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  const id = useId();
  useEffect(() => {
    const dialog = ref.current;
    const previous = document.activeElement;
    dialog?.showModal();
    return () => {
      dialog?.close();
      if (previous instanceof HTMLElement && previous.isConnected)
        previous.focus();
    };
  }, []);
  return (
    <dialog
      ref={ref}
      className="admin-dialog"
      aria-labelledby={id}
      onCancel={(event) => {
        event.preventDefault();
        if (!busy) close();
      }}
    >
      <div className="dialog-heading">
        <h2 id={id}>{title}</h2>
        <button
          type="button"
          className="icon-button"
          aria-label="Close dialog"
          title="Close"
          disabled={busy}
          onClick={close}
        >
          <X size={18} />
        </button>
      </div>
      {children}
    </dialog>
  );
}

export function Actions({
  busy,
  close,
  label = "Save",
  disabled = false,
  icon,
}: {
  busy: boolean;
  close: () => void;
  label?: string;
  disabled?: boolean;
  icon?: React.ReactNode;
}) {
  return (
    <div className="dialog-actions">
      <button
        type="button"
        className="secondary"
        disabled={busy}
        onClick={close}
      >
        Cancel
      </button>
      <button type="submit" disabled={busy || disabled}>
        {busy ? <LoaderCircle size={16} className="spin" /> : icon}
        {busy ? "Working…" : label}
      </button>
    </div>
  );
}
