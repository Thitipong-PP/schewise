// SPDX-License-Identifier: Apache-2.0
import type { components } from "./api-types";
type Schema = components["schemas"]["Schema"];
export function matchingTables(schema: Schema, search: string) {
  const term = search.trim().toLowerCase();
  return schema.tables.filter((table) =>
    `${schema.name} ${table.name} ${table.columns.map((c) => c.name).join(" ")}`
      .toLowerCase()
      .includes(term),
  );
}
export function hasReferencedTable(
  schemas: Schema[],
  reference: { schema: string; table: string },
) {
  return schemas.some(
    (schema) =>
      schema.name === reference.schema &&
      schema.tables.some((table) => table.name === reference.table),
  );
}
export function estimateLabel(rows: number | undefined) {
  return rows === undefined || rows < 0
    ? "Not analyzed"
    : Math.round(rows).toLocaleString();
}
