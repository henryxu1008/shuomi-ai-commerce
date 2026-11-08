package observations

import (
	"context"
	"math/big"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"
	"task-processor/internal/storecenter"
	"time"
)

const MaxOrderAmountCandidates = 250

type OrderAmountQuery struct {
	StoreID    string
	Start, End time.Time
}
type OrderAmountRecord struct {
	StoreID, SyncID, ID, Site, IssuedAt string
	Amount                              *Price
}
type OrderAmountReader interface {
	OrderAmountRecords(context.Context, string, string, OrderAmountQuery, int) ([]OrderAmountRecord, error)
}
type CurrencyOrderAmount struct {
	Currency   string `json:"currency"`
	Amount     string `json:"amount"`
	OrderCount int    `json:"orderCount"`
}
type OrderAmountMissing struct {
	AmountCount      int `json:"amountCount"`
	CurrencyCount    int `json:"currencyCount"`
	UnknownTimeCount int `json:"unknownTimeCount"`
}
type OrderAmountSource struct {
	SyncID     string     `json:"syncId"`
	ObservedAt *time.Time `json:"observedAt"`
	Range      *Window    `json:"range"`
}
type OrderAmountMetrics struct {
	Status            string                 `json:"status"`
	CapturedAt        time.Time              `json:"capturedAt"`
	Source            *OrderAmountSource     `json:"source,omitempty"`
	Reason            string                 `json:"reason,omitempty"`
	MatchedOrderCount *int                   `json:"matchedOrderCount,omitempty"`
	Missing           *OrderAmountMissing    `json:"missing,omitempty"`
	Amounts           *[]CurrencyOrderAmount `json:"amounts,omitempty"`
}

func (q OrderAmountQuery) valid(now time.Time) bool {
	zone := time.FixedZone("UTC+8", 8*3600)
	start, end := q.Start.In(zone), q.End.In(zone)
	midnight := func(t time.Time) bool {
		return t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0
	}
	today := time.Date(now.In(zone).Year(), now.In(zone).Month(), now.In(zone).Day(), 0, 0, 0, 0, zone)
	return ValidID(q.StoreID) && start.Year() > 0 && start.Before(end) && end.Sub(start) <= 30*24*time.Hour && !end.After(today) && midnight(start) && midnight(end)
}
func (s *Service) SavedOrderAmounts(ctx context.Context, scope Scope, q OrderAmountQuery) (OrderAmountMetrics, error) {
	result := OrderAmountMetrics{Status: "coverage_incomplete", Reason: "no_saved_sync"}
	if !s.available() || s.Directory == nil {
		return result, ErrUnavailable
	}
	now := s.now()
	result.CapturedAt = now
	if !scope.Valid() {
		return result, ErrForbidden
	}
	if !q.valid(now) {
		return result, ErrInvalid
	}
	reader, ok := s.Repository.(OrderAmountReader)
	if !ok {
		return result, ErrUnavailable
	}
	if _, e := s.allowedStores(ctx, scope, Orders, false, []string{q.StoreID}, "shein"); e != nil {
		return OrderAmountMetrics{}, e
	}
	heads, latest, e := s.Repository.Heads(ctx, scope.OrganizationID, Orders, []string{q.StoreID})
	if e != nil {
		return OrderAmountMetrics{}, ErrUnavailable
	}
	heads, latest = freezeOrderAmountSources(heads), freezeOrderAmountSources(latest)
	var handle Merchant
	if len(heads) > 1 || len(latest) > 1 {
		return OrderAmountMetrics{}, ErrUnavailable
	}
	if len(heads) == 1 {
		head := heads[0]
		if !ValidID(head.ID) || head.StoreID != q.StoreID || head.Kind != Orders || head.Owner.OrganizationID != scope.OrganizationID {
			return OrderAmountMetrics{}, ErrUnavailable
		}
		handle, e = s.Access.Open(ctx, scope, q.StoreID, Orders, false, &head.Binding)
		if e != nil {
			return OrderAmountMetrics{}, e
		}
		binding := handle.Binding()
		if binding != head.Binding || binding.OrganizationID != scope.OrganizationID || binding.StoreID != q.StoreID || binding.Site != "shein-us" || (binding.Platform != "" && binding.Platform != storecenter.PlatformShein) {
			return OrderAmountMetrics{}, ErrUnavailable
		}
		result.Source = &OrderAmountSource{SyncID: head.ID, ObservedAt: head.ObservedAt, Range: head.Range}
		result.Reason = "sync_incomplete"
		if head.Status == "completed" && !head.Progress.Incomplete && head.ObservedAt != nil && !head.ObservedAt.IsZero() {
			result.Reason = "newer_sync_incomplete"
			if len(latest) == 1 && reflect.DeepEqual(head, latest[0]) {
				result.Reason = "range_not_covered"
				if head.Range != nil && !head.Range.Start.After(q.Start) && !head.Range.End.Before(q.End.Add(-time.Second)) {
					rows, err := reader.OrderAmountRecords(ctx, scope.OrganizationID, head.ID, q, MaxOrderAmountCandidates+1)
					if err != nil {
						return OrderAmountMetrics{}, ErrUnavailable
					}
					if len(rows) > MaxOrderAmountCandidates {
						result.Status = "limited"
						result.Reason = ""
					} else {
						result, err = aggregateOrderAmounts(result, rows, q)
						if err != nil {
							return OrderAmountMetrics{}, err
						}
					}
				}
			}
		}
	}
	if ctx.Err() != nil {
		return OrderAmountMetrics{}, ErrUnavailable
	}
	if handle != nil {
		if e = handle.Check(ctx); e != nil {
			return OrderAmountMetrics{}, e
		}
	}
	if _, e = s.allowedStores(ctx, scope, Orders, false, []string{q.StoreID}, "shein"); e != nil {
		return OrderAmountMetrics{}, e
	}
	current, currentLatest, e := s.Repository.Heads(ctx, scope.OrganizationID, Orders, []string{q.StoreID})
	if e != nil || !reflect.DeepEqual(heads, current) || !reflect.DeepEqual(latest, currentLatest) {
		return OrderAmountMetrics{}, ErrUnavailable
	}
	return result, nil
}

var orderAmountDecimal = regexp.MustCompile(`^[0-9]{1,20}(\.[0-9]{1,8})?$`)
var orderAmountCurrency = regexp.MustCompile(`^[A-Z]{3}$`)

func aggregateOrderAmounts(result OrderAmountMetrics, rows []OrderAmountRecord, q OrderAmountQuery) (OrderAmountMetrics, error) {
	return aggregateOrderAmountsWithDecimal(result, rows, q, orderAmountDecimal, "shein-us")
}

func aggregateOrderAmountsWithDecimal(result OrderAmountMetrics, rows []OrderAmountRecord, q OrderAmountQuery, decimal *regexp.Regexp, site string) (OrderAmountMetrics, error) {
	matched := 0
	missing := OrderAmountMissing{}
	seen := map[string]OrderAmountRecord{}
	totals := map[string]*big.Int{}
	counts := map[string]int{}
	for _, row := range rows {
		if row.StoreID != q.StoreID || result.Source == nil || row.SyncID != result.Source.SyncID || !Identity(row.ID) || row.Site != site {
			return OrderAmountMetrics{}, ErrUnavailable
		}
		if prior, exists := seen[row.ID]; exists {
			if !reflect.DeepEqual(prior, row) {
				return OrderAmountMetrics{}, ErrUnavailable
			}
			continue
		}
		seen[row.ID] = row
		if row.IssuedAt == "" {
			missing.UnknownTimeCount++
			continue
		}
		issued, e := time.Parse(time.RFC3339, row.IssuedAt)
		if e != nil {
			return OrderAmountMetrics{}, ErrUnavailable
		}
		if issued.Before(q.Start) || !issued.Before(q.End) {
			continue
		}
		matched++
		if row.Amount == nil || row.Amount.Value == "" {
			missing.AmountCount++
		}
		if row.Amount == nil || row.Amount.Currency == "" {
			missing.CurrencyCount++
		}
		if row.Amount == nil {
			continue
		}
		if row.Amount.Value != "" && !decimal.MatchString(row.Amount.Value) || row.Amount.Currency != "" && !orderAmountCurrency.MatchString(row.Amount.Currency) {
			return OrderAmountMetrics{}, ErrUnavailable
		}
		if row.Amount.Value == "" || row.Amount.Currency == "" {
			continue
		}
		parts := strings.SplitN(row.Amount.Value, ".", 2)
		fraction := ""
		if len(parts) == 2 {
			fraction = parts[1]
		}
		scaled, ok := new(big.Int).SetString(parts[0]+fraction+strings.Repeat("0", 8-len(fraction)), 10)
		if !ok {
			return OrderAmountMetrics{}, ErrUnavailable
		}
		currency := row.Amount.Currency
		if totals[currency] == nil {
			totals[currency] = new(big.Int)
		}
		totals[currency].Add(totals[currency], scaled)
		counts[currency]++
	}
	result.Reason = ""
	result.MatchedOrderCount = &matched
	if missing.AmountCount+missing.CurrencyCount+missing.UnknownTimeCount > 0 {
		result.Status = "missing_fields"
		result.Missing = &missing
		return result, nil
	}
	amounts := []CurrencyOrderAmount{}
	currencies := []string{}
	for c := range totals {
		currencies = append(currencies, c)
	}
	sort.Strings(currencies)
	for _, c := range currencies {
		text := totals[c].String()
		if len(text) < 9 {
			text = strings.Repeat("0", 9-len(text)) + text
		}
		whole, fraction := text[:len(text)-8], strings.TrimRight(text[len(text)-8:], "0")
		if len(fraction) < 2 {
			fraction += strings.Repeat("0", 2-len(fraction))
		}
		amounts = append(amounts, CurrencyOrderAmount{Currency: c, Amount: whole + "." + fraction, OrderCount: counts[c]})
	}
	result.Status = "complete"
	result.Amounts = &amounts
	return result, nil
}

// Keep the initial metadata stable even when a repository reuses decoded values.
func freezeOrderAmountSources(values []Sync) []Sync {
	result := slices.Clone(values)
	for i := range result {
		v := &result[i]
		if v.Range != nil {
			r := *v.Range
			v.Range = &r
		}
		if v.ObservedAt != nil {
			at := *v.ObservedAt
			v.ObservedAt = &at
		}
		if v.Progress.ExpectedTotal != nil {
			total := *v.Progress.ExpectedTotal
			v.Progress.ExpectedTotal = &total
		}
		v.Progress.Windows = slices.Clone(v.Progress.Windows)
		v.Progress.Notes = slices.Clone(v.Progress.Notes)
		v.Progress.TemuTokenHashes = slices.Clone(v.Progress.TemuTokenHashes)
		if v.Progress.TikTokOrders != nil {
			p := *v.Progress.TikTokOrders
			p.TokenHashes = slices.Clone(p.TokenHashes)
			if p.Total != nil {
				total := *p.Total
				p.Total = &total
			}
			v.Progress.TikTokOrders = &p
		}
		if v.Progress.WalmartOrders != nil {
			p := *v.Progress.WalmartOrders
			p.CursorHashes = slices.Clone(p.CursorHashes)
			if p.Total != nil {
				n := *p.Total
				p.Total = &n
			}
			v.Progress.WalmartOrders = &p
		}
	}
	return result
}
