package researchpersistence

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/research"
	"time"
)

const assessmentTable = "public.research_assessments"

type assessmentRow struct {
	OrganizationID, ActorID, ProjectID, RequestKey, CandidateID string
	CandidateRevision                                           int64
	RecordJSON                                                  []byte
	CreatedAt                                                   time.Time
}

func (assessmentRow) TableName() string      { return "research_assessments" }
func (r *Repository) AssessmentsReady() bool { return r.assessments }
func (r *Repository) CommitAssessment(ctx context.Context, s collection.Scope, p, key string, c research.Command, proof research.AssessmentProof, validation error) (research.Receipt, error) {
	if !r.assessments || c.Action != research.SaveAssessment && c.AssessmentRef == nil {
		return research.Receipt{}, research.ErrInvalid
	}
	if validation == nil {
		if proof.Expires.IsZero() {
			return research.Receipt{}, research.ErrUnavailable
		}
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, proof.Expires)
		defer cancel()
	}
	return r.commit(ctx, s, p, key, c, nil, validation, nil, &proof)
}
func insertAssessment(tx *gorm.DB, s collection.Scope, p, key string, c research.Command, revision int64, now time.Time, proof research.AssessmentProof) error {
	rec := proof.Record
	if rec.RuleVersion != research.CostRuleVersion || rec.InputHash != collection.Digest(rec.Input) || rec.Result.Status == "INVALID_INPUT" || rec.ContextHash == "" || rec.RequestHash != collection.Digest(c) {
		return research.ErrConflict
	}
	raw, e := json.Marshal(rec)
	if e != nil || len(raw) > 32<<10 {
		return research.ErrInvalid
	}
	row := assessmentRow{OrganizationID: s.OrganizationID, ActorID: s.ActorID, ProjectID: p, RequestKey: key, CandidateID: c.CandidateID, CandidateRevision: revision, RecordJSON: raw, CreatedAt: now}
	if tx.Session(&gorm.Session{Logger: logger.Discard}).Create(&row).Error != nil {
		return research.ErrUnavailable
	}
	return nil
}
func decodeAssessment(row assessmentRow) (research.AssessmentRow, error) {
	var rec research.AssessmentRecord
	if json.Unmarshal(row.RecordJSON, &rec) != nil || rec.RuleVersion == "" {
		return research.AssessmentRow{}, research.ErrUnavailable
	}
	return research.AssessmentRow{ID: row.RequestKey, Revision: row.CandidateRevision, At: row.CreatedAt, Record: rec}, nil
}
func loadAssessment(db *gorm.DB, s collection.Scope, p, candidate, id string) (research.AssessmentRow, error) {
	var row assessmentRow
	e := scoped(db, s, p).Where("candidate_id=? AND request_key=?", candidate, id).Take(&row).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return research.AssessmentRow{}, research.ErrNotFound
	}
	if e != nil {
		return research.AssessmentRow{}, research.ErrUnavailable
	}
	return decodeAssessment(row)
}
func (r *Repository) LoadAssessment(ctx context.Context, s collection.Scope, p, candidate, id string) (research.AssessmentRow, error) {
	return loadAssessment(r.db.WithContext(ctx), s, p, candidate, id)
}
func (r *Repository) ListAssessments(ctx context.Context, s collection.Scope, p, candidate string, before int64) ([]research.AssessmentRow, error) {
	q := scoped(r.db.WithContext(ctx), s, p).Where("candidate_id=? AND record_json->>'memberId'=?", candidate, s.MemberID)
	if before > 0 {
		q = q.Where("candidate_revision<?", before)
	}
	var rows []assessmentRow
	if q.Order("candidate_revision DESC").Limit(6).Find(&rows).Error != nil {
		return nil, research.ErrUnavailable
	}
	out := make([]research.AssessmentRow, 0, len(rows))
	for _, row := range rows {
		v, e := decodeAssessment(row)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, nil
}
func (r *Repository) LoadSupplierReview(ctx context.Context, s collection.Scope, p, candidate, key string) (research.SupplierReviewRow, error) {
	var row supplierReviewRow
	e := scoped(r.db.WithContext(ctx), s, p).Where("candidate_id=? AND request_key=?", candidate, key).Take(&row).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return research.SupplierReviewRow{}, research.ErrNotFound
	}
	if e != nil {
		return research.SupplierReviewRow{}, research.ErrUnavailable
	}
	var rec research.SupplierReviewRecord
	if json.Unmarshal(row.RecordJSON, &rec) != nil {
		return research.SupplierReviewRow{}, research.ErrUnavailable
	}
	return research.SupplierReviewRow{RequestKey: row.RequestKey, Revision: row.CandidateRevision, At: row.CreatedAt, Record: rec}, nil
}
func scopedReceipt(db *gorm.DB, s collection.Scope, p string, row commandRow) (research.Receipt, error) {
	out, e := decodeReceipt(row)
	if e != nil {
		return out, e
	}
	if out.AssessmentRef != nil {
		if row.CandidateID == nil {
			return research.Receipt{}, research.ErrUnavailable
		}
		saved, e := loadAssessment(db, s, p, *row.CandidateID, out.AssessmentRef.ID)
		if e != nil {
			return research.Receipt{}, e
		}
		if saved.Record.MemberID != s.MemberID {
			return research.Receipt{}, research.ErrNotFound
		}
	}
	return out, nil
}
