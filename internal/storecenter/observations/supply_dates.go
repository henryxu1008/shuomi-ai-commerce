package observations

import "time"

// This is a range of source calendar labels, not instants in a platform zone.
type SupplyDateRange struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

func validSupplyDate(s string) bool {
	if len(s) != 10 {
		return false
	}
	d, e := time.Parse(time.DateOnly, s)
	return e == nil && d.Year() > 0 && d.Format(time.DateOnly) == s
}
func (q SupplyDateRange) Valid() bool {
	return validSupplyDate(q.StartDate) && validSupplyDate(q.EndDate) && q.StartDate <= q.EndDate
}

type SupplyDateGroup struct {
	OrderIDs []string       `json:"orderIds"`
	Ordered  SupplyQuantity `json:"ordered"`
}
type SupplyUndatedOrder struct {
	ID    string `json:"id"`
	Field string `json:"field"`
	State string `json:"state"`
}
type SupplyOriginalDates struct {
	Range                  SupplyDateRange      `json:"range"`
	Urgent                 SupplyDateGroup      `json:"urgent"`
	Stock                  SupplyDateGroup      `json:"stock"`
	Undated                []SupplyUndatedOrder `json:"undated"`
	MotherIDs              []string             `json:"motherIds"`
	UnknownMotherIDs       []string             `json:"unknownMotherIds"`
	UnknownPurchaseTypeIDs []string             `json:"unknownPurchaseTypeIds"`
	OutsideRangeIDs        []string             `json:"outsideRangeIds"`
}
type supplyDateAccumulator struct {
	resultData    SupplyOriginalDates
	urgent, stock supplySummaryAccumulator
}

func (a *supplyDateAccumulator) add(order SupplyOrder, q SupplyDateRange) {
	r := &a.resultData
	if order.Mother == nil {
		r.UnknownMotherIDs = append(r.UnknownMotherIDs, order.ID)
		return
	}
	if *order.Mother {
		r.MotherIDs = append(r.MotherIDs, order.ID)
		return
	}
	field := "addTime"
	if order.Type == nil || (*order.Type != 1 && *order.Type != 2) {
		r.UnknownPurchaseTypeIDs = append(r.UnknownPurchaseTypeIDs, order.ID)
		return
	}
	if *order.Type == 2 {
		field = "allocateTime"
	}
	value := order.BusinessTimes.Values[field]
	// The caller verifies the whole original bundle. Recheck the selected text
	// before indexing it; do not trim or normalize an offset-bearing timestamp.
	readable := value.RawType == "string" && value.Raw != nil && (value.State == "local_datetime" || value.State == "offset_datetime") && supplyStringState(*value.Raw) == value.State
	if !readable {
		r.Undated = append(r.Undated, SupplyUndatedOrder{ID: order.ID, Field: field, State: value.State})
		return
	}
	date := (*value.Raw)[:10]
	if !validSupplyDate(date) {
		r.Undated = append(r.Undated, SupplyUndatedOrder{ID: order.ID, Field: field, State: "unusable_date"})
		return
	}
	if date < q.StartDate || date > q.EndDate {
		r.OutsideRangeIDs = append(r.OutsideRangeIDs, order.ID)
		return
	}
	if *order.Type == 1 {
		r.Urgent.OrderIDs = append(r.Urgent.OrderIDs, order.ID)
		a.urgent.add(order)
	} else {
		r.Stock.OrderIDs = append(r.Stock.OrderIDs, order.ID)
		a.stock.add(order)
	}
}
func (a *supplyDateAccumulator) result(q SupplyDateRange) SupplyOriginalDates {
	r := a.resultData
	r.Range = q
	if r.Urgent.OrderIDs == nil {
		r.Urgent.OrderIDs = []string{}
	}
	if r.Stock.OrderIDs == nil {
		r.Stock.OrderIDs = []string{}
	}
	if r.Undated == nil {
		r.Undated = []SupplyUndatedOrder{}
	}
	if r.MotherIDs == nil {
		r.MotherIDs = []string{}
	}
	if r.UnknownMotherIDs == nil {
		r.UnknownMotherIDs = []string{}
	}
	if r.UnknownPurchaseTypeIDs == nil {
		r.UnknownPurchaseTypeIDs = []string{}
	}
	if r.OutsideRangeIDs == nil {
		r.OutsideRangeIDs = []string{}
	}
	r.Urgent.Ordered = a.urgent.result().Quantities.Ordered
	r.Stock.Ordered = a.stock.result().Quantities.Ordered
	return r
}
