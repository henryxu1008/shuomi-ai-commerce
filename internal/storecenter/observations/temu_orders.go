package observations

import (
	"context"
	"regexp"
	"time"
)

type TemuOrder struct {
	ID       string `json:"id"`
	Site     string `json:"site"`
	RegionID int64  `json:"regionId"`
	Date     string `json:"date"`
	Status   string `json:"status,omitempty"`
	Amount   *Price `json:"amount"`
}

var temuOrderDecimal = regexp.MustCompile(`^[0-9]{1,17}\.[0-9]{2}$`)

func (o TemuOrder) Valid() bool {
	if !Identity(o.ID) || o.Site != "temu-order-us" || o.RegionID != 211 || !Text(o.Status, 64) {
		return false
	}
	if o.Date != "" {
		v, e := time.Parse(time.RFC3339, o.Date)
		if e != nil || v.Year() < 1 || v.Year() > 9999 || v.Nanosecond() != 0 {
			return false
		}
	}
	return o.Amount == nil || (o.Amount.Special == "" && (o.Amount.Value == "" || temuOrderDecimal.MatchString(o.Amount.Value)) && (o.Amount.Currency == "" || o.Amount.Currency == "USD"))
}

type TemuOrderQuery struct {
	Start, End time.Time
	Page       int
}

// ClosedCreationRange is the exact official inclusive seconds query.
func (q TemuOrderQuery) ClosedCreationRange() (int64, int64, error) {
	if q.Page < 1 || !q.Start.Before(q.End) || q.Start.Nanosecond() != 0 || q.End.Nanosecond() != 0 {
		return 0, 0, ErrInvalid
	}
	return q.Start.Unix(), q.End.Unix() - 1, nil
}

type TemuOrderPage struct {
	Total int
	Items []TemuOrder
}
type TemuOrderReader interface {
	ReadTemuOrders(context.Context, Scope, Binding, TemuOrderQuery) (TemuOrderPage, error)
}
type TemuOrderProgress struct {
	Total    *int `json:"total,omitempty"`
	Complete bool `json:"complete"`
}

func advanceTemuOrders(current Checkpoint, count, total int) (Checkpoint, string, error) {
	c := copyCheckpoint(current)
	if c.TemuOrders == nil {
		c.TemuOrders = &TemuOrderProgress{}
	}
	p := c.TemuOrders
	if p.Complete || c.Page < 1 || len(c.Windows) == 0 || count < 0 || count > 100 || total < 0 {
		return c, "", ErrInvalid
	}
	c.Pages++
	c.Seen += count
	if p.Total == nil {
		v := total
		p.Total = &v
	} else if *p.Total != total {
		c.Note("temu_order_count_changed")
		return c, "partial", nil
	}
	if total > 10000 || c.Seen > total || c.Pages >= 30000 {
		c.Note("temu_order_limit")
		return c, "partial", nil
	}
	if c.Seen < total {
		if count < 100 || c.Page >= 100 {
			c.Note("temu_order_short_page")
			return c, "partial", nil
		}
		c.Page++
		return c, "running", nil
	}
	c.Page = 1
	c.Seen = 0
	p.Total = nil
	c.Windows = c.Windows[1:]
	if len(c.Windows) > 0 {
		return c, "running", nil
	}
	p.Complete = !c.Incomplete
	return finish(c)
}
func (s *Service) stepTemuOrders(ctx context.Context, source Sync, merchant Merchant) (Sync, error) {
	if s.TemuOrders == nil || !source.Binding.ValidTemuOrderApplication() || source.Range == nil || len(source.Progress.Windows) == 0 {
		return Sync{}, ErrUnavailable
	}
	c := copyCheckpoint(source.Progress)
	w := c.Windows[0]
	page, e := s.TemuOrders.ReadTemuOrders(ctx, source.Owner, source.Binding, TemuOrderQuery{Start: w.Start, End: w.End, Page: c.Page})
	if e != nil {
		return Sync{}, e
	}
	if len(page.Items) > 100 {
		return Sync{}, ErrUnavailable
	}
	rows := []Record{}
	seen := map[string]bool{}
	for _, order := range page.Items {
		if !order.Valid() || seen[order.ID] {
			return Sync{}, ErrUnavailable
		}
		seen[order.ID] = true
		if order.Date != "" {
			date, _ := time.Parse(time.RFC3339, order.Date)
			if date.Before(w.Start) || !date.Before(w.End) {
				return Sync{}, ErrUnavailable
			}
		}
		value := order
		rows = append(rows, Record{StoreID: source.StoreID, SyncID: source.ID, ID: order.ID, TemuOrder: &value, WindowKey: w.Key(), ObservedAt: s.now()})
	}
	if e = s.Access.Authorize(ctx, source.Owner, Orders, true); e != nil {
		return Sync{}, e
	}
	if e = merchant.Check(ctx); e != nil {
		return Sync{}, e
	}
	if merchant.Binding() != source.Binding {
		return Sync{}, ErrConflict
	}
	next, status, e := advanceTemuOrders(c, len(page.Items), page.Total)
	if e != nil {
		return Sync{}, e
	}
	return s.Repository.CommitPage(ctx, source, next, rows, status, s.now())
}
