package httpapi_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Thitipong-PP/schewise/internal/auth"
	"github.com/Thitipong-PP/schewise/internal/config"
	"github.com/Thitipong-PP/schewise/internal/database"
	"github.com/Thitipong-PP/schewise/internal/httpapi"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

const testOrigin = "https://localhost:8443"
const testPassword = "Synthetic-test-password-42!"

type fixture struct {
	db      *sql.DB
	app     *fiber.App
	service *auth.Service
	logs    *observer.ObservedLogs
}

func newFixture(t *testing.T) *fixture {
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
	schema := "auth_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, err := admin.ExecContext(ctx, `DROP SCHEMA `+schema+` CASCADE`)
		if err != nil {
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
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal("migrations are not idempotent", err)
	}
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	service, err := auth.New(db, logger, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.Bootstrap(ctx, "admin@example.test", testPassword)
	if err != nil || !created {
		t.Fatal("bootstrap failed", err)
	}
	return &fixture{db: db, app: httpapi.NewApp(httpapi.Config{Auth: service, Logger: logger, Origin: testOrigin}), service: service, logs: logs}
}

type result struct {
	status int
	body   []byte
	cookie *http.Cookie
	header http.Header
}

func (f *fixture) request(t *testing.T, method, path string, input any, cookie *http.Cookie, csrf, origin string) result {
	t.Helper()
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		body = bytes.NewReader(encoded)
	}
	req := httptest.NewRequest(method, testOrigin+path, body)
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	response, err := f.app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second, FailOnTimeout: true})
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	got := result{status: response.StatusCode, body: data, header: response.Header}
	for _, cookie := range response.Cookies() {
		if cookie.Name == httpapi.CookieName {
			got.cookie = cookie
		}
	}
	return got
}

func expectStatus(t *testing.T, result result, status int) {
	t.Helper()
	if result.status != status {
		t.Fatalf("status = %d, want %d; body = %s", result.status, status, result.body)
	}
}

func (f *fixture) csrf(t *testing.T, cookie *http.Cookie) (*http.Cookie, string) {
	t.Helper()
	got := f.request(t, "GET", "/api/v1/auth/csrf", nil, cookie, "", "")
	expectStatus(t, got, 200)
	var data struct {
		Token string `json:"csrf_token"`
	}
	if err := json.Unmarshal(got.body, &data); err != nil {
		t.Fatal(err)
	}
	if got.cookie != nil {
		cookie = got.cookie
	}
	if cookie == nil || data.Token == "" {
		t.Fatal("missing session/CSRF bootstrap")
	}
	return cookie, data.Token
}

func (f *fixture) login(t *testing.T, email string) (*http.Cookie, string, auth.User) {
	t.Helper()
	cookie, csrf := f.csrf(t, nil)
	got := f.request(t, "POST", "/api/v1/auth/login", map[string]string{"email": email, "password": testPassword}, cookie, csrf, testOrigin)
	expectStatus(t, got, 200)
	if got.cookie == nil || got.cookie.Value == cookie.Value {
		t.Fatal("login did not rotate session")
	}
	var data struct {
		User auth.User `json:"user"`
	}
	if err := json.Unmarshal(got.body, &data); err != nil {
		t.Fatal(err)
	}
	cookie, csrf = f.csrf(t, got.cookie)
	return cookie, csrf, data.User
}

func TestAuthenticationLifecycle(t *testing.T) {
	f := newFixture(t)
	credentials := map[string]string{"email": "admin@example.test", "password": testPassword}
	cookie, csrf := f.csrf(t, nil)
	if !cookie.Secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" || cookie.Domain != "" {
		t.Fatal("session cookie policy is incomplete")
	}
	expectStatus(t, f.request(t, "GET", "/api/v1/auth/me", nil, cookie, "", ""), 401)
	expectStatus(t, f.request(t, "POST", "/api/v1/auth/login", credentials, cookie, "", testOrigin), 403)
	expectStatus(t, f.request(t, "POST", "/api/v1/auth/login", credentials, cookie, csrf, "https://attacker.example"), 403)
	expectStatus(t, f.request(t, "POST", "/api/v1/auth/login", credentials, cookie, csrf, ""), 403)
	expectStatus(t, f.request(t, "POST", "/api/v1/auth/register", credentials, cookie, csrf, testOrigin), 401)
	wrong := f.request(t, "POST", "/api/v1/auth/login", map[string]string{"email": "admin@example.test", "password": "Incorrect-password-42!"}, cookie, csrf, testOrigin)
	unknown := f.request(t, "POST", "/api/v1/auth/login", map[string]string{"email": "unknown@example.test", "password": "Incorrect-password-42!"}, cookie, csrf, testOrigin)
	expectStatus(t, wrong, 401)
	expectStatus(t, unknown, 401)
	if !bytes.Equal(wrong.body, unknown.body) {
		t.Fatal("login discloses account existence")
	}
	loggedIn := f.request(t, "POST", "/api/v1/auth/login", credentials, cookie, csrf, testOrigin)
	expectStatus(t, loggedIn, 200)
	if loggedIn.cookie == nil || loggedIn.cookie.Value == cookie.Value {
		t.Fatal("session fixation: token was not rotated")
	}
	expectStatus(t, f.request(t, "GET", "/api/v1/auth/me", nil, cookie, "", ""), 401)
	cookie, csrf = f.csrf(t, loggedIn.cookie)
	me := f.request(t, "GET", "/api/v1/auth/me", nil, cookie, "", "")
	expectStatus(t, me, 200)
	if bytes.Contains(me.body, []byte("password")) || bytes.Contains(me.body, []byte(cookie.Value)) {
		t.Fatal("identity response contains credentials")
	}
	_, otherCSRF := f.csrf(t, nil)
	expectStatus(t, f.request(t, "POST", "/api/v1/auth/register", credentials, cookie, otherCSRF, testOrigin), 403)
	input := map[string]string{"email": "second@example.test", "password": testPassword}
	created := f.request(t, "POST", "/api/v1/auth/register", input, cookie, csrf, testOrigin)
	expectStatus(t, created, 201)
	var registered struct {
		User auth.User `json:"user"`
	}
	if err := json.Unmarshal(created.body, &registered); err != nil {
		t.Fatal(err)
	}
	if _, err := uuid.Parse(registered.User.ID); err != nil || registered.User.Role != "Administrator" {
		t.Fatal("invalid identity/role")
	}
	var storedHash string
	if err := f.db.QueryRow(`SELECT password_hash FROM users WHERE id=$1`, registered.User.ID).Scan(&storedHash); err != nil {
		t.Fatal(err)
	}
	if storedHash == testPassword || !strings.HasPrefix(storedHash, "$argon2id$") {
		t.Fatal("database did not store an Argon2id hash")
	}
	expectStatus(t, f.request(t, "POST", "/api/v1/auth/register", input, cookie, csrf, testOrigin), 409)
	injected := map[string]string{"email": "third@example.test", "password": testPassword, "role": "Administrator"}
	expectStatus(t, f.request(t, "POST", "/api/v1/auth/register", injected, cookie, csrf, testOrigin), 400)
	second, _, user := f.login(t, "second@example.test")
	expectStatus(t, f.request(t, "DELETE", "/api/v1/users/"+user.ID+"/sessions", nil, cookie, csrf, testOrigin), 204)
	expectStatus(t, f.request(t, "GET", "/api/v1/auth/me", nil, second, "", ""), 401)
	loggedOut := f.request(t, "POST", "/api/v1/auth/logout", nil, cookie, csrf, testOrigin)
	expectStatus(t, loggedOut, 204)
	if loggedOut.cookie == nil || loggedOut.cookie.MaxAge != -1 || !loggedOut.cookie.Secure {
		t.Fatal("logout did not expire the secure cookie")
	}
	expectStatus(t, f.request(t, "GET", "/api/v1/auth/me", nil, cookie, "", ""), 401)
	for _, action := range []string{"user.bootstrap", "auth.login", "auth.csrf", "user.register", "session.revoke_user", "auth.logout"} {
		var count int
		if err := f.db.QueryRow(`SELECT count(*) FROM audit_events WHERE action=$1`, action).Scan(&count); err != nil || count == 0 {
			t.Fatalf("missing durable audit event %s", action)
		}
	}
	for _, entry := range f.logs.All() {
		encoded, _ := json.Marshal(entry.ContextMap())
		for _, secret := range []string{testPassword, cookie.Value, csrf} {
			if bytes.Contains(encoded, []byte(secret)) {
				t.Fatal("Zap logs contain secret material")
			}
		}
	}
}

func TestSessionExpiryAndPermissions(t *testing.T) {
	f := newFixture(t)
	cookie, csrf, user := f.login(t, "admin@example.test")
	if _, err := f.db.Exec(`INSERT INTO roles(name) VALUES('TestOnlyRestricted')`); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(`UPDATE users SET role_id=(SELECT id FROM roles WHERE name='TestOnlyRestricted') WHERE id=$1`, user.ID); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request(t, "POST", "/api/v1/auth/register", map[string]string{"email": "denied@example.test", "password": testPassword}, cookie, csrf, testOrigin), 403)
	expectStatus(t, f.request(t, "DELETE", "/api/v1/users/"+user.ID+"/sessions", nil, cookie, csrf, testOrigin), 403)
	if _, err := f.db.Exec(`UPDATE sessions SET expires_at=now()-interval '1 second' WHERE user_id=$1`, user.ID); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request(t, "GET", "/api/v1/auth/me", nil, cookie, "", ""), 401)
}

func TestBootstrapDoesNotResetExistingAdministrator(t *testing.T) {
	f := newFixture(t)
	created, err := f.service.Bootstrap(context.Background(), "other@example.test", "Changed-installation-password-42!")
	if err != nil || created {
		t.Fatal("bootstrap changed an initialized installation")
	}
	_, _, user := f.login(t, "admin@example.test")
	if user.Email != "admin@example.test" {
		t.Fatal("bootstrap changed the admin")
	}
	var count int
	if err := f.db.QueryRow(`SELECT count(*) FROM users`).Scan(&count); err != nil || count != 1 {
		t.Fatal("bootstrap duplicated accounts")
	}
}

func TestFailedAuditRollsBackAccountCreation(t *testing.T) {
	f := newFixture(t)
	cookie, csrf, _ := f.login(t, "admin@example.test")
	if _, err := f.db.Exec(`ALTER TABLE audit_events RENAME TO temporarily_unavailable_audit`); err != nil {
		t.Fatal(err)
	}
	got := f.request(t, "POST", "/api/v1/auth/register", map[string]string{"email": "rolled-back@example.test", "password": testPassword}, cookie, csrf, testOrigin)
	expectStatus(t, got, 503)
	var count int
	if err := f.db.QueryRow(`SELECT count(*) FROM users WHERE email='rolled-back@example.test'`).Scan(&count); err != nil || count != 0 {
		t.Fatal("unaudited account creation was committed")
	}
}

func TestLoginRateLimit(t *testing.T) {
	f := newFixture(t)
	for attempt := 0; attempt < 10; attempt++ {
		expectStatus(t, f.request(t, "POST", "/api/v1/auth/login", nil, nil, "", testOrigin), 403)
	}
	expectStatus(t, f.request(t, "POST", "/api/v1/auth/login", nil, nil, "", testOrigin), 429)
}
