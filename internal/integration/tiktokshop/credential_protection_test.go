package tiktokshop

import (
	"strings"
	"testing"

	authorization "task-processor/internal/storecenter/tiktokshop"
)

func TestCredentialCiphertextBindsOrgAppPurposeAndGeneration(t *testing.T) {
	protection, err := NewCredentialProtection("fixture-key", []byte(strings.Repeat("k", 32)))
	if err != nil {
		t.Fatal(err)
	}
	binding := authorization.CredentialBinding{OrganizationID: "org-a", AppKey: "app-a", AppRevision: "r1", Purpose: "grant", ID: "00000000-0000-4000-8000-000000000001", Generation: 1}
	tokens := authorization.TokenSet{AccessToken: "fixture-access", RefreshToken: "fixture-refresh", OpenID: "seller-a", UserType: 5, GrantedScopes: []string{"seller.authorization.info"}, AccessExpiresAt: fixtureTime.AddDate(0, 0, 1), RefreshExpiresAt: fixtureTime.AddDate(0, 0, 7)}
	key, sealed, err := protection.Seal(binding, tokens)
	if err != nil || strings.Contains(sealed, "fixture-") {
		t.Fatal("credential encryption failed")
	}
	opened, err := protection.Open(binding, key, sealed)
	if err != nil || opened.AccessToken != tokens.AccessToken || opened.RefreshToken != tokens.RefreshToken || opened.OpenID != tokens.OpenID || !opened.AccessExpiresAt.Equal(tokens.AccessExpiresAt) {
		t.Fatal("credential round trip failed")
	}
	for _, field := range []string{"org", "app", "revision", "id", "generation", "purpose"} {
		other := binding
		switch field {
		case "org":
			other.OrganizationID = "org-b"
		case "app":
			other.AppKey = "app-b"
		case "revision":
			other.AppRevision = "r2"
		case "id":
			other.ID = "00000000-0000-4000-8000-000000000002"
		case "generation":
			other.Generation++
		case "purpose":
			other.Purpose = "attempt"
			other.Generation = 0
		}
		if _, err := protection.Open(other, key, sealed); err == nil {
			t.Fatalf("%s mismatch accepted", field)
		}
	}
	if _, err := protection.Open(binding, "other-key", sealed); err == nil {
		t.Fatal("wrong key accepted")
	}
}
