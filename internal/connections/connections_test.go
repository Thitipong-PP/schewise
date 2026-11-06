// SPDX-License-Identifier: Apache-2.0
package connections

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/netip"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

const testKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

func TestEncryptionBindsRecordAndUsesFreshNonce(t *testing.T) {
	v, err := newVault(testKey)
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.New()
	secret := "Synthetic-target-password-42!"
	a, n, err := v.encrypt(id, secret)
	if err != nil {
		t.Fatal(err)
	}
	b, m, err := v.encrypt(id, secret)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(n, m) || bytes.Equal(a, b) || bytes.Contains(a, []byte(secret)) {
		t.Fatal("ciphertext or nonce reused/plaintext stored")
	}
	plain, err := v.decrypt(id, a, n)
	if err != nil || plain != secret {
		t.Fatal("round trip failed")
	}
	wrong, _ := newVault(base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32)))
	empty, _ := newVault("")
	for _, key := range []vault{wrong, empty} {
		if _, err := key.decrypt(id, a, n); !errors.Is(err, ErrSecretKey) {
			t.Fatal("wrong/missing key accepted")
		}
	}
	if _, err := v.decrypt(uuid.New(), a, n); !errors.Is(err, ErrSecretKey) {
		t.Fatal("record substitution accepted")
	}
	a[0] ^= 1
	if _, err := v.decrypt(id, a, n); !errors.Is(err, ErrSecretKey) {
		t.Fatal("tampering accepted")
	}
	for _, key := range []string{"bad", base64.StdEncoding.EncodeToString(make([]byte, 31))} {
		if _, err := newVault(key); !errors.Is(err, ErrSecretKey) {
			t.Fatal("invalid key accepted")
		}
	}
}
func TestStructuralHashIndependentOfOrderingAndStatistics(t *testing.T) {
	a := Content{Schemas: []Schema{{Name: "z"}, {Name: "a", Tables: []Table{{Name: "z"}, {Name: "a", Columns: []Column{{Name: "b", Ordinal: 2}, {Name: "a", Ordinal: 1}}, Constraints: []Constraint{{Name: "z"}, {Name: "a"}}, Indexes: []Index{{Name: "z"}, {Name: "a"}}}}}}}
	raw, hash, err := encodeContent(&a)
	if err != nil {
		t.Fatal(err)
	}
	var b Content
	_ = json.Unmarshal(raw, &b)
	b.Schemas[0], b.Schemas[1] = b.Schemas[1], b.Schemas[0]
	b.Schemas[1].Tables[0], b.Schemas[1].Tables[1] = b.Schemas[1].Tables[1], b.Schemas[1].Tables[0]
	_, same, _ := encodeContent(&b)
	if same != hash {
		t.Fatal("ordering changed hash")
	}
	comment := "Updated comment"
	b.Schemas[0].Comment = &comment
	_, changed, _ := encodeContent(&b)
	if changed == hash {
		t.Fatal("comment omitted from hash")
	}
	if !containsSecret(Content{Schemas: []Schema{{Name: "public", Comment: &comment}}}, "Updated") {
		t.Fatal("secret in metadata not detected")
	}
	if containsSecret(a, "comment") {
		t.Fatal("JSON key incorrectly treated as a secret")
	}
}
func TestSafeErrorsAndDestinationPolicy(t *testing.T) {
	for _, tc := range []struct{ code, want string }{{"28P01", "auth_failed"}, {"3D000", "database_not_found"}, {"42501", "permission_denied"}, {"57014", "timeout"}} {
		safe := targetError(&pgconn.PgError{Code: tc.code, Message: "password=secret", Detail: "secret"})
		if safe.Code != tc.want || bytes.Contains([]byte(safe.Error()), []byte("secret")) {
			t.Fatal("unsafe error mapping")
		}
	}
	if targetError(context.DeadlineExceeded).Code != "timeout" {
		t.Fatal("deadline mapping")
	}
	if targetError(errors.New("password=secret")).Code != "cannot_reach_host" {
		t.Fatal("network mapping")
	}
	p, err := newHostPolicy([]string{"db.example.test", "10.0.0.0/8"}, []string{"10.1.0.0/16"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		host, ip string
		want     bool
	}{{"db.example.test", "192.0.2.1", true}, {"other.test", "10.2.0.1", true}, {"db.example.test", "10.1.0.1", false}, {"other.test", "192.0.2.1", false}} {
		if p.permits(tc.host, netip.MustParseAddr(tc.ip)) != tc.want {
			t.Fatal("host policy mismatch")
		}
	}
	for _, host := range []string{"/var/run/postgresql", "db1,db2", "host=secret", "db:5432", "", "localhost?password=x"} {
		if validHost(host) {
			t.Fatal("unsafe host accepted")
		}
	}
}
