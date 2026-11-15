package scheduledreport

import (
	"encoding/json"
	"fmt"
	"math/big"
	c "task-processor/internal/operationscockpit"
	rc "task-processor/internal/reportcenter"
)

func money(v int64) string {
	n := big.NewInt(v)
	negative := n.Sign() < 0
	n.Abs(n)
	q, r := new(big.Int), new(big.Int)
	q.QuoRem(n, big.NewInt(100), r)
	sign := ""
	if negative {
		sign = "-"
	}
	return fmt.Sprintf("%s%s.%02d 元", sign, q.String(), r.Int64())
}

// Render presents the Cockpit's existing calculations without reinterpreting
// missing coverage as a zero, or reimplementing financial business rules.
func Render(o rc.ScheduledOrigin, name, platform string, s c.Snapshot) rc.Snapshot {
	current := s.Stores[o.StoreID]
	status := "资料不全"
	if current.Complete {
		status = "资料完整"
	}
	fields := []rc.Field{{Label: "资料状态", Value: status}, {Label: "统计日期", Value: o.Day + "（UTC+8）"}, {Label: "店铺", Value: name}, {Label: "平台", Value: platform}, {Label: "资料取得时间", Value: s.CapturedAt.UTC().Format("2006-01-02 15:04:05 UTC")}, {Label: "统计来源", Value: "已保存的经营财务资料；未实时查询平台"}}
	finance := []rc.Field{}
	if len(current.Records) == 0 {
		finance = append(finance, rc.Field{Label: "经营财务", Value: "尚无完整覆盖该日的财务资料，金额未知"})
	} else {
		label := ""
		if !current.Complete {
			label = "已覆盖部分 · "
		}
		for _, p := range []struct {
			name  string
			value int64
		}{{"收入", current.Totals.Revenue}, {"退款", current.Totals.Refunds}, {"采购成本", current.Totals.Procurement}, {"物流成本", current.Totals.Logistics}, {"平台费用", current.Totals.Platform}, {"广告费用", current.Totals.Advertising}, {"其他费用", current.Totals.Other}, {"净收入", current.Totals.NetRevenue}, {"净利润", current.Totals.NetProfit}} {
			finance = append(finance, rc.Field{Label: label + p.name, Value: money(p.value)})
		}
	}
	growth := "资料不足，无法比较"
	if g := c.Growth(current, s.Previous[o.StoreID]); g != nil {
		n, ok := new(big.Rat).SetString(g.Numerator + "/" + g.Denominator)
		if ok {
			growth = new(big.Rat).Mul(n, big.NewRat(100, 1)).FloatString(2) + "%"
		}
	}
	finance = append(finance, rc.Field{Label: "净利润较前一日", Value: growth})
	evidence, _ := json.Marshal(struct {
		Included, Excluded []c.RecordEvidence
		Gaps               []c.Period
	}{current.Records, current.Excluded, current.Gaps})
	return rc.Snapshot{SourceInfo: rc.SourceInfo{Ref: rc.SourceRef{Kind: rc.StoreDailyReportKind, ID: o.OccurrenceID, Version: "1"}, Title: "店铺经营日报 · " + o.Day, StoreID: o.StoreID, SourceAt: &s.CapturedAt, ScheduledOrigin: &o}, Content: rc.Document{SchemaVersion: 1, Sections: []rc.Section{{Title: "经营日报", Fields: fields}, {Title: "经营财务", Fields: finance}, {Title: "资料依据", Fields: []rc.Field{{Label: "原财务记录与覆盖范围", Value: string(evidence)}}}}}}
}
