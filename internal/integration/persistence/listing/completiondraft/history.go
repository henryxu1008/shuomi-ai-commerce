package completiondraftpersistence

import (
	"context"
	"encoding/json"
	"errors"

	"task-processor/internal/listing/completiondraft"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/collection"
)

// Read the qualified retained source and observed head once. History is bounded
// by that head; later saves never change already committed records.
func (r *Repository) historyHead(ctx context.Context, scope collection.Scope, sourceID string) (completiondraft.HistoryPage, error) {
	page := completiondraft.HistoryPage{Items: []completiondraft.Record{}, DraftID: collection.StableID(scope.OrganizationID, scope.ActorID, sourceID, "completion-draft")}
	if ctx == nil || scope.Validate() != nil || !collection.ValidID(sourceID) {
		return page, preparation.ErrInvalid
	}
	db := r.db.WithContext(ctx)
	var source sourceRow
	if err := dbError(filter(db.Table("listing_preparation_sources"), scope).Where("id=?", sourceID).Take(&source).Error); err != nil {
		return page, err
	}
	page.Source = source.value()
	var head headRow
	err := dbError(filter(db, scope).Where("source_id=?", sourceID).Take(&head).Error)
	if errors.Is(err, preparation.ErrNotFound) {
		return page, nil
	}
	if err != nil {
		return page, err
	}
	record, err := readRecord(db, scope, head.RecordID)
	if err != nil || record.Source != page.Source || record.DraftID != page.DraftID || head.DraftID != page.DraftID || record.Revision != head.Revision || head.Revision > completiondraft.MaxHistoryRevision {
		return page, preparation.ErrUnavailable
	}
	page.CurrentRevision = head.Revision
	return page, nil
}

func (r *Repository) ListHistory(ctx context.Context, scope collection.Scope, sourceID string, q completiondraft.HistoryQuery) (completiondraft.HistoryPage, error) {
	q, err := q.Normalize()
	if err != nil {
		return completiondraft.HistoryPage{}, err
	}
	page, err := r.historyHead(ctx, scope, sourceID)
	if err != nil || page.CurrentRevision == 0 {
		return page, err
	}
	base := filter(r.db.WithContext(ctx), scope).Where("source_id=? AND draft_id=? AND revision<=?", sourceID, page.DraftID, page.CurrentRevision)
	if q.Before > 0 {
		base = base.Where("revision<?", q.Before)
	}
	var rows []recordRow
	if base.Order("revision DESC").Limit(q.Limit+1).Find(&rows).Error != nil {
		return completiondraft.HistoryPage{}, preparation.ErrUnavailable
	}
	if len(rows) > q.Limit {
		rows = rows[:q.Limit]
		page.NextBefore = rows[len(rows)-1].Revision
	}
	encodedBytes := 0
	for _, row := range rows {
		record, err := row.value()
		if err != nil || record.Source != page.Source || record.DraftID != page.DraftID {
			return completiondraft.HistoryPage{}, preparation.ErrUnavailable
		}
		raw, err := json.Marshal(record)
		if err != nil {
			return completiondraft.HistoryPage{}, preparation.ErrUnavailable
		}
		if encodedBytes+len(raw)+1 > (2<<20)-4096 {
			if len(page.Items) == 0 {
				return completiondraft.HistoryPage{}, preparation.ErrUnavailable
			}
			page.NextBefore = page.Items[len(page.Items)-1].Revision
			break
		}
		encodedBytes += len(raw) + 1
		page.Items = append(page.Items, record)
	}
	return page, nil
}

// ReadRevision is source/revision-qualified, separate from the internal Target
// consumer's record-ID-based ReadVersion contract.
func (r *Repository) ReadRevision(ctx context.Context, scope collection.Scope, sourceID string, revision int64) (completiondraft.Record, error) {
	if revision < 1 || revision > completiondraft.MaxHistoryRevision {
		return completiondraft.Record{}, preparation.ErrInvalid
	}
	page, err := r.historyHead(ctx, scope, sourceID)
	if err != nil {
		return completiondraft.Record{}, err
	}
	if page.CurrentRevision < revision {
		return completiondraft.Record{}, preparation.ErrNotFound
	}
	var row recordRow
	if err = dbError(filter(r.db.WithContext(ctx), scope).Where("source_id=? AND draft_id=? AND revision=?", sourceID, page.DraftID, revision).Take(&row).Error); err != nil {
		return completiondraft.Record{}, err
	}
	record, err := row.value()
	if err != nil || record.Source != page.Source || record.DraftID != page.DraftID {
		return completiondraft.Record{}, preparation.ErrUnavailable
	}
	return record, nil
}
