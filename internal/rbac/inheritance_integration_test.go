// SPDX-License-Identifier: Apache-2.0
package rbac_test

import (
	"context"
	"errors"
	"github.com/Thitipong-PP/schewise/internal/database"
	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/google/uuid"
	"slices"
	"testing"
)

func TestInheritanceMutationPreviewVisibilityAndMove(t *testing.T) {
	f := newStoreFixture(t)
	ctx := context.Background()
	payments := f.units["Payments"]
	ledger, err := f.repo.CreateUnit(ctx, f.alice, payments, rbac.UnitInput{Name: "Ledger"})
	if err != nil {
		t.Fatal(err)
	}
	f.connections[id("LedgerDB")] = ledger.ID
	if !ledger.InheritAccess {
		t.Fatal("new units must inherit by default")
	}
	before := countRows(t, f.db, "audit_log")
	impact, err := f.repo.DryRunInheritAccess(ctx, id("carol"), payments, false)
	if err != nil || !impact.Executable || len(impact.Connections) != 1 {
		t.Fatalf("preview: %+v %v", impact, err)
	}
	bob := findChange(impact, id("bob"), rbac.UnitScope(ledger.ID))
	if !slices.Contains(bob.Lost, rbac.DocsEdit) || len(bob.BeforeRoles) != 1 || bob.BeforeRoles[0].Name != "Developer" || len(bob.AfterRoles) != 0 || bob.UserEmail != "bob@example.test" {
		t.Fatalf("wrong before/after access: %+v", bob)
	}
	if len(findChange(impact, f.alice, f.scope("Payments")).Lost) > 0 || len(findChange(impact, id("carol"), f.scope("Payments")).Lost) > 0 {
		t.Fatal("preview removed direct or bypass access")
	}
	if unit, err := f.repo.Unit(ctx, payments); err != nil || !unit.InheritAccess {
		t.Fatal("preview mutated unit")
	}
	if countRows(t, f.db, "audit_log") != before {
		t.Fatal("preview wrote mutation audit")
	}
	if err := f.repo.SetInheritAccess(ctx, id("bob"), payments, false); !errors.Is(err, rbac.ErrForbidden) {
		t.Fatal("Developer changed inheritance", err)
	}
	if err := f.repo.SetInheritAccess(ctx, f.alice, rbac.RootScope().ID, false); !errors.Is(err, rbac.ErrRootProtected) {
		t.Fatal("root changed", err)
	}
	if err := f.repo.SetInheritAccess(ctx, id("carol"), payments, false); err != nil {
		t.Fatal(err)
	}
	if countRows(t, f.db, "audit_log") != before+1 {
		t.Fatal("missing transactional inheritance audit")
	}
	for _, resource := range []rbac.Scope{f.scope("Payments"), rbac.UnitScope(ledger.ID), {Type: rbac.ScopeConnection, ID: id("LedgerDB")}} {
		for _, tc := range []struct {
			user uuid.UUID
			want bool
		}{{f.alice, true}, {id("bob"), false}, {id("carol"), true}, {id("eve"), false}} {
			allowed, err := f.repo.Engine().Can(ctx, tc.user, rbac.UnitRead, resource)
			if err != nil || allowed != tc.want {
				t.Fatalf("access %s %v: %v %v", tc.user, resource, allowed, err)
			}
		}
	}
	visible, err := f.repo.VisibleUnits(ctx, id("bob"))
	if err != nil {
		t.Fatal(err)
	}
	for _, unit := range visible {
		if unit.ID == payments || unit.ID == ledger.ID {
			t.Fatal("unreadable isolated subtree visible")
		}
	}
	members, err := f.repo.Members(ctx, id("carol"), payments, false, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, member := range members {
		if member.User.ID == id("bob") || member.User.ID == id("eve") {
			t.Fatal("blocked ancestor still listed as a member")
		}
	}
	if err := f.repo.MoveUnit(ctx, f.alice, payments, f.units["Data"]); err != nil {
		t.Fatal(err)
	}
	moved, err := f.repo.Unit(ctx, payments)
	if err != nil || moved.InheritAccess || moved.ParentID != f.units["Data"] {
		t.Fatal("move changed flag", moved, err)
	}
	gained, err := f.repo.DryRunInheritAccess(ctx, id("carol"), payments, true)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(findChange(gained, id("dave"), f.scope("Payments")).Gained, rbac.UnitRead) {
		t.Fatal("restoring inheritance missed new parent grants")
	}
}

func TestInheritanceSelfLockoutAndAuditRollback(t *testing.T) {
	f := newStoreFixture(t)
	ctx := context.Background()
	if _, err := f.repo.Assign(ctx, f.alice, id("frank"), uuid.MustParse(rbac.ManagerRoleID), f.scope("Engineering"), ""); err != nil {
		t.Fatal(err)
	}
	impact, err := f.repo.DryRunInheritAccess(ctx, id("frank"), f.units["Payments"], false)
	if err != nil || impact.Executable || !slices.Contains(impact.Blockers, "self_lockout") {
		t.Fatal("self-lockout preview not blocked", impact, err)
	}
	if err := f.repo.SetInheritAccess(ctx, id("frank"), f.units["Payments"], false); !errors.Is(err, rbac.ErrSelfLockout) {
		t.Fatal("self-lockout accepted", err)
	}
	if err := f.repo.SetInheritAccess(ctx, f.alice, f.units["Payments"], false); err != nil {
		t.Fatal("root administrator could not isolate unit", err)
	}
	if _, err := f.db.Exec(`ALTER TABLE audit_log RENAME TO unavailable_audit_log`); err != nil {
		t.Fatal(err)
	}
	if err := f.repo.SetInheritAccess(ctx, id("carol"), f.units["Payments"], true); err == nil {
		t.Fatal("unaudited change committed")
	}
	unit, err := f.repo.Unit(ctx, f.units["Payments"])
	if err != nil || unit.InheritAccess {
		t.Fatal("audit failure failed to roll back flag", err)
	}
}

func TestInheritanceMigrationIsAdditiveAndRootIsGuarded(t *testing.T) {
	db := isolatedDB(t)
	ctx := context.Background()
	provider, err := database.Provider(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 2); err != nil {
		t.Fatal(err)
	}
	child := uuid.New()
	if _, err := db.Exec(`INSERT INTO units(id,parent_id,name,ancestors) VALUES($1,$2,'Existing',ARRAY[$2::uuid])`, child, rbac.RootScope().ID); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal("migration not idempotent", err)
	}
	for _, id := range []uuid.UUID{rbac.RootScope().ID, child} {
		var flag bool
		if err := db.QueryRow(`SELECT inherit_access FROM units WHERE id=$1`, id).Scan(&flag); err != nil || !flag {
			t.Fatal("existing unit did not retain inherited access", err)
		}
	}
	if _, err := db.Exec(`UPDATE units SET inherit_access=false WHERE parent_id IS NULL`); err == nil {
		t.Fatal("database allowed root flag change")
	}
}
