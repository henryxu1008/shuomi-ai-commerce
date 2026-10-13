// Package credentialprotection contains the existing official-credential
// encryption behavior. Providers own typed envelopes and associated data;
// this package knows no merchant or connection lifecycle.
package credentialprotection

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

const maxCiphertext = 16384

var errProtection = errors.New("official credential protection unavailable")

type AESGCM struct {
	keyID string
	aead  cipher.AEAD
}

func NewAESGCM(keyID string, key []byte) (*AESGCM, error) {
	if len(key) != 32 || len(keyID) < 1 || len(keyID) > 128 || strings.TrimSpace(keyID) != keyID {
		return nil, errProtection
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errProtection
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errProtection
	}
	return &AESGCM{keyID: keyID, aead: aead}, nil
}
func (p *AESGCM) KeyID() string { return p.keyID }
func (p *AESGCM) Seal(plain, aad []byte) (string, error) {
	if len(plain) == 0 || base64.StdEncoding.EncodedLen(p.aead.NonceSize()+len(plain)+p.aead.Overhead()) > maxCiphertext {
		return "", errProtection
	}
	nonce := make([]byte, p.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", errProtection
	}
	sealed := p.aead.Seal(nonce, nonce, plain, aad)
	return base64.StdEncoding.EncodeToString(sealed), nil
}
func (p *AESGCM) Open(keyID, ciphertext string, aad []byte) ([]byte, error) {
	if keyID != p.keyID || len(ciphertext) > maxCiphertext {
		return nil, errProtection
	}
	data, err := base64.StdEncoding.Strict().DecodeString(ciphertext)
	if err != nil || len(data) < p.aead.NonceSize()+p.aead.Overhead() {
		return nil, errProtection
	}
	plain, err := p.aead.Open(nil, data[:p.aead.NonceSize()], data[p.aead.NonceSize():], aad)
	if err != nil {
		return nil, errProtection
	}
	return plain, nil
}
