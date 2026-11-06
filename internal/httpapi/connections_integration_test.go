// SPDX-License-Identifier: Apache-2.0
package httpapi_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Thitipong-PP/schewise/internal/connections"
	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/google/uuid"
)

func TestConnectionHTTPPermissionsRedactionAndDelete(t *testing.T) {
	f := newAdminFixture(t)
	admin, csrf, _ := f.login(t, "admin@example.test")
	viewer, viewerCSRF, _ := f.login(t, "eve@example.test")
	outsider, _, _ := f.login(t, "frank@example.test")
	secret := "Synthetic-target-secret-42!"
	in := connections.CreateInput{Input: connections.Input{UnitID: f.units["Payments"], Name: "Payments fixture", Engine: "postgres", Host: "db.example.test", Port: 5432, Database: "synthetic", Username: "reader", SSLMode: "require"}, Password: secret}
	response := f.request(t, "POST", "/api/v1/connections", in, admin, csrf, testOrigin)
	expectStatus(t, response, 201)
	connection := decode[connections.Result](t, response).Connection
	path := "/api/v1/connections/" + connection.ID.String()
	for _, endpoint := range []string{path, path + "/schema", path + "/snapshots"} {
		expectStatus(t, f.request(t, "GET", endpoint, nil, nil, "", ""), 401)
		expectStatus(t, f.request(t, "GET", endpoint, nil, outsider, "", ""), 403)
	}
	visible := f.request(t, "GET", path, nil, viewer, "", "")
	expectStatus(t, visible, 200)
	if strings.Contains(string(visible.body), "db.example.test") || strings.Contains(string(visible.body), "reader") || strings.Contains(string(visible.body), secret) {
		t.Fatal("connection details leaked")
	}
	if decode[connections.Result](t, visible).Connection.Details != nil {
		t.Fatal("Viewer received host details")
	}
	expectStatus(t, f.request(t, "POST", path+"/sync", map[string]any{}, viewer, viewerCSRF, testOrigin), 403)
	expectStatus(t, f.request(t, "DELETE", path, map[string]any{"name": connection.Name}, viewer, viewerCSRF, testOrigin), 403)
	expectStatus(t, f.request(t, "PATCH", path, in.Input, admin, "", testOrigin), 403)
	// Real connection scopes can be assigned and revoked through the existing API.
	assignmentInput := map[string]any{"user_id": f.users["frank"], "role_id": rbac.ViewerRoleID, "scope": rbac.Scope{Type: rbac.ScopeConnection, ID: connection.ID}, "note": "Synthetic connection grant"}
	assigned := f.request(t, "POST", "/api/v1/assignments", assignmentInput, admin, csrf, testOrigin)
	expectStatus(t, assigned, 201)
	expectStatus(t, f.request(t, "GET", path, nil, outsider, "", ""), 200)
	contextResponse := f.request(t, "GET", "/api/v1/access/context", nil, outsider, "", "")
	expectStatus(t, contextResponse, 200)
	var access rbac.AccessContext
	if err := json.Unmarshal(contextResponse.body, &access); err != nil {
		t.Fatal(err)
	}
	if access.CanReadUnits || !access.CanReadConnections {
		t.Fatal("connection-only workspace capabilities are incorrect")
	}

	explanation := f.request(t, "POST", "/api/v1/access/explain", map[string]any{"user_id": f.users["frank"], "permission": "schema:read", "resource": rbac.Scope{Type: rbac.ScopeConnection, ID: connection.ID}}, admin, csrf, testOrigin)
	expectStatus(t, explanation, 200)
	var grant struct {
		Assignment struct {
			ID uuid.UUID `json:"id"`
		} `json:"assignment"`
	}
	if err := json.Unmarshal(assigned.body, &grant); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request(t, "DELETE", "/api/v1/assignments/"+grant.Assignment.ID.String(), nil, admin, csrf, testOrigin), 204)
	preview := f.request(t, "POST", path+"/dry-run/delete", map[string]any{}, admin, csrf, testOrigin)
	expectStatus(t, preview, 200)
	if decode[connections.DeleteImpact](t, preview).Name != in.Name {
		t.Fatal("delete impact omitted name")
	}
	expectStatus(t, f.request(t, "DELETE", path, map[string]any{"name": "wrong"}, admin, csrf, testOrigin), 400)
	expectStatus(t, f.request(t, "DELETE", path, map[string]any{"name": connection.Name}, admin, csrf, testOrigin), 204)
	if err := f.repo.DeleteUnit(context.Background(), f.alice, f.units["Payments"]); err == nil {
		t.Fatal("unit still has its direct Manager assignment")
	}
	logs, _ := json.Marshal(f.logs.All())
	if strings.Contains(string(logs), secret) {
		t.Fatal("target password logged")
	}
}
