package shopify

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/credentialprotection"
)

const pairingLifetime = 10 * time.Minute

var ErrPairingCredential = errors.New("Shopify pairing credential is unavailable")

// PairingBinding is the immutable context of a staged, online owner token.
// SessionDigest is a server-derived digest, not the session cookie.
type PairingBinding struct {
	OrganizationID string
	ActorID        string
	MemberID       string
	StoreID        string
	AttemptID      string
	SessionDigest  string
	Installation   Installation
	Generation     int64
	ExpiresAt      time.Time
}

type PairingProtection struct {
	encryption *credentialprotection.AESGCM
}

func NewPairingProtection(keyID string, key []byte) (*PairingProtection, error) {
	cipher, err := credentialprotection.NewAESGCM(keyID, key)
	if err != nil {
		return nil, ErrPairingCredential
	}
	return &PairingProtection{encryption: cipher}, nil
}

// Seal stores only the online credential needed for the confirmation check.
// Store Center must enforce proof consumption, revocation and ciphertext disposal.
func (p *PairingProtection) Seal(binding PairingBinding, token OwnerAccessToken, now time.Time) (keyID, ciphertext string, err error) {
	binding.ExpiresAt = binding.ExpiresAt.UTC().Truncate(time.Microsecond)
	aad, ok := binding.associatedData(now)
	if p == nil || p.encryption == nil || !ok || !validToken(token.Value) || token.ExpiresAt.Before(binding.ExpiresAt) {
		return "", "", ErrPairingCredential
	}
	plain := []byte(token.Value)
	defer clear(plain)
	ciphertext, err = p.encryption.Seal(plain, aad)
	if err != nil {
		return "", "", ErrPairingCredential
	}
	return p.encryption.KeyID(), ciphertext, nil
}

func (p *PairingProtection) Open(binding PairingBinding, keyID, ciphertext string, now time.Time) (OwnerAccessToken, error) {
	binding.ExpiresAt = binding.ExpiresAt.UTC().Truncate(time.Microsecond)
	aad, ok := binding.associatedData(now)
	if p == nil || p.encryption == nil || !ok {
		return OwnerAccessToken{}, ErrPairingCredential
	}
	raw, err := p.encryption.Open(keyID, ciphertext, aad)
	if err != nil {
		return OwnerAccessToken{}, ErrPairingCredential
	}
	defer clear(raw)
	token := string(raw)
	if !validToken(token) {
		return OwnerAccessToken{}, ErrPairingCredential
	}
	return OwnerAccessToken{AccessToken{Value: token, ExpiresAt: binding.ExpiresAt}}, nil
}

func (b PairingBinding) associatedData(now time.Time) ([]byte, bool) {
	if now.IsZero() || !b.ExpiresAt.After(now) || b.ExpiresAt.Sub(now) > pairingLifetime ||
		!authidentity.IsBoundedIdentifier(b.OrganizationID) || !authidentity.IsBoundedIdentifier(b.ActorID) || !authidentity.IsBoundedIdentifier(b.MemberID) ||
		uuid.Validate(b.StoreID) != nil || uuid.Validate(b.AttemptID) != nil || !sourceKeyPattern.MatchString(b.SessionDigest) || b.Generation < 0 {
		return nil, false
	}
	installation := b.Installation
	if !validGID(installation.AppID, "App") || !validGID(installation.ShopID, "Shop") || !validGID(installation.ID, "AppInstallation") ||
		!shopDomainPattern.MatchString(installation.Domain) || !currencyPattern.MatchString(installation.Currency) {
		return nil, false
	}
	wire, err := json.Marshal(struct {
		Purpose string
		PairingBinding
	}{"storecenter/shopify/owner-pairing/v1", b})
	return wire, err == nil
}
