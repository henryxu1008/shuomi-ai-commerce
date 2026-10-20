package submissionpersistence

import (
	"context"
	"encoding/json"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	recordstore "task-processor/internal/integration/persistence/listing/record"
	"task-processor/internal/listing/submission"
	marketplace "task-processor/internal/marketplace/walmart"
	"task-processor/internal/product/collection"
)

// WalmartSKUState belongs to the original Submission transaction. Occupancy
// survives disconnect, app replacement, new records, failed sends and repair.
type WalmartSKUState struct {
	Environment, Market, MerchantIdentityHash, SellerSKU                                        string
	OrganizationID, StoreID, ProductKey, SourceSKUKey, SubjectID, FirstIntentKey, LastAttemptID string
	ReadGeneration                                                                              int64
	LatestObservationID                                                                         string
}

func (WalmartSKUState) TableName() string { return "listing_walmart_sku_bindings" }
func (r *Repository) requireWalmartEffect(ctx context.Context, tx *gorm.DB, a submission.ExecutionAttempt, previousID string) error {
	var row struct {
		ActorID, MemberID, RecordID, Kind, InputHash, BodyHash string
		BodyJSON                                               []byte
	}
	if tx.WithContext(ctx).Table("listing_submission_official_intents").Where("organization_id=? AND intent_key=?", a.OrganizationID, a.IntentKey).Take(&row).Error != nil {
		return submission.ErrExecutionEvidenceRequired
	}
	var i submission.OfficialIntent
	if len(row.BodyJSON) > submission.MaxExecutionPayloadBytes || json.Unmarshal(row.BodyJSON, &i) != nil || submission.ValidateOfficialIntent(i) != nil || i.Walmart == nil || i.Target != a.Target || i.Key != a.IntentKey || i.Owner.OrganizationID != a.OrganizationID || i.Owner.ActorID != row.ActorID || i.Owner.MemberID != row.MemberID || i.RecordID != row.RecordID || i.Kind != row.Kind || i.InputHash != row.InputHash || collection.Digest(i) != row.BodyHash || i.PayloadFingerprint != a.PayloadFingerprint || a.Action != submission.WalmartSubmitAction {
		return submission.ErrExecutionEvidenceRequired
	}
	var ref struct{ TargetID string }
	if tx.Raw("SELECT target_id FROM listing_target_records WHERE organization_id=? AND actor_id=? AND member_id=? AND id=?", i.Owner.OrganizationID, i.Owner.ActorID, i.Owner.MemberID, i.RecordID).Scan(&ref).Error != nil || ref.TargetID == "" {
		return submission.ErrExecutionEvidenceRequired
	}
	if tx.Exec("SELECT 1 FROM listing_preparation_targets WHERE organization_id=? AND actor_id=? AND member_id=? AND id=? FOR SHARE", i.Owner.OrganizationID, i.Owner.ActorID, i.Owner.MemberID, ref.TargetID).Error != nil {
		return submission.ErrExecutionUnavailable
	}
	records, e := recordstore.NewRepository(ctx, tx)
	if e != nil {
		return e
	}
	saved, e := records.ReadTargetHead(ctx, i.Owner, ref.TargetID)
	if e != nil || !submission.WalmartIntentMatchesRecord(i, saved) {
		return submission.ErrExecutionIntentConflict
	}
	if previousID != "" {
		if i.Walmart.RepairFrom == "" {
			return submission.ErrExecutionTargetSucceeded
		}

		if i.Walmart.RepairFrom != previousID {
			current, e := r.readWalmartReceipt(ctx, tx, a.OrganizationID, previousID)
			if e != nil {
				return submission.ErrExecutionEvidenceRequired
			}
			var currentHead struct {
				ID, BodyHash string
				BodyJSON     []byte
				Generation   int64
			}
			if e = tx.Table("listing_walmart_feed_observations").Where("organization_id=? AND receipt_id=?", a.OrganizationID, previousID).Order("generation DESC").Take(&currentHead).Error; e != nil {
				return submission.ErrExecutionTargetSucceeded
			}
			var currentObservation marketplace.FeedObservation
			if json.Unmarshal(currentHead.BodyJSON, &currentObservation) != nil || collection.Digest(currentObservation) != currentHead.BodyHash || currentObservation.FeedID != current.Walmart.FeedID || !currentObservation.CompleteFor(submission.WalmartExpectedSKUs(current.Walmart.SKUs)) || currentObservation.FeedStatus != "PROCESSED" && currentObservation.FeedStatus != "ERROR" {
				return submission.ErrExecutionTargetSucceeded
			}
			for _, item := range currentObservation.Items {
				if item.IngestionStatus != "SUCCESS" && item.IngestionStatus != "DATA_ERROR" {
					return submission.ErrExecutionTargetSucceeded
				}
			}
			for _, sku := range current.Walmart.SKUs {
				var state WalmartSKUState
				if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("environment=? AND market='US' AND merchant_identity_hash=? AND seller_sku=?", i.Binding.Walmart.Environment, i.Binding.SupplierIdentityHash, sku.SellerSKU).Take(&state).Error != nil || state.LastAttemptID != previousID || state.ReadGeneration != currentHead.Generation || state.LatestObservationID != currentHead.ID {
					return submission.ErrExecutionEvidenceRequired
				}
			}
		}
		prior, e := r.readWalmartReceipt(ctx, tx, a.OrganizationID, i.Walmart.RepairFrom)
		if e != nil || !submission.WalmartCompatibleBinding(prior.Binding, i.Binding) || prior.ProductKey != i.Source.Source.ProductKey || prior.Walmart.WireHash == i.Walmart.WireHash {
			return submission.ErrExecutionEvidenceRequired
		}
		var head struct {
			ID, BodyHash string
			BodyJSON     []byte
			Generation   int64
		}
		e = tx.Table("listing_walmart_feed_observations").Where("organization_id=? AND receipt_id=?", a.OrganizationID, i.Walmart.RepairFrom).Order("generation DESC").Take(&head).Error
		var observation marketplace.FeedObservation
		if e != nil || len(head.BodyJSON) > submission.MaxExecutionPayloadBytes || json.Unmarshal(head.BodyJSON, &observation) != nil || collection.Digest(observation) != head.BodyHash || observation.FeedID != prior.Walmart.FeedID || observation.ObservedAt.IsZero() || !observation.AllowsRepair(submission.WalmartExpectedSKUs(prior.Walmart.SKUs), submission.WalmartExpectedSKUs(i.Walmart.SKUs)) {
			return submission.ErrExecutionTargetSucceeded
		}
		for _, s := range i.Walmart.SKUs {
			var occupied WalmartSKUState
			if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("environment=? AND market='US' AND merchant_identity_hash=? AND seller_sku=?", i.Binding.Walmart.Environment, i.Binding.SupplierIdentityHash, s.SellerSKU).Take(&occupied).Error != nil || occupied.ReadGeneration != head.Generation || occupied.LatestObservationID != head.ID || occupied.LastAttemptID != i.Walmart.RepairFrom {
				return submission.ErrExecutionEvidenceRequired
			}
		}
	} else if i.Walmart.RepairFrom != "" {
		return submission.ErrExecutionEvidenceRequired
	}
	for _, s := range i.Walmart.SKUs {
		if s.SellerSKU != marketplace.SellerSKU(marketplace.SKUNamespace(i.Binding.SupplierIdentityHash, i.Binding.Walmart.GrantID), i.Source.Source.ProductKey, s.SourceSKUKey) {
			return submission.ErrExecutionEvidenceRequired
		}
		occupied := WalmartSKUState{Environment: i.Binding.Walmart.Environment, Market: "US", MerchantIdentityHash: i.Binding.SupplierIdentityHash, SellerSKU: s.SellerSKU, OrganizationID: a.OrganizationID, StoreID: a.Target.StoreID, ProductKey: i.Source.Source.ProductKey, SourceSKUKey: s.SourceSKUKey, SubjectID: a.Target.SubjectID, FirstIntentKey: i.Key, LastAttemptID: a.AttemptID}
		if e := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&occupied).Error; e != nil {
			return e
		}
		var actual WalmartSKUState
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("environment=? AND market='US' AND merchant_identity_hash=? AND seller_sku=?", occupied.Environment, occupied.MerchantIdentityHash, occupied.SellerSKU).Take(&actual).Error; e != nil {
			return e
		}
		if actual.OrganizationID != occupied.OrganizationID || actual.StoreID != occupied.StoreID || actual.ProductKey != occupied.ProductKey || actual.SourceSKUKey != occupied.SourceSKUKey || actual.SubjectID != occupied.SubjectID {
			return submission.ErrExecutionIntentConflict
		}
		if actual.LastAttemptID != a.AttemptID && previousID != "" && actual.LastAttemptID != i.Walmart.RepairFrom {
			return submission.ErrExecutionEvidenceRequired
		}
		if e := tx.Model(&WalmartSKUState{}).Where("environment=? AND market='US' AND merchant_identity_hash=? AND seller_sku=?", occupied.Environment, occupied.MerchantIdentityHash, occupied.SellerSKU).Updates(map[string]any{"last_attempt_id": a.AttemptID, "latest_observation_id": ""}).Error; e != nil {
			return e
		}
	}
	return nil
}
func (r *Repository) readWalmartReceipt(ctx context.Context, tx *gorm.DB, org, id string) (submission.OfficialReceipt, error) {
	var row struct {
		BodyJSON                                                         []byte
		BodyHash, ActorID, MemberID, StoreID, SubjectID, IntentKey, Kind string
	}
	if e := tx.Table("listing_submission_official_receipts").Where("organization_id=? AND attempt_id=?", org, id).Take(&row).Error; e != nil {
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return submission.OfficialReceipt{}, submission.ErrExecutionNotFound
		}
		return submission.OfficialReceipt{}, e
	}
	var rcp submission.OfficialReceipt
	if len(row.BodyJSON) > submission.MaxExecutionPayloadBytes || json.Unmarshal(row.BodyJSON, &rcp) != nil || submission.ValidateOfficialReceipt(rcp) != nil || rcp.Walmart == nil || rcp.ID != id || rcp.Owner.OrganizationID != org || rcp.Owner.ActorID != row.ActorID || rcp.Owner.MemberID != row.MemberID || rcp.Target.StoreID != row.StoreID || rcp.Target.SubjectID != row.SubjectID || rcp.Kind != row.Kind || rcp.IntentKey != row.IntentKey || collection.Digest(rcp) != row.BodyHash {
		return rcp, submission.ErrExecutionEvidenceRequired
	}
	a, found, e := r.findByID(ctx, tx, org, id, false)
	if e != nil || !found || a.Status != submission.ExecutionSucceeded || a.Evidence == nil || a.Evidence.Reference != id || a.Evidence.Fingerprint != rcp.ResponseHash || !a.Evidence.ObservedAt.Equal(rcp.ObservedAt) || a.Target != rcp.Target || a.IntentKey != rcp.IntentKey || a.PayloadFingerprint != rcp.PayloadFingerprint {
		return rcp, submission.ErrExecutionEvidenceRequired
	}
	return rcp, nil
}
