package scheduledreport

import (
	"context"
	"fmt"
	storeapp "task-processor/internal/app/storecenter"
	observationapp "task-processor/internal/app/storeobservations"
	rc "task-processor/internal/reportcenter"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
	"time"
)

func ordersUnavailable(reason string) rc.Section {
	return rc.Section{Title: "已保存订单观察", Fields: []rc.Field{{Label: "资料状态", Value: reason}}}
}

type oneStoreDirectory struct {
	app    *Application
	origin rc.ScheduledOrigin
}

func (d oneStoreDirectory) ListStores(ctx context.Context, s o.Scope, k o.Kind, platform string) ([]string, error) {
	if s != (o.Scope{OrganizationID: d.origin.OrganizationID, ActorID: d.origin.ActorID, MemberID: d.origin.MemberID}) || k != o.Orders || platform != "shein" {
		return nil, o.ErrForbidden
	}
	store, e := d.app.Store(ctx, d.origin, true)
	if e != nil {
		return nil, e
	}
	if store.Platform() != storecenter.PlatformShein {
		return nil, o.ErrUnsupported
	}
	return []string{store.ID()}, nil
}
func (a *Application) savedOrders(ctx context.Context, origin rc.ScheduledOrigin, platform storecenter.Platform) (rc.Section, func(context.Context) error) {
	if platform != storecenter.PlatformShein {
		return ordersUnavailable("该平台的已保存订单尚未接入日报"), nil
	}
	if a.d.Observations == nil || a.d.Observations.Repository == nil || a.d.Applications == nil {
		return ordersUnavailable("尚无可读取的已保存订单来源"), nil
	}
	stores, e := storecenter.NewMemberScopedStoreRepository(a.d.StoreDB, memberAccess{a, origin, true})
	if e != nil {
		return ordersUnavailable("店铺订单来源暂不可用"), nil
	}
	authorization := observationapp.Authorization{Client: a.d.IAM, ServiceToken: a.d.ServiceToken, ProjectID: a.d.ProjectID, Permissions: a.d.Policy, OrganizationStatus: a.d.OrganizationStatus}
	official, e := storeapp.NewOfficialObservationAccess(stores, authorization, a.d.Applications)
	if e != nil {
		return ordersUnavailable("店铺订单来源暂不可用"), nil
	}
	access := observationapp.Access{Authorization: authorization, Official: official}
	service := o.Service{Repository: a.d.Observations.Repository, Access: access, Directory: oneStoreDirectory{a, origin}}
	scope := o.Scope{OrganizationID: origin.OrganizationID, ActorID: origin.ActorID, MemberID: origin.MemberID}
	// Freeze the original saved-source binding before reading the existing narrow
	// projection. A newer/different head is not silently substituted at commit.
	heads, _, e := service.Repository.Heads(ctx, origin.OrganizationID, o.Orders, []string{origin.StoreID})
	if e != nil || len(heads) > 1 {
		return ordersUnavailable("已保存订单来源暂不可用"), nil
	}
	day, e := time.ParseInLocation(time.DateOnly, origin.Day, time.FixedZone("UTC+8", 8*3600))
	if e != nil {
		return ordersUnavailable("统计日期暂不可用"), nil
	}
	metrics, e := service.SavedOrderAmounts(ctx, scope, o.OrderAmountQuery{StoreID: origin.StoreID, Start: day, End: day.AddDate(0, 0, 1)})
	if e != nil {
		return ordersUnavailable("当前无法读取该店铺的已保存订单"), nil
	}
	if metrics.Source == nil {
		return renderOrders(metrics), nil
	}
	if len(heads) != 1 || heads[0].ID != metrics.Source.SyncID {
		return ordersUnavailable("订单来源发生变化，请等待下次报告"), nil
	}
	handle, e := access.Open(ctx, scope, origin.StoreID, o.Orders, false, &heads[0].Binding)
	if e != nil {
		return ordersUnavailable("当前无法读取该店铺的已保存订单"), nil
	}
	return renderOrders(metrics), func(ctx context.Context) error {
		if _, e := a.Store(ctx, origin, true); e != nil {
			return e
		}
		if e := access.Authorize(ctx, scope, o.Orders, false); e != nil {
			return e
		}
		return handle.Check(ctx)
	}
}
func renderOrders(m o.OrderAmountMetrics) rc.Section {
	status := map[string]string{"complete": "资料完整", "coverage_incomplete": "同步范围不完整", "missing_fields": "部分订单字段缺失", "limited": "超过现有读取上限（250 条），未计算完整结果"}[m.Status]
	if status == "" {
		return ordersUnavailable("已保存订单资料暂不可用")
	}
	fields := []rc.Field{{Label: "资料状态", Value: status}, {Label: "统计口径", Value: "按订单创建时间统计已保存的记录，不按状态额外过滤、不改原订单状态；金额为订单商品金额（原币），不代表销售额、实收或利润"}}
	if m.Source != nil {
		fields = append(fields, rc.Field{Label: "原同步记录", Value: m.Source.SyncID})
		if m.Source.ObservedAt != nil {
			fields = append(fields, rc.Field{Label: "原同步取得时间", Value: m.Source.ObservedAt.UTC().Format(time.RFC3339)})
		}
		if m.Source.Range != nil {
			fields = append(fields, rc.Field{Label: "原同步范围", Value: m.Source.Range.Start.Format(time.RFC3339) + " 至 " + m.Source.Range.End.Format(time.RFC3339)})
		}
	}
	if m.Status == "complete" && m.MatchedOrderCount != nil && m.Amounts != nil {
		fields = append(fields, rc.Field{Label: "已保存订单数", Value: fmt.Sprint(*m.MatchedOrderCount)})
		if len(*m.Amounts) == 0 {
			fields = append(fields, rc.Field{Label: "已保存订单商品金额", Value: "该日无已保存订单"})
		}
		for _, amount := range *m.Amounts {
			fields = append(fields, rc.Field{Label: "已保存订单商品金额 · " + amount.Currency, Value: amount.Amount})
		}
	} else if m.Missing != nil {
		fields = append(fields, rc.Field{Label: "缺失字段", Value: fmt.Sprintf("金额 %d 条，币种 %d 条，创建时间 %d 条；未显示完整当日金额或订单数", m.Missing.AmountCount, m.Missing.CurrencyCount, m.Missing.UnknownTimeCount)})
	}
	return rc.Section{Title: "已保存订单观察", Fields: fields}
}
