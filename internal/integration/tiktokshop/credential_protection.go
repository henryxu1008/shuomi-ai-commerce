package tiktokshop

import (
	"encoding/json"
	"time"

	"task-processor/internal/integration/credentialprotection"
	authorization "task-processor/internal/storecenter/tiktokshop"
)

type CredentialProtection struct{ encryption *credentialprotection.AESGCM }

var _ authorization.CredentialProtection = (*CredentialProtection)(nil)

func NewCredentialProtection(keyID string, key []byte) (*CredentialProtection, error) {
	encryption, err := credentialprotection.NewAESGCM(keyID, key)
	if err != nil {
		return nil, authorization.ErrUnavailable
	}
	return &CredentialProtection{encryption}, nil
}

// Only this private envelope serializes secrets for encryption. TokenSet's
// normal JSON projection never contains them. Reuse the current AES-GCM owner.
type credentialEnvelope struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	OpenID           string    `json:"open_id"`
	UserType         int       `json:"user_type"`
	Scopes           []string  `json:"scopes"`
	SellerBaseRegion string    `json:"seller_base_region"`
	AccessExpiry     time.Time `json:"access_expiry"`
	RefreshExpiry    time.Time `json:"refresh_expiry"`
}

func (p *CredentialProtection) Seal(binding authorization.CredentialBinding, tokens authorization.TokenSet) (string, string, error) {
	aad, err := binding.AssociatedData()
	if err != nil || p == nil || !validCredential(tokens) {
		return "", "", authorization.ErrUnavailable
	}
	raw, err := json.Marshal(credentialEnvelope{tokens.AccessToken, tokens.RefreshToken, tokens.OpenID, tokens.UserType, tokens.GrantedScopes, tokens.SellerBaseRegion, tokens.AccessExpiresAt, tokens.RefreshExpiresAt})
	if err != nil {
		return "", "", authorization.ErrUnavailable
	}
	defer clear(raw)
	sealed, err := p.encryption.Seal(raw, aad)
	if err != nil {
		return "", "", authorization.ErrUnavailable
	}
	return p.encryption.KeyID(), sealed, nil
}

func (p *CredentialProtection) Open(binding authorization.CredentialBinding, keyID, ciphertext string) (authorization.TokenSet, error) {
	aad, err := binding.AssociatedData()
	if err != nil || p == nil {
		return authorization.TokenSet{}, authorization.ErrUnavailable
	}
	raw, err := p.encryption.Open(keyID, ciphertext, aad)
	if err != nil {
		return authorization.TokenSet{}, authorization.ErrUnavailable
	}
	defer clear(raw)
	var envelope credentialEnvelope
	if !decode(raw, &envelope) {
		return authorization.TokenSet{}, authorization.ErrUnavailable
	}
	tokens := authorization.TokenSet{AccessToken: envelope.AccessToken, RefreshToken: envelope.RefreshToken, OpenID: envelope.OpenID, UserType: envelope.UserType, GrantedScopes: envelope.Scopes, SellerBaseRegion: envelope.SellerBaseRegion, AccessExpiresAt: envelope.AccessExpiry, RefreshExpiresAt: envelope.RefreshExpiry}
	if !validCredential(tokens) {
		return authorization.TokenSet{}, authorization.ErrUnavailable
	}
	return tokens, nil
}

func validCredential(tokens authorization.TokenSet) bool {
	if !bounded(tokens.AccessToken, 8192) || !bounded(tokens.RefreshToken, 8192) || !bounded(tokens.OpenID, 512) ||
		tokens.UserType != 0 && tokens.UserType != 4 && tokens.UserType != 5 || tokens.AccessExpiresAt.IsZero() || tokens.RefreshExpiresAt.IsZero() || len(tokens.GrantedScopes) == 0 || len(tokens.GrantedScopes) > 128 {
		return false
	}
	seen := map[string]bool{}
	for _, scope := range tokens.GrantedScopes {
		if !bounded(scope, 128) || seen[scope] {
			return false
		}
		seen[scope] = true
	}
	return seen["seller.authorization.info"]
}
