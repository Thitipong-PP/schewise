// SPDX-License-Identifier: Apache-2.0
import type { components } from "./api-types";

export type Unit = components["schemas"]["UnitView"];
export type Permission = Unit["permissions"][number];
export type Assignment = components["schemas"]["AssignmentView"];
export type Role = components["schemas"]["RoleView"];
export type UserInfo = components["schemas"]["UserInfo"];
export type Context = components["schemas"]["AccessContext"];
export type Impact = components["schemas"]["Impact"];
export const hiddenSourceMessage =
  "Inherited from a higher-level unit you don't have access to";

export function sourceLabel(assignment: Assignment): string {
  if (assignment.source === "inherited_hidden") return hiddenSourceMessage;
  if (assignment.source === "direct") return "Assigned here";
  return assignment.source_unit
    ? `Inherited from ${assignment.source_unit.name}`
    : "Inherited assignment";
}

export function orderedUnits(units: Unit[]): { unit: Unit; level: number }[] {
  const ids = new Set(units.map((u) => u.id));
  const result: { unit: Unit; level: number }[] = [];
  const seen = new Set<string>();
  const visit = (unit: Unit, level: number) => {
    if (seen.has(unit.id)) return;
    seen.add(unit.id);
    result.push({ unit, level });
    units
      .filter((u) => u.parent_id === unit.id)
      .forEach((u) => visit(u, level + 1));
  };
  units
    .filter((u) => !u.parent_id || !ids.has(u.parent_id))
    .forEach((u) => visit(u, 0));
  return result;
}

export function moveDestinations(units: Unit[], moving: Unit): Unit[] {
  const blocked = new Set([moving.id]);
  let changed = true;
  while (changed) {
    changed = false;
    for (const u of units) {
      if (u.parent_id && blocked.has(u.parent_id) && !blocked.has(u.id)) {
        blocked.add(u.id);
        changed = true;
      }
    }
  }
  return units.filter(
    (u) =>
      !blocked.has(u.id) &&
      u.id !== moving.parent_id &&
      u.permissions.includes("unit:manage"),
  );
}
