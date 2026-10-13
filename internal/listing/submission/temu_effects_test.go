package submission

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

func temuExecutionBinding(mode storecenter.OfficialMerchantMode) storecenter.ProductMerchantBinding {
	b := storecenter.ProductMerchantBinding{Platform: storecenter.PlatformTemu, OrganizationID: "org-a", StoreID: uuid.NewString(), MerchantMode: mode, IdentityPartition: storecenter.PartitionCN, Site: "temu-supplier", ApplicationID: "app-a", ApplicationRevision: "v1:temu:CN:manual_token:6:r1", StoreVersion: 1, ConnectionRevision: 1, SupplierIdentityHash: collection.Digest("merchant-a"), ScopesHash: collection.Digest("scopes"), VerificationRevision: "official-v1", ServiceExpiresAt: time.Now().Add(time.Hour), CredentialExpiresAt: time.Now().Add(time.Hour)}
	if mode == storecenter.MerchantLocal {
		b.IdentityPartition = storecenter.PartitionEU
		b.RegionID = 20
		b.Site = "temu-local"
		b.ApplicationRevision = "v1:temu:EU:manual_token:1:r1"
	}
	return b
}

func TestTemuProductFenceUsesVerifiedMerchantAndCanonicalProductOnly(t *testing.T) {
	b := temuExecutionBinding(storecenter.MerchantFullyManaged)
	first, err := TemuProductSubjectID("product-a", b)
	require.NoError(t, err)
	b.MerchantMode = storecenter.MerchantSemiManaged
	b.ApplicationID = "new-app"
	b.ApplicationRevision = "v2:temu:CN:manual_token:6:r1"
	b.ConnectionRevision++
	second, err := TemuProductSubjectID("product-a", b)
	require.NoError(t, err)
	require.Equal(t, first, second)
	b.SupplierIdentityHash = collection.Digest("another-merchant")
	other, err := TemuProductSubjectID("product-a", b)
	require.NoError(t, err)
	require.NotEqual(t, first, other)
	local := temuExecutionBinding(storecenter.MerchantLocal)
	before, err := TemuProductSubjectID("product-a", local)
	require.NoError(t, err)
	local.RegionID = 75
	after, err := TemuProductSubjectID("product-a", local)
	require.NoError(t, err)
	require.Equal(t, before, after, "a changed site profile cannot reset a local product effect")
}

func TestTemuLocalImageFenceBindsExactBytesMerchantCategoryAndUsage(t *testing.T) {
	b := temuExecutionBinding(storecenter.MerchantLocal)
	hash := collection.Digest("exact bytes")
	first, err := TemuLocalImageSubjectID(hash, 34, 3, b)
	require.NoError(t, err)
	second, err := TemuLocalImageSubjectID(hash, 35, 3, b)
	require.NoError(t, err)
	require.NotEqual(t, first, second)
	second, err = TemuLocalImageSubjectID(hash, 34, 2, b)
	require.NoError(t, err)
	require.NotEqual(t, first, second)
	b.ApplicationID = "new-app"
	b.ApplicationRevision = "v2:temu:EU:manual_token:1:r1"
	second, err = TemuLocalImageSubjectID(hash, 34, 3, b)
	require.NoError(t, err)
	require.Equal(t, first, second)
	_, err = TemuLocalImageSubjectID(hash, 0, 3, b)
	require.Error(t, err)
}
