package officialpersistence

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	submissionstore "task-processor/internal/integration/persistence/listing/submission"
	"task-processor/internal/listing/submission"
	marketplace "task-processor/internal/marketplace/walmart"
	"task-processor/internal/product/collection"
	"time"
)

type walmartObservationRow struct {
	OrganizationID, ID, ReceiptID string
	Generation                    int64
	BodyJSON                      []byte
	BodyHash                      string
	ObservedAt                    time.Time
}

func (walmartObservationRow) TableName() string { return WalmartObservationTable }
func lockWalmartTarget(tx *gorm.DB, receipt submission.OfficialReceipt) error {
	return advisoryLock(tx, "target", receipt.Owner.OrganizationID, "walmart", receipt.Target.StoreID, receipt.Target.SubjectID)
}
func (r *OfficialRepository) BeginWalmartRead(ctx context.Context, scope collection.Scope, id string) (out submission.WalmartResultRequest, err error) {
	if ctx == nil || scope.Validate() != nil {
		return out, submission.ErrExecutionInvalid
	}
	verified, e := readOfficialReceipt(r.db.WithContext(ctx), scope.OrganizationID, id)
	if e != nil || verified.Walmart == nil || r.validateCompleted(ctx, verified) != nil {
		return out, submission.ErrExecutionEvidenceRequired
	}
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		receipt, e := readOfficialReceipt(tx, scope.OrganizationID, id)
		if e != nil {
			return e
		}
		if receipt.Walmart == nil || collection.Digest(receipt) != collection.Digest(verified) {
			return submission.ErrExecutionEvidenceRequired
		}
		if e = lockWalmartTarget(tx, receipt); e != nil {
			return e
		}
		var rows []submissionstore.WalmartSKUState
		if e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id=? AND environment=? AND market='US' AND merchant_identity_hash=? AND seller_sku IN ?", scope.OrganizationID, receipt.Binding.Walmart.Environment, receipt.Binding.SupplierIdentityHash, submission.WalmartExpectedSKUs(receipt.Walmart.SKUs)).Find(&rows).Error; e != nil {
			return e
		}
		if len(rows) != len(receipt.Walmart.SKUs) {
			return submission.ErrExecutionEvidenceRequired
		}
		generation := int64(1)
		for _, s := range rows {
			if s.ReadGeneration >= generation {
				generation = s.ReadGeneration + 1
			}
		}
		if generation <= 0 {
			return submission.ErrExecutionUnavailable
		}
		for _, s := range rows {
			if s.ProductKey != receipt.ProductKey || s.StoreID != receipt.Target.StoreID || s.SubjectID != receipt.Target.SubjectID {
				return submission.ErrExecutionEvidenceRequired
			}
			if s.LastAttemptID == receipt.ID {
				out.TrackedSKUs = append(out.TrackedSKUs, s.SellerSKU)
			}
		}
		if len(out.TrackedSKUs) == 0 {
			return submission.ErrExecutionIntentConflict
		}
		updated := tx.Model(&submissionstore.WalmartSKUState{}).Where("organization_id=? AND environment=? AND market='US' AND merchant_identity_hash=? AND seller_sku IN ? AND last_attempt_id=?", scope.OrganizationID, receipt.Binding.Walmart.Environment, receipt.Binding.SupplierIdentityHash, submission.WalmartExpectedSKUs(receipt.Walmart.SKUs), receipt.ID).Updates(map[string]any{"read_generation": generation, "latest_observation_id": ""})
		if updated.Error != nil {
			return updated.Error
		}
		if int(updated.RowsAffected) != len(out.TrackedSKUs) {
			return submission.ErrExecutionIntentConflict
		}
		out.Receipt = receipt
		out.Generation = generation
		return nil
	})
	return
}
func (r *OfficialRepository) SaveWalmartRead(ctx context.Context, scope collection.Scope, req submission.WalmartResultRequest, o marketplace.FeedObservation) (out submission.WalmartResultRecord, err error) {
	if scope.Validate() != nil || req.Generation < 1 || len(req.TrackedSKUs) == 0 || req.Receipt.Owner.OrganizationID != scope.OrganizationID || req.Receipt.Walmart == nil || submission.ValidateOfficialReceipt(req.Receipt) != nil || o.FeedID != req.Receipt.Walmart.FeedID || o.ObservedAt.IsZero() {
		return out, submission.ErrExecutionEvidenceRequired
	}
	raw, e := json.Marshal(o)
	if e != nil || len(raw) > submission.MaxExecutionPayloadBytes {
		return out, submission.ErrExecutionInvalid
	}
	// Incomplete/missing/duplicate results are retained as partial, never used
	// as a retry admission receipt. The generation head is separate from success.
	if !o.CompleteFor(submission.WalmartExpectedSKUs(req.Receipt.Walmart.SKUs)) {
		o.Partial = true
		raw, _ = json.Marshal(o)
	}
	id := uuid.NewSHA1(uuid.NameSpaceURL, []byte(scope.OrganizationID+":"+req.Receipt.ID+":"+jsonGeneration(req.Generation))).String()
	hash := collection.Digest(o)
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		receipt, e := readOfficialReceipt(tx, scope.OrganizationID, req.Receipt.ID)
		if e != nil || collection.Digest(receipt) != collection.Digest(req.Receipt) {
			return submission.ErrExecutionEvidenceRequired
		}
		if e = lockWalmartTarget(tx, receipt); e != nil {
			return e
		}
		var prior walmartObservationRow
		e = tx.Where("organization_id=? AND receipt_id=? AND generation=?", scope.OrganizationID, receipt.ID, req.Generation).Take(&prior).Error
		if e == nil {
			if prior.BodyHash != hash {
				return submission.ErrExecutionIntentConflict
			}
			var heads int64
			if e = tx.Model(&submissionstore.WalmartSKUState{}).Where("organization_id=? AND last_attempt_id=? AND read_generation=? AND latest_observation_id=?", scope.OrganizationID, receipt.ID, req.Generation, prior.ID).Count(&heads).Error; e != nil {
				return e
			}
			out = submission.WalmartResultRecord{ID: prior.ID, ReceiptID: receipt.ID, Generation: req.Generation, Latest: int(heads) == len(receipt.Walmart.SKUs), Observation: o}
			return nil
		}
		if e != gorm.ErrRecordNotFound {
			return e
		}
		row := walmartObservationRow{OrganizationID: scope.OrganizationID, ID: id, ReceiptID: receipt.ID, Generation: req.Generation, BodyJSON: raw, BodyHash: hash, ObservedAt: o.ObservedAt}
		if e = tx.Create(&row).Error; e != nil {
			return e
		}
		updated := tx.Model(&submissionstore.WalmartSKUState{}).Where("organization_id=? AND environment=? AND market='US' AND merchant_identity_hash=? AND seller_sku IN ? AND last_attempt_id=? AND read_generation=?", scope.OrganizationID, receipt.Binding.Walmart.Environment, receipt.Binding.SupplierIdentityHash, submission.WalmartExpectedSKUs(receipt.Walmart.SKUs), receipt.ID, req.Generation).Update("latest_observation_id", id)
		if updated.Error != nil {
			return updated.Error
		}
		out = submission.WalmartResultRecord{ID: id, ReceiptID: receipt.ID, Generation: req.Generation, Latest: int(updated.RowsAffected) == len(receipt.Walmart.SKUs), Observation: o}
		return nil
	})
	return
}
func jsonGeneration(v int64) string { b, _ := json.Marshal(v); return string(b) }
func (r *OfficialRepository) ReadWalmartResults(ctx context.Context, scope collection.Scope, target submission.ExecutionTarget) ([]submission.WalmartResultRecord, error) {
	if scope.Validate() != nil || target.Platform != "walmart" || !collection.ValidID(target.StoreID) {
		return nil, submission.ErrExecutionInvalid
	}
	var rows []walmartObservationRow
	// One latest observation per receipt that still owns at least one SKU.
	// Repeated reads of a repair Feed must not evict unrelated live SKU facts.
	if e := r.db.WithContext(ctx).Raw(`SELECT * FROM (
 SELECT DISTINCT ON(o.receipt_id) o.* FROM listing_walmart_feed_observations o
 JOIN listing_submission_official_receipts r ON r.organization_id=o.organization_id AND r.attempt_id=o.receipt_id
 WHERE o.organization_id=? AND r.store_id=? AND r.subject_id=?
 AND EXISTS(SELECT 1 FROM listing_walmart_sku_bindings s WHERE s.organization_id=o.organization_id AND s.last_attempt_id=o.receipt_id)
 ORDER BY o.receipt_id,o.generation DESC
 ) live ORDER BY observed_at DESC,generation DESC LIMIT ?`, scope.OrganizationID, target.StoreID, target.SubjectID, marketplace.MaxSKUs).Scan(&rows).Error; e != nil {
		return nil, e
	}

	out := []submission.WalmartResultRecord{}
	for _, row := range rows {
		var obs marketplace.FeedObservation
		if len(row.BodyJSON) > submission.MaxExecutionPayloadBytes || json.Unmarshal(row.BodyJSON, &obs) != nil || collection.Digest(obs) != row.BodyHash {
			return nil, submission.ErrExecutionEvidenceRequired
		}
		receipt, e := readOfficialReceipt(r.db.WithContext(ctx), scope.OrganizationID, row.ReceiptID)
		if e != nil || receipt.Walmart == nil || receipt.Target != target || receipt.Walmart.FeedID != obs.FeedID {
			return nil, submission.ErrExecutionEvidenceRequired
		}
		var count int64
		if e = r.db.WithContext(ctx).Model(&submissionstore.WalmartSKUState{}).Where("organization_id=? AND last_attempt_id=? AND read_generation=? AND latest_observation_id=?", scope.OrganizationID, row.ReceiptID, row.Generation, row.ID).Count(&count).Error; e != nil {
			return nil, e
		}
		available := []string{}
		var skuRows []submissionstore.WalmartSKUState
		if e = r.db.WithContext(ctx).Where("organization_id=? AND last_attempt_id=? AND read_generation=? AND latest_observation_id=?", scope.OrganizationID, row.ReceiptID, row.Generation, row.ID).Order("seller_sku").Find(&skuRows).Error; e != nil {
			return nil, e
		}
		for _, sku := range skuRows {
			if obs.AllowsRepair(submission.WalmartExpectedSKUs(receipt.Walmart.SKUs), []string{sku.SellerSKU}) {
				available = append(available, sku.SellerSKU)
			}
		}
		out = append(out, submission.WalmartResultRecord{ID: row.ID, ReceiptID: row.ReceiptID, Generation: row.Generation, Latest: int(count) == len(receipt.Walmart.SKUs), RepairableSKUs: available, Observation: obs})
	}
	return out, nil
}
