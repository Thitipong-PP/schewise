package config

import "testing"

func TestRequiresHTTPSOrigin(t *testing.T) {
	t.Setenv("PGPASSWORD", "synthetic-database-password")
	for _, origin := range []string{"http://localhost:8443", "https://localhost/path", "https://user:secret@localhost", "https://localhost?token=secret"} {
		t.Setenv("SCHEWISE_ORIGIN", origin)
		if _, err := Load(); err == nil {
			t.Errorf("accepted invalid origin %q", origin)
		}
	}
	t.Setenv("SCHEWISE_ORIGIN", "https://localhost:8443")
	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
}

func TestSessionTTLBounds(t *testing.T) {
	t.Setenv("PGPASSWORD", "synthetic-database-password")
	t.Setenv("SCHEWISE_ORIGIN", "https://localhost:8443")
	for _, ttl := range []string{"0s", "-1h", "10000h", "invalid"} {
		t.Setenv("SCHEWISE_SESSION_TTL", ttl)
		if _, err := Load(); err == nil {
			t.Errorf("accepted invalid lifetime %q", ttl)
		}
	}
}
