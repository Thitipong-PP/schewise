// SPDX-License-Identifier: Apache-2.0
package connections

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type TargetError struct {
	Code    string
	Message string
}

func (e *TargetError) Error() string { return e.Message }
func targetError(err error) *TargetError {
	var known *TargetError
	if errors.As(err, &known) {
		return known
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "28P01", "28000":
			return &TargetError{"auth_failed", "Authentication failed. Check the connection credentials."}
		case "3D000":
			return &TargetError{"database_not_found", "The database does not exist or is unavailable."}
		case "42501":
			return &TargetError{"permission_denied", "The database account cannot read the required catalogs."}
		case "57014", "55P03":
			return &TargetError{"timeout", "The database operation timed out."}
		}
	}
	var cert *tls.CertificateVerificationError
	if errors.As(err, &cert) {
		return &TargetError{"tls_error", "TLS verification failed. Check SSL mode and certificate trust."}
	}
	var timeout net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &timeout) && timeout.Timeout()) {
		return &TargetError{"timeout", "The database operation timed out."}
	}
	return &TargetError{"cannot_reach_host", "Cannot reach the database. Check the host, port, and network access."}
}

type hostPolicy struct{ allow, deny []string }

func newHostPolicy(allow, deny []string) (hostPolicy, error) {
	policy := hostPolicy{}
	for i, list := range [][]string{allow, deny} {
		for _, item := range list {
			item = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(item), "."))
			if item == "" {
				continue
			}
			if strings.Contains(item, "/") {
				if _, err := netip.ParsePrefix(item); err != nil {
					return policy, errors.New("connection host lists accept exact hostnames, IPs, or CIDRs")
				}
			} else if !validHost(item) {
				return policy, errors.New("invalid connection host-list entry")
			}
			if i == 0 {
				policy.allow = append(policy.allow, item)
			} else {
				policy.deny = append(policy.deny, item)
			}
		}
	}
	return policy, nil
}
func validHost(host string) bool {
	if _, err := netip.ParseAddr(host); err == nil {
		return true
	}
	if host == "" || len(host) > 253 {
		return false
	}
	for _, part := range strings.Split(strings.TrimSuffix(host, "."), ".") {
		if part == "" || len(part) > 63 || part[0] == '-' || part[len(part)-1] == '-' {
			return false
		}
		for _, c := range part {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	return true
}
func ruleMatches(rule, host string, ip netip.Addr) bool {
	if rule == host {
		return true
	}
	if prefix, err := netip.ParsePrefix(rule); err == nil && ip.IsValid() {
		return prefix.Contains(ip.Unmap())
	}
	if addr, err := netip.ParseAddr(rule); err == nil && ip.IsValid() {
		return addr.Unmap() == ip.Unmap()
	}
	return false
}
func (p hostPolicy) permits(host string, ip netip.Addr) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	for _, rule := range p.deny {
		if ruleMatches(rule, host, ip) {
			return false
		}
	}
	if len(p.allow) == 0 {
		return true
	}
	for _, rule := range p.allow {
		if ruleMatches(rule, host, ip) {
			return true
		}
	}
	return false
}

// Only explicit profile credentials are used. The constant parse seed cannot
// import an application password/passfile or a client certificate into targets.
func (p hostPolicy) connect(ctx context.Context, input Input, password string, statementTimeout time.Duration) (*pgx.Conn, error) {
	cfg, err := pgx.ParseConfig("host=127.0.0.1 port=5432 dbname=postgres user=schewise_target password=unused sslmode=disable passfile=/dev/null")
	if err != nil {
		return nil, &TargetError{"target_config", "Target connection configuration is unavailable."}
	}
	cfg.Host, cfg.Port, cfg.Database, cfg.User, cfg.Password = input.Host, uint16(input.Port), input.Database, input.Username, password
	cfg.ConnectTimeout = 5 * time.Second
	cfg.RuntimeParams = map[string]string{"application_name": "schewise_schema", "default_transaction_read_only": "on", "statement_timeout": strconv.FormatInt(statementTimeout.Milliseconds(), 10), "lock_timeout": "3000", "search_path": "pg_catalog", "client_encoding": "UTF8"}
	cfg.DefaultQueryExecMode = pgx.QueryExecModeExec
	cfg.Fallbacks = nil
	cfg.TLSConfig = nil
	secure := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: input.Host}
	switch input.SSLMode {
	case "require", "prefer", "allow":
		secure.InsecureSkipVerify = true // Explicit libpq modes do not verify server identity.
	case "verify-ca":
		roots, err := x509.SystemCertPool()
		if err != nil {
			return nil, &TargetError{"tls_error", "System certificate trust is unavailable."}
		}
		secure.InsecureSkipVerify = true // Verify the chain below, intentionally omitting hostname verification for verify-ca.
		secure.VerifyConnection = func(state tls.ConnectionState) error {
			if len(state.PeerCertificates) == 0 {
				return errors.New("missing server certificate")
			}
			chain := x509.NewCertPool()
			for _, c := range state.PeerCertificates[1:] {
				chain.AddCert(c)
			}
			_, err := state.PeerCertificates[0].Verify(x509.VerifyOptions{Roots: roots, Intermediates: chain})
			return err
		}
	}
	if input.SSLMode != "disable" {
		cfg.TLSConfig = secure
	}
	if input.SSLMode == "prefer" {
		cfg.Fallbacks = []*pgconn.FallbackConfig{{Host: input.Host, Port: uint16(input.Port)}}
	}
	if input.SSLMode == "allow" {
		cfg.TLSConfig = nil
		cfg.Fallbacks = []*pgconn.FallbackConfig{{Host: input.Host, Port: uint16(input.Port), TLSConfig: secure}}
	}
	// Resolve once per attempt, check every address, then dial only checked IPs.
	cfg.LookupFunc = func(ctx context.Context, host string) ([]string, error) {
		addresses, err := net.DefaultResolver.LookupHost(ctx, host)
		if err != nil {
			return nil, err
		}
		for _, text := range addresses {
			ip, err := netip.ParseAddr(text)
			if err != nil || !p.permits(input.Host, ip) {
				return nil, &TargetError{"host_denied", "The destination is blocked by the host policy."}
			}
		}
		return addresses, nil
	}
	cfg.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(address)
		ip, parseErr := netip.ParseAddr(host)
		if err != nil || parseErr != nil || network != "tcp" || !p.permits(input.Host, ip) {
			return nil, &TargetError{"host_denied", "The destination is blocked by the host policy."}
		}
		return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, network, address)
	}
	cfg.OnNotice = func(*pgconn.PgConn, *pgconn.Notice) {}
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return nil, targetError(err)
	}
	return conn, nil
}
