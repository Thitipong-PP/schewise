// SPDX-License-Identifier: Apache-2.0

// Package onetimetoken provides opaque high-entropy tokens and storage digests.
// Callers enforce purpose, expiry, and single consumption transactionally.
package onetimetoken

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
)

func Generate() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func Hash(token string) []byte {
	digest := sha256.Sum256([]byte(token))
	return digest[:]
}

func Valid(token string) bool {
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(token)
	return err == nil && len(decoded) == 32
}

func Matches(token string, digest []byte) bool {
	return Valid(token) && subtle.ConstantTimeCompare(Hash(token), digest) == 1
}
