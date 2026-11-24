package shopify

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestPairingCredentialCannotBeMovedBetweenAuthorizations(t *testing.T) {
	now := time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC)
	protection, err := NewPairingProtection("fixture-key", bytes.Repeat([]byte{21}, 32))
	if err != nil {
		t.Fatal(err)
	}
	binding := PairingBinding{
		OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a",
		StoreID: "887ef9cb-6e29-4995-9094-9eec57d41507", AttemptID: "8b6f6912-6e29-4995-9094-9eec57d41507",
		SessionDigest: fixtureSourceKey, Installation: productInstallation(), Generation: 2, ExpiresAt: now.Add(5 * time.Minute),
	}
	credential := OwnerAccessToken{AccessToken{Value: "fixture-owner-token", ExpiresAt: now.Add(time.Hour)}}
	key, sealed, err := protection.Seal(binding, credential, now)
	if err != nil || key != "fixture-key" || bytes.Contains([]byte(sealed), []byte(credential.Value)) {
		t.Fatalf("sealing pairing credential: %v", err)
	}
	got, err := protection.Open(binding, key, sealed, now.Add(time.Minute))
	if err != nil || got.Value != credential.Value || !got.ExpiresAt.Equal(binding.ExpiresAt) {
		t.Fatalf("expected opened credential with pairing expiry, got %v", err)
	}
	for _, change := range []func(*PairingBinding){
		func(b *PairingBinding) { b.OrganizationID = "org-b" },
		func(b *PairingBinding) { b.MemberID = "member-b" },
		func(b *PairingBinding) { b.ActorID = "actor-b" },
		func(b *PairingBinding) { b.StoreID = "a667f9cb-6e29-4995-9094-9eec57d41507" },
		func(b *PairingBinding) { b.AttemptID = "a667f9cb-6e29-4995-9094-9eec57d41507" },
		func(b *PairingBinding) {
			b.SessionDigest = "442d8383cde574aacd35cc667d0da6248521414e9f256a6c4502e72d8f037dc0"
		},
		func(b *PairingBinding) { b.Installation.ID = "gid://shopify/AppInstallation/41" },
		func(b *PairingBinding) { b.Installation.AppID = "gid://shopify/App/40" },
		func(b *PairingBinding) { b.Installation.ShopID = "gid://shopify/Shop/20" },
		func(b *PairingBinding) { b.Generation++ },
		func(b *PairingBinding) { b.ExpiresAt = b.ExpiresAt.Add(time.Second) },
	} {
		other := binding
		change(&other)
		if _, err := protection.Open(other, key, sealed, now); !errors.Is(err, ErrPairingCredential) {
			t.Fatalf("different pairing opened a credential: %v", err)
		}
	}
	if _, err := protection.Open(binding, key, sealed, binding.ExpiresAt); !errors.Is(err, ErrPairingCredential) {
		t.Fatalf("expired pairing opened a credential: %v", err)
	}
	if _, err := protection.Open(binding, "other-key", sealed, now); !errors.Is(err, ErrPairingCredential) {
		t.Fatalf("wrong key opened a credential: %v", err)
	}
}
