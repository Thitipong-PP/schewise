package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Thitipong-PP/schewise/internal/httpapi"
)

func TestHealthContract(t *testing.T) {
	recorder := httptest.NewRecorder()
	httpapi.NewHandler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	var body struct {
		Status  string `json:"status"`
		Service string `json:"service"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid health JSON: %v", err)
	}
	if body.Status != "ok" || body.Service != "schewise-api" {
		t.Errorf("unexpected health response: %+v", body)
	}
}

func TestRouteBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name   string
		method string
		path   string
		status int
	}{
		{"head health", http.MethodHead, "/api/v1/health", http.StatusOK},
		{"reject mutation", http.MethodPost, "/api/v1/health", http.StatusMethodNotAllowed},
		{"unknown path", http.MethodGet, "/api/v1/missing", http.StatusNotFound},
		{"no catch-all health", http.MethodGet, "/", http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			httpapi.NewHandler().ServeHTTP(recorder, httptest.NewRequest(tc.method, tc.path, nil))
			if recorder.Code != tc.status {
				t.Errorf("status = %d, want %d", recorder.Code, tc.status)
			}
		})
	}
}
