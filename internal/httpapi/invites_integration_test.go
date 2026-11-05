// SPDX-License-Identifier: Apache-2.0
package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	api "github.com/Thitipong-PP/schewise/internal/apicontract"
	"github.com/Thitipong-PP/schewise/internal/auth"
	"github.com/Thitipong-PP/schewise/internal/httpapi"
	"github.com/Thitipong-PP/schewise/internal/onetimetoken"
	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

func inviteToken(value auth.InviteLink) string {
	parts := strings.Split(value.Link, "/invite/")
	return parts[len(parts)-1]
}
func createInvite(t *testing.T, f *fixture, email string) auth.InviteLink {
	t.Helper()
	cookie, csrf, _ := f.login(t, "admin@example.test")
	got := f.request(t, "POST", "/api/v1/invites", auth.InviteInput{Email: email, RoleID: uuid.MustParse(rbac.ViewerRoleID), Scope: rbac.RootScope(), Note: "Synthetic invite"}, cookie, csrf, testOrigin)
	expectStatus(t, got, 201)
	return decode[auth.InviteLink](t, got)
}
func TestInviteLifecycleAndPrivacy(t *testing.T) {
	f := newFixture(t)
	cookie, csrf, admin := f.login(t, "admin@example.test")
	input := auth.InviteInput{Email: "  Teammate@Example.test ", RoleID: uuid.MustParse(rbac.ViewerRoleID), Scope: rbac.RootScope(), Note: "Synthetic invitation"}
	expectStatus(t, f.request(t, "POST", "/api/v1/invites", input, cookie, "", testOrigin), 403)
	created := f.request(t, "POST", "/api/v1/invites", input, cookie, csrf, testOrigin)
	expectStatus(t, created, 201)
	value := decode[auth.InviteLink](t, created)
	token := inviteToken(value)
	if value.Invite.Email != "teammate@example.test" || !strings.HasPrefix(value.Link, testOrigin+"/invite/") || !onetimetoken.Valid(token) {
		t.Fatal("invalid link or email")
	}
	if ttl := time.Until(value.Invite.ExpiresAt); ttl < 167*time.Hour || ttl > 169*time.Hour {
		t.Fatal("wrong default expiry")
	}
	var digest []byte
	if err := f.db.QueryRow(`SELECT token_hash FROM invites WHERE id=$1`, value.Invite.ID).Scan(&digest); err != nil || !onetimetoken.Matches(token, digest) {
		t.Fatal("token not stored as a digest", err)
	}
	listed := f.request(t, "GET", "/api/v1/invites", nil, cookie, "", "")
	expectStatus(t, listed, 200)
	if bytes.Contains(listed.body, []byte(token)) || bytes.Contains(listed.body, []byte("token_hash")) || bytes.Contains(listed.body, []byte("\"link\"")) {
		t.Fatal("list leaked token")
	}
	public := f.request(t, "GET", "/api/v1/invites/"+token, nil, nil, "", "")
	expectStatus(t, public, 200)
	display := decode[auth.InviteDisplay](t, public)
	if display.Email != value.Invite.Email || display.RoleName != "Viewer" || display.ScopeName != "Organization" {
		t.Fatal("incorrect invite display")
	}
	if bytes.Contains(public.body, []byte("note")) || bytes.Contains(public.body, []byte(admin.ID)) {
		t.Fatal("public inspection leaked private metadata")
	}
	previous, acceptCSRF := f.csrf(t, nil)
	accept := api.AcceptInviteRequest{DisplayName: "Synthetic Teammate", Password: testPassword}
	expectStatus(t, f.request(t, "POST", "/api/v1/invites/"+token+"/accept", accept, previous, acceptCSRF, "https://attacker.example"), 403)
	accepted := f.request(t, "POST", "/api/v1/invites/"+token+"/accept", accept, previous, acceptCSRF, testOrigin)
	expectStatus(t, accepted, 201)
	identity := decode[api.IdentityResponse](t, accepted).User
	if identity.Email != value.Invite.Email || identity.DisplayName != accept.DisplayName || accepted.cookie == nil || accepted.cookie.Value == previous.Value || !accepted.cookie.Secure || !accepted.cookie.HttpOnly {
		t.Fatal("account/session creation failed")
	}
	me := f.request(t, "GET", "/api/v1/auth/me", nil, accepted.cookie, "", "")
	expectStatus(t, me, 200)
	if decode[api.IdentityResponse](t, me).User.DisplayName != accept.DisplayName {
		t.Fatal("display name not persisted")
	}
	expectStatus(t, f.request(t, "GET", "/api/v1/auth/me", nil, previous, "", ""), 401)
	var role, creator, user, storedHash string
	if err := f.db.QueryRow(`SELECT a.role_id,a.created_by,i.used_by_user_id,u.password_hash FROM invites i JOIN users u ON u.id=i.used_by_user_id JOIN assignments a ON a.user_id=u.id WHERE i.id=$1 AND i.used_at IS NOT NULL`, value.Invite.ID).Scan(&role, &creator, &user, &storedHash); err != nil || role != rbac.ViewerRoleID || creator != admin.ID || user != identity.ID || !auth.VerifyPassword(testPassword, storedHash) {
		t.Fatal("non-atomic or incorrect acceptance", err)
	}
	for _, action := range []string{"invite.created", "invite.accepted", "assignment.create"} {
		var count int
		if err := f.db.QueryRow(`SELECT count(*) FROM audit_log WHERE action=$1`, action).Scan(&count); err != nil || count == 0 {
			t.Fatal("missing audit", action, err)
		}
	}
	var audit string
	if err := f.db.QueryRow(`SELECT string_agg(details::text,'') FROM audit_log`).Scan(&audit); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(audit, token) || strings.Contains(audit, testPassword) {
		t.Fatal("audit leaked secrets")
	}
	for _, entry := range f.logs.All() {
		encoded, _ := json.Marshal(entry.ContextMap())
		if strings.Contains(string(encoded), token) || strings.Contains(string(encoded), testPassword) {
			t.Fatal("Zap leaked secrets")
		}
	}
}

func TestInviteCreationAndDelegation(t *testing.T) {
	f := newAdminFixture(t)
	admin, csrf, _ := f.login(t, "admin@example.test")
	input := auth.InviteInput{Email: "new@example.test", RoleID: uuid.MustParse(rbac.ViewerRoleID), Scope: rbac.UnitScope(f.units["Payments"])}
	manager, managerCSRF, _ := f.login(t, "carol@example.test")
	expectStatus(t, f.request(t, "POST", "/api/v1/invites", input, manager, managerCSRF, testOrigin), 403)
	expectStatus(t, f.request(t, "GET", "/api/v1/invites", nil, manager, "", ""), 403)
	for _, role := range []string{rbac.AdministratorRoleID, uuid.NewString()} {
		input.RoleID = uuid.MustParse(role)
		got := f.request(t, "POST", "/api/v1/invites", input, admin, csrf, testOrigin)
		if got.status != 400 && got.status != 404 {
			t.Fatal("invalid role/scope accepted", got.status)
		}
	}
	input.RoleID = uuid.MustParse(rbac.ViewerRoleID)
	input.Scope = rbac.UnitScope(uuid.New())
	expectStatus(t, f.request(t, "POST", "/api/v1/invites", input, admin, csrf, testOrigin), 404)
	input.Scope = rbac.UnitScope(f.units["Payments"])
	role, err := f.repo.CreateRole(context.Background(), f.alice, rbac.RoleInput{Name: "Account operator", Permissions: []rbac.Permission{rbac.UserManage, rbac.UnitRead}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.Assign(context.Background(), f.alice, f.users["frank"], role.ID, rbac.RootScope(), ""); err != nil {
		t.Fatal(err)
	}
	limited, limitedCSRF, _ := f.login(t, "frank@example.test")
	expectStatus(t, f.request(t, "GET", "/api/v1/invites", nil, limited, "", ""), 200)
	expectStatus(t, f.request(t, "POST", "/api/v1/invites", input, limited, limitedCSRF, testOrigin), 403)
	created := f.request(t, "POST", "/api/v1/invites", input, admin, csrf, testOrigin)
	expectStatus(t, created, 201)
	input.Email = "NEW@EXAMPLE.TEST"
	expectStatus(t, f.request(t, "POST", "/api/v1/invites", input, admin, csrf, testOrigin), 409)
	input.Email = "bob@example.test"
	existing := f.request(t, "POST", "/api/v1/invites", input, admin, csrf, testOrigin)
	expectStatus(t, existing, 409)
	if decode[api.Error](t, existing).Code != "account_exists" {
		t.Fatal("admin not told to assign directly")
	}
	// A manager cannot revoke or regenerate even if the role is delegable there.
	id := decode[auth.InviteLink](t, created).Invite.ID.String()
	for _, action := range []string{"revoke", "regenerate"} {
		expectStatus(t, f.request(t, "POST", "/api/v1/invites/"+id+"/"+action, map[string]any{}, manager, managerCSRF, testOrigin), 403)
	}
}

func TestUnavailableInvitesShareOneError(t *testing.T) {
	f := newFixture(t)
	values := []auth.InviteLink{createInvite(t, f, "expired@example.test"), createInvite(t, f, "revoked@example.test"), createInvite(t, f, "used@example.test")}
	if _, err := f.db.Exec(`UPDATE invites SET expires_at=now()-interval '1 second' WHERE id=$1`, values[0].Invite.ID); err != nil {
		t.Fatal(err)
	}
	_, _, admin := f.login(t, "admin@example.test")
	if err := f.service.RevokeInvite(context.Background(), &admin, values[1].Invite.ID); err != nil {
		t.Fatal(err)
	}
	previous, err := f.service.Anonymous(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AcceptInvite(context.Background(), inviteToken(values[2]), "Synthetic used", testPassword, previous, auth.Event{}); err != nil {
		t.Fatal(err)
	}
	unknown, err := onetimetoken.Generate()
	if err != nil {
		t.Fatal(err)
	}
	var expected []byte
	cookie, csrf := f.csrf(t, nil)
	for _, token := range []string{inviteToken(values[0]), inviteToken(values[1]), inviteToken(values[2]), unknown, "wrong-token"} {
		get := f.request(t, "GET", "/api/v1/invites/"+token, nil, nil, "", "")
		expectStatus(t, get, 400)
		post := f.request(t, "POST", "/api/v1/invites/"+token+"/accept", api.AcceptInviteRequest{DisplayName: "Synthetic", Password: testPassword}, cookie, csrf, testOrigin)
		expectStatus(t, post, 400)
		if expected == nil {
			expected = get.body
		}
		if !bytes.Equal(expected, get.body) || !bytes.Equal(expected, post.body) {
			t.Fatal("token status disclosed")
		}
	}
}

func TestInviteRegenerateRevokeAndPagination(t *testing.T) {
	f := newFixture(t)
	value := createInvite(t, f, "regenerate@example.test")
	cookie, csrf, _ := f.login(t, "admin@example.test")
	regenerated := f.request(t, "POST", "/api/v1/invites/"+value.Invite.ID.String()+"/regenerate", map[string]any{}, cookie, csrf, testOrigin)
	expectStatus(t, regenerated, 201)
	next := decode[auth.InviteLink](t, regenerated)
	if next.Link == value.Link || next.Invite.ID == value.Invite.ID {
		t.Fatal("regeneration reused token/record")
	}
	expectStatus(t, f.request(t, "GET", "/api/v1/invites/"+inviteToken(value), nil, nil, "", ""), 400)
	expectStatus(t, f.request(t, "GET", "/api/v1/invites/"+inviteToken(next), nil, nil, "", ""), 200)
	page := decode[auth.InvitePage](t, f.request(t, "GET", "/api/v1/invites?limit=1", nil, cookie, "", ""))
	if len(page.Invites) != 1 || page.NextBefore == nil {
		t.Fatal("missing pagination")
	}
	last := decode[auth.InvitePage](t, f.request(t, "GET", "/api/v1/invites?limit=1&before="+page.NextBefore.String(), nil, cookie, "", ""))
	if len(last.Invites) != 1 || last.NextBefore != nil || last.Invites[0].ID == page.Invites[0].ID {
		t.Fatal("incorrect cursor")
	}
	expectStatus(t, f.request(t, "POST", "/api/v1/invites/"+next.Invite.ID.String()+"/revoke", map[string]any{}, cookie, csrf, testOrigin), 204)
	expectStatus(t, f.request(t, "GET", "/api/v1/invites/"+inviteToken(next), nil, nil, "", ""), 400)
	for _, action := range []string{"invite.revoked", "invite.regenerated"} {
		var n int
		if err := f.db.QueryRow(`SELECT count(*) FROM audit_log WHERE action=$1`, action).Scan(&n); err != nil || n == 0 {
			t.Fatal("missing invite audit", err)
		}
	}
}

func TestInviteRevalidationAndAtomicity(t *testing.T) {
	for _, cause := range []string{"scope_deleted", "role_deleted", "invalid_pair", "inviter_lost_authority", "session_insert_failed"} {
		t.Run(cause, func(t *testing.T) {
			f := newAdminFixture(t)
			ctx := context.Background()
			_, _, admin := f.login(t, "admin@example.test")
			unit, err := f.repo.CreateUnit(ctx, f.alice, f.units["root"], rbac.UnitInput{Name: "Temporary invite scope"})
			if err != nil {
				t.Fatal(err)
			}
			role, err := f.repo.CreateRole(ctx, f.alice, rbac.RoleInput{Name: "Invite viewer", Permissions: []rbac.Permission{rbac.UnitRead}})
			if err != nil {
				t.Fatal(err)
			}
			value, err := f.service.CreateInvite(ctx, &admin, auth.InviteInput{Email: "atomic@example.test", RoleID: role.ID, Scope: rbac.UnitScope(unit.ID)})
			if err != nil {
				t.Fatal(err)
			}
			previous, err := f.service.Anonymous(ctx)
			if err != nil {
				t.Fatal(err)
			}
			switch cause {
			case "scope_deleted":
				err = f.repo.DeleteUnit(ctx, f.alice, unit.ID)
			case "role_deleted":
				err = f.repo.DeleteRole(ctx, f.alice, role.ID)
			case "invalid_pair":
				_, err = f.repo.UpdateRole(ctx, f.alice, role.ID, rbac.RoleInput{Name: role.Name, Permissions: []rbac.Permission{rbac.UnitRead, rbac.UserManage}})
			case "inviter_lost_authority":
				// Preserve another active Administrator, then remove the issuer's grant.
				_, err = f.repo.Assign(ctx, f.alice, f.users["frank"], uuid.MustParse(rbac.AdministratorRoleID), rbac.RootScope(), "")
				if err == nil {
					var grants []rbac.Assignment
					grants, err = f.repo.Assignments(ctx, f.alice)
					if err == nil {
						err = f.repo.Revoke(ctx, f.users["frank"], grants[0].ID)
					}
				}
			case "session_insert_failed":
				_, err = f.db.Exec(`ALTER TABLE sessions ADD CONSTRAINT synthetic_no_authenticated_session CHECK (user_id IS NULL) NOT VALID`)
			}
			if err != nil {
				t.Fatal(err)
			}
			_, err = f.service.AcceptInvite(ctx, inviteToken(value), "Synthetic Atomic", testPassword, previous, auth.Event{})
			if err == nil {
				t.Fatal("stale or failed invite accepted")
			}
			if cause != "session_insert_failed" && !errors.Is(err, auth.ErrInviteInvalid) {
				t.Fatal("stale invite did not return generic error", err)
			}
			var users, grants int
			var used bool
			if err := f.db.QueryRow(`SELECT count(*) FROM users WHERE email='atomic@example.test'`).Scan(&users); err != nil {
				t.Fatal(err)
			}
			if err := f.db.QueryRow(`SELECT count(*) FROM assignments WHERE role_id=$1 AND scope_id=$2`, role.ID, unit.ID).Scan(&grants); err != nil {
				t.Fatal(err)
			}
			if err := f.db.QueryRow(`SELECT used_at IS NOT NULL FROM invites WHERE id=$1`, value.Invite.ID).Scan(&used); err != nil {
				t.Fatal(err)
			}
			if users != 0 || grants != 0 || used {
				t.Fatal("partial acceptance escaped rollback")
			}
			if _, err := f.service.Lookup(ctx, previous.Token); err != nil {
				t.Fatal("rollback consumed pre-accept session", err)
			}
		})
	}
}

func TestBootstrapOnlyRegistration(t *testing.T) {
	f := newAuthFixture(t, false)
	cookie, csrf := f.csrf(t, nil)
	input := map[string]string{"email": "first@example.test", "password": testPassword}
	created := f.request(t, "POST", "/api/v1/auth/register", input, cookie, csrf, testOrigin)
	expectStatus(t, created, 201)
	user := decode[api.IdentityResponse](t, created).User
	allowed, err := f.service.Can(context.Background(), &user, rbac.UserManage, rbac.RootScope())
	if err != nil || !allowed {
		t.Fatal("first user not root Administrator", err)
	}
	for _, email := range []string{"first@example.test", "other@example.test"} {
		input["email"] = email
		closed := f.request(t, "POST", "/api/v1/auth/register", input, cookie, csrf, testOrigin)
		expectStatus(t, closed, 403)
		if decode[api.Error](t, closed).Error != "Registration is unavailable. Use an invitation." {
			t.Fatal("non-generic closed registration")
		}
	}
	var n int
	if err := f.db.QueryRow(`SELECT count(*) FROM users`).Scan(&n); err != nil || n != 1 {
		t.Fatal("registration created extra account", err)
	}
}

func TestInvitePasswordPolicy(t *testing.T) {
	f := newFixture(t)
	value := createInvite(t, f, "policy@example.test")
	cookie, csrf := f.csrf(t, nil)
	for _, password := range []string{"short", strings.Repeat("a", 129), "Password1234", "123456789012"} {
		expectStatus(t, f.request(t, "POST", "/api/v1/invites/"+inviteToken(value)+"/accept", api.AcceptInviteRequest{DisplayName: "Policy test", Password: password}, cookie, csrf, testOrigin), 400)
	}
	expectStatus(t, f.request(t, "POST", "/api/v1/invites/"+inviteToken(value)+"/accept", map[string]string{"display_name": "Policy test", "password": testPassword, "email": "changed@example.test"}, cookie, csrf, testOrigin), 400)
	expectStatus(t, f.request(t, "POST", "/api/v1/invites/"+inviteToken(value)+"/accept", api.AcceptInviteRequest{DisplayName: "Policy test", Password: "long passphrase without composition rules"}, cookie, csrf, testOrigin), 201)
}

func TestInviteRateLimits(t *testing.T) {
	f := newFixture(t)
	for i := 0; i < 20; i++ {
		expectStatus(t, f.request(t, "GET", "/api/v1/invites/wrong", nil, nil, "", ""), 400)
	}
	expectStatus(t, f.request(t, "GET", "/api/v1/invites/wrong", nil, nil, "", ""), 429)
	// A separate application instance has fresh per-process buckets. Distinct IPs
	// avoid the per-IP cap and exercise the shared GET/POST global budget.
	app := httpapi.NewApp(httpapi.Config{Auth: f.service, RBAC: rbac.NewRepository(f.db, zap.NewNop(), nil), Origin: testOrigin})
	handler := app.Handler()
	for i := 0; i < 201; i++ {
		var request fasthttp.Request
		request.SetRequestURI("http://localhost/api/v1/invites/wrong")
		request.Header.SetMethod("GET")
		var ctx fasthttp.RequestCtx
		ctx.Init(&request, &net.TCPAddr{IP: net.ParseIP(fmt.Sprintf("192.0.2.%d", i+1)), Port: 1234}, nil)
		handler(&ctx)
		want := 400
		if i == 200 {
			want = 429
		}
		if ctx.Response.StatusCode() != want {
			t.Fatalf("global attempt %d = %d, want %d", i, ctx.Response.StatusCode(), want)
		}
	}

}

func TestConcurrentInviteCreationAndAcceptance(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	_, _, admin := f.login(t, "admin@example.test")
	var wg sync.WaitGroup
	values := make(chan auth.InviteLink, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Go(func() {
			value, err := f.service.CreateInvite(ctx, &admin, auth.InviteInput{Email: "race@example.test", RoleID: uuid.MustParse(rbac.ViewerRoleID), Scope: rbac.RootScope()})
			if err != nil {
				errs <- err
			} else {
				values <- value
			}
		})
	}
	wg.Wait()
	close(values)
	close(errs)
	if len(values) != 1 || len(errs) != 1 {
		t.Fatal("concurrent duplicate invites created")
	}
	if err := <-errs; !errors.Is(err, auth.ErrInviteConflict) {
		t.Fatal(err)
	}
	value := <-values
	errResults := make(chan error, 2)
	for i := 0; i < 2; i++ {
		previous, err := f.service.Anonymous(ctx)
		if err != nil {
			t.Fatal(err)
		}
		wg.Go(func() {
			_, err := f.service.AcceptInvite(ctx, inviteToken(value), "Synthetic race", testPassword, previous, auth.Event{})
			errResults <- err
		})
	}
	wg.Wait()
	close(errResults)
	success, invalid := 0, 0
	for err := range errResults {
		if err == nil {
			success++
		} else if errors.Is(err, auth.ErrInviteInvalid) {
			invalid++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || invalid != 1 {
		t.Fatal("invite was not consumed once")
	}
}
