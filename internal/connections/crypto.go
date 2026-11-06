// SPDX-License-Identifier: Apache-2.0
package connections

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"

	"github.com/google/uuid"
)

var ErrSecretKey = errors.New("configure SCHEWISE_SECRET_KEY with the original base64-encoded 32-byte encryption key")

type vault struct{ aead cipher.AEAD }

func newVault(encoded string) (vault, error) {
	if encoded == "" {
		return vault{}, nil
	}
	key, err := base64.StdEncoding.Strict().DecodeString(encoded)
	if err != nil || len(key) != 32 {
		return vault{}, ErrSecretKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return vault{}, ErrSecretKey
	}
	aead, err := cipher.NewGCM(block)
	return vault{aead: aead}, err
}
func (v vault) encrypt(id uuid.UUID, password string) ([]byte, []byte, error) {
	if v.aead == nil {
		return nil, nil, ErrSecretKey
	}
	nonce := make([]byte, v.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	return v.aead.Seal(nil, nonce, []byte(password), []byte("schewise.connection:"+id.String())), nonce, nil
}
func (v vault) decrypt(id uuid.UUID, encrypted, nonce []byte) (string, error) {
	if v.aead == nil || len(nonce) != v.aead.NonceSize() {
		return "", ErrSecretKey
	}
	plain, err := v.aead.Open(nil, nonce, encrypted, []byte("schewise.connection:"+id.String()))
	if err != nil {
		return "", ErrSecretKey
	}
	return string(plain), nil
}
