package scheduledreport

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	c "task-processor/internal/operationscockpit"
	rc "task-processor/internal/reportcenter"
	o "task-processor/internal/storecenter/observations"
	"testing"
	"time"
)

func TestDailyReportKeepsKnownZeroSeparateFromMissing(t *testing.T) {
	now := time.Date(2026, 11, 15, 1, 0, 0, 0, time.UTC)
	origin := rc.ScheduledOrigin{OrganizationID: "org", ActorID: "user", MemberID: "member", PlanID: uuid.NewString(), PlanRevision: 2, OccurrenceID: uuid.NewString(), ScheduledAt: now, StoreID: uuid.NewString(), Day: "2026-11-14", RendererVersion: 1}
	snapshot := c.Snapshot{CapturedAt: now, Stores: map[string]c.StoreAggregate{}, Previous: map[string]c.StoreAggregate{}}
	missing := Render(origin, "合成店铺", "shein", snapshot)
	raw, _, e := missing.Bytes()
	require.NoError(t, e)
	require.Contains(t, string(raw), "金额未知")
	require.NotContains(t, string(raw), "0.00 元")
	snapshot.Stores[origin.StoreID] = c.StoreAggregate{StoreID: origin.StoreID, Period: c.Period{Start: origin.Day, End: origin.Day}, Complete: true, Records: []c.RecordEvidence{{ID: uuid.NewString(), Revision: 1, Period: c.Period{Start: origin.Day, End: origin.Day}}}}
	zero := Render(origin, "合成店铺", "shein", snapshot)
	raw, _, e = zero.Bytes()
	require.NoError(t, e)
	require.Contains(t, string(raw), "0.00 元")
	require.Contains(t, string(raw), "资料完整")
	require.NotContains(t, string(raw), "金额未知")
	aggregate := snapshot.Stores[origin.StoreID]
	aggregate.Complete = false
	snapshot.Stores[origin.StoreID] = aggregate
	raw, _, e = Render(origin, "合成店铺", "shein", snapshot).Bytes()
	require.NoError(t, e)
	require.Contains(t, string(raw), "资料不全")
	require.Contains(t, string(raw), "已覆盖部分")
}
func TestSavedOrderPresentationDoesNotInventCompleteCountsOrAmounts(t *testing.T) {
	zero := 0
	amounts := []o.CurrencyOrderAmount{}
	complete := renderOrders(o.OrderAmountMetrics{Status: "complete", MatchedOrderCount: &zero, Amounts: &amounts})
	raw, _ := json.Marshal(complete)
	require.Contains(t, string(raw), `"value":"0"`)
	for _, status := range []string{"coverage_incomplete", "missing_fields", "limited"} {
		section := renderOrders(o.OrderAmountMetrics{Status: status, MatchedOrderCount: &zero, Amounts: &amounts, Missing: &o.OrderAmountMissing{UnknownTimeCount: 1}})
		raw, _ := json.Marshal(section)
		require.NotContains(t, string(raw), `"label":"已保存订单数"`)
		require.NotContains(t, string(raw), `"label":"已保存订单商品金额"`)
	}
	grouped := []o.CurrencyOrderAmount{{Currency: "USD", Amount: "1.25", OrderCount: 1}, {Currency: "CNY", Amount: "9.00", OrderCount: 1}}
	two := 2
	raw, _ = json.Marshal(renderOrders(o.OrderAmountMetrics{Status: "complete", MatchedOrderCount: &two, Amounts: &grouped}))
	require.Contains(t, string(raw), "USD")
	require.Contains(t, string(raw), "1.25")
	require.Contains(t, string(raw), "9.00")
	require.Contains(t, string(raw), "不代表销售额、实收或利润")
}
