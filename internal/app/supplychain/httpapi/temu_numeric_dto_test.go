package supplychainhttp

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	supplyapp "task-processor/internal/app/supplychain"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/product/collection"
	"testing"
)

func TestTemuBrowserBoundaryPreservesLongIdentifiersAndExplicitZero(t *testing.T) {
	zero := int64(0)
	long := int64(9007199254740993)
	owner := record.TargetRecord{Input: record.TargetInput{Temu: &temugoods.DraftInput{Mode: "local", Selection: model.RuleSelection{CatID: long}, Local: &model.LocalGoodsV2{Basic: model.LocalGoodsBasic{CatID: long}, SKUs: []model.LocalSKU{{Quantity: &zero}}}}}}
	raw, err := json.Marshal(supplyPublicResponse(owner))
	require.NoError(t, err)
	require.Contains(t, string(raw), `"catId":"9007199254740993"`)
	require.Contains(t, string(raw), `"quantity":"0"`)
	require.NotContains(t, string(raw), `"draft":`)
	require.NotContains(t, string(raw), `"result":`)
	native, err := json.Marshal(owner.Input.Temu.Local)
	require.NoError(t, err)
	require.Contains(t, string(native), `"catId":9007199254740993`, "official protocol must remain numeric")
	var body supplyTargetBody
	require.NoError(t, json.Unmarshal([]byte(`{"sourceId":"source","storeId":"store","effectiveVersion":"1","temu":{"mode":"local","selection":{"catId":"9007199254740993"},"local":{"goodsBasic":{"catId":"9007199254740993"},"skuList":[{"quantity":"0"}]}}}`), &body))
	decoded, err := body.domain()
	require.NoError(t, err)
	require.Equal(t, long, decoded.Temu.Selection.CatID)
	require.Equal(t, long, decoded.Temu.Local.Basic.CatID)
	require.NotNil(t, decoded.Temu.Local.SKUs[0].Quantity)
	require.Zero(t, *decoded.Temu.Local.SKUs[0].Quantity)
	body.Temu = json.RawMessage(`{"mode":"fully_managed","supplier":{"productSpecPropertyReqs":[{"refPid":"9007199254740993","specId":"9007199254740993"}]}}`)
	supplier, err := body.domain()
	require.NoError(t, err)
	require.Equal(t, long, *supplier.Temu.Supplier.SpecProperties[0].RefPID)
	for _, invalid := range []string{`{"mode":"local","selection":{"catId":"9223372036854775808"}}`, `{"mode":"local","selection":{"catId":"01"}}`, `{"mode":"local","selection":{"catId":"1","catId":"2"}}`, `{"mode":"local","local":{"mallId":"123"}}`} {
		body.Temu = json.RawMessage(invalid)
		_, err = body.domain()
		require.Error(t, err)
	}
}

func TestTemuConfirmedPublicViewsKeepNativeLongIdentifiers(t *testing.T) {
	id := int64(9007199254740993)
	created := &temugoods.CreatedGoods{Local: &model.LocalGoodsCreated{GoodsID: &id}, ResponseHash: collection.Digest("official response")}
	for _, value := range []any{
		supplyapp.PublicationView{TemuProduct: created},
		supplyapp.UploadResult{TemuProduct: created},
		collection.Page[supplyapp.OperationItemView]{Items: []supplyapp.OperationItemView{{OperationItem: preparation.OperationItem{}, ConfirmedTemuProduct: created}}},
	} {
		raw, err := json.Marshal(supplyPublicResponse(value))
		require.NoError(t, err)
		require.Contains(t, string(raw), `"goodsId":"9007199254740993"`)
		require.NotContains(t, string(raw), `"product":`)
		require.NotContains(t, string(raw), `"confirmedProduct":`)
	}
}
