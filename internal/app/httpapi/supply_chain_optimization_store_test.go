package httpapi

import (
	"context"
	"github.com/stretchr/testify/require"
	supplyapp "task-processor/internal/app/supplychain"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"testing"
)

type optimizationMerchantFixture struct {
	binding storecenter.ProductMerchantBinding
}

func (f optimizationMerchantFixture) ReadTargetMerchant(context.Context, collection.Scope, string) (storecenter.ProductMerchantBinding, error) {
	return f.binding, nil
}

func TestSupplyTitleOptimizationRejectsNativeStoreBeforeIntake(t *testing.T) {
	scope := collection.Scope{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a"}
	for _, platform := range []storecenter.Platform{storecenter.PlatformShein, storecenter.PlatformTemu, "unsupported"} {
		bridge := &supplyProductAgent{app: &supplyapp.Application{TargetMerchants: optimizationMerchantFixture{storecenter.ProductMerchantBinding{Platform: platform, OrganizationID: scope.OrganizationID, StoreID: "store-a"}}}}
		err := bridge.requireOptimizationStore(context.Background(), scope, "store-a")
		if platform == storecenter.PlatformShein {
			require.NoError(t, err)
		} else {
			require.ErrorIs(t, err, record.ErrNotReady)
		}
	}
	bridge := &supplyProductAgent{app: &supplyapp.Application{TargetMerchants: optimizationMerchantFixture{storecenter.ProductMerchantBinding{Platform: storecenter.PlatformShein, OrganizationID: "other-org", StoreID: "store-a"}}}}
	require.ErrorIs(t, bridge.requireOptimizationStore(context.Background(), scope, "store-a"), record.ErrNotReady)
}
