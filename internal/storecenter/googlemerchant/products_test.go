package googlemerchant

import (
	"errors"
	"testing"
)

func TestStableOfferIdentityUsesReservedNamespace(t *testing.T) {
	target := Target{
		AccountID: "123", DataSourceID: "456", ContentLanguage: "en",
		FeedLabel: "US", OfferPrefix: "sm-0123456789abcdef0123-",
	}
	first, err := target.OfferID("product-17:variant-blue")
	if err != nil || !target.OwnsOfferID(first) || len(first) > 50 {
		t.Fatalf("invalid Merchant offer ID %q: %v", first, err)
	}
	otherSource := target
	otherSource.DataSourceID = "789"
	if same, err := otherSource.OfferID("product-17:variant-blue"); err != nil || same != first {
		t.Fatalf("source changes must not produce another ID: %q %v", same, err)
	}
	second, err := target.OfferID("product-17:variant-red")
	if err != nil || second == first {
		t.Fatalf("different variants must be distinct: %q %v", second, err)
	}
	otherNamespace := target
	otherNamespace.OfferPrefix = "sm-fedcba9876543210fedc-"
	if otherNamespace.OwnsOfferID(first) {
		t.Fatal("another namespace may not send the original offer ID")
	}
	if _, err := target.OfferID(""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty variant identity accepted: %v", err)
	}
}
