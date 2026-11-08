package storecenter

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestRegionalTemuOrderIdentityCannotUseSupplierProductAuthority(t *testing.T) {
	f := OfficialVerificationFacts{Platform: PlatformTemu, IdentityPartition: PartitionGlobal, Namespace: MerchantRegionalSemiNamespace, MerchantID: "123", SemiUniqueID: "semi-a", Mode: MerchantSemiManaged, RegionID: 211}
	h, e := f.MerchantIdentityHash()
	require.NoError(t, e)
	f.SemiUniqueID = ""
	_, e = f.MerchantIdentityHash()
	require.Error(t, e)
	b := ProductMerchantBinding{Platform: PlatformTemu, IdentityPartition: PartitionGlobal, MerchantMode: MerchantSemiManaged, RegionID: 211, Site: "temu-order-us", ApplicationID: "app", ApplicationRevision: "v1:temu:GLOBAL:manual_token:3:r2", VerificationRevision: "temu-regional-order-identity-v2", SupplierIdentityHash: h, ScopesHash: strings.Repeat("a", 64), CredentialExpiresAt: time.Now().Add(time.Hour)}
	require.True(t, b.ValidTemuOrderApplication())
	require.False(t, b.ValidTemuApplication())
	b.Site = "temu-supplier"
	require.False(t, b.ValidTemuOrderApplication())
	b.Site = "temu-order-us"
	b.IdentityPartition = PartitionCN
	require.False(t, b.ValidTemuOrderApplication())
}
