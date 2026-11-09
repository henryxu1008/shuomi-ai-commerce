package shein

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	o "task-processor/internal/storecenter/observations"
	"testing"
)

func TestSupplyBusinessTimesPreserveSourceAndNeverInventDates(t *testing.T) {
	receipt := `{"code":"0","info":{"count":1,"list":[{"orderNo":"A","addTime":"2026-11-01 12:34:56","allocateTime":null,"updateTime":"","requestDeliveryTime":" 2026-11-02 00:00:00 ","requestReceiptTime":"2026-11-02T00:00:00-05:00","reserveTime":"1970-01-01 08:00:01","receiptTime":123.00,"checkTime":false,"storageTime":"2026-02-30 00:00:00","returnTime":"2026-11-02","requestCompleteTime":"\t"}]}}`
	client := testClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) { return observationResponse(receipt), nil }))
	page, err := client.QueryObservedSupplyOrders(context.Background(), goodsCredential(), []string{"A"})
	require.NoError(t, err)
	raw, err := json.Marshal(page.Items[0])
	require.NoError(t, err)
	var saved map[string]any
	require.NoError(t, json.Unmarshal(raw, &saved))
	bundle, ok := saved["businessTimes"].(map[string]any)
	require.True(t, ok, "business time contract must be saved")
	require.Equal(t, "shein-supply-business-times/v1", bundle["contract"])
	require.Equal(t, "/open-api/order/purchase-order-infos", bundle["source"])
	require.Equal(t, "unconfirmed", bundle["timezone"])
	values := bundle["values"].(map[string]any)
	require.Len(t, values, 13)
	for field, want := range map[string][]any{
		"addTime": {"local_datetime", "string", "2026-11-01 12:34:56"}, "allocateTime": {"null", "null", nil}, "updateTime": {"empty", "string", ""},
		"deliveryTime": {"missing", "missing", nil}, "requestDeliveryTime": {"unrecognized", "string", " 2026-11-02 00:00:00 "},
		"requestReceiptTime": {"offset_datetime", "string", "2026-11-02T00:00:00-05:00"}, "reserveTime": {"local_datetime", "string", "1970-01-01 08:00:01"},
		"receiptTime": {"unexpected_type", "number", "123.00"}, "checkTime": {"unexpected_type", "boolean", "false"}, "storageTime": {"unrecognized", "string", "2026-02-30 00:00:00"},
	} {
		v := values[field].(map[string]any)
		require.Equal(t, want[0], v["state"], field)
		require.Equal(t, want[1], v["rawType"], field)
		require.Equal(t, want[2], v["raw"], field)
	}
	for _, forbidden := range []string{"businessDate", "normalizedAt", "timezoneOffset"} {
		require.NotContains(t, string(raw), forbidden)
	}
}

func TestSupplyBusinessTimesRejectUnsaveableWithoutPartialPage(t *testing.T) {
	for _, value := range []string{`{}`, `[]`, `"` + strings.Repeat("x", 257) + `"`, `"\u0000"`} {
		client := testClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
			return observationResponse(`{"code":"0","info":{"count":2,"list":[{"orderNo":"A"},{"orderNo":"B","addTime":` + value + `}]}}`), nil
		}))
		page, err := client.QueryObservedSupplyOrders(context.Background(), goodsCredential(), []string{"A", "B"})
		require.ErrorIs(t, err, o.ErrUnavailable)
		require.Empty(t, page.Items)
	}
}

func TestSupplyTimeConflictsUseOriginalIdentityRules(t *testing.T) {
	client := testClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		return observationResponse(`{"code":"0","info":{"count":2,"list":[{"orderNo":"A","addTime":"2026-11-01 00:00:00"},{"orderNo":"A","addTime":"2026-11-02 00:00:00"}]}}`), nil
	}))
	page, err := client.QueryObservedSupplyOrders(context.Background(), goodsCredential(), []string{"A"})
	require.NoError(t, err)
	require.Empty(t, page.Items)
	require.Contains(t, page.Notes, "supply_identity_conflict")
}
