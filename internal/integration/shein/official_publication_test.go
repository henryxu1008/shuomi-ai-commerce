package shein

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"strings"
	"task-processor/internal/marketplace/shein/model"
	"testing"
)

func TestOfficialShelfRetainsBoundedFailureWithoutClaimingSuccess(t *testing.T) {
	body := `{"code":"0","msg":"partial","info":{"total_count":1,"success_count":0,"failure_count":1,"failure_results":[{"identity":"skc-1","code":"0","msg":"商品无库存无法上架"}]}}`
	client := testClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	}))
	got, e := client.ShelfOn(context.Background(), goodsCredential(), "skc-1")
	require.ErrorIs(t, e, ErrGoodsOutcomeUnknown)
	require.Equal(t, goodsHash([]byte(body)), got.Response.ResponseHash)
	require.Equal(t, "skc-1", got.Response.Failures[0].Identity)
	require.Equal(t, "0", got.Response.Failures[0].Code)
	require.Equal(t, "商品无库存无法上架", got.Response.Failures[0].Message)
}
func TestNewOfficialPublishCannotSucceedWithoutExactVersion(t *testing.T) {
	body := `{"code":"0","info":{"success":true,"spu_name":"spu-a","skc_list":[{"skc_name":"skc-a","sku_list":[{"sku_code":"code-a","supplier_sku":"sku-a"}]}]}}`
	client := testClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	}))
	_, e := client.PublishProduct(context.Background(), goodsCredential(), goodsProduct())
	require.ErrorIs(t, e, ErrGoodsOutcomeUnknown)
}

func TestOfficialShelfRequiresExactCountersAndNeverRetries(t *testing.T) {
	body := `{"code":"0","info":{"total_count":1,"success_count":1,"failure_count":0,"failure_results":[]}}`
	calls := 0
	client := testClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "/open-api/goods/modify-skc-shelf", r.URL.Path)
		require.NotEmpty(t, r.Header.Get("x-lt-signature"))
		raw, _ := io.ReadAll(r.Body)
		var v struct {
			Items []model.ShelfChange `json:"skc_site_info_list"`
		}
		require.NoError(t, json.Unmarshal(raw, &v))
		require.Equal(t, []model.ShelfChange{{SKCName: "skc-1", Sites: []string{"shein-us"}, State: 1}}, v.Items)
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	}))
	_, err := client.ShelfOn(context.Background(), goodsCredential(), "skc-1")
	require.NoError(t, err)
	for _, bad := range []string{
		`{"code":"0","info":{"total_count":1,"success_count":0,"failure_count":1,"failure_results":[{"code":"0","identity":"skc-1","msg":"商品无库存无法上架"}]}}`,
		`{"code":"0","info":{"total_count":2,"success_count":1,"failure_count":0,"failure_results":[]}}`,
		`{"code":"0","info":{"success_count":1,"failure_results":[]}}`,
	} {
		body = bad
		_, err = client.ShelfOn(context.Background(), goodsCredential(), "skc-1")
		require.ErrorIs(t, err, ErrGoodsOutcomeUnknown)
	}
	require.Equal(t, 4, calls)
}
func TestOfficialAuditBindsRequestedVersionAndPreservesUnknownState(t *testing.T) {
	body := `{"code":"0","info":{"data":[{"spuName":"spu-1","version":"v1","skcList":[{"skcName":"skc-1","documentSn":"doc-1","documentState":3,"failedReason":[{"language":"zh-cn","content":"需补资料"}]}]}]}}`
	client := testClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "/open-api/goods/query-document-state", r.URL.Path)
		raw, _ := io.ReadAll(r.Body)
		require.JSONEq(t, `{"spuList":[{"spuName":"spu-1","version":"v1"}]}`, string(raw))
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	}))
	got, err := client.QueryDocumentState(context.Background(), goodsCredential(), "spu-1", "v1")
	require.NoError(t, err)
	require.Equal(t, 3, *got.SKCs[0].State)
	require.Equal(t, "需补资料", got.SKCs[0].Reasons[0].Content)
	body = strings.ReplaceAll(body, `"version":"v1"`, `"version":"v2"`)
	_, err = client.QueryDocumentState(context.Background(), goodsCredential(), "spu-1", "v1")
	require.Error(t, err)
}
