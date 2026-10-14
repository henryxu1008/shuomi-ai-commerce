package temu

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
	"testing"
	"time"
)

func TestObservationPageAllowsSequentialReadsBeyondTenSeconds(t *testing.T) {
	for _, partition := range []storecenter.OfficialIdentityPartition{storecenter.PartitionCN, storecenter.PartitionGlobal} {
		t.Run(string(partition), func(t *testing.T) {
			t.Parallel()
			calls := 0
			var identityDeadline time.Time
			c := identityClient(t, partition, roundTripFunc(func(r *http.Request) (*http.Response, error) {
				fields := map[string]json.RawMessage{}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&fields))
				var api string
				require.NoError(t, json.Unmarshal(fields["type"], &api))
				calls++
				deadline, ok := r.Context().Deadline()
				require.True(t, ok)
				require.LessOrEqual(t, time.Until(deadline), 10*time.Second, "each HTTP request remains bounded")
				delay := 4 * time.Second
				var response string
				switch api {
				case supplierInfoAPI:
					identityDeadline = deadline
					require.LessOrEqual(t, time.Until(deadline), 5*time.Second)
					delay = 2 * time.Second
					response = `{"success":true,"result":{"uniqueId":"pa-mall-1","expiredTime":1900000000,"apiScopeList":["bg.glo.goods.list.get","bg.glo.product.search"]}}`
				case supplierModeAPI:
					require.Equal(t, identityDeadline, deadline, "supplier identity cannot renew its five-second total budget")
					delay = 2 * time.Second
					response = `{"success":true,"result":{"semiManagedMall":true}}`
				case supplierGoodsListAPI:
					response = `{"success":true,"result":{"totalCount":1,"data":[{"productId":123,"productSkcId":21,"productName":"Saved supplier item","productSkuSummaries":[{"productSkuId":31,"virtualStock":0}]}]}}`
				case supplierLifecycleAPI:
					response = `{"success":true,"result":{"total":1,"dataList":[{"productId":123,"skcList":[{"skcId":21,"skuList":[{"skuId":31,"applyJitStatus":3}]}]}]}}`
				case localInfoAPI:
					require.LessOrEqual(t, time.Until(deadline), 5*time.Second)
					delay = 2 * time.Second
					response = `{"success":true,"result":{"mallId":123,"mallType":100,"regionId":211,"expiredTime":1900000000,"apiScopeList":["temu.local.goods.list.retrieve","bg.local.goods.publish.status.get"]}}`
				case localGoodsListAPI:
					delay = 5 * time.Second
					response = `{"success":true,"result":{"total":1,"pagination":{},"goodsList":[{"goodsId":"123","goodsName":"Saved local item","goodsStatus":"Active","skuInfoList":[{"skuId":31,"stock":0,"specList":[]}]}]}}`
				case localPublishStatusAPI:
					delay = 5 * time.Second
					response = `{"success":true,"result":{"goodsPublishStatusList":[{"goodsId":123,"status":1}]}}`
				default:
					t.Fatalf("unexpected API %s", api)
				}
				timer := time.NewTimer(delay)
				defer timer.Stop()
				select {
				case <-timer.C:
					return jsonResponse(response), nil
				case <-r.Context().Done():
					return nil, r.Context().Err()
				}
			}))
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			page, err := c.QueryObservedTemuProducts(ctx, tokenCredential(), o.ProductQuery{Page: 1})
			require.NoError(t, err, "identity, list and status may cumulatively exceed ten seconds within the worker deadline")
			require.Len(t, page.TemuItems, 1)
			if partition == storecenter.PartitionCN {
				require.Equal(t, 4, calls)
				require.NotNil(t, page.TemuItems[0].SKUs[0].ApplyJITStatus)
			} else {
				require.Equal(t, 3, calls)
				require.NotNil(t, page.TemuItems[0].PublishStatus)
			}
		})
	}
}

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
