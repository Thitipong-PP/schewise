// SPDX-License-Identifier: Apache-2.0
import assert from "node:assert/strict";
import test from "node:test";
import {
  sourceLabel,
  accessSearchMode,
  filterAccessMembers,
  blockingLabel,
  hiddenSourceMessage,
  orderedUnits,
  moveDestinations,
} from "../src/lib/admin-model.ts";
import { csrfHeaders, result } from "../src/lib/admin-api.ts";

const makeUnit = (
  id,
  parent_id = null,
  permissions = ["unit:read", "unit:manage"],
) => ({
  id,
  parent_id,
  permissions,
  name: id,
  description: "",
  kind: "",
  depth: 1,
  is_root: !parent_id,
  can_move: !!parent_id,
  inherit_access: true,
});

test("members and Explain always label hidden origins without reading supplied source metadata", () => {
  assert.equal(
    sourceLabel({
      source: "inherited_hidden",
      source_unit: { id: "secret-id", name: "Engineering" },
    }),
    hiddenSourceMessage,
  );
  assert.equal(sourceLabel({ source: "direct" }), "Assigned here");
  assert.equal(
    sourceLabel({
      source: "inherited",
      source_unit: { id: "visible-id", name: "Payments" },
    }),
    "Inherited from Payments",
  );
});

test("organization forest keeps readable orphan units and relative indentation", () => {
  const units = [
    makeUnit("Payments", "unreadable"),
    makeUnit("API", "Payments"),
    makeUnit("Mobile", "unreadable"),
  ];
  assert.deepEqual(
    orderedUnits(units).map(({ unit, level }) => [unit.id, level]),
    [
      ["Payments", 0],
      ["API", 1],
      ["Mobile", 0],
    ],
  );
});

test("move picker excludes current parent, self, descendants, and unmanaged destinations", () => {
  const root = makeUnit("Organization");
  const moving = makeUnit("Engineering", root.id);
  const units = [
    root,
    moving,
    makeUnit("Payments", moving.id),
    makeUnit("API", "Payments"),
    makeUnit("Data", root.id),
    makeUnit("Read only", root.id, ["unit:read"]),
  ];
  assert.deepEqual(
    moveDestinations(units, moving).map((unit) => unit.id),
    ["Data"],
  );
});

test("mutations obtain a current CSRF token and fail closed when initialization fails", async () => {
  const calls = [];
  const client = {
    GET: async (path, options) => {
      calls.push({ path, options });
      return {
        data: { csrf_token: "synthetic-current-token" },
        response: new Response(null, { status: 200 }),
      };
    },
  };
  assert.deepEqual(await csrfHeaders(client), {
    "X-CSRF-Token": "synthetic-current-token",
  });
  assert.deepEqual(calls, [
    { path: "/api/v1/auth/csrf", options: { cache: "no-store" } },
  ]);
  await assert.rejects(
    csrfHeaders({
      GET: async () => ({
        error: { error: "Session unavailable" },
        response: new Response(null, { status: 503 }),
      }),
    }),
    /Session unavailable/,
  );
});

test("API failures expose the server reason instead of treating a rejection as success", () => {
  assert.throws(
    () =>
      result({
        response: new Response(null, { status: 409 }),
        error: {
          code: "last_administrator",
          error: "The last active root Administrator cannot be revoked.",
        },
      }),
    /last active root Administrator/,
  );
  assert.throws(
    () => result({ response: new Response(null, { status: 503 }) }),
    /could not be completed/,
  );
});

test("hidden blocking units never enter readable Explain labels", () => {
  const hidden = {
    source: "blocked_hidden",
    unit: { id: "private-id", name: "Payments" },
    reason_text: "Blocked by Payments",
  };
  assert.equal(
    blockingLabel(hidden),
    "Access from above is blocked by an inheritance boundary you do not have access to.",
  );
  assert.equal(
    blockingLabel({ ...hidden, source: "blocked" }),
    "Blocked by Payments",
  );
});

test("access search accepts resource-only, user-only, optional permission, and a complete Explain query", () => {
  const mode = (unitID, userID, permission = "") =>
    accessSearchMode({ unitID, userID, permission });
  assert.equal(mode("", ""), null);
  assert.equal(mode("payments", ""), "members");
  assert.equal(mode("", "bob"), "user");
  assert.equal(mode("payments", "bob"), "members");
  assert.equal(mode("payments", "", "docs:edit"), "members");
  assert.equal(mode("payments", "bob", "docs:edit"), "explain");
  assert.equal(mode("", "bob", "docs:edit"), null);
});

test("resource search filters only server-evaluated effective access", () => {
  const rows = [
    { user: { id: "bob" }, permissions: ["unit:read", "docs:edit"] },
    { user: { id: "eve" }, permissions: ["unit:read"] },
  ];
  const filters = { unitID: "payments", userID: "", permission: "" };
  assert.deepEqual(filterAccessMembers(rows, filters), rows);
  assert.deepEqual(
    filterAccessMembers(rows, { ...filters, permission: "docs:edit" }),
    [rows[0]],
  );
  assert.deepEqual(filterAccessMembers(rows, { ...filters, userID: "eve" }), [
    rows[1],
  ]);
  assert.deepEqual(
    filterAccessMembers(rows, {
      ...filters,
      userID: "eve",
      permission: "docs:edit",
    }),
    [],
  );
});

test("schema search matches schema, table and column names without regex interpretation", async () => {
  const { matchingTables } = await import("../src/lib/schema-model.ts");
  const schema = {
    name: "Commerce",
    tables: [
      { name: "Orders", columns: [{ name: "customer_id" }] },
      { name: "Customers", columns: [{ name: "email.address" }] },
    ],
  };
  assert.deepEqual(
    matchingTables(schema, "CUSTOMER_ID").map((t) => t.name),
    ["Orders"],
  );
  assert.equal(matchingTables(schema, "commerce").length, 2);
  assert.deepEqual(
    matchingTables(schema, "email.address").map((t) => t.name),
    ["Customers"],
  );
  assert.equal(matchingTables(schema, "[invalid").length, 0);
  assert.equal(matchingTables(schema, "   ").length, 2);
});
test("declared FK links require the exact schema/table, and unknown estimates remain distinct from zero", async () => {
  const { hasReferencedTable, estimateLabel } =
    await import("../src/lib/schema-model.ts");
  const schemas = [
    { name: "sales", tables: [{ name: "orders" }] },
    { name: "other", tables: [{ name: "customers" }] },
  ];
  assert.equal(
    hasReferencedTable(schemas, { schema: "sales", table: "customers" }),
    false,
  );
  assert.equal(
    hasReferencedTable(schemas, { schema: "other", table: "customers" }),
    true,
  );
  assert.equal(estimateLabel(-1), "Not analyzed");
  assert.equal(estimateLabel(0), "0");
  assert.equal(estimateLabel(undefined), "Not analyzed");
});
