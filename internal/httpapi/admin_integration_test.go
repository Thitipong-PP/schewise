// SPDX-License-Identifier: Apache-2.0

package httpapi_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	api "github.com/Thitipong-PP/schewise/internal/apicontract"
	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type adminFixture struct {
	*fixture
	repo  *rbac.Repository
	alice uuid.UUID
	units map[string]uuid.UUID
	users map[string]uuid.UUID
}

func newAdminFixture(t *testing.T) *adminFixture {
	t.Helper()
	f := newFixture(t)
	_, _, alice := f.login(t, "admin@example.test")
	a := &adminFixture{fixture: f, repo: rbac.NewRepository(f.db, zap.NewNop(), nil), alice: uuid.MustParse(alice.ID), units: map[string]uuid.UUID{"root": rbac.RootScope().ID}, users: map[string]uuid.UUID{"alice": uuid.MustParse(alice.ID)}}
	for _, pair := range [][2]string{{"Engineering", "root"}, {"Payments", "Engineering"}, {"Mobile", "Engineering"}, {"Data", "root"}} {
		unit, err := a.repo.CreateUnit(context.Background(), a.alice, a.units[pair[1]], rbac.UnitInput{Name: pair[0]})
		if err != nil {
			t.Fatal(err)
		}
		a.units[pair[0]] = unit.ID
	}
	var hash string
	if err := f.db.QueryRow(`SELECT password_hash FROM users WHERE id=$1`, a.alice).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	for _, user := range []string{"bob", "carol", "dave", "eve", "frank"} {
		id := uuid.New()
		a.users[user] = id
		if _, err := f.db.Exec(`INSERT INTO users(id,email,password_hash) VALUES($1,$2,$3)`, id, user+"@example.test", hash); err != nil {
			t.Fatal(err)
		}
	}
	for _, grant := range []struct{ user, role, unit string }{{"bob", rbac.DeveloperRoleID, "Engineering"}, {"carol", rbac.ManagerRoleID, "Payments"}, {"dave", rbac.DeveloperRoleID, "Mobile"}, {"dave", rbac.ViewerRoleID, "Data"}, {"eve", rbac.ViewerRoleID, "root"}} {
		if _, err := a.repo.Assign(context.Background(), a.alice, a.users[grant.user], uuid.MustParse(grant.role), rbac.UnitScope(a.units[grant.unit]), "Source metadata: "+grant.unit); err != nil {
			t.Fatal(err)
		}
	}
	return a
}
func decode[T any](t *testing.T, got result) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(got.body, &value); err != nil {
		t.Fatal(err)
	}
	return value
}
func TestAdminEndpointsRequireAuthenticationAndAuthorization(t *testing.T) {
	f := newAdminFixture(t)
	cookie, csrf, _ := f.login(t, "frank@example.test")
	for _, op := range api.Operations() {
		if op.Existing || op.Public || op.Self {
			continue
		}
		t.Run(op.ID, func(t *testing.T) {
			path := strings.ReplaceAll(op.Path, "{id}", f.units["Payments"].String())
			path = strings.ReplaceAll(path, "{sid}", uuid.NewString())
			if strings.Contains(op.Path, "/users/{id}") {
				path = strings.ReplaceAll(op.Path, "{id}", f.users["frank"].String())
			}
			var input any
			if op.Request != nil {
				input = map[string]any{}
			}
			expectStatus(t, f.request(t, op.Method, path, input, nil, "", testOrigin), 401)
			denied := f.request(t, op.Method, path, input, cookie, csrf, testOrigin)
			expectStatus(t, denied, 403)
			failure := decode[api.Error](t, denied)
			if failure.Code != "forbidden" || failure.Error == "" {
				t.Fatal("inconsistent error envelope")
			}
		})
	}
}
func TestInheritedSourcePrivacyAcrossMembersAndExplain(t *testing.T) {
	f := newAdminFixture(t)
	manager, managerCSRF, _ := f.login(t, "carol@example.test")
	administrator, adminCSRF, _ := f.login(t, "admin@example.test")
	path := "/api/v1/units/" + f.units["Payments"].String()
	for _, endpoint := range []string{"/members", "/access"} {
		hidden := f.request(t, "GET", path+endpoint, nil, manager, "", "")
		expectStatus(t, hidden, 200)
		if strings.Contains(string(hidden.body), f.units["Engineering"].String()) || strings.Contains(string(hidden.body), "Engineering") {
			t.Fatal("hidden origin leaked through member response")
		}
		rows := decode[api.MembersResponse](t, hidden)
		found := false
		for _, member := range rows.Members {
			if member.User.ID != f.users["bob"] {
				continue
			}
			found = true
			for _, assignment := range member.Assignments {
				if assignment.Source != rbac.SourceHidden || assignment.Scope != nil || assignment.SourceUnit != nil || assignment.ID != nil || assignment.Note != nil || assignment.CanRevoke || assignment.ReadOnlyReason == "" {
					t.Fatalf("unsafe hidden assignment: %+v", assignment)
				}
			}
		}
		if !found {
			t.Fatal("inherited member disappeared")
		}
		full := f.request(t, "GET", path+endpoint, nil, administrator, "", "")
		expectStatus(t, full, 200)
		if !strings.Contains(string(full.body), f.units["Engineering"].String()) || !strings.Contains(string(full.body), "Engineering") {
			t.Fatal("Administrator lost readable source")
		}
	}
	input := api.ExplainRequest{UserID: f.users["bob"], Permission: rbac.DocsEdit, Resource: rbac.UnitScope(f.units["Payments"])}
	hidden := f.request(t, "POST", "/api/v1/access/explain", input, manager, managerCSRF, testOrigin)
	expectStatus(t, hidden, 200)
	value := decode[rbac.ExplanationView](t, hidden)
	if !value.Allowed || len(value.Grants) != 1 || value.Grants[0].Source != rbac.SourceHidden || value.Grants[0].Scope != nil || value.ReasonText == "" {
		t.Fatal("Explain did not redact source")
	}
	if strings.Contains(string(hidden.body), f.units["Engineering"].String()) || strings.Contains(string(hidden.body), "Engineering") {
		t.Fatal("Explain leaked ancestor")
	}
	full := f.request(t, "POST", "/api/v1/access/explain", input, administrator, adminCSRF, testOrigin)
	expectStatus(t, full, 200)
	if !strings.Contains(string(full.body), "Engineering") {
		t.Fatal("readable Explain origin missing")
	}
	assignments, err := f.repo.Assignments(context.Background(), f.users["bob"])
	if err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request(t, "DELETE", "/api/v1/assignments/"+assignments[0].ID.String(), nil, manager, managerCSRF, testOrigin), 403)
	// Redaction is per assignment: Bob also receives a directly visible Viewer.
	if _, err := f.repo.Assign(context.Background(), f.alice, f.users["bob"], uuid.MustParse(rbac.ViewerRoleID), rbac.UnitScope(f.units["Payments"]), "Visible note"); err != nil {
		t.Fatal(err)
	}
	mixed := decode[api.MembersResponse](t, f.request(t, "GET", path+"/members", nil, manager, "", ""))
	for _, member := range mixed.Members {
		if member.User.ID == f.users["bob"] {
			visible, hidden := 0, 0
			for _, grant := range member.Assignments {
				if grant.Source == rbac.SourceHidden {
					hidden++
				} else if grant.SourceUnit != nil && grant.SourceUnit.ID == f.units["Payments"] {
					visible++
				}
			}
			if visible != 1 || hidden != 1 {
				t.Fatal("redaction applied to entire response instead of assignment")
			}
		}
	}
}
func TestHiddenAndMissingUnitsAreIndistinguishable(t *testing.T) {
	f := newAdminFixture(t)
	cookie, csrf, _ := f.login(t, "carol@example.test")
	hidden, missing := f.units["Engineering"].String(), uuid.NewString()
	for _, op := range api.Operations() {
		if !strings.Contains(op.Path, "/units/{id}") {
			continue
		}
		t.Run(op.ID, func(t *testing.T) {
			var input any
			if op.Request != nil {
				input = map[string]any{}
			}
			a := f.request(t, op.Method, strings.ReplaceAll(op.Path, "{id}", hidden), input, cookie, csrf, testOrigin)
			b := f.request(t, op.Method, strings.ReplaceAll(op.Path, "{id}", missing), input, cookie, csrf, testOrigin)
			expectStatus(t, a, 403)
			expectStatus(t, b, 403)
			if string(a.body) != string(b.body) {
				t.Fatal("hidden/missing response difference")
			}
		})
	}
	visible := f.request(t, "GET", "/api/v1/units", nil, cookie, "", "")
	expectStatus(t, visible, 200)
	if strings.Contains(string(visible.body), hidden) || strings.Contains(string(visible.body), "Engineering") {
		t.Fatal("visible forest leaked hidden parent")
	}
}
func TestAdminRoleAndAssignmentLifecycle(t *testing.T) {
	f := newAdminFixture(t)
	admin, csrf, _ := f.login(t, "admin@example.test")
	manager, managerCSRF, _ := f.login(t, "carol@example.test")
	roles := f.request(t, "GET", "/api/v1/roles", nil, admin, "", "")
	expectStatus(t, roles, 200)
	if len(decode[api.RolesResponse](t, roles).Roles) != 4 {
		t.Fatal("system roles missing")
	}
	catalog := f.request(t, "GET", "/api/v1/permissions", nil, manager, "", "")
	expectStatus(t, catalog, 200)
	for _, p := range decode[api.CatalogResponse](t, catalog).Permissions {
		if p.Description == "" || p.Area == "" {
			t.Fatal("permission description missing")
		}
	}
	request := api.RoleRequest{Name: "Reviewer", Description: "Synthetic custom role", Permissions: []rbac.Permission{rbac.UnitRead, rbac.SchemaRead}}
	created := f.request(t, "POST", "/api/v1/roles", request, admin, csrf, testOrigin)
	expectStatus(t, created, 201)
	role := decode[api.RoleResponse](t, created).Role
	assign := api.AssignmentRequest{UserID: f.users["frank"], RoleID: role.ID, Scope: rbac.UnitScope(f.units["Payments"]), Note: "Synthetic review access"}
	granted := f.request(t, "POST", "/api/v1/assignments", assign, manager, managerCSRF, testOrigin)
	expectStatus(t, granted, 201)
	assignment := decode[api.AssignmentResponse](t, granted).Assignment
	if assignment.ID == nil {
		t.Fatal("new direct assignment ID missing")
	}
	expectStatus(t, f.request(t, "DELETE", "/api/v1/roles/"+role.ID.String(), nil, admin, csrf, testOrigin), 409)
	mixed := api.RoleRequest{Name: role.Name, Description: role.Description, Permissions: []rbac.Permission{rbac.UnitRead, rbac.UserManage}}
	expectStatus(t, f.request(t, "PATCH", "/api/v1/roles/"+role.ID.String(), mixed, admin, csrf, testOrigin), 400)
	escalation := api.AssignmentRequest{UserID: f.users["frank"], RoleID: uuid.MustParse(rbac.AdministratorRoleID), Scope: rbac.UnitScope(f.units["Payments"])}
	expectStatus(t, f.request(t, "POST", "/api/v1/assignments", escalation, manager, managerCSRF, testOrigin), 403)
	escalation.Scope = rbac.RootScope()
	expectStatus(t, f.request(t, "POST", "/api/v1/assignments", escalation, manager, managerCSRF, testOrigin), 403)
	cloned := f.request(t, "POST", "/api/v1/roles/"+rbac.ViewerRoleID+"/clone", api.CloneRequest{Name: "Viewer copy"}, admin, csrf, testOrigin)
	expectStatus(t, cloned, 201)
	system := api.RoleRequest{Name: "Renamed", Permissions: []rbac.Permission{rbac.UnitRead}}
	expectStatus(t, f.request(t, "PATCH", "/api/v1/roles/"+rbac.ViewerRoleID, system, admin, csrf, testOrigin), 409)
	expectStatus(t, f.request(t, "DELETE", "/api/v1/roles/"+rbac.ViewerRoleID, nil, admin, csrf, testOrigin), 409)
	expectStatus(t, f.request(t, "DELETE", "/api/v1/assignments/"+assignment.ID.String(), nil, manager, managerCSRF, testOrigin), 204)
	expectStatus(t, f.request(t, "DELETE", "/api/v1/roles/"+role.ID.String(), nil, admin, csrf, testOrigin), 204)
	original, err := f.repo.Assignments(context.Background(), f.alice)
	if err != nil {
		t.Fatal(err)
	}
	last := f.request(t, "DELETE", "/api/v1/assignments/"+original[0].ID.String(), nil, admin, csrf, testOrigin)
	expectStatus(t, last, 409)
	if decode[api.Error](t, last).Code != "last_administrator" {
		t.Fatal("last admin protection missing")
	}
}
func TestAdminUnitPreviewAndAudit(t *testing.T) {
	f := newAdminFixture(t)
	cookie, csrf, _ := f.login(t, "admin@example.test")
	viewer, _, _ := f.login(t, "eve@example.test")
	create := api.CreateUnitRequest{ParentID: f.units["Payments"], UnitRequest: api.UnitRequest{Name: "Empty", Kind: "team"}}
	got := f.request(t, "POST", "/api/v1/units", create, cookie, csrf, testOrigin)
	expectStatus(t, got, 201)
	unit := decode[api.UnitResponse](t, got).Unit
	path := "/api/v1/units/" + unit.ID.String()
	rename := f.request(t, "PATCH", path, api.UnitRequest{Name: "Renamed", Description: "Synthetic", Kind: "project"}, cookie, csrf, testOrigin)
	expectStatus(t, rename, 200)
	moved := api.MoveRequest{ParentID: f.units["Data"]}
	preview := f.request(t, "POST", path+"/dry-run/move", moved, cookie, csrf, testOrigin)
	expectStatus(t, preview, 200)
	if !decode[rbac.Impact](t, preview).Executable {
		t.Fatal("move preview invalid")
	}
	current, err := f.repo.Unit(context.Background(), unit.ID)
	if err != nil || current.ParentID != f.units["Payments"] {
		t.Fatal("preview moved unit")
	}
	expectStatus(t, f.request(t, "POST", path+"/move", moved, cookie, csrf, testOrigin), 204)
	deletion := f.request(t, "POST", path+"/dry-run/delete", api.EmptyRequest{}, cookie, csrf, testOrigin)
	expectStatus(t, deletion, 200)
	if !decode[rbac.Impact](t, deletion).Executable {
		t.Fatal("empty deletion blocked")
	}
	expectStatus(t, f.request(t, "DELETE", path, nil, cookie, csrf, testOrigin), 204)
	blocked := f.request(t, "POST", "/api/v1/units/"+f.units["Payments"].String()+"/dry-run/delete", api.EmptyRequest{}, cookie, csrf, testOrigin)
	expectStatus(t, blocked, 200)
	if decode[rbac.Impact](t, blocked).Executable {
		t.Fatal("nonempty unit deletion permitted")
	}
	expectStatus(t, f.request(t, "GET", "/api/v1/audit", nil, viewer, "", ""), 403)
	log := f.request(t, "GET", "/api/v1/audit?action=unit.create&actor_id="+f.alice.String()+"&limit=2", nil, cookie, "", "")
	expectStatus(t, log, 200)
	page := decode[rbac.AuditPage](t, log)
	if len(page.Entries) == 0 || len(page.Entries) > 2 || !page.HasMore {
		t.Fatal("audit pagination missing")
	}
	for _, entry := range page.Entries {
		if entry.Action != "unit.create" || entry.ActorUserID != f.alice {
			t.Fatal("audit filters not applied")
		}
	}
	expectStatus(t, f.request(t, "GET", "/api/v1/audit?from=invalid", nil, cookie, "", ""), 400)
	expectStatus(t, f.request(t, "GET", "/api/v1/audit?from="+time.Now().Add(time.Hour).UTC().Format(time.RFC3339)+"&to="+time.Now().UTC().Format(time.RFC3339), nil, cookie, "", ""), 400)
}

func TestScopedUserAccessAndDirectory(t *testing.T) {
	f := newAdminFixture(t)
	manager, csrf, _ := f.login(t, "carol@example.test")
	viewer, viewerCSRF, _ := f.login(t, "eve@example.test")
	path := "/api/v1/users/" + f.users["bob"].String()
	for _, endpoint := range []string{"/assignments", "/access"} {
		got := f.request(t, "GET", path+endpoint, nil, manager, "", "")
		expectStatus(t, got, 200)
		if strings.Contains(string(got.body), "Engineering") || strings.Contains(string(got.body), f.units["Engineering"].String()) {
			t.Fatal("user access exposed an unreadable ancestor")
		}
		expectStatus(t, f.request(t, "GET", path+endpoint, nil, viewer, "", ""), 403)
	}
	access := decode[rbac.UserAccess](t, f.request(t, "GET", path+"/access", nil, manager, "", ""))
	if len(access.Units) != 1 || access.Units[0].Unit.ID != f.units["Payments"] {
		t.Fatal("user access escaped the manageable subtree")
	}
	self := api.ExplainRequest{UserID: f.users["eve"], Permission: rbac.UnitRead, Resource: rbac.UnitScope(f.units["Payments"])}
	expectStatus(t, f.request(t, "POST", "/api/v1/access/explain", self, viewer, viewerCSRF, testOrigin), 200)
	self.UserID = f.users["bob"]
	expectStatus(t, f.request(t, "POST", "/api/v1/access/explain", self, viewer, viewerCSRF, testOrigin), 403)
	expectStatus(t, f.request(t, "GET", "/api/v1/users", nil, manager, "", ""), 403)
	users := f.request(t, "GET", "/api/v1/users?unit_id="+f.units["Payments"].String()+"&q=bob", nil, manager, "", "")
	expectStatus(t, users, 200)
	if rows := decode[api.UsersResponse](t, users).Users; len(rows) != 1 || rows[0].ID != f.users["bob"] {
		t.Fatal("assignment directory search failed")
	}
	roles := f.request(t, "GET", "/api/v1/units/"+f.units["Payments"].String()+"/assignable-roles", nil, manager, "", "")
	expectStatus(t, roles, 200)
	for _, role := range decode[api.AssignableRolesResponse](t, roles).Roles {
		for _, p := range role.Permissions {
			if p.SystemLevel() {
				t.Fatal("root-only role offered below root")
			}
		}
	}
	// New HTTP mutations retain the same synchronizer-CSRF boundary.
	expectStatus(t, f.request(t, "POST", "/api/v1/access/explain", self, manager, "", testOrigin), 403)
	expectStatus(t, f.request(t, "POST", "/api/v1/access/explain", self, manager, csrf, "https://untrusted.example"), 403)
}

func TestRoleEditorCannotEscalateAndCanRelinquishOwnPermission(t *testing.T) {
	f := newAdminFixture(t)
	ctx := context.Background()
	role, err := f.repo.CreateRole(ctx, f.alice, rbac.RoleInput{Name: "Role editor", Permissions: []rbac.Permission{rbac.RoleManage, rbac.UnitRead}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.Assign(ctx, f.alice, f.users["frank"], role.ID, rbac.RootScope(), "Synthetic role editor"); err != nil {
		t.Fatal(err)
	}
	cookie, csrf, _ := f.login(t, "frank@example.test")
	path := "/api/v1/roles/" + role.ID.String()
	escalated := api.RoleRequest{Name: role.Name, Permissions: []rbac.Permission{rbac.RoleManage, rbac.UnitRead, rbac.UserManage}}
	expectStatus(t, f.request(t, "PATCH", path, escalated, cookie, csrf, testOrigin), 403)
	unchanged, err := f.repo.Role(ctx, role.ID)
	if err != nil || len(unchanged.Permissions) != 2 {
		t.Fatal("escalation changed the role")
	}
	// The successful write must not be reported as 403 after its own gate is removed.
	reduced := api.RoleRequest{Name: role.Name, Permissions: []rbac.Permission{rbac.UnitRead}}
	expectStatus(t, f.request(t, "PATCH", path, reduced, cookie, csrf, testOrigin), 200)
	expectStatus(t, f.request(t, "GET", "/api/v1/roles", nil, cookie, "", ""), 403)
}
