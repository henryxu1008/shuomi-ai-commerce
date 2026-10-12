package temu

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/storecenter"
	"testing"
)

func TestLocalCreationAndImageUseFixedRegionalOuterProtocolWithoutRetry(t *testing.T) {
	for _, partition := range []storecenter.OfficialIdentityPartition{storecenter.PartitionUS, storecenter.PartitionEU, storecenter.PartitionGlobal} {
		t.Run(string(partition), func(t *testing.T) {
			calls := []string{}
			c := identityClient(t, partition, roundTripFunc(func(r *http.Request) (*http.Response, error) {
				var fields map[string]json.RawMessage
				require.NoError(t, json.NewDecoder(r.Body).Decode(&fields))
				var api string
				require.NoError(t, json.Unmarshal(fields["type"], &api))
				calls = append(calls, api)
				require.Equal(t, localAPIOrigin(partition)+"/openapi/router", r.URL.String())
				if api == localInfoAPI {
					return jsonResponse(`{"success":true,"result":{"mallId":123,"mallType":100,"regionId":211,"expiredTime":1900000000,"apiScopeList":["temu.local.goods.v2.add","temu.local.goods.image.v2.upload"]}}`), nil
				}
				require.NotContains(t, fields, "request")
				require.NotContains(t, fields, "mallId")
				require.NotContains(t, fields, "siteId")
				if api == localGoodsAddAPI {
					require.Contains(t, fields, "goodsBasic")
					return jsonResponse(`{"success":true,"result":{"goodsId":9007199254740993,"productType":1,"skuInfoList":[{"skuId":21,"outSkuSn":"sku-a","specList":[{"parentSpecId":10,"specId":101,"specName":"Blue"}]}],"warnings":[{"message":"Check Seller Center"}]}}`), nil
				}
				require.Equal(t, localImageUploadAPI, api)
				require.JSONEq(t, `34`, string(fields["catId"]))
				require.JSONEq(t, `3`, string(fields["usage"]))
				return jsonResponse(`{"success":true,"result":{"images":[{"url":"https://temu-image.example.org/1.png"},{"url":"https://temu-image.example.org/2.png"}]}}`), nil
			}))
			created, err := c.CreateLocalGoods(context.Background(), tokenCredential(), model.LocalGoodsV2{Basic: model.LocalGoodsBasic{Name: "Explicit title", CatID: 34}})
			require.NoError(t, err)
			require.EqualValues(t, 9007199254740993, *created.Local.GoodsID)
			require.Len(t, created.ResponseHash, 64)
			require.Len(t, created.Local.Warnings, 1)
			image, hash, err := c.UploadLocalImage(context.Background(), tokenCredential(), model.LocalImageUpload{FileURL: "https://images.example.org/a.png", CatID: 34, Usage: 3})
			require.NoError(t, err)
			require.Len(t, image.Images, 2)
			require.Len(t, hash, 64)
			require.Equal(t, []string{localInfoAPI, localGoodsAddAPI, localInfoAPI, localImageUploadAPI}, calls)
		})
	}
}
