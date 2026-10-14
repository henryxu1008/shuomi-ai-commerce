package supplychainapp

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/authidentity"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

func TestPublicationReadUsesConfirmedOwnerReceiptWithoutAnotherSend(t *testing.T) {
	uploader, scope, records, merchant, _, official, auth := uploadFixture(t)
	stores := &rulesFixture{binding: merchant.Binding()}
	app := Application{Sources: uploader.dependencies.Sources.(*preparation.SourceSelector), Authorization: auth, PublicationReceipts: official, PublicationStores: stores, TargetMerchants: stores}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Minute)})
	value, err := app.Publication(ctx, records.saved.Source.ID, records.saved.Merchant.StoreID)
	require.NoError(t, err)
	require.Nil(t, value.Product)
	_, err = uploader.Upload(context.Background(), scope, uuid.NewString(), records.saved.ID)
	require.NoError(t, err)
	value, err = app.Publication(ctx, records.saved.Source.ID, records.saved.Merchant.StoreID)
	require.NoError(t, err)
	require.NotNil(t, value.Product)
	require.Equal(t, "sku-code-a", value.Product.SKCs[0].SKUs[0].SKUCode)
	require.Equal(t, records.saved.ID, value.RecordID)
	require.Equal(t, 1, merchant.publishes)
	_, err = app.Publication(ctx, uuid.NewString(), records.saved.Merchant.StoreID)
	require.ErrorIs(t, err, preparation.ErrNotFound)
	stores.binding.SupplierIdentityHash = "other-supplier"
	_, err = app.Publication(ctx, records.saved.Source.ID, records.saved.Merchant.StoreID)
	require.ErrorIs(t, err, record.ErrConflict)
	stores.binding.Platform = storecenter.PlatformTemu
	stores.binding.IdentityPartition = storecenter.PartitionGlobal
	stores.binding.MerchantMode = storecenter.MerchantLocal
	stores.binding.Site = "temu-local"
	stores.binding.RegionID = 75
	stores.binding.ApplicationType = ""
	stores.binding.ApplicationRevision = "v1:temu:GLOBAL:manual_token:1:r1"
	stores.binding.VerificationRevision = "official-v1"
	stores.binding.ScopesHash = collection.Digest("scopes")
	stores.binding.SupplierIdentityHash = collection.Digest("native merchant")
	stores.binding.CredentialExpiresAt = time.Now().Add(time.Hour)
	value, err = app.Publication(ctx, records.saved.Source.ID, records.saved.Merchant.StoreID)
	require.NoError(t, err, "an in-scope unsupported publishing site can still read its own empty projection")
	require.Nil(t, value.Product)
	require.Equal(t, 1, merchant.publishes)
}
