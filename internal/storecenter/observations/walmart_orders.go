package observations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"task-processor/internal/marketplace/walmart"
	"time"
)

// The runtime intentionally does not register a network reader in this phase.
// An implementation must pin merchant/market/range, include replacement orders,
// avoid status filters, check the live orders scope and use only a fixed GET.
type WalmartOrderReader interface {
	ReadWalmartOrders(context.Context, Scope, Binding, WalmartOrderQuery) (walmart.OrderPage, error)
}
type WalmartOrderQuery struct {
	Start, End time.Time
	Partition  int
	Cursor     string
}

var WalmartOrderPartitions = [3]string{"SellerFulfilled", "WFSFulfilled", "3PLFulfilled"}

type WalmartOrderProgress struct {
	Partition    int      `json:"partition"`
	Cursor       string   `json:"cursor,omitempty"`
	CursorHashes []string `json:"cursorHashes,omitempty"`
	Total        *int     `json:"total,omitempty"`
	Complete     bool     `json:"complete"`
}

func advanceWalmartOrders(current Checkpoint, count, total int, cursor string) (Checkpoint, string, error) {
	c := copyCheckpoint(current)
	if c.WalmartOrders == nil {
		c.WalmartOrders = &WalmartOrderProgress{}
	}
	p := c.WalmartOrders
	if p.Partition < 0 || p.Partition > 2 || p.Complete || c.Page < 1 || len(c.Windows) == 0 || count < 0 || count > 30 || total < 0 || len(cursor) > 8192 || strings.ContainsAny(cursor, "\r\n\x00") {
		return c, "", ErrInvalid
	}
	c.Pages++
	c.Seen += count
	if p.Total == nil {
		v := total
		p.Total = &v
	} else if *p.Total != total {
		c.Note("walmart_order_count_changed")
	}
	if total > 10000 || c.Seen > 10000 || c.Pages >= 30000 {
		c.Note("walmart_order_limit")
		return c, "partial", nil
	}
	if cursor != "" {
		hash := sha256.Sum256([]byte(cursor))
		digest := hex.EncodeToString(hash[:])
		for _, old := range p.CursorHashes {
			if old == digest {
				c.Note("provider_cursor_cycle")
				return c, "partial", nil
			}
		}
		if c.Page >= 512 || len(p.CursorHashes) >= 512 {
			c.Note("provider_checkpoint_limit")
			return c, "partial", nil
		}
		p.CursorHashes = append(p.CursorHashes, digest)
		p.Cursor = cursor
		c.Page++
		return c, "running", nil
	}
	// A cursor remains the end signal; inconsistent counts retain partial
	// coverage. Same-window duplicate pages are independently detected in SQL.
	if c.Seen != *p.Total {
		c.Note("walmart_order_count_mismatch")
	}
	p.Partition++
	p.Cursor = ""
	p.CursorHashes = nil
	p.Total = nil
	c.Page = 1
	c.Seen = 0
	if p.Partition < 3 {
		return c, "running", nil
	}
	p.Partition = 0
	c.Windows = c.Windows[1:]
	if len(c.Windows) > 0 {
		return c, "running", nil
	}
	p.Complete = !c.Incomplete
	return finish(c)
}
func (s *Service) stepWalmartOrders(ctx context.Context, source Sync, merchant Merchant) (Sync, error) {
	if s.WalmartOrders == nil || source.Range == nil || len(source.Progress.Windows) == 0 {
		return Sync{}, ErrUnavailable
	}
	c := copyCheckpoint(source.Progress)
	partition := 0
	cursor := ""
	if c.WalmartOrders != nil {
		partition = c.WalmartOrders.Partition
		cursor = c.WalmartOrders.Cursor
	}
	if partition < 0 || partition > 2 {
		return Sync{}, ErrConflict
	}
	w := c.Windows[0]
	page, e := s.WalmartOrders.ReadWalmartOrders(ctx, source.Owner, source.Binding, WalmartOrderQuery{Start: w.Start.Add(-time.Second), End: w.End.Add(time.Second), Partition: partition, Cursor: cursor})
	if e != nil {
		return Sync{}, e
	}
	if len(page.Items) > 30 {
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
			date, _ := time.Parse(time.RFC3339Nano, order.Date)
			if date.Before(source.Range.Start) || !date.Before(source.Range.End) {
				continue
			}
		}
		value := order
		rows = append(rows, Record{StoreID: source.StoreID, SyncID: source.ID, ID: order.ID, WalmartOrder: &value, WindowKey: w.Key() + "/" + string(rune('0'+partition)), ObservedAt: s.now()})
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
	next, status, e := advanceWalmartOrders(c, len(page.Items), page.Total, page.NextCursor)
	if e != nil {
		return Sync{}, e
	}
	return s.Repository.CommitPage(ctx, source, next, rows, status, s.now())
}
