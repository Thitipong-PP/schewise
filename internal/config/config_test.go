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

func TestInviteConfiguration(t *testing.T) {
	t.Setenv("PGPASSWORD", "synthetic-database-password")
	t.Setenv("SCHEWISE_ORIGIN", "https://localhost:8443")
	t.Setenv("PUBLIC_BASE_URL", "")
	t.Setenv("SCHEWISE_INVITE_TTL", "")
	cfg, err := Load()
	if err != nil || cfg.PublicBaseURL != cfg.Origin || cfg.InviteTTL.Hours() != 168 {
		t.Fatal("invalid invite defaults", err)
	}
	for _, value := range []string{"http://localhost:8443", "https://attacker.example", "https://localhost:8443/path", "https://localhost:8443?token=secret"} {
		t.Setenv("PUBLIC_BASE_URL", value)
		if _, err := Load(); err == nil {
			t.Error("accepted incompatible public URL")
		}
	}
	t.Setenv("PUBLIC_BASE_URL", cfg.Origin)
	for _, ttl := range []string{"0s", "-1h", "721h", "invalid"} {
		t.Setenv("SCHEWISE_INVITE_TTL", ttl)
		if _, err := Load(); err == nil {
			t.Errorf("accepted invite lifetime %q", ttl)
		}
	}
	t.Setenv("SCHEWISE_INVITE_TTL", "48h")
	if cfg, err := Load(); err != nil || cfg.InviteTTL.Hours() != 48 {
		t.Fatal("custom invite lifetime rejected", err)
	}
}
