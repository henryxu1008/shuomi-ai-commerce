package submissionpersistence

import (
	"context"
	"encoding/json"
	"gorm.io/gorm"
	"strings"
	recordstore "task-processor/internal/integration/persistence/listing/record"
	"task-processor/internal/listing/submission"
	"task-processor/internal/product/collection"
)

func isShelfTarget(a submission.ExecutionAttempt) bool {
	return a.Target.Platform == "shein" && (a.Action == submission.SheinShelfAction || strings.HasPrefix(a.Target.SubjectID, "shein-shelf-on-us-v1-"))
}
func (r *Repository) requireSheinShelf(ctx context.Context, tx *gorm.DB, a submission.ExecutionAttempt) error {
	var row struct {
		ActorID, MemberID, RecordID, Kind, InputHash, BodyHash string
		BodyJSON                                               []byte
	}
	if !r.immutableOfficialEffects || tx.WithContext(ctx).Table("listing_submission_official_intents").Where("organization_id=? AND intent_key=?", a.OrganizationID, a.IntentKey).Take(&row).Error != nil {
		return submission.ErrExecutionEvidenceRequired
	}
	var v submission.OfficialIntent
	if len(row.BodyJSON) > submission.MaxExecutionPayloadBytes || json.Unmarshal(row.BodyJSON, &v) != nil || submission.ValidateOfficialIntent(v) != nil || v.Shelf == nil || v.Owner.OrganizationID != a.OrganizationID || v.Owner.ActorID != row.ActorID || v.Owner.MemberID != row.MemberID || v.Key != a.IntentKey || v.RecordID != row.RecordID || v.Kind != row.Kind || v.InputHash != row.InputHash || collection.Digest(v) != row.BodyHash || v.Target != a.Target || v.PayloadFingerprint != a.PayloadFingerprint || a.Action != submission.SheinShelfAction || !r.now().Before(v.Shelf.ExpiresAt) {
		return submission.ErrExecutionEvidenceRequired
	}
	var receiptRow struct {
		ActorID, MemberID, IntentKey, StoreID, SubjectID, Kind, BodyHash string
		BodyJSON                                                         []byte
	}
	if tx.WithContext(ctx).Table("listing_submission_official_receipts").Where("organization_id=? AND attempt_id=?", a.OrganizationID, v.Shelf.OriginalReceiptID).Take(&receiptRow).Error != nil {
		return submission.ErrExecutionEvidenceRequired
	}
	var original submission.OfficialReceipt
	if len(receiptRow.BodyJSON) > submission.MaxExecutionPayloadBytes || json.Unmarshal(receiptRow.BodyJSON, &original) != nil || submission.ValidateOfficialReceipt(original) != nil || original.Owner != v.Owner || original.ID != v.Shelf.OriginalReceiptID || original.IntentKey != receiptRow.IntentKey || original.Owner.ActorID != receiptRow.ActorID || original.Owner.MemberID != receiptRow.MemberID || original.Kind != receiptRow.Kind || original.Target.StoreID != receiptRow.StoreID || original.Target.SubjectID != receiptRow.SubjectID || collection.Digest(original) != receiptRow.BodyHash || receiptRow.BodyHash != v.Shelf.OriginalReceiptHash || !submission.CompatibleShelfBinding(original.Binding, v.Binding) || original.Product.Version != v.Shelf.Version {
		return submission.ErrExecutionEvidenceRequired
	}
	previous, found, e := r.findByID(ctx, tx, a.OrganizationID, original.ID, false)
	if e != nil || !found || previous.Status != submission.ExecutionSucceeded || previous.Target != original.Target || previous.IntentKey != original.IntentKey || previous.PayloadFingerprint != original.PayloadFingerprint || previous.Evidence == nil || previous.Evidence.Reference != original.ID || previous.Evidence.Fingerprint != original.ResponseHash || !previous.Evidence.ObservedAt.Equal(original.ObservedAt) {
		return submission.ErrExecutionEvidenceRequired
	}
	records, e := recordstore.NewRepository(ctx, tx)
	if e != nil {
		return submission.ErrExecutionUnavailable
	}
	saved, e := records.ReadTargetRecord(ctx, v.Owner, v.RecordID)
	if e != nil || collection.Digest(saved) != v.RecordHash || saved.Source != v.Source {
		return submission.ErrExecutionEvidenceRequired
	}
	if _, e = submission.OriginalShelfPlan(saved, original, v.Shelf.SKC); e != nil {
		return e
	}
	return nil
}
