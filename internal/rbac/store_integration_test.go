// SPDX-License-Identifier: Apache-2.0

package rbac_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Thitipong-PP/schewise/internal/auth"
	"github.com/Thitipong-PP/schewise/internal/config"
	"github.com/Thitipong-PP/schewise/internal/database"
	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func isolatedDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("SCHEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		if os.Getenv("SCHEWISE_INTEGRATION_TESTS") != "1" {
			t.Skip("set SCHEWISE_TEST_DATABASE_URL or run make integration-test")
		}
		cfg, err := config.Load()
		if err != nil {
			t.Fatal(err)
		}
		dsn = cfg.DatabaseURL
	}
	ctx := context.Background()
	admin, err := database.Open(ctx, dsn)
	if err != nil {
		t.Fatal("test database unavailable")
	}
	schema := "rbac_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		_ = admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.ExecContext(ctx, `DROP SCHEMA `+schema+` CASCADE`); err != nil {
			t.Error(err)
		}
		_ = admin.Close()
	})
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	db, err := database.Open(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

type fakeConnections map[uuid.UUID]uuid.UUID

func (c fakeConnections) Owner(_ context.Context, _ rbac.Queryer, id uuid.UUID) (uuid.UUID, error) {
	owner, ok := c[id]
	if !ok {
		return uuid.Nil, rbac.ErrNotFound
	}
	return owner, nil
}
func (c fakeConnections) InUnits(_ context.Context, _ rbac.Queryer, units []uuid.UUID) ([]rbac.Connection, error) {
	result := []rbac.Connection{}
	for connection, owner := range c {
		if slices.Contains(units, owner) {
			result = append(result, rbac.Connection{ID: connection, UnitID: owner})
		}
	}
	return result, nil
}

type storeFixture struct {
	db          *sql.DB
	repo        *rbac.Repository
	alice       uuid.UUID
	units       map[string]uuid.UUID
	connections fakeConnections
}

func newStoreFixture(t *testing.T) *storeFixture {
	t.Helper()
	ctx := context.Background()
	db := isolatedDB(t)
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	service, err := auth.New(db, zap.NewNop(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Bootstrap(ctx, "alice@example.test", "Synthetic-test-password-42!"); err != nil {
		t.Fatal(err)
	}
	f := &storeFixture{db: db, units: map[string]uuid.UUID{"Organization": rbac.RootScope().ID}, connections: fakeConnections{}}
	if err := db.QueryRow(`SELECT id FROM users WHERE email='alice@example.test'`).Scan(&f.alice); err != nil {
		t.Fatal(err)
	}
	f.repo = rbac.NewRepository(db, zap.NewNop(), f.connections)
	for _, pair := range [][2]string{{"Engineering", "Organization"}, {"Payments", "Engineering"}, {"Mobile", "Engineering"}, {"Data", "Organization"}} {
		unit, err := f.repo.CreateUnit(ctx, f.alice, f.units[pair[1]], rbac.UnitInput{Name: pair[0], Kind: "team"})
		if err != nil {
			t.Fatal(err)
		}
		f.units[pair[0]] = unit.ID
	}
	hash, err := auth.HashPassword("Synthetic-test-password-42!")
	if err != nil {
		t.Fatal(err)
	}
	for _, user := range []string{"bob", "carol", "dave", "eve", "frank"} {
		if _, err := db.Exec(`INSERT INTO users(id,email,password_hash) VALUES($1,$2,$3)`, id(user), user+"@example.test", hash); err != nil {
			t.Fatal(err)
		}
	}
	for _, a := range []struct{ user, role, scope string }{{"bob", rbac.DeveloperRoleID, "Engineering"}, {"carol", rbac.ManagerRoleID, "Payments"}, {"dave", rbac.DeveloperRoleID, "Mobile"}, {"dave", rbac.ViewerRoleID, "Data"}, {"eve", rbac.ViewerRoleID, "Organization"}} {
		if _, err := f.repo.Assign(ctx, f.alice, id(a.user), uuid.MustParse(a.role), rbac.UnitScope(f.units[a.scope]), ""); err != nil {
			t.Fatal(err)
		}
	}
	return f
}
func (f *storeFixture) scope(name string) rbac.Scope { return rbac.UnitScope(f.units[name]) }
func countRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var count int
	// Test callers provide only constant table names.
	if err := db.QueryRow(`SELECT count(*) FROM ` + table).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
func TestMigrationPreservesIdentityAndAudits(t *testing.T) {
	ctx := context.Background()
	db := isolatedDB(t)
	provider, err := database.Provider(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 1); err != nil {
		t.Fatal(err)
	}
	user := id("legacy-admin")
	hash, err := auth.HashPassword("Synthetic-legacy-password-42!")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO users(id,email,password_hash,role_id) VALUES($1,'legacy@example.test',$2,(SELECT id FROM roles WHERE name='Administrator'))`, user, hash); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO roles(id,name) VALUES($1,'LegacyRestricted')`, id("LegacyRestricted")); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO users(id,email,password_hash,role_id) VALUES($1,'restricted@example.test',$2,$3)`, id("legacy-restricted"), hash, id("LegacyRestricted")); err != nil {
		t.Fatal(err)
	}
	token, err := auth.RandomToken()
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(token))
	if _, err := db.Exec(`INSERT INTO sessions(token_hash,user_id,csrf_token,expires_at) VALUES($1,$2,'synthetic-csrf',now()+interval '1 hour')`, digest[:], user); err != nil {
		t.Fatal(err)
	}
	audit := id("legacy-audit")
	if _, err := db.Exec(`INSERT INTO audit_events(id,request_id,actor_id,target_id,action,outcome,remote_ip) VALUES($1,'migration-test',$2,$2,'user.bootstrap','success','127.0.0.1')`, audit, user); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := database.Migrate(ctx, db); err != nil {
			t.Fatal(err)
		}
	}
	repo := rbac.NewRepository(db, zap.NewNop(), nil)
	assignments, err := repo.Assignments(ctx, user)
	if err != nil || len(assignments) != 1 || assignments[0].RoleID != uuid.MustParse(rbac.AdministratorRoleID) || assignments[0].Scope != rbac.RootScope() {
		t.Fatalf("admin migration: %+v %v", assignments, err)
	}
	service, err := auth.New(db, zap.NewNop(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	session, err := service.Lookup(ctx, token)
	if err != nil || session.User.ID != user.String() || session.User.Role != "Administrator" {
		t.Fatal("existing session did not survive migration", err)
	}
	var preserved string
	if err := db.QueryRow(`SELECT password_hash FROM users WHERE id=$1`, user).Scan(&preserved); err != nil || preserved != hash {
		t.Fatal("password hash changed")
	}
	var outcome string
	if err := db.QueryRow(`SELECT details->>'outcome' FROM audit_log WHERE id=$1`, audit).Scan(&outcome); err != nil || outcome != "success" {
		t.Fatal("audit history lost", err)
	}
	var legacyColumn bool
	if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='users' AND column_name IN ('role_id','is_admin'))`).Scan(&legacyColumn); err != nil || legacyColumn {
		t.Fatal("old authority column retained", err)
	}
	for _, p := range rbac.Permissions() {
		if allowed, err := repo.Engine().Can(ctx, user, p, rbac.RootScope()); !allowed || err != nil {
			t.Fatal("migrated admin denied", p, err)
		}
	}
	if allowed, err := repo.Engine().Can(ctx, id("legacy-restricted"), rbac.UnitRead, rbac.RootScope()); allowed || err != nil {
		t.Fatal("legacy unknown role was elevated")
	}
	if _, err := provider.Down(ctx); err == nil {
		t.Fatal("lossy downgrade unexpectedly allowed")
	}
	if countRows(t, db, "assignments") != 2 || countRows(t, db, "units") != 1 {
		t.Fatal("downgrade failure mutated data")
	}
}
func TestStoreDelegationAndRoleGuards(t *testing.T) {
	f := newStoreFixture(t)
	ctx := context.Background()
	for _, test := range []struct {
		name, user, role, scope string
		want                    error
	}{{"manager grants developer", "carol", rbac.DeveloperRoleID, "Payments", nil}, {"manager grants viewer", "carol", rbac.ViewerRoleID, "Payments", nil}, {"manager cannot grant admin", "carol", rbac.AdministratorRoleID, "Organization", rbac.ErrForbidden}, {"system role below root", "alice", rbac.AdministratorRoleID, "Payments", rbac.ErrInvalidScope}, {"manager cannot grant parent", "carol", rbac.ManagerRoleID, "Engineering", rbac.ErrForbidden}, {"developer cannot delegate", "bob", rbac.ViewerRoleID, "Mobile", rbac.ErrForbidden}} {
		t.Run(test.name, func(t *testing.T) {
			actor := id(test.user)
			if test.user == "alice" {
				actor = f.alice
			}
			_, err := f.repo.Assign(ctx, actor, id("frank"), uuid.MustParse(test.role), f.scope(test.scope), "Synthetic delegation")
			if !errors.Is(err, test.want) {
				t.Fatalf("Assign=%v,want %v", err, test.want)
			}
		})
	}
	if _, err := f.repo.Assign(ctx, f.alice, id("frank"), uuid.MustParse(rbac.ViewerRoleID), rbac.Scope{Type: "group", ID: id("group")}, ""); !errors.Is(err, rbac.ErrInvalidScope) {
		t.Fatalf("unknown scope: %v", err)
	}
	if _, err := f.repo.Assign(ctx, f.alice, id("frank"), uuid.MustParse(rbac.ViewerRoleID), rbac.Scope{Type: rbac.ScopeConnection, ID: id("missing")}, ""); !errors.Is(err, rbac.ErrNotFound) {
		t.Fatalf("unknown connection: %v", err)
	}
	connection := id("PaymentsDB")
	f.connections[connection] = f.units["Payments"]
	if _, err := f.db.Exec(`INSERT INTO connections(id,unit_id,name,engine,host,port,database,username,ssl_mode,encrypted_password,nonce) VALUES($1,$2,'Fixture','postgres','localhost',5432,'fixture','fixture','disable','\x00'::bytea,decode(repeat('00',12),'hex'))`, connection, f.units["Payments"]); err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.Assign(ctx, id("carol"), id("frank"), uuid.MustParse(rbac.ViewerRoleID), rbac.Scope{Type: rbac.ScopeConnection, ID: connection}, ""); err != nil {
		t.Fatal(err)
	}
	outside, err := f.repo.Assign(ctx, f.alice, id("frank"), uuid.MustParse(rbac.ViewerRoleID), f.scope("Engineering"), "")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repo.Revoke(ctx, id("carol"), outside.ID); !errors.Is(err, rbac.ErrForbidden) {
		t.Fatalf("revoked ancestor assignment: %v", err)
	}
	for _, roleID := range []string{rbac.AdministratorRoleID, rbac.ManagerRoleID, rbac.DeveloperRoleID, rbac.ViewerRoleID} {
		if err := f.repo.DeleteRole(ctx, f.alice, uuid.MustParse(roleID)); !errors.Is(err, rbac.ErrSystemRole) {
			t.Fatal("system role deleted", err)
		}
		if _, err := f.repo.UpdateRole(ctx, f.alice, uuid.MustParse(roleID), rbac.RoleInput{Name: "Changed"}); !errors.Is(err, rbac.ErrSystemRole) {
			t.Fatal("system role changed", err)
		}
	}
	clone, err := f.repo.CloneRole(ctx, f.alice, uuid.MustParse(rbac.ViewerRoleID), "CustomViewer")
	if err != nil || clone.IsSystem || len(clone.Permissions) != 3 {
		t.Fatal("clone invalid", err)
	}
	if _, err := f.repo.Assign(ctx, f.alice, id("frank"), clone.ID, f.scope("Payments"), ""); err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.UpdateRole(ctx, f.alice, clone.ID, rbac.RoleInput{Name: clone.Name, Permissions: []rbac.Permission{rbac.UserManage}}); !errors.Is(err, rbac.ErrInvalidScope) {
		t.Fatal("assigned role gained root-only permission", err)
	}
	if err := f.repo.DeleteRole(ctx, f.alice, clone.ID); !errors.Is(err, rbac.ErrNotEmpty) {
		t.Fatal("assigned role deleted", err)
	}
	unassigned, err := f.repo.CreateRole(ctx, f.alice, rbac.RoleInput{Name: "Temporary", Permissions: []rbac.Permission{rbac.UnitRead}})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := f.repo.UpdateRole(ctx, f.alice, unassigned.ID, rbac.RoleInput{Name: "Updated", Description: "Synthetic", Permissions: []rbac.Permission{rbac.UnitRead, rbac.DocsEdit}})
	if err != nil || len(updated.Permissions) != 2 {
		t.Fatal("custom role update failed", err)
	}
	if err := f.repo.DeleteRole(ctx, f.alice, unassigned.ID); err != nil {
		t.Fatal(err)
	}
	for _, permissions := range [][]rbac.Permission{{rbac.Permission("unknown:permission")}, {rbac.UnitRead, rbac.UnitRead}} {
		if _, err := f.repo.CreateRole(ctx, f.alice, rbac.RoleInput{Name: "Invalid", Permissions: permissions}); !errors.Is(err, rbac.ErrInvalidInput) {
			t.Fatal("invalid permissions accepted", err)
		}
	}
	// A role editor cannot elevate an already assigned role to permissions they lack.
	editor, err := f.repo.CreateRole(ctx, f.alice, rbac.RoleInput{Name: "RoleEditor", Permissions: []rbac.Permission{rbac.RoleManage}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.Assign(ctx, f.alice, id("frank"), editor.ID, rbac.RootScope(), ""); err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.UpdateRole(ctx, id("frank"), clone.ID, rbac.RoleInput{Name: clone.Name, Permissions: []rbac.Permission{rbac.DataQuery}}); !errors.Is(err, rbac.ErrForbidden) {
		t.Fatal("role edit escalated privileges", err)
	}
}

func TestLastAdministratorAndConcurrentMutations(t *testing.T) {
	t.Run("last active admin and sessions", func(t *testing.T) {
		f := newStoreFixture(t)
		ctx := context.Background()
		assignments, err := f.repo.Assignments(ctx, f.alice)
		if err != nil {
			t.Fatal(err)
		}
		if err := f.repo.Revoke(ctx, f.alice, assignments[0].ID); !errors.Is(err, rbac.ErrLastAdministrator) {
			t.Fatal("last admin revoked", err)
		}
		if err := f.repo.SetUserActive(ctx, f.alice, f.alice, false); !errors.Is(err, rbac.ErrLastAdministrator) {
			t.Fatal("last admin deactivated", err)
		}
		if _, err := f.repo.Assign(ctx, f.alice, id("bob"), uuid.MustParse(rbac.AdministratorRoleID), rbac.RootScope(), ""); err != nil {
			t.Fatal(err)
		}
		if _, err := f.db.Exec(`INSERT INTO sessions(token_hash,user_id,csrf_token,expires_at) VALUES($1,$2,'synthetic-csrf',now()+interval '1 hour')`, make([]byte, 32), id("bob")); err != nil {
			t.Fatal(err)
		}
		if err := f.repo.SetUserActive(ctx, f.alice, id("bob"), false); err != nil {
			t.Fatal(err)
		}
		if countRows(t, f.db, "sessions") != 0 {
			t.Fatal("deactivation left sessions active")
		}
		if allowed, err := f.repo.Engine().Can(ctx, id("bob"), rbac.UnitRead, f.scope("Engineering")); allowed || err != nil {
			t.Fatal("inactive user retained permissions")
		}
		if err := f.repo.Revoke(ctx, f.alice, assignments[0].ID); !errors.Is(err, rbac.ErrLastAdministrator) {
			t.Fatal("inactive Administrator counted as replacement", err)
		}
		if err := f.repo.SetUserActive(ctx, f.alice, id("bob"), true); err != nil {
			t.Fatal(err)
		}
		if err := f.repo.Revoke(ctx, f.alice, assignments[0].ID); err != nil {
			t.Fatal("replacement Administrator did not permit revocation", err)
		}
	})
	for _, mode := range []string{"revoke", "deactivate", "mixed"} {
		t.Run(mode, func(t *testing.T) {
			f := newStoreFixture(t)
			ctx := context.Background()
			bob, err := f.repo.Assign(ctx, f.alice, id("bob"), uuid.MustParse(rbac.AdministratorRoleID), rbac.RootScope(), "")
			if err != nil {
				t.Fatal(err)
			}
			alice, err := f.repo.Assignments(ctx, f.alice)
			if err != nil {
				t.Fatal(err)
			}
			operator, err := f.repo.CloneRole(ctx, f.alice, uuid.MustParse(rbac.AdministratorRoleID), "RecoveryOperator")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.repo.Assign(ctx, f.alice, id("frank"), operator.ID, rbac.RootScope(), ""); err != nil {
				t.Fatal(err)
			}
			jobs := []rbac.Assignment{alice[0], bob}
			start := make(chan struct{})
			results := make(chan error, 2)
			var group sync.WaitGroup
			for i, a := range jobs {
				group.Add(1)
				go func(i int, a rbac.Assignment) {
					defer group.Done()
					<-start
					if mode == "deactivate" || (mode == "mixed" && i == 1) {
						results <- f.repo.SetUserActive(ctx, id("frank"), a.UserID, false)
					} else {
						results <- f.repo.Revoke(ctx, id("frank"), a.ID)
					}
				}(i, a)
			}
			close(start)
			group.Wait()
			close(results)
			successes, protected := 0, 0
			for err := range results {
				if err == nil {
					successes++
				} else if errors.Is(err, rbac.ErrLastAdministrator) {
					protected++
				} else {
					t.Fatal(err)
				}
			}
			if successes != 1 || protected != 1 {
				t.Fatalf("concurrent results: successes=%d protected=%d", successes, protected)
			}
			var count int
			if err := f.db.QueryRow(`SELECT count(*) FROM assignments a JOIN users u ON u.id=a.user_id WHERE a.role_id=$1 AND a.scope_type='unit' AND a.scope_id=$2 AND u.is_active`, uuid.MustParse(rbac.AdministratorRoleID), rbac.RootScope().ID).Scan(&count); err != nil || count != 1 {
				t.Fatal("concurrency lost last admin", count, err)
			}
		})
	}
}
func findChange(impact rbac.Impact, user uuid.UUID, scope rbac.Scope) rbac.AccessChange {
	for _, change := range impact.Changes {
		if change.UserID == user && change.Resource == scope {
			return change
		}
	}
	return rbac.AccessChange{}
}
func TestUnitMovesDeletesAndImpact(t *testing.T) {
	f := newStoreFixture(t)
	ctx := context.Background()
	billing, err := f.repo.CreateUnit(ctx, f.alice, f.units["Payments"], rbac.UnitInput{Name: "Billing", Description: "Synthetic child", Kind: "free-form label"})
	if err != nil {
		t.Fatal(err)
	}
	f.connections[id("PaymentsDB")] = f.units["Payments"]
	f.connections[id("BillingDB")] = billing.ID
	for _, test := range []struct {
		name                string
		actor, unit, parent uuid.UUID
		want                error
	}{
		{"root is permanent", f.alice, rbac.RootScope().ID, f.units["Data"], rbac.ErrRootProtected},
		{"descendant cycle", f.alice, f.units["Engineering"], billing.ID, rbac.ErrCycle},
		{"self cycle", f.alice, f.units["Payments"], f.units["Payments"], rbac.ErrCycle},
		{"manager cannot escape subtree", id("carol"), f.units["Payments"], f.units["Data"], rbac.ErrForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := f.repo.MoveUnit(ctx, test.actor, test.unit, test.parent); !errors.Is(err, test.want) {
				t.Fatalf("MoveUnit=%v,want %v", err, test.want)
			}
		})
	}
	auditCount := countRows(t, f.db, "audit_log")
	impact, err := f.repo.DryRunMove(ctx, f.alice, f.units["Payments"], f.units["Data"])
	if err != nil {
		t.Fatal(err)
	}
	if !impact.Executable || len(impact.Connections) != 2 || len(impact.Blockers) != 0 {
		t.Fatalf("move impact: %+v", impact)
	}
	if !slices.Contains(findChange(impact, id("bob"), f.scope("Payments")).Lost, rbac.DataQuery) {
		t.Fatal("move did not report bob's lost access")
	}
	if !slices.Contains(findChange(impact, id("dave"), f.scope("Payments")).Gained, rbac.UnitRead) {
		t.Fatal("move did not report dave's gained read access")
	}
	if !slices.Contains(findChange(impact, id("bob"), rbac.Scope{Type: rbac.ScopeConnection, ID: id("BillingDB")}).Lost, rbac.ConnectionRead) {
		t.Fatal("connection access loss missing")
	}
	for _, change := range impact.Changes {
		if change.UserID == f.alice || change.UserID == id("eve") || change.UserID == id("carol") {
			t.Fatal("unchanged access reported as changed")
		}
	}
	before, err := f.repo.Unit(ctx, f.units["Payments"])
	if err != nil || before.ParentID != f.units["Engineering"] || countRows(t, f.db, "audit_log") != auditCount {
		t.Fatal("dry run mutated hierarchy or audit log", err)
	}
	if err := f.repo.MoveUnit(ctx, f.alice, f.units["Payments"], f.units["Data"]); err != nil {
		t.Fatal(err)
	}
	chain, err := f.repo.GetChain(ctx, billing.ID)
	if err != nil {
		t.Fatal(err)
	}
	got := []uuid.UUID{}
	for _, unit := range chain {
		got = append(got, unit.ID)
	}
	if !slices.Equal(got, []uuid.UUID{rbac.RootScope().ID, f.units["Data"], f.units["Payments"], billing.ID}) {
		t.Fatalf("descendant path not rebuilt: %v", got)
	}
	if allowed, err := f.repo.Engine().Can(ctx, id("bob"), rbac.DataQuery, rbac.UnitScope(billing.ID)); allowed || err != nil {
		t.Fatal("move left stale inherited permission")
	}
	fifth, err := f.repo.CreateUnit(ctx, f.alice, billing.ID, rbac.UnitInput{Name: "Depth5"})
	if err != nil {
		t.Fatal(err)
	}
	sixth, err := f.repo.CreateUnit(ctx, f.alice, fifth.ID, rbac.UnitInput{Name: "Depth6"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.CreateUnit(ctx, f.alice, sixth.ID, rbac.UnitInput{Name: "TooDeep"}); !errors.Is(err, rbac.ErrDepth) {
		t.Fatal("depth seven accepted", err)
	}
	if err := f.repo.MoveUnit(ctx, f.alice, f.units["Data"], f.units["Engineering"]); !errors.Is(err, rbac.ErrDepth) {
		t.Fatal("subtree move exceeded depth", err)
	}
	if _, err := f.repo.DryRunMove(ctx, f.alice, f.units["Data"], f.units["Engineering"]); !errors.Is(err, rbac.ErrDepth) {
		t.Fatal("dry run ignored depth", err)
	}
	blocked, err := f.repo.DryRunDelete(ctx, f.alice, f.units["Payments"])
	if err != nil {
		t.Fatal(err)
	}
	if blocked.Executable || len(blocked.Changes) != 0 || len(blocked.Connections) != 2 || !slices.Contains(blocked.Blockers, "child_units") || !slices.Contains(blocked.Blockers, "direct_assignments") {
		t.Fatalf("nonempty delete report: %+v", blocked)
	}
	if err := f.repo.DeleteUnit(ctx, f.alice, f.units["Payments"]); !errors.Is(err, rbac.ErrNotEmpty) {
		t.Fatal("nonempty unit deleted", err)
	}
	if err := f.repo.DeleteUnit(ctx, f.alice, rbac.RootScope().ID); !errors.Is(err, rbac.ErrRootProtected) {
		t.Fatal("root deleted", err)
	}
	leaf, err := f.repo.CreateUnit(ctx, f.alice, f.units["Mobile"], rbac.UnitInput{Name: "Empty"})
	if err != nil {
		t.Fatal(err)
	}
	auditCount = countRows(t, f.db, "audit_log")
	deletion, err := f.repo.DryRunDelete(ctx, f.alice, leaf.ID)
	if err != nil || !deletion.Executable || len(deletion.Connections) != 0 {
		t.Fatal("empty delete preview failed", err)
	}
	if len(findChange(deletion, id("bob"), rbac.UnitScope(leaf.ID)).Lost) == 0 || countRows(t, f.db, "audit_log") != auditCount {
		t.Fatal("empty deletion impact missing or mutated audit")
	}
	if err := f.repo.DeleteUnit(ctx, f.alice, leaf.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.Unit(ctx, leaf.ID); !errors.Is(err, rbac.ErrNotFound) {
		t.Fatal("empty unit still exists")
	}
	connected, err := f.repo.CreateUnit(ctx, f.alice, f.units["Mobile"], rbac.UnitInput{Name: "Connected"})
	if err != nil {
		t.Fatal(err)
	}
	f.connections[id("ConnectedDB")] = connected.ID
	if err := f.repo.DeleteUnit(ctx, f.alice, connected.ID); !errors.Is(err, rbac.ErrNotEmpty) {
		t.Fatal("connection owner deleted", err)
	}
	assigned, err := f.repo.CreateUnit(ctx, f.alice, f.units["Mobile"], rbac.UnitInput{Name: "Assigned"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.Assign(ctx, f.alice, id("frank"), uuid.MustParse(rbac.ViewerRoleID), rbac.UnitScope(assigned.ID), ""); err != nil {
		t.Fatal(err)
	}
	if err := f.repo.DeleteUnit(ctx, f.alice, assigned.ID); !errors.Is(err, rbac.ErrNotEmpty) {
		t.Fatal("assigned unit deleted", err)
	}
	if _, err := f.repo.DryRunDelete(ctx, id("carol"), f.units["Data"]); !errors.Is(err, rbac.ErrForbidden) {
		t.Fatal("unauthorized dry run disclosed data", err)
	}
}
func TestDatabaseGuardsAndTransactionalAudit(t *testing.T) {
	f := newStoreFixture(t)
	ctx := context.Background()
	for _, statement := range []string{
		`DELETE FROM units WHERE parent_id IS NULL`,
		`INSERT INTO units(name) VALUES('Second root')`,
		`UPDATE units SET ancestors=ARRAY[]::uuid[] WHERE name='Mobile'`,
		`UPDATE units SET parent_id=(SELECT id FROM units WHERE name='Data'),ancestors=ARRAY['00000000-0000-4000-8000-000000000001'::uuid,(SELECT id FROM units WHERE name='Data')] WHERE name='Engineering'`,
		`UPDATE roles SET name='Changed' WHERE is_system`,
		`DELETE FROM role_permissions WHERE role_id='00000000-0000-4000-8000-000000000101' AND permission='unit:read'`,
		`DELETE FROM assignments WHERE role_id='00000000-0000-4000-8000-000000000101'`,
		`UPDATE users SET is_active=false WHERE email='alice@example.test'`,
	} {
		if _, err := f.db.Exec(statement); err == nil {
			t.Fatalf("database guard failed: %s", statement)
		}
	}
	if _, err := f.repo.AuditLog(ctx, id("eve"), 100); !errors.Is(err, rbac.ErrForbidden) {
		t.Fatal("Viewer read audit log", err)
	}
	for _, roleID := range []string{rbac.AdministratorRoleID, rbac.ManagerRoleID, rbac.DeveloperRoleID, rbac.ViewerRoleID} {
		role, err := f.repo.Role(ctx, uuid.MustParse(roleID))
		if err != nil || !role.IsSystem {
			t.Fatal("system seed missing", roleID, err)
		}

		expected := []rbac.Permission{}
		for _, permission := range rbac.Permissions() {
			switch roleID {
			case rbac.AdministratorRoleID:
				expected = append(expected, permission)
			case rbac.ManagerRoleID:
				if !permission.SystemLevel() {
					expected = append(expected, permission)
				}
			case rbac.DeveloperRoleID:
				if !permission.SystemLevel() && permission != rbac.UnitManageMembers {
					expected = append(expected, permission)
				}
			case rbac.ViewerRoleID:
				if permission == rbac.UnitRead || permission == rbac.ConnectionRead || permission == rbac.SchemaRead {
					expected = append(expected, permission)
				}
			}
		}
		slices.Sort(expected)
		slices.Sort(role.Permissions)
		if !slices.Equal(expected, role.Permissions) {
			t.Fatalf("seed permission mismatch for %s: %v", roleID, role.Permissions)
		}
	}
	unit, err := f.repo.UpdateUnit(ctx, f.alice, f.units["Payments"], rbac.UnitInput{Name: "Payments", Description: "Updated synthetic description", Kind: "project"})
	if err != nil || unit.Kind != "project" {
		t.Fatal(err)
	}
	role, err := f.repo.CreateRole(ctx, f.alice, rbac.RoleInput{Name: "AuditProbe", Permissions: []rbac.Permission{rbac.UnitRead}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.UpdateRole(ctx, f.alice, role.ID, rbac.RoleInput{Name: "AuditProbeUpdated", Permissions: []rbac.Permission{rbac.UnitRead}}); err != nil {
		t.Fatal(err)
	}
	assignment, err := f.repo.Assign(ctx, f.alice, id("frank"), role.ID, f.scope("Payments"), "")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repo.Revoke(ctx, f.alice, assignment.ID); err != nil {
		t.Fatal(err)
	}
	if err := f.repo.DeleteRole(ctx, f.alice, role.ID); err != nil {
		t.Fatal(err)
	}
	if err := f.repo.SetUserActive(ctx, f.alice, id("frank"), false); err != nil {
		t.Fatal(err)
	}
	if err := f.repo.SetUserActive(ctx, f.alice, id("frank"), true); err != nil {
		t.Fatal(err)
	}
	audit, err := f.repo.AuditLog(ctx, f.alice, 200)
	if err != nil {
		t.Fatal(err)
	}
	actions := map[string]bool{}
	for _, row := range audit {
		actions[row.Action] = true
		if row.ID == uuid.Nil || row.CreatedAt.IsZero() || row.Details == nil {
			t.Fatal("incomplete audit row")
		}
	}
	for _, action := range []string{"unit.create", "unit.update", "assignment.create", "assignment.revoke", "role.create", "role.update", "role.delete", "user.set_active"} {
		if !actions[action] {
			t.Fatal("missing audit", action)
		}
	}
	unitsBefore, assignmentsBefore := countRows(t, f.db, "units"), countRows(t, f.db, "assignments")
	if _, err := f.db.Exec(`ALTER TABLE audit_log RENAME TO unavailable_audit_log`); err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.CreateUnit(ctx, f.alice, f.units["Payments"], rbac.UnitInput{Name: "Rollback"}); err == nil {
		t.Fatal("unaudited unit creation committed")
	}
	if _, err := f.repo.Assign(ctx, f.alice, id("frank"), uuid.MustParse(rbac.ViewerRoleID), f.scope("Payments"), ""); err == nil {
		t.Fatal("unaudited assignment committed")
	}
	if err := f.repo.SetUserActive(ctx, f.alice, id("frank"), false); err == nil {
		t.Fatal("unaudited deactivation committed")
	}
	if unitsBefore != countRows(t, f.db, "units") || assignmentsBefore != countRows(t, f.db, "assignments") {
		t.Fatal("audit failure did not roll back mutation")
	}
	var active bool
	if err := f.db.QueryRow(`SELECT is_active FROM users WHERE id=$1`, id("frank")).Scan(&active); err != nil || !active {
		t.Fatal("deactivation survived audit rollback", err)
	}
}

func TestLegacyRoleNameConflictRollsBackMigration(t *testing.T) {
	db := isolatedDB(t)
	ctx := context.Background()
	provider, err := database.Provider(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO roles(name) VALUES('Manager')`); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, db); err == nil {
		t.Fatal("legacy custom Manager was silently promoted")
	}
	var oldColumn bool
	if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='users' AND column_name='role_id')`).Scan(&oldColumn); err != nil || !oldColumn {
		t.Fatal("failed migration did not restore legacy schema", err)
	}
	if countRows(t, db, "roles") != 2 || countRows(t, db, "audit_events") != 0 {
		t.Fatal("failed migration changed old metadata")
	}
}
func TestDefaultConnectionResolverRejectsMissingRow(t *testing.T) {
	f := newStoreFixture(t)
	repo := rbac.NewRepository(f.db, zap.NewNop(), nil)
	ctx := context.Background()
	scope := rbac.Scope{Type: rbac.ScopeConnection, ID: id("missing-connection")}
	if allowed, err := repo.Engine().Can(ctx, f.alice, rbac.ConnectionRead, scope); allowed || !errors.Is(err, rbac.ErrNotFound) {
		t.Fatal("missing connection must fail closed", allowed, err)
	}
	if _, err := repo.Assign(ctx, f.alice, id("frank"), uuid.MustParse(rbac.ViewerRoleID), scope, ""); !errors.Is(err, rbac.ErrNotFound) {
		t.Fatal("assignment to missing connection", err)
	}
}

func TestAuthorizationNeverCombinesDifferentCommittedStates(t *testing.T) {
	f := newStoreFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	assignment, err := f.repo.Assign(ctx, f.alice, id("frank"), uuid.MustParse(rbac.DeveloperRoleID), f.scope("Data"), "")
	if err != nil {
		t.Fatal(err)
	}
	// In both committed states Frank's assignment is in the other subtree:
	// Payments@Engineering + Developer@Data, or Payments@Data + Developer@Engineering.
	// Mixing a path from one state with assignments from the other would grant edit.
	writer := make(chan error, 1)
	go func() {
		for i := 0; i < 100; i++ {
			parent, scope := f.units["Data"], f.units["Engineering"]
			if i%2 == 1 {
				parent, scope = scope, parent
			}
			tx, err := f.db.BeginTx(ctx, nil)
			if err != nil {
				writer <- err
				return
			}
			if err = rbac.Lock(ctx, tx); err == nil {
				_, err = tx.ExecContext(ctx, `UPDATE units SET parent_id=$2,ancestors=ARRAY[$3::uuid,$2::uuid] WHERE id=$1`, f.units["Payments"], parent, rbac.RootScope().ID)
			}
			if err == nil {
				_, err = tx.ExecContext(ctx, `UPDATE assignments SET scope_id=$2 WHERE id=$1`, assignment.ID, scope)
			}
			if err == nil {
				err = tx.Commit()
			} else {
				_ = tx.Rollback()
			}
			if err != nil {
				writer <- err
				return
			}
		}
		writer <- nil
	}()
	engine := f.repo.Engine()
	for i := 0; i < 200; i++ {
		allowed, err := engine.Can(ctx, id("frank"), rbac.DocsEdit, f.scope("Payments"))
		if err != nil || allowed {
			cancel()
			<-writer
			t.Fatalf("mixed authorization snapshot granted access: %v %v", allowed, err)
		}
	}
	if err := <-writer; err != nil {
		t.Fatal(err)
	}
}
