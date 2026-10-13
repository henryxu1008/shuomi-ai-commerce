package temu

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
	"testing"
)

func TestLocalObservationUsesOpaqueTokenAndKeepsCreationReviewSalesSeparate(t *testing.T) {
	calls := []string{}
	c := identityClient(t, storecenter.PartitionGlobal, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		fields := map[string]json.RawMessage{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&fields))
		var api string
		require.NoError(t, json.Unmarshal(fields["type"], &api))
		calls = append(calls, api)
		switch api {
		case localInfoAPI:
			return jsonResponse(`{"success":true,"result":{"mallId":123,"mallType":100,"regionId":75,"expiredTime":1900000000,"apiScopeList":["temu.local.goods.list.retrieve","bg.local.goods.publish.status.get"]}}`), nil
		case localGoodsListAPI:
			require.NotContains(t, fields, "page")
			require.NotContains(t, fields, "pageNo")
			require.NotContains(t, fields, "siteId")
			require.NotContains(t, fields, "request")
			require.JSONEq(t, `"opaque-next"`, string(fields["pageToken"]))
			require.JSONEq(t, `"ALL"`, string(fields["goodsSearchType"]))
			return jsonResponse(`{"success":true,"result":{"total":1,"pagination":{},"goodsList":[{"goodsId":"9007199254740993","goodsName":"Actual product","goodsStatus":"Active","skuInfoList":[{"skuId":9007199254740994,"skuSn":"platform-code","stock":0,"specList":[]}]}]}}`), nil
		case localPublishStatusAPI:
			require.JSONEq(t, `[9007199254740993]`, string(fields["goodsIdList"]))
			return jsonResponse(`{"success":true,"result":{"goodsPublishStatusList":[{"goodsId":9007199254740993,"status":1,"subStatus":2}]}}`), nil
		}
		t.Fatalf("unadmitted API %s", api)
		return nil, nil
	}))
	page, err := c.QueryObservedTemuProducts(context.Background(), tokenCredential(), o.ProductQuery{Page: 2, PageToken: "opaque-next"})
	require.NoError(t, err)
	require.True(t, page.TokenPagination)
	require.Nil(t, page.NextToken)
	require.Empty(t, page.Items)
	require.Len(t, page.TemuItems, 1)
	p := page.TemuItems[0]
	require.Equal(t, "9007199254740993", p.ID)
	require.Equal(t, "active", p.SaleState)
	require.Equal(t, int64(1), *p.PublishStatus.Status)
	require.Empty(t, p.SKUs[0].ExternalCode)
	require.Equal(t, "platform-code", p.SKUs[0].SerialNumber)
	require.Equal(t, int64(0), *p.SKUs[0].Stock)
	require.Equal(t, []string{localInfoAPI, localGoodsListAPI, localPublishStatusAPI}, calls)
}

func TestSupplierObservationKeepsSKCEntityAndRawStatusWithoutInferringOnSale(t *testing.T) {
	for _, semi := range []bool{false, true} {
		c := identityClient(t, storecenter.PartitionCN, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			fields := map[string]json.RawMessage{}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&fields))
			var api string
			require.NoError(t, json.Unmarshal(fields["type"], &api))
			switch api {
			case supplierInfoAPI:
				return jsonResponse(`{"success":true,"result":{"uniqueId":"pa-mall-1","expiredTime":1900000000,"apiScopeList":["bg.glo.goods.list.get","bg.glo.product.search"]}}`), nil
			case supplierModeAPI:
				if semi {
					return jsonResponse(`{"success":true,"result":{"semiManagedMall":true}}`), nil
				}
				return jsonResponse(`{"success":true,"result":{"semiManagedMall":false}}`), nil
			case supplierGoodsListAPI:
				require.Equal(t, "https://openapi-b-partner.temu.com/openapi/router", r.URL.String())
				require.JSONEq(t, `3`, string(fields["page"]))
				return jsonResponse(`{"success":true,"result":{"totalCount":2,"data":[{"productId":9007199254740993,"productSkcId":21,"productName":"Actual title","skcSiteStatus":100,"productSkuSummaries":[{"productSkuId":31,"extCode":"sku-a","virtualStock":0}]},{"productId":9007199254740993,"productSkcId":22,"productName":"Actual title","productSkuSummaries":[{"productSkuId":32,"extCode":"sku-b"}]}]}}`), nil
			case supplierLifecycleAPI:
				require.JSONEq(t, `[31,32]`, string(fields["productSkuIdList"]))
				return jsonResponse(`{"success":true,"result":{"total":1,"dataList":[{"productId":9007199254740993,"skcList":[{"skcId":21,"selectStatus":7,"skuList":[{"skuId":31,"applyJitStatus":3,"suggestCloseJit":false}]},{"skcId":22,"skuList":[{"skuId":32}]}]}]}}`), nil
			}
			t.Fatalf("unexpected API %s", api)
			return nil, nil
		}))
		page, err := c.QueryObservedTemuProducts(context.Background(), tokenCredential(), o.ProductQuery{Page: 3})
		require.NoError(t, err)
		require.Len(t, page.TemuItems, 2)
		require.Equal(t, "21", page.TemuItems[0].ID)
		require.Equal(t, "9007199254740993", page.TemuItems[0].ProductID)
		require.Equal(t, "supplier_skc", page.TemuItems[0].Entity)
		require.Equal(t, "unknown", page.TemuItems[0].SaleState)
		require.Equal(t, "100", page.TemuItems[0].RawStatus)
		require.Equal(t, int64(7), *page.TemuItems[0].SelectionStatus)
		require.Equal(t, "virtualStock", page.TemuItems[0].SKUs[0].StockKind)
		if semi {
			require.Equal(t, "semi_managed", page.TemuItems[0].Mode)
		} else {
			require.Equal(t, "fully_managed", page.TemuItems[0].Mode)
		}
	}
}
