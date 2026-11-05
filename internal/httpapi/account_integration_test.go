// SPDX-License-Identifier: Apache-2.0

package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	api "github.com/Thitipong-PP/schewise/internal/apicontract"
	"github.com/Thitipong-PP/schewise/internal/auth"
	"github.com/Thitipong-PP/schewise/internal/config"
	"github.com/Thitipong-PP/schewise/internal/database"
	"github.com/Thitipong-PP/schewise/internal/onetimetoken"
	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const changedPassword = "Synthetic-updated-password-43!"

func TestAccountEndpointsRequireAuthentication(t *testing.T) {
	f := newFixture(t)
	for _, op := range api.Operations() {
		if !op.Self {
			continue
		}
		t.Run(op.ID, func(t *testing.T) {
			path := strings.ReplaceAll(op.Path, "{id}", uuid.NewString())
			var input any
			if op.Request != nil {
				input = map[string]any{}
			}
			expectStatus(t, f.request(t, op.Method, path, input, nil, "", testOrigin), 401)
		})
	}
}

func TestProfileAndSessionOwnership(t *testing.T) {
	f := newAdminFixture(t)
	member, csrf, bob := f.login(t, "bob@example.test")
	admin, _, _ := f.login(t, "admin@example.test")
	profile := f.request(t, "GET", "/api/v1/me/profile", nil, member, "", "")
	expectStatus(t, profile, 200)
	if decode[api.IdentityResponse](t, profile).User.ID != bob.ID {
		t.Fatal("wrong profile")
	}
	updated := f.request(t, "PATCH", "/api/v1/me/profile", api.ProfileRequest{DisplayName: "  Synthetic Bob  "}, member, csrf, testOrigin)
	expectStatus(t, updated, 200)
	if decode[api.IdentityResponse](t, updated).User.DisplayName != "Synthetic Bob" {
		t.Fatal("display name not normalized")
	}
	expectStatus(t, f.request(t, "PATCH", "/api/v1/me/profile", map[string]string{"display_name": "Hacked", "email": "changed@example.test"}, member, csrf, testOrigin), 400)
	expectStatus(t, f.request(t, "PATCH", "/api/v1/me/profile", api.ProfileRequest{DisplayName: " "}, member, csrf, testOrigin), 400)
	adminSession, err := f.service.Lookup(context.Background(), admin.Value)
	if err != nil {
		t.Fatal(err)
	}
	listing := f.request(t, "GET", "/api/v1/me/sessions?user_id="+f.alice.String(), nil, member, "", "")
	expectStatus(t, listing, 200)
	sessions := decode[auth.SessionList](t, listing)
	if len(sessions.Sessions) != 1 || !sessions.Sessions[0].Current || sessions.Sessions[0].ID == adminSession.ID {
		t.Fatal("other user's sessions were disclosed")
	}
	if bytes.Contains(listing.body, []byte("token_hash")) || bytes.Contains(listing.body, []byte("csrf")) || bytes.Contains(listing.body, []byte(member.Value)) {
		t.Fatal("session list contains bearer secrets")
	}
	victim := f.request(t, "DELETE", "/api/v1/me/sessions/"+adminSession.ID.String(), nil, member, csrf, testOrigin)
	missing := f.request(t, "DELETE", "/api/v1/me/sessions/"+uuid.NewString(), nil, member, csrf, testOrigin)
	expectStatus(t, victim, 404)
	expectStatus(t, missing, 404)
	if !bytes.Equal(victim.body, missing.body) {
		t.Fatal("session ownership leaked")
	}
	expectStatus(t, f.request(t, "GET", "/api/v1/auth/me", nil, admin, "", ""), 200)
	self := sessions.Sessions[0].ID
	revoked := f.request(t, "DELETE", "/api/v1/me/sessions/"+self.String(), nil, member, csrf, testOrigin)
	expectStatus(t, revoked, 204)
	if revoked.cookie == nil || revoked.cookie.MaxAge != -1 {
		t.Fatal("current session cookie not cleared")
	}
	expectStatus(t, f.request(t, "GET", "/api/v1/auth/me", nil, member, "", ""), 401)
}

func TestChangePasswordKeepsCurrentAndRevokesOthers(t *testing.T) {
	f := newAdminFixture(t)
	cookie, csrf, user := f.login(t, "bob@example.test")
	other, _, _ := f.login(t, user.Email)
	admin, adminCSRF, _ := f.login(t, "admin@example.test")
	pending := createReset(t, f.fixture, user.ID, admin, adminCSRF)
	request := api.PasswordChangeRequest{CurrentPassword: "Wrong-password-42!", Password: changedPassword}
	wrong := f.request(t, "POST", "/api/v1/me/password", request, cookie, csrf, testOrigin)
	expectStatus(t, wrong, 400)
	if decode[api.Error](t, wrong).Code != "invalid_current_password" {
		t.Fatal("wrong current password not rejected explicitly")
	}
	expectStatus(t, f.request(t, "GET", "/api/v1/auth/me", nil, other, "", ""), 200)
	request.CurrentPassword = testPassword
	request.Password = "Password1234"
	expectStatus(t, f.request(t, "POST", "/api/v1/me/password", request, cookie, csrf, testOrigin), 400)
	request.Password = changedPassword
	expectStatus(t, f.request(t, "POST", "/api/v1/me/password", request, cookie, csrf, testOrigin), 204)
	expectStatus(t, f.request(t, "GET", "/api/v1/reset-password/"+resetToken(pending), nil, nil, "", ""), 400)
	expectStatus(t, f.request(t, "GET", "/api/v1/auth/me", nil, cookie, "", ""), 200)
	expectStatus(t, f.request(t, "GET", "/api/v1/auth/me", nil, other, "", ""), 401)
	anonymous, token := f.csrf(t, nil)
	expectStatus(t, f.request(t, "POST", "/api/v1/auth/login", map[string]string{"email": user.Email, "password": testPassword}, anonymous, token, testOrigin), 401)
	expectStatus(t, f.request(t, "POST", "/api/v1/auth/login", map[string]string{"email": user.Email, "password": changedPassword}, anonymous, token, testOrigin), 200)
	assertAudit(t, f.fixture, "user.password_changed", "session.revoke_others", "user.password_change_failed")
}

func TestPasswordChangeRateLimit(t *testing.T) {
	f := newFixture(t)
	cookie, csrf, _ := f.login(t, "admin@example.test")
	input := api.PasswordChangeRequest{CurrentPassword: "wrong", Password: changedPassword}
	for i := 0; i < 5; i++ {
		expectStatus(t, f.request(t, "POST", "/api/v1/me/password", input, cookie, csrf, testOrigin), 400)
	}
	expectStatus(t, f.request(t, "POST", "/api/v1/me/password", input, cookie, csrf, testOrigin), 429)
	expectStatus(t, f.request(t, "GET", "/api/v1/me/profile", nil, cookie, "", ""), 200)
}

func TestRevokeOtherSessionsIsScoped(t *testing.T) {
	f := newAdminFixture(t)
	cookie, csrf, _ := f.login(t, "bob@example.test")
	other, _, _ := f.login(t, "bob@example.test")
	admin, _, _ := f.login(t, "admin@example.test")
	expectStatus(t, f.request(t, "POST", "/api/v1/me/sessions/revoke-others", map[string]any{}, cookie, csrf, testOrigin), 204)
	expectStatus(t, f.request(t, "GET", "/api/v1/auth/me", nil, cookie, "", ""), 200)
	expectStatus(t, f.request(t, "GET", "/api/v1/auth/me", nil, other, "", ""), 401)
	expectStatus(t, f.request(t, "GET", "/api/v1/auth/me", nil, admin, "", ""), 200)
}

func TestAdminDeactivationAndReactivation(t *testing.T) {
	f := newAdminFixture(t)
	admin, csrf, actor := f.login(t, "admin@example.test")
	member, memberCSRF, bob := f.login(t, "bob@example.test")
	for _, path := range []string{"/api/v1/admin/users", "/api/v1/admin/unknown"} {
		expectStatus(t, f.request(t, "GET", path, nil, member, "", ""), 403)
	}
	for _, action := range []string{"deactivate", "reactivate", "password-reset"} {
		expectStatus(t, f.request(t, "POST", "/api/v1/admin/users/"+actor.ID+"/"+action, map[string]any{}, member, memberCSRF, testOrigin), 403)
	}
	path := "/api/v1/admin/users/" + bob.ID
	reset := f.request(t, "POST", path+"/password-reset", map[string]any{}, admin, csrf, testOrigin)
	expectStatus(t, reset, 201)
	link := decode[auth.PasswordResetLink](t, reset)
	expectStatus(t, f.request(t, "POST", path+"/deactivate", map[string]any{}, admin, csrf, testOrigin), 204)
	expectStatus(t, f.request(t, "GET", "/api/v1/auth/me", nil, member, "", ""), 401)
	anonymous, token := f.csrf(t, nil)
	deactivated := f.request(t, "POST", "/api/v1/auth/login", map[string]string{"email": bob.Email, "password": testPassword}, anonymous, token, testOrigin)
	unknown := f.request(t, "POST", "/api/v1/auth/login", map[string]string{"email": "unknown@example.test", "password": testPassword}, anonymous, token, testOrigin)
	expectStatus(t, deactivated, 401)
	if !bytes.Equal(deactivated.body, unknown.body) {
		t.Fatal("deactivation disclosed by login")
	}
	expectStatus(t, f.request(t, "POST", path+"/password-reset", map[string]any{}, admin, csrf, testOrigin), 409)
	expectStatus(t, f.request(t, "POST", path+"/reactivate", map[string]any{}, admin, csrf, testOrigin), 204)
	expectStatus(t, f.request(t, "POST", "/api/v1/auth/login", map[string]string{"email": bob.Email, "password": testPassword}, anonymous, token, testOrigin), 200)
	expectStatus(t, f.request(t, "GET", "/api/v1/auth/me", nil, member, "", ""), 401)
	expectStatus(t, f.request(t, "GET", "/api/v1/reset-password/"+resetToken(link), nil, nil, "", ""), 400)
	// The last root Administrator cannot deactivate themselves or be deactivated.
	expectStatus(t, f.request(t, "POST", "/api/v1/admin/users/"+actor.ID+"/deactivate", map[string]any{}, admin, csrf, testOrigin), 409)
	if _, err := f.repo.Assign(context.Background(), f.alice, f.users["frank"], uuid.MustParse(rbac.AdministratorRoleID), rbac.RootScope(), ""); err != nil {
		t.Fatal(err)
	}
	// Self-deactivation is allowed only while another active root admin remains.
	expectStatus(t, f.request(t, "POST", "/api/v1/admin/users/"+actor.ID+"/deactivate", map[string]any{}, admin, csrf, testOrigin), 204)
	expectStatus(t, f.request(t, "GET", "/api/v1/auth/me", nil, admin, "", ""), 401)
	last, lastCSRF, _ := f.login(t, "frank@example.test")
	expectStatus(t, f.request(t, "POST", "/api/v1/admin/users/"+f.users["frank"].String()+"/deactivate", map[string]any{}, last, lastCSRF, testOrigin), 409)
	assertAudit(t, f.fixture, "user.deactivated", "user.reactivated", "session.revoke_all")
}

func TestManagedUserSearchAndLastLogin(t *testing.T) {
	f := newAdminFixture(t)
	cookie, csrf, _ := f.login(t, "bob@example.test")
	expectStatus(t, f.request(t, "PATCH", "/api/v1/me/profile", api.ProfileRequest{DisplayName: "Synthetic Search Person"}, cookie, csrf, testOrigin), 200)
	admin, _, _ := f.login(t, "admin@example.test")
	for _, term := range []string{"BOB@EXAMPLE", "search person"} {
		got := f.request(t, "GET", "/api/v1/admin/users?q="+url.QueryEscape(term), nil, admin, "", "")
		expectStatus(t, got, 200)
		page := decode[auth.ManagedUsers](t, got)
		if len(page.Users) != 1 || page.Users[0].ID != f.users["bob"].String() || page.Users[0].LastLoginAt == nil || page.Users[0].Role != "Developer" {
			t.Fatal("search/role summary/last login missing")
		}
	}
	first := decode[auth.ManagedUsers](t, f.request(t, "GET", "/api/v1/admin/users?limit=1", nil, admin, "", ""))
	if len(first.Users) != 1 || first.NextBefore == nil || first.Users[0].CanDeactivate {
		t.Fatal("pagination/last admin projection incorrect")
	}
	next := decode[auth.ManagedUsers](t, f.request(t, "GET", "/api/v1/admin/users?limit=1&before="+first.NextBefore.String(), nil, admin, "", ""))
	if len(next.Users) != 1 || next.Users[0].ID == first.Users[0].ID {
		t.Fatal("pagination repeats users")
	}
}

func TestMyAccessRedactsOriginsAndAllowsEmptyAccounts(t *testing.T) {
	f := newAdminFixture(t)
	ctx := context.Background()
	role, err := f.repo.CreateRole(ctx, f.alice, rbac.RoleInput{Name: "Document writer", Permissions: []rbac.Permission{rbac.DocsEdit}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.Assign(ctx, f.alice, f.users["frank"], role.ID, rbac.UnitScope(f.units["Engineering"]), "Private Engineering note"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.Assign(ctx, f.alice, f.users["frank"], uuid.MustParse(rbac.ViewerRoleID), rbac.UnitScope(f.units["Payments"]), ""); err != nil {
		t.Fatal(err)
	}
	cookie, _, _ := f.login(t, "frank@example.test")
	got := f.request(t, "GET", "/api/v1/me/access", nil, cookie, "", "")
	expectStatus(t, got, 200)
	access := decode[rbac.SelfAccess](t, got)
	if len(access.Units) != 1 || access.Units[0].Unit.ID != f.units["Payments"] {
		t.Fatal("unreadable units included")
	}
	if bytes.Contains(got.body, []byte("Engineering")) || bytes.Contains(got.body, []byte(f.units["Engineering"].String())) {
		t.Fatal("hidden source leaked")
	}
	hidden := false
	for _, grant := range access.Units[0].Assignments {
		if grant.CanRevoke {
			t.Fatal("self access is not read-only")
		}
		if grant.Source == rbac.SourceHidden {
			hidden = true
			if grant.ID != nil || grant.Scope != nil || grant.SourceUnit != nil || grant.Note != nil {
				t.Fatal("hidden assignment metadata leaked")
			}
		}
	}
	if !hidden {
		t.Fatal("expected inherited hidden grant")
	}
	// An active account with no assignments still owns account settings.
	newUser := f.seedUser(t, "empty@example.test")
	grants, err := f.repo.Assignments(ctx, uuid.MustParse(newUser.ID))
	if err != nil {
		t.Fatal(err)
	}
	for _, grant := range grants {
		if err := f.repo.Revoke(ctx, f.alice, grant.ID); err != nil {
			t.Fatal(err)
		}
	}
	empty, _, _ := f.login(t, newUser.Email)
	emptyAccess := f.request(t, "GET", "/api/v1/me/access", nil, empty, "", "")
	expectStatus(t, emptyAccess, 200)
	if len(decode[rbac.SelfAccess](t, emptyAccess).Units) != 0 {
		t.Fatal("empty account got access")
	}
	expectStatus(t, f.request(t, "GET", "/api/v1/me/profile", nil, empty, "", ""), 200)
}

func resetToken(link auth.PasswordResetLink) string {
	return strings.TrimPrefix(link.Link, testOrigin+"/reset-password/")
}
func createReset(t *testing.T, f *fixture, userID string, cookie *http.Cookie, csrf string) auth.PasswordResetLink {
	t.Helper()
	got := f.request(t, "POST", "/api/v1/admin/users/"+userID+"/password-reset", map[string]any{}, cookie, csrf, testOrigin)
	expectStatus(t, got, 201)
	return decode[auth.PasswordResetLink](t, got)
}

func TestPasswordResetLifecycleAndPrivacy(t *testing.T) {
	f := newAdminFixture(t)
	admin, csrf, _ := f.login(t, "admin@example.test")
	member, _, bob := f.login(t, "bob@example.test")
	old := createReset(t, f.fixture, bob.ID, admin, csrf)
	link := createReset(t, f.fixture, bob.ID, admin, csrf)
	if link.Link == old.Link || time.Until(link.ExpiresAt) < 23*time.Hour || time.Until(link.ExpiresAt) > 25*time.Hour {
		t.Fatal("reset link lifetime/rotation invalid")
	}
	token := resetToken(link)
	if !onetimetoken.Valid(token) {
		t.Fatal("bad reset entropy")
	}
	expectStatus(t, f.request(t, "GET", "/api/v1/reset-password/"+resetToken(old), nil, nil, "", ""), 400)
	info := f.request(t, "GET", "/api/v1/reset-password/"+token, nil, nil, "", "")
	expectStatus(t, info, 200)
	if decode[auth.PasswordResetDisplay](t, info).Email != bob.Email {
		t.Fatal("wrong reset account")
	}
	listed := f.request(t, "GET", "/api/v1/admin/users", nil, admin, "", "")
	expectStatus(t, listed, 200)
	for _, secret := range []string{token, resetToken(old), "token_hash", "password_hash", "reset_link"} {
		if bytes.Contains(listed.body, []byte(secret)) {
			t.Fatal("user list leaked reset/credential data")
		}
	}
	anonymous, resetCSRF := f.csrf(t, nil)
	expectStatus(t, f.request(t, "POST", "/api/v1/reset-password/"+token, api.PasswordResetRequest{Password: "Password1234"}, anonymous, resetCSRF, testOrigin), 400)
	used := f.request(t, "POST", "/api/v1/reset-password/"+token, api.PasswordResetRequest{Password: changedPassword}, anonymous, resetCSRF, testOrigin)
	expectStatus(t, used, 204)
	if used.cookie != nil && used.cookie.Value != "" {
		t.Fatal("password reset logged user in")
	}
	expectStatus(t, f.request(t, "GET", "/api/v1/auth/me", nil, anonymous, "", ""), 401)
	expectStatus(t, f.request(t, "GET", "/api/v1/auth/me", nil, member, "", ""), 401)
	// Resetting Bob must not revoke the issuer's unrelated session.
	expectStatus(t, f.request(t, "GET", "/api/v1/auth/me", nil, admin, "", ""), 200)
	expectStatus(t, f.request(t, "POST", "/api/v1/auth/login", map[string]string{"email": bob.Email, "password": changedPassword}, anonymous, resetCSRF, testOrigin), 200)
	expectStatus(t, f.request(t, "GET", "/api/v1/reset-password/"+token, nil, nil, "", ""), 400)
	for _, entry := range f.logs.All() {
		data, _ := json.Marshal(entry.ContextMap())
		if bytes.Contains(data, []byte(token)) || bytes.Contains(data, []byte(changedPassword)) {
			t.Fatal("reset secret logged")
		}
	}
	var audit string
	if err := f.db.QueryRow(`SELECT string_agg(details::text,'') FROM audit_log`).Scan(&audit); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(audit, token) || strings.Contains(audit, changedPassword) {
		t.Fatal("reset secret audited")
	}
	assertAudit(t, f.fixture, "password_reset.created", "password_reset.used", "user.password_changed", "session.revoke_all")
}

func TestPasswordResetUnavailableStates(t *testing.T) {
	f := newAdminFixture(t)
	admin, csrf, _ := f.login(t, "admin@example.test")
	link := createReset(t, f.fixture, f.users["bob"].String(), admin, csrf)
	if _, err := f.db.Exec(`UPDATE password_resets SET expires_at=now()-interval '1 second' WHERE user_id=$1`, f.users["bob"]); err != nil {
		t.Fatal(err)
	}
	expired := resetToken(link)
	revoked := createReset(t, f.fixture, f.users["bob"].String(), admin, csrf)
	used := createReset(t, f.fixture, f.users["bob"].String(), admin, csrf)
	if _, err := f.service.ResetPassword(context.Background(), resetToken(used), changedPassword, auth.Event{}); err != nil {
		t.Fatal(err)
	}
	unknown, err := onetimetoken.Generate()
	if err != nil {
		t.Fatal(err)
	}
	anonymous, token := f.csrf(t, nil)
	var expected []byte
	for _, value := range []string{expired, resetToken(revoked), resetToken(used), unknown, "invalid"} {
		get := f.request(t, "GET", "/api/v1/reset-password/"+value, nil, nil, "", "")
		post := f.request(t, "POST", "/api/v1/reset-password/"+value, api.PasswordResetRequest{Password: changedPassword}, anonymous, token, testOrigin)
		expectStatus(t, get, 400)
		expectStatus(t, post, 400)
		if expected == nil {
			expected = get.body
		}
		if !bytes.Equal(get.body, expected) || !bytes.Equal(post.body, expected) {
			t.Fatal("reset state disclosed")
		}
	}
}

func TestPasswordResetRateLimit(t *testing.T) {
	f := newFixture(t)
	for i := 0; i < 20; i++ {
		expectStatus(t, f.request(t, "GET", "/api/v1/reset-password/invalid", nil, nil, "", ""), 400)
	}
	expectStatus(t, f.request(t, "GET", "/api/v1/reset-password/invalid", nil, nil, "", ""), 429)
}

func TestAccountManagerUsesPermissionsAndResetRechecksIssuer(t *testing.T) {
	f := newAdminFixture(t)
	ctx := context.Background()
	role, err := f.repo.CreateRole(ctx, f.alice, rbac.RoleInput{Name: "Account operator", Permissions: []rbac.Permission{rbac.UserManage}})
	if err != nil {
		t.Fatal(err)
	}
	grant, err := f.repo.Assign(ctx, f.alice, f.users["frank"], role.ID, rbac.RootScope(), "")
	if err != nil {
		t.Fatal(err)
	}
	cookie, csrf, _ := f.login(t, "frank@example.test")
	expectStatus(t, f.request(t, "GET", "/api/v1/admin/users", nil, cookie, "", ""), 200)
	expectStatus(t, f.request(t, "POST", "/api/v1/admin/users/"+f.alice.String()+"/deactivate", map[string]any{}, cookie, csrf, testOrigin), 409)
	link := createReset(t, f.fixture, f.users["bob"].String(), cookie, csrf)
	if err := f.repo.Revoke(ctx, f.alice, grant.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.InspectPasswordReset(ctx, resetToken(link)); !errors.Is(err, auth.ErrResetInvalid) {
		t.Fatal("reset outlived issuer authority", err)
	}
	if _, err := f.service.ResetPassword(ctx, resetToken(link), changedPassword, auth.Event{}); !errors.Is(err, auth.ErrResetInvalid) {
		t.Fatal("revoked issuer reset accepted", err)
	}
	var unused bool
	if err := f.db.QueryRow(`SELECT used_at IS NULL FROM password_resets WHERE token_hash=$1`, onetimetoken.Hash(resetToken(link))).Scan(&unused); err != nil || !unused {
		t.Fatal("invalidated reset was consumed", err)
	}
}

func TestPasswordMutationsRollBackWhenAuditFails(t *testing.T) {
	for _, operation := range []string{"change", "reset", "replace_reset"} {
		t.Run(operation, func(t *testing.T) {
			f := newAdminFixture(t)
			ctx := context.Background()
			admin, csrf, actor := f.login(t, "admin@example.test")
			bobCookie, _, bob := f.login(t, "bob@example.test")
			link := createReset(t, f.fixture, bob.ID, admin, csrf)
			var before string
			if err := f.db.QueryRow(`SELECT password_hash FROM users WHERE id=$1`, bob.ID).Scan(&before); err != nil {
				t.Fatal(err)
			}
			constraint := map[string]string{"change": "user.password_changed", "reset": "password_reset.used", "replace_reset": "password_reset.created"}[operation]
			if _, err := f.db.Exec(`ALTER TABLE audit_log ADD CONSTRAINT synthetic_block_account_audit CHECK (action <> '` + constraint + `') NOT VALID`); err != nil {
				t.Fatal(err)
			}
			var err error
			switch operation {
			case "change":
				session, e := f.service.Lookup(ctx, bobCookie.Value)
				if e != nil {
					t.Fatal(e)
				}
				err = f.service.ChangePassword(ctx, session, testPassword, changedPassword, auth.Event{})
			case "reset":
				_, err = f.service.ResetPassword(ctx, resetToken(link), changedPassword, auth.Event{})
			case "replace_reset":
				_, err = f.service.CreatePasswordReset(ctx, &actor, uuid.MustParse(bob.ID))
			}
			if err == nil {
				t.Fatal("unaudited mutation committed")
			}
			var after string
			if err := f.db.QueryRow(`SELECT password_hash FROM users WHERE id=$1`, bob.ID).Scan(&after); err != nil || after != before {
				t.Fatal("password escaped rollback", err)
			}
			if _, err := f.service.Lookup(ctx, bobCookie.Value); err != nil {
				t.Fatal("session revocation escaped rollback", err)
			}
			if _, err := f.service.InspectPasswordReset(ctx, resetToken(link)); err != nil {
				t.Fatal("reset consumption/revocation escaped rollback", err)
			}
		})
	}
}

func TestSessionMetadataAndSeenWriteThrottle(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	previous, err := f.service.Anonymous(ctx)
	if err != nil {
		t.Fatal(err)
	}
	agent := "Mozilla/5.0 Windows Chrome/123.0 " + strings.Repeat("x", 400)
	session, err := f.service.Login(ctx, previous, "admin@example.test", testPassword, auth.Event{RemoteIP: "198.51.100.17", UserAgent: agent})
	if err != nil {
		t.Fatal(err)
	}
	var stored string
	var ip string
	if err := f.db.QueryRow(`SELECT user_agent,ip FROM sessions WHERE id=$1`, session.ID).Scan(&stored, &ip); err != nil || len(stored) != 256 || ip != "198.51.100.17" {
		t.Fatal("metadata not truncated/stored correctly", err)
	}
	listing, err := f.service.Sessions(ctx, session)
	if err != nil || len(listing.Sessions) != 1 || listing.Sessions[0].UserAgent != "Chrome · Windows" {
		t.Fatal("agent is not coarse", err)
	}
	if _, err := f.db.Exec(`UPDATE sessions SET last_seen_at=now()-interval '10 minutes' WHERE id=$1`, session.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.Lookup(ctx, session.Token); err != nil {
		t.Fatal(err)
	}
	var first, second time.Time
	if err := f.db.QueryRow(`SELECT last_seen_at FROM sessions WHERE id=$1`, session.ID).Scan(&first); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.Lookup(ctx, session.Token); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRow(`SELECT last_seen_at FROM sessions WHERE id=$1`, session.ID).Scan(&second); err != nil || !first.Equal(second) {
		t.Fatal("last_seen updated more than once per interval", err)
	}
	if err := f.service.ConfigureAccount(auth.AccountConfig{DisableSessionIP: true}); err != nil {
		t.Fatal(err)
	}
	listing, err = f.service.Sessions(ctx, session)
	if err != nil || listing.Sessions[0].IP != nil {
		t.Fatal("privacy flag did not hide IP", err)
	}
	previous, err = f.service.Anonymous(ctx)
	if err != nil {
		t.Fatal(err)
	}
	next, err := f.service.Login(ctx, previous, "admin@example.test", testPassword, auth.Event{RemoteIP: "198.51.100.18"})
	if err != nil {
		t.Fatal(err)
	}
	var omitted bool
	if err := f.db.QueryRow(`SELECT ip IS NULL FROM sessions WHERE id=$1`, next.ID).Scan(&omitted); err != nil || !omitted {
		t.Fatal("privacy flag stored session IP", err)
	}
}

func TestLoginCannotReuseAPasswordVerifiedBeforeChange(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	previous, err := f.service.Anonymous(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// Give the login connection a distinct application name so waiting for the
	// shared lock is observable without a timing assumption or production hook.
	dsn := os.Getenv("SCHEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		cfg, err := config.Load()
		if err != nil {
			t.Fatal(err)
		}
		dsn = cfg.DatabaseURL
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	var schema string
	if err := f.db.QueryRow(`SELECT current_schema()`).Scan(&schema); err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	name := "stale_login_" + uuid.NewString()
	q.Set("application_name", name)
	u.RawQuery = q.Encode()
	db, err := database.Open(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service, err := auth.New(db, zap.NewNop(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := auth.HashPassword(changedPassword)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := f.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err := rbac.Lock(ctx, tx); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := service.Login(ctx, previous, "admin@example.test", testPassword, auth.Event{})
		done <- err
	}()
	waiting := false
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if err := f.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE application_name=$1 AND wait_event_type='Lock' AND wait_event='advisory')`, name).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !waiting {
		t.Fatal("login never reached the credential serialization point")
	}
	if _, err := tx.Exec(`UPDATE users SET password_hash=$1 WHERE email='admin@example.test'`, hash); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, auth.ErrCredentials) {
		t.Fatal("stale credential login was accepted", err)
	}
	var n int
	if err := f.db.QueryRow(`SELECT count(*) FROM sessions WHERE user_id IS NOT NULL`).Scan(&n); err != nil || n != 0 {
		t.Fatal("stale login created a session", err)
	}
}

func assertAudit(t *testing.T, f *fixture, actions ...string) {
	t.Helper()
	for _, action := range actions {
		var n int
		if err := f.db.QueryRow(`SELECT count(*) FROM audit_log WHERE action=$1`, action).Scan(&n); err != nil || n == 0 {
			t.Fatalf("missing audit %s: %v", action, err)
		}
	}
}
