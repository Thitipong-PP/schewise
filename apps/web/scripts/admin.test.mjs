// SPDX-License-Identifier: Apache-2.0
import assert from "node:assert/strict";
import test from "node:test";
import {
  sourceLabel,
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
