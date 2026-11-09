package observations

import (
	"context"
	"reflect"
	"task-processor/internal/storecenter"
)

func ValidSupplyProgress(p *SupplyProgress) bool {
	if p == nil || p.Next < 0 || p.Next > len(p.OrderIDs) {
		return false
	}
	ids, e := NormalizeSupplyOrderIDs(p.OrderIDs)
	if e != nil || !reflect.DeepEqual(ids, p.OrderIDs) {
		return false
	}
	processed := map[string]bool{}
	for _, id := range ids[:p.Next] {
		processed[id] = true
	}
	seen := map[string]bool{}
	for _, id := range p.MissingIDs {
		if !processed[id] || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}

// This validates the new kind within the original CAS transaction, not a new state owner.
func ValidateSupplyCommit(current Sync, next Checkpoint, records []Record, status string) error {
	p, q := current.Progress.SupplyOrders, next.SupplyOrders
	if current.Kind != SupplyOrders || !ValidSupplyProgress(p) || !ValidSupplyProgress(q) || !reflect.DeepEqual(p.OrderIDs, q.OrderIDs) || current.Binding.Platform != storecenter.PlatformShein || current.Binding.ApplicationType != storecenter.ApplicationFullyManaged {
		return ErrInvalid
	}
	if status == "partial" && len(records) == 0 && q.Next == p.Next && reflect.DeepEqual(p.MissingIDs, q.MissingIDs) && next.Seen == current.Progress.Seen && next.Pages == current.Progress.Pages && next.Page == current.Progress.Page && next.Incomplete {
		return nil
	}
	end := min(p.Next+SupplyChunkSize, len(p.OrderIDs))
	if p.Next == end || q.Next != end {
		return ErrInvalid
	}
	allowed := map[string]bool{}
	for _, id := range p.OrderIDs[p.Next:end] {
		allowed[id] = true
	}
	seen := map[string]bool{}
	for _, record := range records {
		if !allowed[record.ID] || seen[record.ID] || record.SupplyOrder == nil || record.SupplyOrder.ID != record.ID || !record.SupplyOrder.Valid() {
			return ErrInvalid
		}
		seen[record.ID] = true
	}
	missing := append([]string{}, p.MissingIDs...)
	for _, id := range p.OrderIDs[p.Next:end] {
		if !seen[id] {
			missing = append(missing, id)
		}
	}
	if !reflect.DeepEqual(missing, q.MissingIDs) || next.Seen != current.Progress.Seen+len(records) || next.Pages != current.Progress.Pages+1 || next.Page != current.Progress.Page+1 {
		return ErrInvalid
	}
	if status == "completed" && (q.Next != len(q.OrderIDs) || len(missing) > 0 || next.Incomplete) {
		return ErrInvalid
	}
	if q.Next < len(q.OrderIDs) && status != "running" {
		return ErrInvalid
	}
	return nil
}

func (s *Service) stepSupplyOrders(ctx context.Context, sync Sync, merchant Merchant) (Sync, error) {
	reader, ok := merchant.(SupplyOrderReader)
	if !ok || !ValidSupplyProgress(sync.Progress.SupplyOrders) {
		return Sync{}, ErrUnavailable
	}
	c := copyCheckpoint(sync.Progress)
	p := c.SupplyOrders
	end := min(p.Next+SupplyChunkSize, len(p.OrderIDs))
	if p.Next == end {
		return Sync{}, ErrConflict
	}
	ids := p.OrderIDs[p.Next:end]
	page, e := reader.SupplyOrders(ctx, ids)
	if e != nil {
		return Sync{}, e
	}
	if len(page.Items) > len(ids) {
		return Sync{}, ErrConflict
	}
	allowed, seen := map[string]bool{}, map[string]bool{}
	for _, id := range ids {
		allowed[id] = true
	}
	records := make([]Record, 0, len(page.Items))
	for _, order := range page.Items {
		if !allowed[order.ID] || seen[order.ID] || !order.Valid() {
			return Sync{}, ErrConflict
		}
		seen[order.ID] = true
		item := order
		records = append(records, Record{StoreID: sync.StoreID, SyncID: sync.ID, ID: item.ID, SupplyOrder: &item})
	}
	for _, note := range page.Notes {
		c.Note(note)
	}
	for _, id := range ids {
		if !seen[id] {
			p.MissingIDs = append(p.MissingIDs, id)
			c.Note("supply_ids_not_returned")
		}
	}
	p.Next = end
	c.Seen += len(records)
	c.Pages++
	c.Page++
	status := "running"
	if end == len(p.OrderIDs) {
		status = "completed"
		if c.Incomplete {
			status = "partial"
		}
	}
	if e = merchant.Check(ctx); e != nil {
		return Sync{}, e
	}
	return s.Repository.CommitPage(ctx, sync, c, records, status, s.now())
}
