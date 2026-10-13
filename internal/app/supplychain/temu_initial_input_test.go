package supplychainapp

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"task-processor/internal/product/catalog"
	"task-processor/internal/storecenter"
	"testing"
)

func TestNativeAdaptationKeepsPriceQuantityUnitsCategoryAndConsentMissing(t *testing.T) {
	source := catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{ProductKey: "product-a"}, Snapshot: catalog.ProductSnapshot{Title: "Saved title", Description: "Saved description", Variants: []catalog.Variant{{SourceID: "variant-a", SKU: "source-sku", Stock: 50}}}}
	for _, mode := range []storecenter.OfficialMerchantMode{storecenter.MerchantLocal, storecenter.MerchantFullyManaged, storecenter.MerchantSemiManaged} {
		binding := storecenter.ProductMerchantBinding{MerchantMode: mode, SupplierIdentityHash: "original-merchant"}
		input := initialTemuInput(source, binding)
		require.Equal(t, string(mode), input.Mode)
		require.Empty(t, input.Images)
		require.Empty(t, input.PackageUnits.Weight)
		require.Empty(t, input.PackageUnits.Dimension)
		require.Zero(t, input.Selection.CatID)
		if mode == storecenter.MerchantLocal {
			require.Nil(t, input.Supplier)
			require.Equal(t, source.Snapshot.Title, input.Local.Basic.Name)
			require.Equal(t, source.Snapshot.Description, input.Local.Basic.Description)
			require.Nil(t, input.Local.SKUs[0].Quantity)
			require.Empty(t, input.Local.SKUs[0].Price.BasePrice.Amount)
			require.Empty(t, input.Local.SKUs[0].Price.BasePrice.Currency)
			require.NotEmpty(t, input.Local.Basic.ExternalGoodsID)
		} else {
			require.Nil(t, input.Local)
			require.Nil(t, input.Supplier.Cat1ID)
			require.Nil(t, input.Supplier.SKCs[0].SKUs[0].SupplierPrice)
			require.Nil(t, input.Supplier.SKCs[0].SKUs[0].Stock)
			require.Empty(t, input.Supplier.SKCs[0].SKUs[0].Currency)
			require.Nil(t, input.Supplier.Compliance)
		}
		first, _ := json.Marshal(input)
		again, _ := json.Marshal(initialTemuInput(source, binding))
		require.Equal(t, first, again, "correlation IDs stay stable across original command retries")
	}
}
