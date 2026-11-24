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

// InstallProofBinding is the separate AAD purpose for Shopify-first OAuth.
// Local member and Store authority is deliberately absent until confirmation.
type InstallProofBinding struct {
	AttemptID, SessionDigest string
	Installation             Installation
	Generation               int64
	ExpiresAt                time.Time
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
	aad, _ := binding.associatedData(now)
	return p.seal(aad, token, binding.ExpiresAt)
}

func (p *PairingProtection) SealInstallProof(binding InstallProofBinding, token OwnerAccessToken, now time.Time) (string, string, error) {
	binding.ExpiresAt = binding.ExpiresAt.UTC().Truncate(time.Microsecond)
	return p.seal(binding.associatedData(now), token, binding.ExpiresAt)
}

func (p *PairingProtection) seal(aad []byte, token OwnerAccessToken, expires time.Time) (string, string, error) {
	if p == nil || p.encryption == nil || len(aad) == 0 || !validToken(token.Value) || token.ExpiresAt.Before(expires) {
		return "", "", ErrPairingCredential
	}
	plain := []byte(token.Value)
	defer clear(plain)
	ciphertext, err := p.encryption.Seal(plain, aad)
	if err != nil {
		return "", "", ErrPairingCredential
	}
	return p.encryption.KeyID(), ciphertext, nil
}

func (p *PairingProtection) Open(binding PairingBinding, keyID, ciphertext string, now time.Time) (OwnerAccessToken, error) {
	binding.ExpiresAt = binding.ExpiresAt.UTC().Truncate(time.Microsecond)
	aad, _ := binding.associatedData(now)
	return p.open(aad, keyID, ciphertext, binding.ExpiresAt)
}

func (p *PairingProtection) OpenInstallProof(binding InstallProofBinding, keyID, ciphertext string, now time.Time) (OwnerAccessToken, error) {
	binding.ExpiresAt = binding.ExpiresAt.UTC().Truncate(time.Microsecond)
	return p.open(binding.associatedData(now), keyID, ciphertext, binding.ExpiresAt)
}

func (p *PairingProtection) open(aad []byte, keyID, ciphertext string, expires time.Time) (OwnerAccessToken, error) {
	if p == nil || p.encryption == nil || len(aad) == 0 {
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
	return OwnerAccessToken{AccessToken{Value: token, ExpiresAt: expires}}, nil
}

func (b PairingBinding) associatedData(now time.Time) ([]byte, bool) {
	if now.IsZero() || !b.ExpiresAt.After(now) || b.ExpiresAt.Sub(now) > pairingLifetime ||
		!authidentity.IsBoundedIdentifier(b.OrganizationID) || !authidentity.IsBoundedIdentifier(b.ActorID) || !authidentity.IsBoundedIdentifier(b.MemberID) ||
		uuid.Validate(b.StoreID) != nil || uuid.Validate(b.AttemptID) != nil || !sourceKeyPattern.MatchString(b.SessionDigest) || b.Generation < 0 {
		return nil, false
	}
	if !validCredentialInstallation(b.Installation) {
		return nil, false
	}
	wire, err := json.Marshal(struct {
		Purpose string
		PairingBinding
	}{"storecenter/shopify/owner-pairing/v1", b})
	return wire, err == nil
}

func (b InstallProofBinding) associatedData(now time.Time) []byte {
	if now.IsZero() || !b.ExpiresAt.After(now) || b.ExpiresAt.Sub(now) > pairingLifetime ||
		uuid.Validate(b.AttemptID) != nil || !sourceKeyPattern.MatchString(b.SessionDigest) || b.Generation < 0 || !validCredentialInstallation(b.Installation) {
		return nil
	}
	wire, err := json.Marshal(struct {
		Purpose string
		InstallProofBinding
	}{"storecenter/shopify/unbound-install-proof/v1", b})
	if err != nil {
		return nil
	}
	return wire
}

func validCredentialInstallation(v Installation) bool {
	return validGID(v.AppID, "App") && validGID(v.ShopID, "Shop") && validGID(v.ID, "AppInstallation") &&
		shopDomainPattern.MatchString(v.Domain) && currencyPattern.MatchString(v.Currency)
}
