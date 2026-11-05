package config

import "testing"

func TestAccountConfiguration(t *testing.T) {
	t.Setenv("PGPASSWORD", "synthetic-database-password")
	t.Setenv("SCHEWISE_STORE_SESSION_IP", "")
	t.Setenv("SCHEWISE_PASSWORD_RESET_TTL", "")
	cfg, err := Load()
	if err != nil || !cfg.StoreSessionIP || cfg.ResetTTL.Hours() != 24 {
		t.Fatal("invalid account defaults", err)
	}
	t.Setenv("SCHEWISE_STORE_SESSION_IP", "false")
	t.Setenv("SCHEWISE_PASSWORD_RESET_TTL", "48h")
	cfg, err = Load()
	if err != nil || cfg.StoreSessionIP || cfg.ResetTTL.Hours() != 48 {
		t.Fatal("account configuration ignored", err)
	}
	t.Setenv("SCHEWISE_STORE_SESSION_IP", "invalid")
	if _, err := Load(); err == nil {
		t.Fatal("invalid IP privacy flag accepted")
	}
	t.Setenv("SCHEWISE_STORE_SESSION_IP", "true")
	for _, ttl := range []string{"0s", "-1h", "169h", "invalid"} {
		t.Setenv("SCHEWISE_PASSWORD_RESET_TTL", ttl)
		if _, err := Load(); err == nil {
			t.Errorf("invalid reset lifetime %q accepted", ttl)
		}
	}
}

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
