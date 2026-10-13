package storecenter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTemuMerchantOwnershipAndCreateFenceIdentitySurviveModeRegionAndCredentialChanges(t *testing.T) {
	supplier := OfficialVerificationFacts{Platform: PlatformTemu, IdentityPartition: PartitionCN, Namespace: MerchantSupplierNamespace, MerchantID: "pa-id", Mode: MerchantFullyManaged}
	first, err := supplier.MerchantIdentityHash()
	require.NoError(t, err)
	supplier.Mode = MerchantSemiManaged
	supplier.ExpiresAt = time.Now().Add(time.Hour)
	supplier.Scopes = []string{"changed.scope"}
	next, err := supplier.MerchantIdentityHash()
	require.NoError(t, err)
	require.Equal(t, first, next)

	local := OfficialVerificationFacts{Platform: PlatformTemu, IdentityPartition: PartitionUS, Namespace: MerchantLocalNamespace, MerchantID: "123", Mode: MerchantLocal, RegionID: 211}
	first, err = local.MerchantIdentityHash()
	require.NoError(t, err)
	local.RegionID = 83 // Multiple official regions can map to the same site.
	next, err = local.MerchantIdentityHash()
	require.NoError(t, err)
	require.Equal(t, first, next)
	local.IdentityPartition = PartitionEU
	next, err = local.MerchantIdentityHash()
	require.NoError(t, err)
	require.NotEqual(t, first, next)
	local.Mode = MerchantSemiManaged
	_, err = local.MerchantIdentityHash()
	require.Error(t, err, "regional SEMI must not obtain a Local identity namespace")
}
