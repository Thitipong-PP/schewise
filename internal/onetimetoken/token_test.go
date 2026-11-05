// SPDX-License-Identifier: Apache-2.0
package onetimetoken

import (
	"bytes"
	"strings"
	"testing"
)

func TestTokens(t *testing.T) {
	a, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	b, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	if a == b || !Valid(a) || len(a) != 43 {
		t.Fatal("invalid random token")
	}
	digest := Hash(a)
	if len(digest) != 32 || bytes.Equal(digest, []byte(a)) || !Matches(a, digest) || Matches(b, digest) {
		t.Fatal("digest verification failed")
	}
	for _, token := range []string{"", a + "=", a + "/", strings.Repeat("a", 42), strings.Repeat("a", 44)} {
		if Matches(token, digest) || Valid(token) {
			t.Fatal("malformed token accepted")
		}
	}
}
