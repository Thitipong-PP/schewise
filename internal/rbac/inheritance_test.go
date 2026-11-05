// SPDX-License-Identifier: Apache-2.0
package rbac_test

import (
	"context"
	"errors"
	"github.com/Thitipong-PP/schewise/internal/rbac"
	"slices"
	"testing"
)

func TestInheritanceBoundariesAndAdministratorBypass(t *testing.T) {
	r := memoryFixture()
	ctx := context.Background()
	payments := r.units[id("Payments")]
	payments.InheritAccess = false
	r.units[payments.ID] = payments
	r.units[id("Ledger")] = rbac.Unit{ID: id("Ledger"), ParentID: payments.ID, Name: "Ledger", InheritAccess: true, Ancestors: append(slices.Clone(payments.Ancestors), payments.ID)}
	r.connections[id("LedgerDB")] = id("Ledger")
	// A full-permission custom clone at root is still subject to the boundary.
	r.roles[id("Clone")] = rbac.Role{ID: id("Clone"), Name: "Clone", Permissions: rbac.Permissions()}
	r.assign("frank", "Clone", rbac.RootScope())
	e := rbac.NewEngine(r)
	for _, resource := range []rbac.Scope{unitScope("Payments"), unitScope("Ledger"), {Type: rbac.ScopeConnection, ID: id("LedgerDB")}} {
		for _, tc := range []struct {
			user string
			want bool
		}{{"alice", true}, {"bob", false}, {"carol", true}, {"eve", false}, {"frank", false}} {
			allowed, err := e.Can(ctx, id(tc.user), rbac.UnitRead, resource)
			if err != nil || allowed != tc.want {
				t.Fatalf("%s at %v: %v %v", tc.user, resource, allowed, err)
			}
		}
	}
	if allowed, err := e.Can(ctx, id("bob"), rbac.DocsEdit, unitScope("Mobile")); err != nil || !allowed {
		t.Fatal("boundary affected sibling")
	}
	explanation, err := e.Explain(ctx, id("bob"), rbac.DocsEdit, unitScope("Ledger"))
	if err != nil || explanation.Allowed || explanation.Reason != "blocked_by_inherit_access" || len(explanation.Blocked) != 1 || explanation.Blocked[0].BlockedBy != payments.ID {
		t.Fatalf("blocked provenance: %+v %v", explanation, err)
	}
	r.assign("bob", "Viewer", unitScope("Payments"))
	r.assign("eve", "Developer", unitScope("Ledger"))
	if allowed, err := e.Can(ctx, id("bob"), rbac.UnitRead, unitScope("Ledger")); err != nil || !allowed {
		t.Fatal("boundary removed direct grant")
	}
	if allowed, err := e.Can(ctx, id("eve"), rbac.DocsEdit, unitScope("Ledger")); err != nil || !allowed {
		t.Fatal("boundary removed lower grant")
	}
	if allowed, err := e.Can(ctx, id("eve"), rbac.DocsEdit, unitScope("Payments")); err != nil || allowed {
		t.Fatal("lower grant flowed upward")
	}
}

func TestCanRevokeDelegationSubtreeAndSelf(t *testing.T) {
	r := memoryFixture()
	e := rbac.NewEngine(r)
	ctx := context.Background()
	r.assign("frank", "Developer", unitScope("Payments"))
	r.assign("eve", "Viewer", unitScope("Payments"))
	assistant := rbac.Role{ID: id("Assistant"), Name: "Assistant", Permissions: []rbac.Permission{rbac.UnitRead, rbac.UnitManageMembers}}
	r.roles[assistant.ID] = assistant
	r.assign("dave", "Assistant", unitScope("Payments"))
	find := func(user string, scope rbac.Scope) rbac.Assignment {
		for _, a := range r.assignments {
			if a.UserID == id(user) && a.Scope == scope {
				return a
			}
		}
		t.Fatal("missing fixture assignment")
		return rbac.Assignment{}
	}
	for _, tc := range []struct {
		actor, user string
		scope       rbac.Scope
		want        bool
	}{
		{"carol", "frank", unitScope("Payments"), true}, {"carol", "eve", unitScope("Payments"), true},
		{"carol", "bob", unitScope("Engineering"), false}, {"dave", "carol", unitScope("Payments"), false},
		{"bob", "frank", unitScope("Payments"), false}, {"bob", "bob", unitScope("Engineering"), false},
		{"carol", "carol", unitScope("Payments"), true}, {"carol", "alice", rbac.RootScope(), false},
	} {
		got, err := e.CanRevoke(ctx, id(tc.actor), find(tc.user, tc.scope))
		if err != nil || got != tc.want {
			t.Fatalf("%+v = %v %v", tc, got, err)
		}
	}
	alice := find("alice", rbac.RootScope())
	if got, err := e.CanRevoke(ctx, id("alice"), alice); got || !errors.Is(err, rbac.ErrLastAdministrator) {
		t.Fatal("last admin not protected", got, err)
	}
	r.assign("frank", "Administrator", rbac.RootScope())
	if got, err := e.CanRevoke(ctx, id("alice"), alice); !got || err != nil {
		t.Fatal("self revoke with second admin denied", got, err)
	}
	// The bypass depends on a live root assignment, not a stale identity label.
	r.users[id("alice")] = false
	if got, err := e.Can(ctx, id("alice"), rbac.UnitRead, unitScope("Payments")); got || err != nil {
		t.Fatal("inactive administrator bypassed")
	}
}
