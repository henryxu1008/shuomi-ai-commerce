package walmart

import (
	"encoding/json"
	"time"

	"task-processor/internal/integration/credentialprotection"
	auth "task-processor/internal/storecenter/walmart"
)

type CredentialProtection struct {
	encryption *credentialprotection.AESGCM
}

func NewCredentialProtection(keyID string, key []byte) (*CredentialProtection, error) {
	if keyID == "" || len(keyID) > 128 || len(key) != 32 {
		return nil, auth.ErrInvalid
	}
	b, e := credentialprotection.NewAESGCM(keyID, key)
	if e != nil {
		return nil, auth.ErrInvalid
	}
	return &CredentialProtection{b}, nil
}

type tokenEnvelope struct {
	Access, Refresh, Seller     string
	AccessExpiry, RefreshExpiry time.Time
}

func (p *CredentialProtection) Seal(b auth.CredentialBinding, t auth.TokenSet) (string, string, error) {
	aad, e := b.AssociatedData()
	if e != nil || t.AccessToken == "" || t.RefreshToken == "" || t.SellerID == "" {
		return "", "", auth.ErrInvalid
	}
	raw, e := json.Marshal(tokenEnvelope{t.AccessToken, t.RefreshToken, t.SellerID, t.AccessExpiresAt, t.RefreshExpiresAt})
	if e != nil || len(raw) > 16000 {
		return "", "", auth.ErrInvalid
	}
	defer clear(raw)
	sealed, e := p.encryption.Seal(raw, aad)
	if e != nil {
		return "", "", auth.ErrInvalid
	}
	return p.encryption.KeyID(), sealed, nil
}
func (p *CredentialProtection) Open(b auth.CredentialBinding, key, ciphertext string) (auth.TokenSet, error) {
	var out auth.TokenSet
	aad, e := b.AssociatedData()
	if e != nil {
		return out, auth.ErrInvalid
	}
	plain, e := p.encryption.Open(key, ciphertext, aad)
	if e != nil {
		return out, auth.ErrInvalid
	}
	defer clear(plain)
	var v tokenEnvelope
	if json.Unmarshal(plain, &v) != nil || v.Access == "" || v.Refresh == "" || v.Seller == "" {
		return out, auth.ErrInvalid
	}
	return auth.TokenSet{AccessToken: v.Access, RefreshToken: v.Refresh, SellerID: v.Seller, AccessExpiresAt: v.AccessExpiry, RefreshExpiresAt: v.RefreshExpiry}, nil
}
