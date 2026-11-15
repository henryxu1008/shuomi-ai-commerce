package aiworkbenchpersistence

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"task-processor/internal/aiworkbench"
	rc "task-processor/internal/reportcenter"
	"time"
)

type reportExecutionRow struct {
	OccurrenceID   string
	OrganizationID string
	OwnerUserID    string
	Intent         json.RawMessage `gorm:"type:jsonb"`
	Fingerprint    string
	CaptureStarted bool
	ClosedNoWrite  bool
	ReportID       *string
	ReportDigest   *string
	Proof          *string
}

func (reportExecutionRow) TableName() string { return "ai_workbench.report_execution" }

func (s *Store) ClaimScheduledReport(ctx context.Context, scope aiworkbench.Scope, id string, revision int64, slot, now time.Time) (aiworkbench.ScheduledOccurrence, aiworkbench.ReportExecution, error) {
	var occurrence aiworkbench.ScheduledOccurrence
	var execution aiworkbench.ReportExecution
	if !validScope(scope) || !validKey(id) || revision < 1 || slot.IsZero() || now.IsZero() {
		return occurrence, execution, aiworkbench.ErrInvalid
	}
	e := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row scheduledPlanRow
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND organization_id=? AND owner_user_id=?", id, scope.OrganizationID, scope.ActorID).Take(&row).Error; e != nil {
			return missing(e)
		}
		store := &Store{db: tx}
		if old, e := store.ReadScheduledOccurrence(ctx, scope, id, slot); e == nil {
			occurrence = old
			if old.State == "SKIPPED" || old.State == "BLOCKED" {
				return nil
			}
			execution, e = store.ReadReportExecution(ctx, scope, old.ID)
			return e
		} else if !errors.Is(e, aiworkbench.ErrNotFound) {
			return e
		}
		p, e := schedulePlanValue(row)
		if e != nil {
			return e
		}
		if p.Report == nil || row.Revision != revision || row.RegistrationState != "MATCHED" {
			return aiworkbench.ErrRevisionMismatch
		}
		// A callback may only claim a real slot of this frozen rule.
		next, e := p.Rule.NextAfter(slot.Add(-time.Nanosecond))
		if e != nil || !next.Equal(slot) {
			return aiworkbench.ErrInvalidSchedule
		}
		occurrence, _, e = store.ClaimScheduledOccurrence(ctx, scope, id, slot, now)
		if e != nil {
			return e
		}
		if occurrence.State != "CLAIMED" {
			return nil
		}
		o := rc.ScheduledOrigin{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: p.MemberID, PlanID: id, PlanRevision: revision, OccurrenceID: occurrence.ID, ScheduledAt: slot.UTC(), StoreID: p.Report.StoreID, Day: p.Report.Period(slot), RendererVersion: 1}
		if !o.Valid() {
			return aiworkbench.ErrInvalid
		}
		raw, _ := json.Marshal(o)
		if e = tx.Create(&reportExecutionRow{OccurrenceID: occurrence.ID, OrganizationID: scope.OrganizationID, OwnerUserID: scope.ActorID, Intent: raw, Fingerprint: o.Fingerprint()}).Error; e != nil {
			return unavailable(e)
		}
		execution = aiworkbench.ReportExecution{Origin: o}
		return nil
	})
	return occurrence, execution, e
}
func (s *Store) ReadReportExecution(ctx context.Context, scope aiworkbench.Scope, id string) (aiworkbench.ReportExecution, error) {
	if s == nil || s.db == nil || !validScope(scope) || !validKey(id) {
		return aiworkbench.ReportExecution{}, aiworkbench.ErrInvalid
	}
	var row reportExecutionRow
	if e := s.db.WithContext(ctx).Where("occurrence_id=? AND organization_id=? AND owner_user_id=?", id, scope.OrganizationID, scope.ActorID).Take(&row).Error; e != nil {
		return aiworkbench.ReportExecution{}, missing(e)
	}
	var o rc.ScheduledOrigin
	if json.Unmarshal(row.Intent, &o) != nil || !o.Valid() || o.Scope() != (rc.Scope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}) || o.OccurrenceID != id || o.Fingerprint() != row.Fingerprint {
		return aiworkbench.ReportExecution{}, aiworkbench.ErrUnavailable
	}
	result := aiworkbench.ReportExecution{Origin: o, CaptureStarted: row.CaptureStarted, ClosedNoWrite: row.ClosedNoWrite}
	if row.ReportID != nil {
		result.ReportID = *row.ReportID
	}
	if row.ReportDigest != nil {
		result.ReportDigest = *row.ReportDigest
	}
	if row.Proof != nil {
		result.Proof = *row.Proof
	}
	return result, nil
}

// Successful commit is required before entering Report DB. A delayed worker
// cannot cross a definite-no-write fence established by a concurrent rejection.
func (s *Store) BeginReportCapture(ctx context.Context, o rc.ScheduledOrigin) error {
	if !o.Valid() {
		return aiworkbench.ErrInvalid
	}
	res := s.db.WithContext(ctx).Model(&reportExecutionRow{}).Where("occurrence_id=? AND organization_id=? AND owner_user_id=? AND fingerprint=? AND closed_no_write=FALSE AND report_id IS NULL", o.OccurrenceID, o.OrganizationID, o.ActorID, o.Fingerprint()).Update("capture_started", true)
	if res.Error != nil {
		return unavailable(res.Error)
	}
	if res.RowsAffected != 1 {
		return aiworkbench.ErrRevisionMismatch
	}
	return nil
}
func (s *Store) BlockUnstartedReport(ctx context.Context, o rc.ScheduledOrigin) error {
	if !o.Valid() {
		return aiworkbench.ErrInvalid
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var plan scheduledPlanRow
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND organization_id=? AND owner_user_id=?", o.PlanID, o.OrganizationID, o.ActorID).Take(&plan).Error; e != nil {
			return missing(e)
		}
		res := tx.Model(&reportExecutionRow{}).Where("occurrence_id=? AND organization_id=? AND owner_user_id=? AND fingerprint=? AND capture_started=FALSE", o.OccurrenceID, o.OrganizationID, o.ActorID, o.Fingerprint()).Update("closed_no_write", true)
		if res.Error != nil {
			return unavailable(res.Error)
		}
		if res.RowsAffected != 1 {
			return aiworkbench.ErrRevisionMismatch
		}
		if e := tx.Model(&scheduledOccurrenceRow{}).Where("id=? AND plan_id=? AND organization_id=? AND owner_user_id=? AND business_task_id IS NULL", o.OccurrenceID, o.PlanID, o.OrganizationID, o.ActorID).Updates(map[string]any{"state": "BLOCKED", "reason": "CURRENT_AUTHORIZATION_UNAVAILABLE", "updated_at": time.Now().UTC()}).Error; e != nil {
			return e
		}
		return tx.Model(&scheduledPlanRow{}).Where("id=? AND active_occurrence_id=?", o.PlanID, o.OccurrenceID).Updates(map[string]any{"active_occurrence_id": nil, "updated_at": time.Now().UTC()}).Error
	})
}
func (s *Store) FinishReportCapture(ctx context.Context, r rc.CaptureReceipt) error {
	o := r.Origin
	if !r.ValidFor(o) || !validDigest(r.Digest) {
		return aiworkbench.ErrInvalid
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var plan scheduledPlanRow
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND organization_id=? AND owner_user_id=?", o.PlanID, o.OrganizationID, o.ActorID).Take(&plan).Error; e != nil {
			return missing(e)
		}
		execution, e := (&Store{db: tx}).ReadReportExecution(ctx, aiworkbench.Scope{OrganizationID: o.OrganizationID, ActorID: o.ActorID}, o.OccurrenceID)
		if e != nil {
			return e
		}
		if execution.Origin != o || !execution.CaptureStarted || execution.ClosedNoWrite {
			return aiworkbench.ErrRevisionMismatch
		}
		if execution.ReportID != "" && (execution.ReportID != r.ReportID || execution.ReportDigest != r.Digest || execution.Proof != r.Proof()) {
			return aiworkbench.ErrIdempotencyConflict
		}
		if e = tx.Model(&reportExecutionRow{}).Where("occurrence_id=? AND fingerprint=?", o.OccurrenceID, o.Fingerprint()).Updates(map[string]any{"report_id": r.ReportID, "report_digest": r.Digest, "proof": r.Proof()}).Error; e != nil {
			return unavailable(e)
		}
		if e = tx.Model(&scheduledOccurrenceRow{}).Where("id=? AND organization_id=? AND owner_user_id=? AND plan_id=? AND business_task_id IS NULL", o.OccurrenceID, o.OrganizationID, o.ActorID, o.PlanID).Updates(map[string]any{"state": "RECONCILED", "reason": "REPORT_SAVED", "reconciliation_digest": r.Proof(), "updated_at": time.Now().UTC()}).Error; e != nil {
			return unavailable(e)
		}
		return unavailable(tx.Model(&scheduledPlanRow{}).Where("id=? AND active_occurrence_id=?", o.PlanID, o.OccurrenceID).Updates(map[string]any{"active_occurrence_id": nil, "updated_at": time.Now().UTC()}).Error)
	})
}
