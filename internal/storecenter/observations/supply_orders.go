package observations

import (
	"context"
	"encoding/json"
	"math/big"
	"sort"
	"strings"
)

const SupplyOrders Kind = "supply-orders"
const SupplyChunkSize = 20

func NormalizeSupplyOrderIDs(ids []string) ([]string, error) {
	if len(ids) == 0 || len(ids) > 200 {
		return nil, ErrInvalid
	}
	set := map[string]bool{}
	for _, id := range ids {
		if !Identity(id) || strings.ContainsAny(id, ",/\\?#") {
			return nil, ErrInvalid
		}
		set[id] = true
	}
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Strings(out)
	// A missing receipt can repeat every identity in MissingIDs. Reserve 4 KiB
	// of the original 64 KiB checkpoint for JSONB spacing, counters and notes.
	worst, e := json.Marshal(SupplyProgress{OrderIDs: out, Next: len(out), MissingIDs: out})
	if e != nil || len(worst) > 60<<10 {
		return nil, ErrInvalid
	}
	return out, nil
}

type SupplyLine struct {
	SKU       string `json:"sku"`
	Ordered   *int64 `json:"ordered"`
	Delivered *int64 `json:"delivered"`
	Received  *int64 `json:"received"`
	Stored    *int64 `json:"stored"`
	Defective *int64 `json:"defective"`
}
type SupplyOrder struct {
	BusinessTimes SupplyBusinessTimes `json:"businessTimes"`
	ID            string              `json:"id"`
	Type          *int                `json:"type"`
	Status        *int                `json:"status"`
	Scene         *int                `json:"scene"`
	CountryMarket *int                `json:"countryMarket"`
	Mother        *bool               `json:"mother"`
	MotherRaw     string              `json:"motherRaw"`
	Lines         []SupplyLine        `json:"lines"`
}

func SupplyMother(raw string) *bool {
	var yes bool
	switch raw {
	case "是", "Yes", "yes":
		yes = true
	case "否", "No", "no":
	default:
		return nil
	}
	return &yes
}
func (s SupplyOrder) Valid() bool {
	if !s.BusinessTimes.Valid() {
		return false
	}
	if _, e := NormalizeSupplyOrderIDs([]string{s.ID}); e != nil {
		return false
	}
	if !Text(s.MotherRaw, 24) || len(s.Lines) > 1000 {
		return false
	}
	mother := SupplyMother(s.MotherRaw)
	if (mother == nil) != (s.Mother == nil) || mother != nil && *mother != *s.Mother {
		return false
	}
	for _, v := range []*int{s.Type, s.Status, s.Scene, s.CountryMarket} {
		if v != nil && (*v < 0 || *v > 1000000) {
			return false
		}
	}
	for _, line := range s.Lines {
		if !Identity(line.SKU) {
			return false
		}
		for _, v := range []*int64{line.Ordered, line.Delivered, line.Received, line.Stored, line.Defective} {
			if v != nil && (*v < 0 || *v > 100000000) {
				return false
			}
		}
	}
	return true
}

type SupplyPage struct {
	Items []SupplyOrder
	Notes []string
}
type SupplyOrderReader interface {
	SupplyOrders(context.Context, []string) (SupplyPage, error)
}
type SupplyProgress struct {
	OrderIDs   []string `json:"orderIds"`
	Next       int      `json:"next"`
	MissingIDs []string `json:"missingIds"`
}
type SupplyQuantity struct {
	Value        *string `json:"value"`
	MissingLines int     `json:"missingLines"`
}
type SupplyQuantities struct {
	Ordered   SupplyQuantity `json:"ordered"`
	Delivered SupplyQuantity `json:"delivered"`
	Received  SupplyQuantity `json:"received"`
	Stored    SupplyQuantity `json:"stored"`
	Defective SupplyQuantity `json:"defective"`
}
type SupplySummary struct {
	OrderCount       int              `json:"orderCount"`
	MotherCount      int              `json:"motherCount"`
	UnknownTypeCount int              `json:"unknownTypeCount"`
	Quantities       SupplyQuantities `json:"quantities"`
}

type supplySummaryAccumulator struct {
	summary SupplySummary
	sums    [5]big.Int
	missing [5]int
}

func (a *supplySummaryAccumulator) add(order SupplyOrder) {
	if order.Mother == nil {
		a.summary.UnknownTypeCount++
		return
	}
	if *order.Mother {
		a.summary.MotherCount++
		return
	}
	a.summary.OrderCount++
	seen := map[string]bool{}
	duplicate := false
	for _, line := range order.Lines {
		if seen[line.SKU] {
			duplicate = true
		}
		seen[line.SKU] = true
	}
	if len(order.Lines) == 0 {
		for i := range a.missing {
			a.missing[i]++
		}
		return
	}
	for _, line := range order.Lines {
		for i, value := range []*int64{line.Ordered, line.Delivered, line.Received, line.Stored, line.Defective} {
			if duplicate || value == nil {
				a.missing[i]++
				continue
			}
			a.sums[i].Add(&a.sums[i], big.NewInt(*value))
		}
	}
}

func (a *supplySummaryAccumulator) result() SupplySummary {
	result := a.summary
	fields := []*SupplyQuantity{&result.Quantities.Ordered, &result.Quantities.Delivered, &result.Quantities.Received, &result.Quantities.Stored, &result.Quantities.Defective}
	for i, field := range fields {
		field.MissingLines = a.missing[i]
		if field.MissingLines == 0 {
			value := a.sums[i].String()
			field.Value = &value
		}
	}
	return result
}
