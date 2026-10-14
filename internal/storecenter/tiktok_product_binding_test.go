package storecenter

import (
	"strings"
	"testing"
	"time"
)

func TestTikTokProductBindingRequiresExactCanonicalGrantAndUSShop(t *testing.T) {
	value := ProductMerchantBinding{Platform: PlatformTikTokShop, OrganizationID: "org", StoreID: "00000000-0000-4000-8000-000000000001", Site: "tiktokshop-us", ApplicationID: "app", ApplicationRevision: "r1", SupplierIdentityHash: strings.Repeat("a", 64), ScopesHash: strings.Repeat("b", 64), CredentialExpiresAt: time.Now().Add(time.Hour), TikTok: TikTokMerchantBinding{GrantID: "00000000-0000-4000-8000-000000000002", ShopID: "shop", GrantGeneration: 2, GrantRevision: 4, BindingRevision: 1}}
	if !value.ValidApplication() {
		t.Fatal("valid US canonical binding rejected")
	}
	for _, change := range []func(*ProductMerchantBinding){func(v *ProductMerchantBinding) { v.Site = "tiktokshop-uk" }, func(v *ProductMerchantBinding) { v.TikTok.GrantRevision = 1 }, func(v *ProductMerchantBinding) { v.ApplicationType = ApplicationSelfOperated }, func(v *ProductMerchantBinding) { v.TikTok.ShopID = "" }, func(v *ProductMerchantBinding) { v.IdentityPartition = PartitionUS }} {
		v := value
		change(&v)
		if v.ValidApplication() {
			t.Fatal("ambiguous TikTok authority accepted")
		}
	}
}
