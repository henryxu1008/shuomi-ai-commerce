package httpapi

import (
	"context"
	listingapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/listing"
	record "github.com/henryxu1008/shuomi-ai-commerce/internal/listing/record/target"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	"github.com/stretchr/testify/require"
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
		bridge := &listingProductAgent{app: &listingapp.Application{TargetMerchants: optimizationMerchantFixture{storecenter.ProductMerchantBinding{Platform: platform, OrganizationID: scope.OrganizationID, StoreID: "store-a"}}}}
		err := bridge.requireOptimizationStore(context.Background(), scope, "store-a")
		if platform == storecenter.PlatformShein {
			require.NoError(t, err)
		} else {
			require.ErrorIs(t, err, record.ErrNotReady)
		}
	}
	bridge := &listingProductAgent{app: &listingapp.Application{TargetMerchants: optimizationMerchantFixture{storecenter.ProductMerchantBinding{Platform: storecenter.PlatformShein, OrganizationID: "other-org", StoreID: "store-a"}}}}
	require.ErrorIs(t, bridge.requireOptimizationStore(context.Background(), scope, "store-a"), record.ErrNotReady)
}
