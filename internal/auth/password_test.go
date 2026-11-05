package auth

import (
	"strings"
	"testing"
)

func TestArgon2idPasswordRoundTrip(t *testing.T) {
	password := "Synthetic-test-password-42!"
	first, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	second, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("password hashes must use different salts")
	}
	if !strings.HasPrefix(first, "$argon2id$v=19$") {
		t.Fatal("not an Argon2id PHC hash")
	}
	if !VerifyPassword(password, first) {
		t.Fatal("correct password rejected")
	}
	if VerifyPassword("Incorrect-password-42!", first) {
		t.Fatal("incorrect password accepted")
	}
	for _, hash := range []string{"", password, strings.Replace(first, "m=65536", "m=4294967295", 1), strings.Replace(first, "v=19", "v=18", 1), first + "!"} {
		if VerifyPassword(password, hash) {
			t.Fatal("malformed or unbounded hash accepted")
		}
	}
}

func TestPasswordInputLimits(t *testing.T) {
	for _, password := range []string{"", "short", strings.Repeat("x", 257), string([]byte{0xff, 0xff})} {
		if _, err := HashPassword(password); err == nil {
			t.Fatal("invalid password accepted")
		}
	}
}

func TestNewPasswordPolicy(t *testing.T) {
	for _, password := range []string{"Password1234", "123456789012", strings.Repeat("a", 129), strings.Repeat("😀", 129)} {
		if ValidPassword(password) {
			t.Fatal("weak or oversized new password accepted")
		}
	}
	for _, password := range []string{strings.Repeat("a", 12), strings.Repeat("a", 128), strings.Repeat("😀", 128), "a long phrase with spaces"} {
		if !ValidPassword(password) {
			t.Fatal("valid password rejected by composition/length rules")
		}
	}
}
