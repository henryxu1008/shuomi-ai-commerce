package observations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
)

const shopifyOrderPageLimit = 2500

type ShopifyOrderReader interface {
	ReadShopifyOrders(context.Context, Scope, Binding, ShopifyOrderQuery) (ShopifyOrderPage, error)
}

type ShopifyOrderProgress struct {
	Cursor       string   `json:"cursor,omitempty"`
	CursorHashes []string `json:"cursorHashes,omitempty"`
	Complete     bool     `json:"complete"`
}

func validShopifyOrderStatus(s string) bool {
	switch s {
	case "", "cancelled", "fulfilled", "unfulfilled", "partial", "unknown":
		return true
	default:
		return false
	}
}

func (s *Service) stepShopifyOrders(ctx context.Context, source Sync, merchant Merchant) (Sync, error) {
	if s.ShopifyOrders == nil || !source.Binding.ValidShopifyOrderApplication() || source.Range == nil ||
		len(source.Progress.Windows) != 1 || source.Progress.Windows[0] != *source.Range {
		return Sync{}, ErrUnsupported
	}
	next := copyCheckpoint(source.Progress)
	if next.ShopifyOrders == nil {
		next.ShopifyOrders = &ShopifyOrderProgress{}
	}
	progress := next.ShopifyOrders
	page, err := s.ShopifyOrders.ReadShopifyOrders(ctx, source.Owner, source.Binding, ShopifyOrderQuery{
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
	if len(page.Orders) > 10 {
		return Sync{}, ErrUnavailable
	}
	now := s.now()
	rows := make([]Record, 0, len(page.Orders))
	positions := make(map[string]int, len(page.Orders))
	for _, order := range page.Orders {
		if !order.Valid() || order.CreatedAt.Before(source.Range.Start) || !order.CreatedAt.Before(source.Range.End) {
			next.Note("shopify_order_identity_or_range")
			continue
		}
		if index, seen := positions[order.ID]; seen {
			previous := rows[index].ShopifyOrder
			if order.UpdatedAt.After(previous.UpdatedAt) {
				rows[index].ShopifyOrder = &order
			} else if order.UpdatedAt.Equal(previous.UpdatedAt) && !reflect.DeepEqual(order, *previous) {
				next.Note("shopify_order_identity_conflict")
			}
			continue
		}
		positions[order.ID] = len(rows)
		rows = append(rows, Record{StoreID: source.StoreID, SyncID: source.ID, ID: order.ID,
			ShopifyOrder: &order, WindowKey: source.Range.Key(), ObservedAt: now})
	}
	next.Pages++
	next.Seen += len(page.Orders)
	if page.Incomplete {
		next.Note("shopify_order_page_incomplete")
	}
	if page.Next == "" {
		progress.Cursor = ""
		progress.CursorHashes = nil
		progress.Complete = !next.Incomplete
		next.Windows = []Window{}
		status := "completed"
		if next.Incomplete {
			status = "partial"
		}
		return s.Repository.CommitPage(ctx, source, next, rows, status, now)
	}
	digest := sha256.Sum256([]byte(page.Next))
	hash := hex.EncodeToString(digest[:])
	// Retain fewer hashes than the existing 64 KiB checkpoint allows, and
	// conservatively stop well before the provider's 25,000 order maximum.
	if next.Pages >= shopifyOrderPageLimit || len(progress.CursorHashes) >= 512 || len(page.Next) > 2048 || !Text(page.Next, 2048) {
		next.Note("shopify_order_page_limit")
		return s.Repository.CommitPage(ctx, source, next, rows, "partial", now)
	}
	for _, prior := range progress.CursorHashes {
		if prior == hash {
			next.Note("provider_cursor_cycle")
			return s.Repository.CommitPage(ctx, source, next, rows, "partial", now)
		}
	}
	progress.Cursor = page.Next
	progress.CursorHashes = append(progress.CursorHashes, hash)
	next.Page++
	return s.Repository.CommitPage(ctx, source, next, rows, "running", now)
}
