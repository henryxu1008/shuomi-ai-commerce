package reportcenterpersistence

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	rc "task-processor/internal/reportcenter"
	"time"
)

func scheduledReceipt(db *gorm.DB, o rc.ScheduledOrigin) (rc.CaptureReceipt, bool, error) {
	id, e := lookup(db, o.Scope(), o.Key(), o.Fingerprint())
	if e != nil || id == "" {
		return rc.CaptureReceipt{}, false, e
	}
	var row reportRow
	if e = scoped(db, o.Scope()).Where("id=?", id).Take(&row).Error; e != nil {
		return rc.CaptureReceipt{}, false, rc.ErrUnavailable
	}
	var origin rc.ScheduledOrigin
	if row.Kind != rc.StoreDailyReportKind || row.SourceID != o.OccurrenceID || row.SourceVersion != "1" || row.StoreID != o.StoreID || row.ProductKey != "" || json.Unmarshal(row.ScheduledOrigin, &origin) != nil || origin != o {
		return rc.CaptureReceipt{}, false, rc.ErrConflict
	}
	return rc.CaptureReceipt{ReportID: row.ID, Digest: row.Digest, Origin: origin}, true, nil
}

// Metadata only: an internal reconciliation capability, not a user read API.
func (s *Store) ReadScheduledCaptureReceipt(ctx context.Context, o rc.ScheduledOrigin) (rc.CaptureReceipt, bool, error) {
	if s == nil || s.db == nil || !o.Valid() {
		return rc.CaptureReceipt{}, false, rc.ErrInvalid
	}
	return scheduledReceipt(s.db.WithContext(ctx), o)
}
func (s *Store) SaveScheduledCapture(ctx context.Context, o rc.ScheduledOrigin, snap rc.Snapshot, check func(context.Context, *rc.Snapshot) error) (rc.CaptureReceipt, error) {
	if s == nil || s.db == nil || !o.Valid() || check == nil || snap.ScheduledOrigin == nil || *snap.ScheduledOrigin != o || snap.Ref != (rc.SourceRef{Kind: rc.StoreDailyReportKind, ID: o.OccurrenceID, Version: "1"}) {
		return rc.CaptureReceipt{}, rc.ErrInvalid
	}
	var e error
	var result rc.CaptureReceipt
	e = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Concurrent readers may have captured different live snapshots. Serialize
		// by intent, then replay the first committed winner without comparing bodies.
		if e := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", o.OrganizationID+"\x1f"+o.ActorID+"\x1f"+o.Key()).Error; e != nil {
			return e
		}
		r, found, e := scheduledReceipt(tx, o)
		if e != nil {
			return e
		}
		if found {
			result = r
			return nil
		}
		if e = check(ctx, &snap); e != nil {
			return e
		}
		if snap.ScheduledOrigin == nil || *snap.ScheduledOrigin != o || snap.Ref != (rc.SourceRef{Kind: rc.StoreDailyReportKind, ID: o.OccurrenceID, Version: "1"}) {
			return rc.ErrInvalid
		}
		body, digest, e := snap.Bytes()
		if e != nil {
			return e
		}
		raw, e := json.Marshal(o)
		if e != nil {
			return rc.ErrInvalid
		}
		row := reportRow{ID: o.Key(), OrganizationID: o.OrganizationID, ActorID: o.ActorID, Kind: rc.StoreDailyReportKind, SourceID: o.OccurrenceID, SourceVersion: "1", Title: snap.Title, StoreID: o.StoreID, SourceAt: snap.SourceAt, CapturedAt: time.Now().UTC(), Content: body, Digest: digest, ScheduledOrigin: raw}
		if e = tx.Create(&row).Error; e != nil {
			return e
		}
		if e = tx.Create(&favoriteRow{ReportID: row.ID, OrganizationID: o.OrganizationID, ActorID: o.ActorID, UpdatedAt: row.CapturedAt}).Error; e != nil {
			return e
		}
		if e = tx.Create(&commandRow{o.OrganizationID, o.ActorID, o.Key(), o.Fingerprint(), row.ID}).Error; e != nil {
			return e
		}
		result = rc.CaptureReceipt{ReportID: row.ID, Digest: digest, Origin: o}
		return nil
	})
	if errors.Is(e, gorm.ErrRecordNotFound) {
		e = rc.ErrUnavailable
	}
	return result, e
}
