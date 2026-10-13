package model_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/marketplace/temu/model"
)

func TestLocalV2WireKeepsExactStringsLargeIDsAndExplicitZeroFacts(t *testing.T) {
	zero := int64(0)
	noTrademark := false
	input := model.LocalGoodsV2{
		Basic:          model.LocalGoodsBasic{Name: "Product", CatID: 9007199254740993, Brand: &model.LocalBrand{NoTrademark: &noTrademark}},
		ServicePromise: model.LocalServicePromise{ShipmentLimitDay: &zero, CostTemplateID: "template"},
		SKUs:           []model.LocalSKU{{Quantity: &zero, Price: model.LocalSKUPrice{BasePrice: model.LocalMoney{Amount: "99999999.99", Currency: "USD"}}, PackageInfo: model.LocalPackageInfo{Weight: "0.1", Length: "0.1", Width: "0.1", Height: "0.1"}}},
	}
	raw, err := json.Marshal(input)
	require.NoError(t, err)
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &fields))
	require.Contains(t, fields, "goodsBasic")
	require.Contains(t, fields, "skuList")
	for _, forbidden := range []string{"request", "mallId", "siteId", "app_key", "access_token", "sign", "timestamp"} {
		require.NotContains(t, fields, forbidden)
	}
	require.Contains(t, string(raw), `"catId":9007199254740993`)
	require.Contains(t, string(raw), `"quantity":0`)
	require.Contains(t, string(raw), `"noTrademark":false`)
	require.Contains(t, string(raw), `"amount":"99999999.99"`)
	input.Basic.Brand = nil
	input.SKUs[0].Quantity = nil
	raw, err = json.Marshal(input)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "noTrademark", "do not manufacture a legal brand statement")
	require.Contains(t, string(raw), `"quantity":null`, "a missing required fact is not zero inventory")
}
