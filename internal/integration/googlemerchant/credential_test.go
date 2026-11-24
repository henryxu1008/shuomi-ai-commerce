package googlemerchant

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	domain "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/googlemerchant"
)

func TestCredentialProtectionBindsPurposeStoreAndGeneration(t *testing.T) {
	protection, err := NewCredentialProtection("test-key", make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	attempt := domain.CredentialBinding{
		OrganizationID: "org-a", StoreID: "store-a", ApplicationID: "app-a", ApplicationRevision: "r1",
		ID: "attempt-a", Purpose: domain.AttemptCredentials,
	}
	grant := attempt
	grant.Purpose, grant.ID, grant.Generation = domain.GrantCredentials, "grant-a", 1
	tokens := domain.TokenSet{AccessToken: "example-access", RefreshToken: "example-refresh", ExpiresAt: time.Now().UTC()}
	key, ciphertext, err := protection.Seal(attempt, tokens)
	if err != nil || strings.Contains(ciphertext, tokens.AccessToken) {
		t.Fatalf("invalid sealed credential: %v", err)
	}
	opened, err := protection.Open(attempt, key, ciphertext)
	if err != nil || opened != tokens {
		t.Fatalf("invalid restored credential: %v", err)
	}
	otherStore := attempt
	otherStore.StoreID = "store-b"
	otherOrg := attempt
	otherOrg.OrganizationID = "org-b"
	for _, binding := range []domain.CredentialBinding{otherStore, otherOrg, grant} {
		if _, err := protection.Open(binding, key, ciphertext); !errors.Is(err, domain.ErrInvalidGrantInput) {
			t.Fatalf("cross-bound credential read: %v", err)
		}
	}
	key, ciphertext, err = protection.Seal(grant, tokens)
	if err != nil {
		t.Fatal(err)
	}
	grant.Generation++
	if _, err := protection.Open(grant, key, ciphertext); !errors.Is(err, domain.ErrInvalidGrantInput) {
		t.Fatalf("stale grant credential read: %v", err)
	}
	if raw, err := json.Marshal(tokens); err != nil || strings.Contains(string(raw), tokens.RefreshToken) {
		t.Fatalf("credential failed to redact as JSON: %v", err)
	}
}
