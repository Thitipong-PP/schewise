// SPDX-License-Identifier: Apache-2.0
package httpapi_test

import (
	"context"
	api "github.com/Thitipong-PP/schewise/internal/apicontract"
	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/google/uuid"
	"strings"
	"testing"
)

func TestInheritanceHTTPAndHiddenBlockingUnit(t *testing.T) {
	f := newAdminFixture(t)
	ctx := context.Background()
	admin, adminCSRF, _ := f.login(t, "admin@example.test")
	manager, managerCSRF, _ := f.login(t, "carol@example.test")
	developer, developerCSRF, _ := f.login(t, "bob@example.test")
	path := "/api/v1/units/" + f.units["Payments"].String()
	off := map[string]any{"inherit_access": false}
	expectStatus(t, f.request(t, "PATCH", path+"/inherit-access", off, developer, developerCSRF, testOrigin), 403)
	expectStatus(t, f.request(t, "PATCH", path+"/inherit-access", off, manager, "", testOrigin), 403)
	expectStatus(t, f.request(t, "PATCH", path+"/inherit-access", map[string]any{}, manager, managerCSRF, testOrigin), 400)
	expectStatus(t, f.request(t, "PATCH", "/api/v1/units/"+rbac.RootUnitID+"/inherit-access", off, admin, adminCSRF, testOrigin), 409)
	preview := f.request(t, "POST", path+"/dry-run/inherit-access", off, manager, managerCSRF, testOrigin)
	expectStatus(t, preview, 200)
	if !decode[rbac.Impact](t, preview).Executable {
		t.Fatal("direct manager could not preview")
	}
	expectStatus(t, f.request(t, "PATCH", path+"/inherit-access", off, manager, managerCSRF, testOrigin), 204)
	expectStatus(t, f.request(t, "GET", path, nil, developer, "", ""), 403)
	for _, endpoint := range []string{"/api/v1/units", "/api/v1/units/tree"} {
		tree := f.request(t, "GET", endpoint, nil, developer, "", "")
		expectStatus(t, tree, 200)
		if strings.Contains(string(tree.body), f.units["Payments"].String()) {
			t.Fatal("isolated unit leaked in tree")
		}
	}
	ledger, err := f.repo.CreateUnit(ctx, f.alice, f.units["Payments"], rbac.UnitInput{Name: "Ledger"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.Assign(ctx, f.alice, f.users["dave"], uuid.MustParse(rbac.ManagerRoleID), rbac.UnitScope(ledger.ID), ""); err != nil {
		t.Fatal(err)
	}
	dave, daveCSRF, _ := f.login(t, "dave@example.test")
	input := api.ExplainRequest{UserID: f.users["bob"], Permission: rbac.DocsEdit, Resource: rbac.UnitScope(ledger.ID)}
	hidden := f.request(t, "POST", "/api/v1/access/explain", input, dave, daveCSRF, testOrigin)
	expectStatus(t, hidden, 200)
	explanation := decode[rbac.ExplanationView](t, hidden)
	if explanation.Allowed || explanation.Reason != "blocked_by_inherit_access" || len(explanation.Blocked) != 1 || explanation.Blocked[0].Source != "blocked_hidden" || explanation.Blocked[0].Unit != nil {
		t.Fatal("hidden boundary projection invalid", explanation)
	}
	for _, secret := range []string{"Engineering", f.units["Engineering"].String(), "Payments", f.units["Payments"].String()} {
		if strings.Contains(string(hidden.body), secret) {
			t.Fatal("hidden unit leaked", secret)
		}
	}
	block := explanation.Blocked[0].Assignment
	if block.ID != nil || block.Scope != nil || block.SourceUnit != nil || block.Note != nil || block.CanRevoke {
		t.Fatal("hidden block exposed metadata")
	}
	full := f.request(t, "POST", "/api/v1/access/explain", input, admin, adminCSRF, testOrigin)
	expectStatus(t, full, 200)
	visible := decode[rbac.ExplanationView](t, full)
	if len(visible.Blocked) != 1 || visible.Blocked[0].Source != "blocked" || visible.Blocked[0].Unit == nil || visible.Blocked[0].Unit.ID != f.units["Payments"] {
		t.Fatal("administrator lost full blocking provenance")
	}
	// The same inherited Manager grant is visible at Ledger but its Payments origin is not.
	members := f.request(t, "GET", "/api/v1/units/"+ledger.ID.String()+"/members", nil, dave, "", "")
	expectStatus(t, members, 200)
	if strings.Contains(string(members.body), f.units["Payments"].String()) {
		t.Fatal("member origin leaked")
	}
}

func TestSelfRevocationAndReducedDelegatorOverHTTP(t *testing.T) {
	f := newAdminFixture(t)
	ctx := context.Background()
	assistant, err := f.repo.CreateRole(ctx, f.alice, rbac.RoleInput{Name: "Assistant", Permissions: []rbac.Permission{rbac.UnitRead, rbac.UnitManageMembers}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.Assign(ctx, f.alice, f.users["frank"], assistant.ID, rbac.UnitScope(f.units["Payments"]), ""); err != nil {
		t.Fatal(err)
	}
	assignments, err := f.repo.Assignments(ctx, f.users["carol"])
	if err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/assignments/" + assignments[0].ID.String()
	frank, frankCSRF, _ := f.login(t, "frank@example.test")
	expectStatus(t, f.request(t, "DELETE", path, nil, frank, frankCSRF, testOrigin), 403)
	carol, carolCSRF, _ := f.login(t, "carol@example.test")
	expectStatus(t, f.request(t, "DELETE", path, nil, carol, carolCSRF, testOrigin), 204)
	expectStatus(t, f.request(t, "GET", "/api/v1/units/"+f.units["Payments"].String(), nil, carol, "", ""), 403)
}
