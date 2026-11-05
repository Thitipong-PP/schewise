// SPDX-License-Identifier: Apache-2.0

package rbac_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/google/uuid"
)

func id(name string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte("synthetic-rbac:"+name))
}

type memoryReader struct {
	users       map[uuid.UUID]bool
	units       map[uuid.UUID]rbac.Unit
	roles       map[uuid.UUID]rbac.Role
	assignments []rbac.Assignment
	connections map[uuid.UUID]uuid.UUID
	failure     error
}

func (r *memoryReader) ActiveUser(_ context.Context, user uuid.UUID) (bool, error) {
	return r.users[user], nil
}
func (r *memoryReader) Unit(_ context.Context, unit uuid.UUID) (rbac.Unit, error) {
	value, ok := r.units[unit]
	if !ok {
		return value, rbac.ErrNotFound
	}
	return value, nil
}
func (r *memoryReader) Role(_ context.Context, role uuid.UUID) (rbac.Role, error) {
	value, ok := r.roles[role]
	if !ok {
		return value, rbac.ErrNotFound
	}
	return value, nil
}
func (r *memoryReader) ConnectionUnit(_ context.Context, connection uuid.UUID) (uuid.UUID, error) {
	unit, ok := r.connections[connection]
	if !ok {
		return uuid.Nil, rbac.ErrNotFound
	}
	return unit, nil
}
func (r *memoryReader) Grants(_ context.Context, user uuid.UUID, scopes []rbac.Scope) ([]rbac.RoleGrant, error) {
	if r.failure != nil {
		return nil, r.failure
	}
	grants := []rbac.RoleGrant{}
	for _, a := range r.assignments {
		if a.UserID == user && slices.Contains(scopes, a.Scope) {
			grants = append(grants, rbac.RoleGrant{Assignment: a, Role: r.roles[a.RoleID]})
		}
	}
	return grants, nil
}
func (r *memoryReader) assign(user, role string, scope rbac.Scope) {
	roleID := id(role)
	switch role {
	case "Administrator":
		roleID = uuid.MustParse(rbac.AdministratorRoleID)
	case "Manager":
		roleID = uuid.MustParse(rbac.ManagerRoleID)
	case "Developer":
		roleID = uuid.MustParse(rbac.DeveloperRoleID)
	case "Viewer":
		roleID = uuid.MustParse(rbac.ViewerRoleID)
	}
	r.assignments = append(r.assignments, rbac.Assignment{ID: id(user + role + scope.ID.String()), UserID: id(user), RoleID: roleID, Scope: scope})
}
func unitScope(name string) rbac.Scope {
	if name == "Organization" {
		return rbac.RootScope()
	}
	return rbac.UnitScope(id(name))
}
func memoryFixture() *memoryReader {
	r := &memoryReader{users: map[uuid.UUID]bool{}, units: map[uuid.UUID]rbac.Unit{}, roles: map[uuid.UUID]rbac.Role{}, connections: map[uuid.UUID]uuid.UUID{}}
	for _, name := range []string{"alice", "bob", "carol", "dave", "eve", "frank"} {
		r.users[id(name)] = true
	}
	root := rbac.RootScope().ID
	r.units[root] = rbac.Unit{ID: root, Name: "Organization", Ancestors: []uuid.UUID{}}
	for _, name := range []string{"Engineering", "Data"} {
		r.units[id(name)] = rbac.Unit{ID: id(name), ParentID: root, Name: name, Ancestors: []uuid.UUID{root}}
	}
	for _, name := range []string{"Payments", "Mobile"} {
		r.units[id(name)] = rbac.Unit{ID: id(name), ParentID: id("Engineering"), Name: name, Ancestors: []uuid.UUID{root, id("Engineering")}}
	}
	for _, name := range []string{"Payments", "Mobile", "Data"} {
		r.connections[id(name+"DB")] = id(name)
	}
	all, manager, developer := rbac.Permissions(), []rbac.Permission{}, []rbac.Permission{}
	for _, p := range all {
		if !p.SystemLevel() {
			manager = append(manager, p)
			if p != rbac.UnitManageMembers {
				developer = append(developer, p)
			}
		}
	}
	for _, role := range []rbac.Role{{ID: uuid.MustParse(rbac.AdministratorRoleID), Name: "Administrator", IsSystem: true, Permissions: all}, {ID: uuid.MustParse(rbac.ManagerRoleID), Name: "Manager", IsSystem: true, Permissions: manager}, {ID: uuid.MustParse(rbac.DeveloperRoleID), Name: "Developer", IsSystem: true, Permissions: developer}, {ID: uuid.MustParse(rbac.ViewerRoleID), Name: "Viewer", IsSystem: true, Permissions: []rbac.Permission{rbac.UnitRead, rbac.ConnectionRead, rbac.SchemaRead}}} {
		r.roles[role.ID] = role
	}
	r.assign("alice", "Administrator", rbac.RootScope())
	r.assign("bob", "Developer", unitScope("Engineering"))
	r.assign("carol", "Manager", unitScope("Payments"))
	r.assign("dave", "Developer", unitScope("Mobile"))
	r.assign("dave", "Viewer", unitScope("Data"))
	r.assign("eve", "Viewer", rbac.RootScope())
	return r
}
func TestAuthorizationFixture(t *testing.T) {
	r := memoryFixture()
	engine := rbac.NewEngine(r)
	ctx := context.Background()
	tests := []struct {
		name, user string
		permission rbac.Permission
		resource   rbac.Scope
		want       bool
	}{
		{"developer reaches payments", "bob", rbac.DataQuery, unitScope("Payments"), true},
		{"developer reaches mobile", "bob", rbac.DocsEdit, unitScope("Mobile"), true},
		{"developer cannot reach data", "bob", rbac.UnitRead, unitScope("Data"), false},
		{"developer cannot reach ancestor", "bob", rbac.UnitRead, rbac.RootScope(), false},
		{"developer cannot manage assignments", "bob", rbac.UnitManageMembers, unitScope("Payments"), false},
		{"manager controls payments", "carol", rbac.UnitManageMembers, unitScope("Payments"), true},
		{"manager cannot reach sibling", "carol", rbac.UnitRead, unitScope("Mobile"), false},
		{"manager cannot reach parent", "carol", rbac.UnitManage, unitScope("Engineering"), false},
		{"multiple assignments edit mobile", "dave", rbac.DocsEdit, unitScope("Mobile"), true},
		{"multiple assignments read data", "dave", rbac.SchemaRead, unitScope("Data"), true},
		{"viewer cannot edit data", "dave", rbac.DocsEdit, unitScope("Data"), false},
		{"root viewer inherits read", "eve", rbac.UnitRead, unitScope("Payments"), true},
		{"root viewer cannot query data", "eve", rbac.DataQuery, unitScope("Payments"), false},
		{"connection inherits owner chain", "bob", rbac.ConnectionManage, rbac.Scope{Type: rbac.ScopeConnection, ID: id("PaymentsDB")}, true},
		{"connection outside subtree", "bob", rbac.ConnectionRead, rbac.Scope{Type: rbac.ScopeConnection, ID: id("DataDB")}, false},
		{"unknown permission", "alice", rbac.Permission("database:destroy"), rbac.RootScope(), false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := engine.Can(ctx, id(test.user), test.permission, test.resource)
			if err != nil || got != test.want {
				t.Fatalf("Can = %v, %v; want %v", got, err, test.want)
			}
		})
	}
	for _, resource := range []rbac.Scope{rbac.RootScope(), unitScope("Engineering"), unitScope("Payments"), unitScope("Mobile"), unitScope("Data"), {Type: rbac.ScopeConnection, ID: id("PaymentsDB")}} {
		for _, permission := range rbac.Permissions() {
			if got, err := engine.Can(ctx, id("frank"), permission, resource); got || err != nil {
				t.Fatalf("deny by default: %s %v %v", permission, got, err)
			}
			if got, err := engine.Can(ctx, id("alice"), permission, resource); !got || err != nil {
				t.Fatalf("root Administrator: %s %v %v", permission, got, err)
			}
		}
	}
}
func TestUnionAndExplain(t *testing.T) {
	r := memoryFixture()
	r.assignments = slices.DeleteFunc(r.assignments, func(a rbac.Assignment) bool { return a.UserID == id("bob") })
	r.assign("bob", "Viewer", unitScope("Engineering"))
	r.assign("bob", "Developer", unitScope("Payments"))
	e := rbac.NewEngine(r)
	ctx := context.Background()
	for _, test := range []struct {
		permission rbac.Permission
		scope      string
		want       bool
	}{{rbac.DocsEdit, "Payments", true}, {rbac.UnitRead, "Payments", true}, {rbac.UnitRead, "Mobile", true}, {rbac.DocsEdit, "Mobile", false}, {rbac.UnitRead, "Data", false}} {
		got, err := e.Can(ctx, id("bob"), test.permission, unitScope(test.scope))
		if err != nil || got != test.want {
			t.Fatalf("union: %+v = %v %v", test, got, err)
		}
	}
	explanation, err := e.Explain(ctx, id("bob"), rbac.UnitRead, unitScope("Payments"))
	if err != nil || !explanation.Allowed || len(explanation.Grants) != 2 {
		t.Fatalf("Explain = %+v, %v", explanation, err)
	}
	grants := map[string]rbac.Grant{}
	for _, grant := range explanation.Grants {
		grants[grant.Role] = grant
		if grant.AssignmentID == uuid.Nil || grant.RoleID == uuid.Nil {
			t.Fatal("missing grant identity")
		}
	}
	if !grants["Viewer"].Inherited || grants["Developer"].Inherited || grants["Viewer"].Scope != unitScope("Engineering") || grants["Developer"].Scope != unitScope("Payments") {
		t.Fatalf("wrong inheritance provenance: %+v", grants)
	}
	denied, err := e.Explain(ctx, id("frank"), rbac.UnitRead, unitScope("Payments"))
	if err != nil || denied.Allowed || len(denied.Grants) != 0 || denied.Reason != "no_matching_assignment" {
		t.Fatalf("denied explanation: %+v %v", denied, err)
	}
	// A narrower Viewer can add access, never cancel an inherited Developer grant.
	r.assign("bob", "Developer", unitScope("Engineering"))
	r.assign("bob", "Viewer", unitScope("Payments"))
	if got, err := e.Can(ctx, id("bob"), rbac.DocsEdit, unitScope("Payments")); !got || err != nil {
		t.Fatal("narrow role subtracted permission")
	}
}
func TestSystemPermissionsAndConnectionIsolation(t *testing.T) {
	r := memoryFixture()
	role := rbac.Role{ID: id("RootOnly"), Name: "RootOnly", Permissions: []rbac.Permission{rbac.UserManage, rbac.RoleManage, rbac.AuditRead, rbac.UnitRead}}
	r.roles[role.ID] = role
	for _, scope := range []rbac.Scope{unitScope("Payments"), {Type: rbac.ScopeConnection, ID: id("PaymentsDB")}} {
		r.assign("frank", "RootOnly", scope)
		e := rbac.NewEngine(r)
		for _, p := range []rbac.Permission{rbac.UserManage, rbac.RoleManage, rbac.AuditRead} {
			if got, err := e.Can(context.Background(), id("frank"), p, rbac.Scope{Type: rbac.ScopeConnection, ID: id("PaymentsDB")}); got || err != nil {
				t.Fatal("non-root system grant became effective")
			}
		}
	}
	r.assignments = slices.DeleteFunc(r.assignments, func(a rbac.Assignment) bool { return a.UserID == id("frank") })
	r.assign("frank", "Viewer", rbac.Scope{Type: rbac.ScopeConnection, ID: id("PaymentsDB")})
	e := rbac.NewEngine(r)
	for _, test := range []struct {
		resource rbac.Scope
		want     bool
	}{{rbac.Scope{Type: rbac.ScopeConnection, ID: id("PaymentsDB")}, true}, {unitScope("Payments"), false}, {rbac.Scope{Type: rbac.ScopeConnection, ID: id("MobileDB")}, false}} {
		got, err := e.Can(context.Background(), id("frank"), rbac.UnitRead, test.resource)
		if err != nil || got != test.want {
			t.Fatalf("connection isolation: %v %v", got, err)
		}
	}
}
func TestCanAssign(t *testing.T) {
	e := rbac.NewEngine(memoryFixture())
	ctx := context.Background()
	for _, test := range []struct {
		name, user, role, scope string
		want                    bool
	}{
		{"manager cannot grant Administrator", "carol", rbac.AdministratorRoleID, "Payments", false},
		{"manager cannot grant outside subtree", "carol", rbac.ManagerRoleID, "Engineering", false},
		{"manager delegates Developer", "carol", rbac.DeveloperRoleID, "Payments", true},
		{"manager delegates Viewer", "carol", rbac.ViewerRoleID, "Payments", true},
		{"developer cannot grant Viewer", "bob", rbac.ViewerRoleID, "Engineering", false},
		{"developer cannot grant Developer", "bob", rbac.DeveloperRoleID, "Payments", false},
		{"root Administrator delegates Administrator", "alice", rbac.AdministratorRoleID, "Organization", true},
		{"system role restricted to root", "alice", rbac.AdministratorRoleID, "Payments", false},
		{"unassigned user cannot delegate", "frank", rbac.ViewerRoleID, "Payments", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := e.CanAssign(ctx, id(test.user), uuid.MustParse(test.role), unitScope(test.scope))
			if err != nil || got != test.want {
				t.Fatalf("CanAssign=%v %v, want %v", got, err, test.want)
			}
		})
	}
}
func TestEngineFailsClosed(t *testing.T) {
	r := memoryFixture()
	e := rbac.NewEngine(r)
	ctx := context.Background()
	for _, scope := range []rbac.Scope{{Type: "group", ID: id("Payments")}, {Type: rbac.ScopeUnit, ID: uuid.Nil}, {Type: rbac.ScopeConnection, ID: id("missing")}} {
		if allowed, err := e.Can(ctx, id("alice"), rbac.UnitRead, scope); allowed || err == nil {
			t.Fatalf("invalid scope accepted: %v %v", allowed, err)
		}
	}
	r.users[id("alice")] = false
	if allowed, err := e.Can(ctx, id("alice"), rbac.UserManage, rbac.RootScope()); allowed || err != nil {
		t.Fatal("inactive user retained access")
	}
	r.users[id("alice")] = true
	r.failure = errors.New("store unavailable")
	if allowed, err := e.Can(ctx, id("alice"), rbac.UnitRead, rbac.RootScope()); allowed || err == nil {
		t.Fatal("store failure did not fail closed")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if allowed, err := e.Can(canceled, id("alice"), rbac.UnitRead, rbac.RootScope()); allowed || !errors.Is(err, context.Canceled) {
		t.Fatal("canceled request authorized")
	}
}
