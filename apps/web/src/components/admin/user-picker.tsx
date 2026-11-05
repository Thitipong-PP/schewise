// SPDX-License-Identifier: Apache-2.0
"use client";

import { useCallback, useEffect, useId, useRef, useState } from "react";
import { Check, LoaderCircle, Search, X } from "lucide-react";
import { api, result } from "@/lib/admin-api";
import type { UserInfo } from "@/lib/admin-model";
import { useQuery } from "./shared";

// Keep typed text immediate; query only after a short pause. In-flight responses
// are aborted/ignored by useQuery, and stale suggestions cannot be selected.
export function UserPicker({
  label,
  unitID,
  value,
  onChange,
  disabled = false,
  selfOnly,
  allowGlobal = false,
  required = true,
}: {
  label: string;
  unitID: string;
  value: UserInfo | null;
  onChange: (user: UserInfo | null) => void;
  disabled?: boolean;
  selfOnly?: UserInfo;
  allowGlobal?: boolean;
  required?: boolean;
}) {
  const id = useId();
  const input = useRef<HTMLInputElement>(null);
  const [text, setText] = useState("");
  const [debounced, setDebounced] = useState("");
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(-1);
  useEffect(() => {
    input.current?.setCustomValidity(
      !value && text.trim()
        ? `Select a ${label.toLowerCase()} from the results or clear this field.`
        : "",
    );
  }, [value, text, label]);
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(text), 250);
    return () => clearTimeout(timer);
  }, [text]);
  const query = useQuery(
    useCallback(
      async (signal) => {
        if (selfOnly)
          return selfOnly.email.toLowerCase().includes(debounced.toLowerCase())
            ? [selfOnly]
            : [];
        if (!unitID && !allowGlobal) return [];
        return result(
          await api.GET("/api/v1/users", {
            params: {
              query: { unit_id: unitID || undefined, q: debounced.trim() },
            },
            signal,
          }),
        ).users;
      },
      [unitID, debounced, selfOnly, allowGlobal],
    ),
    true,
  );
  const users = query.data ?? [];
  const pending = text !== debounced || query.loading;
  const highlighted = Math.min(active, users.length - 1);
  const expanded = open && !disabled;
  useEffect(() => {
    if (expanded && highlighted >= 0)
      document
        .getElementById(`${id}-${highlighted}`)
        ?.scrollIntoView({ block: "nearest" });
  }, [expanded, highlighted, id]);
  function choose(user: UserInfo) {
    if (pending || query.error) return;
    onChange(user);
    setOpen(false);
    setActive(-1);
  }
  return (
    <div
      className="user-picker field"
      onBlur={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget)) setOpen(false);
      }}
    >
      <label htmlFor={id}>
        {label}
        {!required && <span className="muted"> (optional)</span>}
      </label>
      <div className="picker-control">
        <Search size={16} aria-hidden="true" />
        <input
          ref={input}
          id={id}
          role="combobox"
          aria-autocomplete="list"
          aria-expanded={expanded}
          aria-controls={expanded ? `${id}-results` : undefined}
          aria-activedescendant={
            expanded && !pending && highlighted >= 0
              ? `${id}-${highlighted}`
              : undefined
          }
          aria-describedby={`${id}-status`}
          autoComplete="off"
          spellCheck={false}
          placeholder="Search by email"
          maxLength={254}
          value={value?.email ?? text}
          disabled={disabled}
          required={required}
          onFocus={() => setOpen(true)}
          onChange={(event) => {
            setText(event.target.value);
            onChange(null);
            setOpen(true);
            setActive(-1);
          }}
          onKeyDown={(event) => {
            if (event.key === "Escape") {
              if (open) {
                event.preventDefault();
                event.stopPropagation();
                setOpen(false);
              }
              return;
            }
            if (event.key === "ArrowDown" || event.key === "ArrowUp") {
              event.preventDefault();
              setOpen(true);
              if (!pending && users.length)
                setActive(
                  event.key === "ArrowDown"
                    ? (highlighted + 1) % users.length
                    : highlighted <= 0
                      ? users.length - 1
                      : highlighted - 1,
                );
              return;
            }
            if (event.key === "Enter" && expanded) {
              event.preventDefault();
              const user = users[highlighted >= 0 ? highlighted : 0];
              if (user) choose(user);
            }
          }}
        />
        {pending ? (
          <LoaderCircle
            size={15}
            className="picker-end spin"
            aria-hidden="true"
          />
        ) : (
          (value || text) && (
            <button
              className="icon-button picker-clear"
              type="button"
              aria-label={`Clear ${label.toLowerCase()}`}
              title="Clear selection"
              disabled={disabled}
              onClick={() => {
                onChange(null);
                setText("");
                setActive(-1);
                input.current?.focus();
                setOpen(required);
              }}
            >
              <X size={14} />
            </button>
          )
        )}
      </div>
      <span className="picker-status muted" id={`${id}-status`} role="status">
        {query.error
          ? query.error
          : pending && open
            ? "Searching…"
            : value
              ? "Selected"
              : ""}
      </span>
      {expanded && (
        <div className="picker-popover">
          <ul
            id={`${id}-results`}
            role="listbox"
            aria-label={`${label} results`}
            aria-busy={pending}
          >
            {users.map((user, index) => (
              <li
                key={user.id}
                id={`${id}-${index}`}
                role="option"
                tabIndex={-1}
                aria-selected={value?.id === user.id}
                aria-disabled={pending || !!query.error}
                data-active={highlighted === index}
                onMouseDown={(event) => event.preventDefault()}
                onMouseMove={() => setActive(index)}
                onClick={() => choose(user)}
              >
                <span>{user.email}</span>
                {value?.id === user.id && (
                  <Check size={15} aria-hidden="true" />
                )}
              </li>
            ))}
          </ul>
          {!pending && !query.error && users.length === 0 && (
            <p className="picker-empty">No matching users.</p>
          )}
          {pending && users.length === 0 && (
            <p className="picker-empty">Searching…</p>
          )}
          {query.error && (
            <button type="button" className="secondary" onClick={query.reload}>
              Retry search
            </button>
          )}
        </div>
      )}
    </div>
  );
}
