package target

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/authidentity"
	"task-processor/internal/listing/preparation"
	temugoods "task-processor/internal/marketplace/temu/goods"
	temumodel "task-processor/internal/marketplace/temu/model"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

type targetTemuRules struct {
	mode storecenter.OfficialMerchantMode
}

func (r targetTemuRules) ReadTemuTargetRules(_ context.Context, scope collection.Scope, storeID string, selection temumodel.RuleSelection) (storecenter.ProductMerchantBinding, temugoods.RuleSnapshot, error) {
	site, partition, region, revision := "temu-local", storecenter.PartitionEU, int64(75), "v1:temu:EU:manual_token:1:r1"
	if r.mode != storecenter.MerchantLocal {
		site, partition, region, revision = "temu-supplier", storecenter.PartitionCN, 0, "v1:temu:CN:manual_token:6:r1"
	}
	binding := storecenter.ProductMerchantBinding{Platform: storecenter.PlatformTemu, OrganizationID: scope.OrganizationID, StoreID: storeID, Site: site, IdentityPartition: partition, RegionID: region, MerchantMode: r.mode, StoreVersion: 1, ConnectionRevision: 1, ApplicationID: "app-a", ApplicationRevision: revision, VerificationRevision: "official-v1", ScopesHash: collection.Digest("scopes"), SupplierIdentityHash: collection.Digest("merchant"), ServiceExpiresAt: time.Now().Add(time.Hour), CredentialExpiresAt: time.Now().Add(time.Hour)}
	return binding, temugoods.RuleSnapshot{Mode: string(r.mode), Selection: selection, SpecificationRevision: "observed-revision", ObservedAt: time.Now()}, nil
}

func TestTemuTargetsUseOriginalSourceAndImmutableCommandForAllVerifiedModes(t *testing.T) {
	for _, mode := range []storecenter.OfficialMerchantMode{storecenter.MerchantLocal, storecenter.MerchantFullyManaged, storecenter.MerchantSemiManaged} {
		t.Run(string(mode), func(t *testing.T) {
			scope := collection.Scope{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a"}
			authority := targetAuth{scope}
			collections, err := collection.NewService(targetCollectionRepo{}, authority, targetUnusedSources{})
			require.NoError(t, err)
			source := preparation.SourceItem{ID: uuid.NewString(), PreparationID: uuid.NewString(), CollectionItemID: uuid.NewString(), CollectionRevision: 1, Source: collection.Source{ProductKey: "product-a", PublicationID: uuid.NewString(), Version: 1, Kind: "own"}}
			repo := targetPrepRepo{source: source}
			preparations, err := preparation.NewService(repo, collections, authority)
			require.NoError(t, err)
			snapshot := catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: "product-a"}, PublicationID: source.Source.PublicationID, Version: 1, Snapshot: catalog.ProductSnapshot{Title: "Original"}}
			selections, err := preparation.NewSourceSelector(preparations, collections, repo, targetCatalog{snapshot})
			require.NoError(t, err)
			records := &targetRecords{values: map[string]TargetReceipt{}, hashes: map[string]string{}}
			service, err := NewTargetService(TargetDependencies{Sources: selections, Products: OriginalTargetProduct{}, Assets: targetAssets{}, Rules: targetRules{}, TemuRules: targetTemuRules{mode}, Records: records, Images: targetTestImages{}, Authorizer: authority})
			require.NoError(t, err)
			ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Minute)})
			key := uuid.NewString()
			input := TargetInput{SourceID: source.ID, StoreID: uuid.NewString(), EffectiveVersion: 1, Temu: &temugoods.DraftInput{Mode: string(mode), Selection: temumodel.RuleSelection{CatID: 34}}}
			if mode == storecenter.MerchantLocal {
				input.Temu.Local = &temumodel.LocalGoodsV2{Basic: temumodel.LocalGoodsBasic{Name: "Explicit local title", CatID: 34}}
			} else {
				input.Temu.Supplier = &temumodel.SupplierGoods{Name: "Explicit supplier title"}
			}
			receipt, err := service.Create(ctx, key, input)
			require.NoError(t, err)
			require.NotNil(t, receipt.Record.Temu)
			require.False(t, receipt.Record.Temu.ReadyForUpload)
			require.NotEmpty(t, receipt.Record.Temu.Issues)
			require.False(t, receipt.Record.Result.ReadyForUpload)
			require.Equal(t, storecenter.PlatformTemu, receipt.Record.Merchant.Platform)
			require.Equal(t, source, receipt.Record.Source)
			require.NotEmpty(t, receipt.Record.RulesHash)
			replay, err := service.Create(ctx, key, input)
			require.NoError(t, err)
			require.True(t, replay.Replayed)
			require.Equal(t, 1, records.writes)
			input.Temu.Mode = "different-mode"
			_, err = service.Create(ctx, key, input)
			require.ErrorIs(t, err, ErrConflict)
		})
	}
}
