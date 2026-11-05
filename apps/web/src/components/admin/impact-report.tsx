// SPDX-License-Identifier: Apache-2.0
import type { Impact, Unit } from "@/lib/admin-model";

export function ImpactReport({
  impact,
  units,
}: {
  impact: Impact;
  units: Unit[];
}) {
  return (
    <section className="impact-report" aria-label="Impact preview">
      <h3>Impact preview</h3>
      {impact.blockers.map((blocker) => (
        <p className="notice" key={blocker}>
          {{
            child_units: "This unit contains child units.",
            direct_assignments:
              "Revoke direct assignments before deleting this unit.",
            connections: "This unit contains connections.",
            self_lockout:
              "This change would remove your permission to manage assignments at this unit. Add a direct management assignment before disabling inheritance.",
          }[blocker] ?? blocker}
        </p>
      ))}
      <p className="muted">
        {impact.changes.length} user/resource access changes ·{" "}
        {impact.connections.length} connections
      </p>
      {impact.changes.length > 0 ? (
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>User / resource</th>
                <th>Gained</th>
                <th>Lost</th>
              </tr>
            </thead>
            <tbody>
              {impact.changes.map((change) => (
                <tr
                  key={`${change.user_id}:${change.resource.type}:${change.resource.id}`}
                >
                  <td>
                    <strong title={change.user_id}>
                      {change.user_email || change.user_id}
                    </strong>
                    <span className="row-description">
                      {units.find((u) => u.id === change.resource.id)?.name ??
                        change.resource.id}
                    </span>
                    <span className="row-description">
                      Before:{" "}
                      {change.before_roles.map((r) => r.name).join(", ") ||
                        "No access"}
                      <br />
                      After:{" "}
                      {change.after_roles.map((r) => r.name).join(", ") ||
                        "No access"}
                    </span>
                  </td>
                  <td>{change.gained.join(", ") || "—"}</td>
                  <td>{change.lost.join(", ") || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <p>No effective permissions change.</p>
      )}
    </section>
  );
}
