package httpapi_test

import (
	"encoding/json"
	"mime"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Thitipong-PP/schewise/internal/httpapi"
)

func TestHealthContract(t *testing.T) {
	response, err := httpapi.NewApp(httpapi.Config{}).Test(httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	if got, _, err := mime.ParseMediaType(response.Header.Get("Content-Type")); err != nil || got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	var body struct {
		Status  string `json:"status"`
		Service string `json:"service"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
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
			response, err := httpapi.NewApp(httpapi.Config{}).Test(httptest.NewRequest(tc.method, tc.path, nil))
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != tc.status {
				t.Errorf("status = %d, want %d", response.StatusCode, tc.status)
			}
		})
	}
}
