package temu

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/storecenter"
)

func TestSupplierCategoryReadsUseCNScopeAndRouteWithExactToken(t *testing.T) {
	calls := []string{}
	var firstDeadline time.Time
	c := identityClient(t, storecenter.PartitionCN, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body map[string]json.RawMessage
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		var api, token string
		require.NoError(t, json.Unmarshal(body["type"], &api))
		require.NoError(t, json.Unmarshal(body["access_token"], &token))
		require.Equal(t, "test-token", token)
		require.Equal(t, "openapi.kuajingmaihuo.com", r.URL.Host)
		deadline, ok := r.Context().Deadline()
		require.True(t, ok)
		require.LessOrEqual(t, time.Until(deadline), 10*time.Second)
		if firstDeadline.IsZero() {
			firstDeadline = deadline
		} else {
			require.Equal(t, firstDeadline, deadline, "metadata cannot renew the scope read deadline")
		}
		calls = append(calls, api)
		if api == "bg.open.accesstoken.info.get" {
			return jsonResponse(`{"success":true,"result":{"mallId":123,"expiredTime":1900000000,"apiScopeList":["bg.goods.cats.get"]}}`), nil
		}
		require.Equal(t, "bg.goods.cats.get", api)
		require.JSONEq(t, `12`, string(body["parentCatId"]))
		require.JSONEq(t, `100`, string(body["siteId"]))
		require.JSONEq(t, `false`, string(body["showHidden"]))
		return jsonResponse(`{"success":true,"result":{"categoryDTOList":[{"catId":34,"catName":"Packaging","parentCatId":12,"catType":1,"isLeaf":true,"catLevel":2,"isHidden":false,"hiddenType":0}]}}`), nil
	}))
	items, err := c.QuerySupplierCategories(context.Background(), tokenCredential(), model.SupplierCategoryQuery{ParentCatID: 12, SiteID: 100})
	require.NoError(t, err)
	require.Equal(t, []string{"bg.open.accesstoken.info.get", "bg.goods.cats.get"}, calls)
	require.Len(t, items, 1)
	require.True(t, items[0].SelectableLeaf())
}

func TestCategoryRootEmptyAndUnknownAvailabilityRemainDistinct(t *testing.T) {
	for _, result := range []string{
		`{"categoryDTOList":[]}`,
		`{"categoryDTOList":[{"catId":1,"catName":"Unknown flags"}]}`,
		`{"categoryDTOList":[{"catId":1,"catName":"Unknown hidden flag","parentCatId":0,"catType":1,"catLevel":1,"isLeaf":true}]}`,
	} {
		t.Run(result, func(t *testing.T) {
			calls := 0
			c := identityClient(t, storecenter.PartitionCN, roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return jsonResponse(`{"success":true,"result":{"mallId":123,"expiredTime":1900000000,"apiScopeList":["bg.goods.cats.get"]}}`), nil
				}
				var body map[string]json.RawMessage
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.NotContains(t, body, "parentCatId", "CN root lookup omits parent, while local lookup sends zero")
				require.NotContains(t, body, "siteId")
				return jsonResponse(`{"success":true,"result":` + result + `}`), nil
			}))
			items, err := c.QuerySupplierCategories(context.Background(), tokenCredential(), model.SupplierCategoryQuery{})
			require.NoError(t, err)
			require.NotNil(t, items)
			for _, item := range items {
				require.False(t, item.SelectableLeaf(), "unknown flags cannot authorize leaf selection")
			}
		})
	}
}

func TestCategoryScopeExpiredAndCancellationCannotSendMetadata(t *testing.T) {
	for _, partition := range []storecenter.OfficialIdentityPartition{storecenter.PartitionCN, storecenter.PartitionGlobal} {
		t.Run(string(partition), func(t *testing.T) {
			calls := 0
			c := identityClient(t, partition, roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				return jsonResponse(`{"success":true,"result":{"mallId":123,"mallType":100,"regionId":211,"expiredTime":1759999999,"apiScopeList":["bg.goods.cats.get","bg.local.goods.cats.get"]}}`), nil
			}))
			read := func(ctx context.Context) error {
				if partition == storecenter.PartitionCN {
					_, err := c.QuerySupplierCategories(ctx, tokenCredential(), model.SupplierCategoryQuery{})
					return err
				}
				_, err := c.QueryLocalCategories(ctx, tokenCredential(), model.LocalCategoryQuery{})
				return err
			}
			require.ErrorIs(t, read(context.Background()), storecenter.ErrOfficialCredentialExpired)
			require.Equal(t, 1, calls)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			require.Error(t, read(ctx))
			require.Equal(t, 1, calls)
		})
	}
}

func TestLocalCategoryReadsUseOnlyOriginalPartitionAndExplicitAvailability(t *testing.T) {
	for partition, origin := range map[storecenter.OfficialIdentityPartition]string{
		storecenter.PartitionUS:     "https://openapi-b-us.temu.com",
		storecenter.PartitionEU:     "https://openapi-b-eu.temu.com",
		storecenter.PartitionGlobal: "https://openapi-b-global.temu.com",
	} {
		t.Run(string(partition), func(t *testing.T) {
			calls := 0
			c := identityClient(t, partition, roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				var body map[string]json.RawMessage
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				var api string
				require.NoError(t, json.Unmarshal(body["type"], &api))
				require.Equal(t, origin+"/openapi/router", r.URL.String())
				if api == "bg.open.accesstoken.info.get" {
					return jsonResponse(`{"success":true,"result":{"mallId":9007199254740993,"mallType":100,"regionId":211,"expiredTime":1900000000,"apiScopeList":["bg.local.goods.cats.get"]}}`), nil
				}
				require.Equal(t, "bg.local.goods.cats.get", api)
				require.JSONEq(t, `0`, string(body["parentCatId"]))
				require.JSONEq(t, `"en"`, string(body["language"]))
				require.NotContains(t, body, "siteId", "local metadata is for the authorized mall, not an arbitrary site selector")
				return jsonResponse(`{"success":true,"result":{"goodsCatsList":[{"catId":34,"catName":"Packaging","parentId":0,"catType":1,"leaf":true,"level":1,"availableStatus":0},{"catId":35,"catName":"Unavailable","parentId":0,"catType":1,"leaf":true,"level":1,"availableStatus":1},{"catId":36,"catName":"Missing availability","parentId":0,"catType":1,"leaf":true,"level":1}]}}`), nil
			}))
			items, err := c.QueryLocalCategories(context.Background(), tokenCredential(), model.LocalCategoryQuery{Language: "en"})
			require.NoError(t, err)
			require.Equal(t, 2, calls)
			require.Len(t, items, 3)
			require.True(t, items[0].SelectableLeaf())
			require.False(t, items[1].SelectableLeaf())
			require.False(t, items[2].SelectableLeaf(), "missing availability is not zero")
		})
	}
}

func TestCategoryReadCannotUseAnotherGatewayScopeOrInvalidCredential(t *testing.T) {
	for _, scopes := range []string{`[]`, `["bg.glo.goods.add"]`, `["bg.local.goods.cats.get"]`} {
		t.Run(scopes, func(t *testing.T) {
			calls := 0
			c := identityClient(t, storecenter.PartitionCN, roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				return jsonResponse(`{"success":true,"result":{"mallId":123,"expiredTime":1900000000,"apiScopeList":` + scopes + `}}`), nil
			}))
			_, err := c.QuerySupplierCategories(context.Background(), tokenCredential(), model.SupplierCategoryQuery{})
			require.ErrorIs(t, err, storecenter.ErrOfficialAuthorizationRejected)
			require.Equal(t, 1, calls)
		})
	}
	requests := 0
	c := identityClient(t, storecenter.PartitionCN, roundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		t.Fatal("invalid query must not read scope or metadata")
		return nil, nil
	}))
	_, err := c.QueryLocalCategories(context.Background(), tokenCredential(), model.LocalCategoryQuery{})
	require.Error(t, err)
	_, err = c.QuerySupplierCategories(context.Background(), tokenCredential(), model.SupplierCategoryQuery{ParentCatID: -1})
	require.Error(t, err)
	credential := tokenCredential()
	credential.AppID = "another-app"
	_, err = c.QuerySupplierCategories(context.Background(), credential, model.SupplierCategoryQuery{})
	require.Error(t, err)
	credential = tokenCredential()
	credential.SecretKey = "mixed-shein-secret"
	_, err = c.QuerySupplierCategories(context.Background(), credential, model.SupplierCategoryQuery{})
	require.Error(t, err)
	require.Zero(t, requests)
}

func TestCategoryResponseMissingListDuplicatesAndWrongParentAreUnavailable(t *testing.T) {
	for index, result := range []string{`{}`, `{"categoryDTOList":null}`, `{"categoryDTOList":[{"catId":1,"catName":"A","parentCatId":5}]}`, `{"categoryDTOList":[{"catId":1,"catName":"A"},{"catId":1,"catName":"B"}]}`} {
		t.Run(strconv.Itoa(index), func(t *testing.T) {
			calls := 0
			c := identityClient(t, storecenter.PartitionCN, roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return jsonResponse(`{"success":true,"result":{"mallId":123,"expiredTime":1900000000,"apiScopeList":["bg.goods.cats.get"]}}`), nil
				}
				return jsonResponse(`{"success":true,"result":` + result + `}`), nil
			}))
			_, err := c.QuerySupplierCategories(context.Background(), tokenCredential(), model.SupplierCategoryQuery{})
			require.ErrorIs(t, err, storecenter.ErrOfficialConnectionUnavailable)
		})
	}
}
