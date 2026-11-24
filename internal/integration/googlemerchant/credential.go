package googlemerchant

import (
	"encoding/json"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/credentialprotection"
	domain "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/googlemerchant"
)

type CredentialProtection struct {
	encryption *credentialprotection.AESGCM
}

func NewCredentialProtection(keyID string, key []byte) (*CredentialProtection, error) {
	if keyID == "" || len(keyID) > 128 || len(key) != 32 {
		return nil, domain.ErrInvalidGrantInput
	}
	encryption, err := credentialprotection.NewAESGCM(keyID, key)
	if err != nil {
		return nil, domain.ErrInvalidGrantInput
	}
	return &CredentialProtection{encryption: encryption}, nil
}

type tokenEnvelope struct {
	Access, Refresh string
	Expiry          time.Time
}

func (p *CredentialProtection) Seal(binding domain.CredentialBinding, token domain.TokenSet) (string, string, error) {
	aad, err := binding.AssociatedData()
	if err != nil || p == nil || !usableEnvelope(token) {
		return "", "", domain.ErrInvalidGrantInput
	}
	raw, err := json.Marshal(tokenEnvelope{Access: token.AccessToken, Refresh: token.RefreshToken, Expiry: token.ExpiresAt})
	if err != nil {
		return "", "", domain.ErrInvalidGrantInput
	}
	defer clear(raw)
	ciphertext, err := p.encryption.Seal(raw, aad)
	if err != nil {
		return "", "", domain.ErrInvalidGrantInput
	}
	return p.encryption.KeyID(), ciphertext, nil
}

func (p *CredentialProtection) Open(binding domain.CredentialBinding, keyID, ciphertext string) (domain.TokenSet, error) {
	aad, err := binding.AssociatedData()
	if err != nil || p == nil {
		return domain.TokenSet{}, domain.ErrInvalidGrantInput
	}
	raw, err := p.encryption.Open(keyID, ciphertext, aad)
	if err != nil {
		return domain.TokenSet{}, domain.ErrInvalidGrantInput
	}
	defer clear(raw)
	var envelope tokenEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return domain.TokenSet{}, domain.ErrInvalidGrantInput
	}
	token := domain.TokenSet{AccessToken: envelope.Access, RefreshToken: envelope.Refresh, ExpiresAt: envelope.Expiry}
	if !usableEnvelope(token) {
		return domain.TokenSet{}, domain.ErrInvalidGrantInput
	}
	return token, nil
}

func usableEnvelope(token domain.TokenSet) bool {
	return validToken(token.AccessToken) && validToken(token.RefreshToken) && !token.ExpiresAt.IsZero()
}
