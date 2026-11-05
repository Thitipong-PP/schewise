// Package config reads process environment variables without loading dotenv files.
package config

import (
	"errors"
	"net"
	"net/url"
	"os"
	"strings"
	"time"
)

type Config struct {
	Address       string
	DatabaseURL   string
	Origin        string
	AdminEmail    string
	AdminPassword string
	SessionTTL    time.Duration
	PublicBaseURL string
	InviteTTL     time.Duration
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func Load() (Config, error) {
	c := Config{Address: env("SCHEWISE_HTTP_ADDR", "127.0.0.1:8080"), Origin: strings.TrimRight(env("SCHEWISE_ORIGIN", "https://localhost:8443"), "/"),
		AdminEmail: os.Getenv("SCHEWISE_ADMIN_EMAIL"), AdminPassword: os.Getenv("SCHEWISE_ADMIN_PASSWORD"), SessionTTL: 24 * time.Hour}
	origin, err := url.Parse(c.Origin)
	if err != nil || origin.Scheme != "https" || origin.Host == "" || origin.User != nil || origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" {
		return c, errors.New("SCHEWISE_ORIGIN must be an HTTPS origin without a path, query, or credentials")
	}
	if value := os.Getenv("SCHEWISE_SESSION_TTL"); value != "" {
		c.SessionTTL, err = time.ParseDuration(value)
		if err != nil || c.SessionTTL < time.Minute || c.SessionTTL > 30*24*time.Hour {
			return c, errors.New("SCHEWISE_SESSION_TTL must be between 1m and 720h")
		}
	}
	c.PublicBaseURL = strings.TrimRight(env("PUBLIC_BASE_URL", c.Origin), "/")
	if c.PublicBaseURL != c.Origin {
		return c, errors.New("PUBLIC_BASE_URL must match SCHEWISE_ORIGIN (the browser-facing HTTPS origin)")
	}
	c.InviteTTL = 7 * 24 * time.Hour
	if value := os.Getenv("SCHEWISE_INVITE_TTL"); value != "" {
		c.InviteTTL, err = time.ParseDuration(value)
		if err != nil || c.InviteTTL < time.Minute || c.InviteTTL > 30*24*time.Hour {
			return c, errors.New("SCHEWISE_INVITE_TTL must be between 1m and 720h")
		}
	}
	c.DatabaseURL = os.Getenv("SCHEWISE_DATABASE_URL")
	if c.DatabaseURL == "" {
		password := os.Getenv("PGPASSWORD")
		if password == "" {
			return c, errors.New("set PGPASSWORD or SCHEWISE_DATABASE_URL for the application database")
		}
		connection := url.URL{Scheme: "postgres", Host: net.JoinHostPort(env("PGHOST", "127.0.0.1"), env("PGPORT", "5432")), Path: env("PGDATABASE", "schewise"), User: url.UserPassword(env("PGUSER", "schewise"), password)}
		query := url.Values{"sslmode": {env("PGSSLMODE", "disable")}, "connect_timeout": {"5"}}
		connection.RawQuery = query.Encode()
		c.DatabaseURL = connection.String()
	}
	return c, nil
}
