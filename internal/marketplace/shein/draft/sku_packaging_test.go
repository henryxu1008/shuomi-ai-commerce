package draft

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/listing/record"
	"task-processor/internal/marketplace/shein/offlinepackage"
	"task-processor/internal/marketplace/shein/workspace"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/catalog"
)

func TestOwnSKUPackagingBuildsExactOfflineFactsAndKeepsPartialGaps(t *testing.T) {
	var snapshot catalog.ProductSnapshot
	require.NoError(t, json.Unmarshal([]byte(`{"title":"Measured","variants":[{"source_id":"red","sku":"RED","stock":0,"own_stock_declared":true,"packaging":{"weightGrams":125.5,"lengthCm":10,"widthCm":8,"heightCm":2}},{"source_id":"blue","sku":"BLUE","stock":7,"own_stock_declared":true,"packaging":{"lengthCm":20}}]}`), &snapshot))
	raw, err := (Builder{}).Build(context.Background(), snapshot, asset.ApprovedAssetInventory{}, record.Input{Country: "US", Language: "en"})
	require.NoError(t, err)
	var pkg offlinepackage.Package
	require.NoError(t, json.Unmarshal(raw, &pkg))
	skus := pkg.DraftPayload.SKCList[0].SKUList
	require.Len(t, skus, 2)
	require.Equal(t, "RED", skus[0].SupplierSKU)
	require.Equal(t, 125.5, skus[0].Weight)
	require.Equal(t, "g", skus[0].WeightUnit)
	require.Equal(t, "10", skus[0].Length)
	require.Equal(t, "8", skus[0].Width)
	require.Equal(t, "2", skus[0].Height)
	require.Equal(t, "cm", skus[0].LengthUnit)
	require.Zero(t, skus[0].StockCount)
	require.True(t, skus[0].OwnStockDeclared)
	require.Equal(t, "BLUE", skus[1].SupplierSKU)
	require.Zero(t, skus[1].Weight)
	require.Empty(t, skus[1].WeightUnit)
	require.Equal(t, "20", skus[1].Length)
	require.Empty(t, skus[1].Width)
	require.Empty(t, skus[1].Height)
	checks := workspace.BuildOriginalSKUReadinessChecks(&pkg)
	require.Len(t, checks, 1)
	require.False(t, checks[0].OK)
	result, err := json.Marshal(checks)
	require.NoError(t, err)
	for _, text := range []string{"BLUE", "重量待补", "宽待补", "高待补", "库存 0"} {
		require.Contains(t, string(result), text)
	}
}
