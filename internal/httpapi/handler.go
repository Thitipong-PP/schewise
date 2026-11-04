// Package httpapi implements the public HTTP boundary of the Go API.
package httpapi

import "net/http"

// NewHandler builds the API routes without opening a network listener.
func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte("{\"status\":\"ok\",\"service\":\"schewise-api\"}\n"))
	})
	return mux
}
