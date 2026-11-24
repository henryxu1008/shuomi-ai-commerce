package observations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"time"
)

const amazonOrderCoverageLag = 2 * time.Minute

type AmazonOrderReader interface {
	ReadAmazonOrders(context.Context, Scope, Binding, AmazonOrderQuery) (AmazonOrderPage, error)
}

type AmazonOrderProgress struct {
	Cursor       string   `json:"cursor,omitempty"`
	CursorHashes []string `json:"cursorHashes,omitempty"`
	Complete     bool     `json:"complete"`
}

func (s *Service) stepAmazonOrders(ctx context.Context, source Sync, merchant Merchant) (Sync, error) {
	if s.AmazonOrders == nil || !source.Binding.ValidAmazonOrderApplication() || source.Range == nil ||
		len(source.Progress.Windows) != 1 || source.Progress.Windows[0] != *source.Range {
		return Sync{}, ErrUnsupported
	}
	next := copyCheckpoint(source.Progress)
	if next.AmazonOrders == nil {
		next.AmazonOrders = &AmazonOrderProgress{}
	}
	progress := next.AmazonOrders
	page, err := s.AmazonOrders.ReadAmazonOrders(ctx, source.Owner, source.Binding, AmazonOrderQuery{
		Start: source.Range.Start, End: source.Range.End, Cursor: progress.Cursor,
	})
	if err != nil {
		return Sync{}, err
	}
	if err := s.Access.Authorize(ctx, source.Owner, Orders, true); err != nil {
		return Sync{}, err
	}
	if err := merchant.Check(ctx); err != nil {
		return Sync{}, err
	}
	if merchant.Binding() != source.Binding {
		return Sync{}, ErrConflict
	}
	if len(page.Orders) > AmazonOrderPageSize {
		return Sync{}, ErrUnavailable
	}
	now := s.now()
	rows := amazonOrderRecords(source, &next, page.Orders, now)
	next.Pages++
	next.Seen += len(page.Orders)
	if page.Incomplete {
		next.Note("amazon_order_page_incomplete")
	}
	status := advanceAmazonOrderPage(&next, page.Next)
	return s.Repository.CommitPage(ctx, source, next, rows, status, now)
}

func amazonOrderRecords(source Sync, next *Checkpoint, orders []AmazonOrder, at time.Time) []Record {
	rows := make([]Record, 0, len(orders))
	positions := make(map[string]int, len(orders))
	for _, order := range orders {
		if !order.Valid() || order.CreatedAt.Before(source.Range.Start) || order.CreatedAt.After(source.Range.End) {
			next.Note("amazon_order_identity_or_range")
			continue
		}
		if index, seen := positions[order.ID]; seen {
			previous := rows[index].AmazonOrder
			if order.UpdatedAt.After(previous.UpdatedAt) {
				rows[index].AmazonOrder = &order
			} else if order.UpdatedAt.Equal(previous.UpdatedAt) && !reflect.DeepEqual(order, *previous) {
				next.Note("amazon_order_identity_conflict")
			}
			continue
		}
		positions[order.ID] = len(rows)
		rows = append(rows, Record{StoreID: source.StoreID, SyncID: source.ID, ID: order.ID,
			AmazonOrder: &order, WindowKey: source.Range.Key(), ObservedAt: at})
	}
	return rows
}

func advanceAmazonOrderPage(next *Checkpoint, cursor string) string {
	progress := next.AmazonOrders
	if cursor == "" {
		progress.Cursor = ""
		progress.CursorHashes = nil
		progress.Complete = !next.Incomplete
		next.Windows = []Window{}
		if next.Incomplete {
			return "partial"
		}
		return "completed"
	}
	if len(progress.CursorHashes) >= 512 || !Text(cursor, 2048) {
		next.Note("amazon_order_page_limit")
		return "partial"
	}
	digest := sha256.Sum256([]byte(cursor))
	hash := hex.EncodeToString(digest[:])
	for _, prior := range progress.CursorHashes {
		if prior == hash {
			next.Note("provider_cursor_cycle")
			return "partial"
		}
	}
	progress.Cursor = cursor
	progress.CursorHashes = append(progress.CursorHashes, hash)
	next.Page++
	return "running"
}
