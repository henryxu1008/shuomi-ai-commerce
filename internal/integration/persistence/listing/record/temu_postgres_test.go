package recordpersistence_test

import (
	"context"
	temugoods "task-processor/internal/marketplace/temu/goods"
	temumodel "task-processor/internal/marketplace/temu/model"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"time"
)

type currentTemuSaveRules struct {
	mode storecenter.OfficialMerchantMode
}

func (r currentTemuSaveRules) ReadTemuTargetRules(_ context.Context, scope collection.Scope, storeID string, selection temumodel.RuleSelection) (storecenter.ProductMerchantBinding, temugoods.RuleSnapshot, error) {
	site, partition, region, revision := "temu-local", storecenter.PartitionEU, int64(75), "app-v1:temu:EU:manual_token:1:r1"
	if r.mode != storecenter.MerchantLocal {
		site, partition, region, revision = "temu-supplier", storecenter.PartitionCN, 0, "app-v1:temu:CN:manual_token:6:r1"
	}
	return storecenter.ProductMerchantBinding{Platform: storecenter.PlatformTemu, OrganizationID: scope.OrganizationID, StoreID: storeID, IdentityPartition: partition, RegionID: region, MerchantMode: r.mode, Site: site, StoreVersion: 1, ConnectionRevision: 1, ApplicationID: "temu-app", ApplicationRevision: revision, VerificationRevision: "official-test-v1", ScopesHash: collection.Digest("scopes"), SupplierIdentityHash: collection.Digest("verified-merchant"), ServiceExpiresAt: time.Now().Add(time.Hour).UTC().Truncate(time.Microsecond), CredentialExpiresAt: time.Now().Add(time.Hour).UTC().Truncate(time.Microsecond)},
		temugoods.RuleSnapshot{Selection: selection, Mode: string(r.mode), SpecificationRevision: "observed-version", ObservedAt: time.Now().UTC().Truncate(time.Microsecond)}, nil
}
