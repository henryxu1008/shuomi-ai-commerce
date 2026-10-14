package submissionpersistence

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"

	recordstore "task-processor/internal/integration/persistence/listing/record"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	"task-processor/internal/product/collection"
)

// TikTok stages use the original Product transaction and stable target fence.
// Retained preceding receipts are immutable; each is joined to its completed
// kernel attempt before the next stage can obtain a send permit.
func (r *Repository) requireTikTokEffect(ctx context.Context, tx *gorm.DB, a submission.ExecutionAttempt) error {
	var row struct {
		ActorID, MemberID, RecordID, Kind, InputHash, BodyHash string
		BodyJSON                                               []byte
	}
	if tx.WithContext(ctx).Table("listing_submission_official_intents").Where("organization_id=? AND intent_key=?", a.OrganizationID, a.IntentKey).Take(&row).Error != nil {
		return submission.ErrExecutionEvidenceRequired
	}
	var intent submission.OfficialIntent
	if len(row.BodyJSON) > submission.MaxExecutionPayloadBytes || json.Unmarshal(row.BodyJSON, &intent) != nil || submission.ValidateOfficialIntent(intent) != nil || intent.Target.Platform != "tiktokshop" || intent.Key != a.IntentKey || intent.Owner.OrganizationID != a.OrganizationID || intent.Owner.ActorID != row.ActorID || intent.Owner.MemberID != row.MemberID || intent.RecordID != row.RecordID || intent.Kind != row.Kind || intent.InputHash != row.InputHash || collection.Digest(intent) != row.BodyHash || intent.Target != a.Target || intent.PayloadFingerprint != a.PayloadFingerprint || a.Action != submission.TikTokStageAction(intent.Kind) {
		return submission.ErrExecutionEvidenceRequired
	}
	var reference struct{ TargetID string }
	if tx.WithContext(ctx).Raw("SELECT target_id FROM listing_target_records WHERE organization_id=? AND actor_id=? AND member_id=? AND id=?", intent.Owner.OrganizationID, intent.Owner.ActorID, intent.Owner.MemberID, intent.RecordID).Scan(&reference).Error != nil || reference.TargetID == "" {
		return submission.ErrExecutionEvidenceRequired
	}
	if tx.WithContext(ctx).Exec("SELECT 1 FROM listing_preparation_targets WHERE organization_id=? AND actor_id=? AND member_id=? AND id=? FOR SHARE", intent.Owner.OrganizationID, intent.Owner.ActorID, intent.Owner.MemberID, reference.TargetID).Error != nil {
		return submission.ErrExecutionUnavailable
	}
	records, err := recordstore.NewRepository(ctx, tx)
	if err != nil {
		return submission.ErrExecutionUnavailable
	}
	saved, err := records.ReadTargetHead(ctx, intent.Owner, reference.TargetID)
	if err != nil || saved.ID != intent.RecordID || collection.Digest(saved) != intent.RecordHash || saved.Source != intent.Source || saved.Merchant != intent.Binding || !record.PublicationReady(saved) {
		return submission.ErrExecutionIntentConflict
	}
	receipts := []submission.OfficialReceipt{}
	var global *submission.OfficialReceipt
	for _, ref := range intent.TikTok.Images {
		receipt, err := r.readTikTokPrecedingReceipt(ctx, tx, intent.Owner.OrganizationID, ref)
		if err != nil {
			return err
		}
		receipts = append(receipts, receipt)
	}
	if intent.TikTok.Global != nil {
		receipt, err := r.readTikTokPrecedingReceipt(ctx, tx, intent.Owner.OrganizationID, *intent.TikTok.Global)
		if err != nil {
			return err
		}
		global = &receipt
	}
	raw, effect, err := submission.ResolveTikTokPayload(saved, intent.Kind, intent.Key, intent.TikTok.Image, receipts, global)
	if err != nil || collection.Digest(effect) != collection.Digest(intent.TikTok) || !submission.TikTokPayloadMatchesIntent(intent, raw) {
		return submission.ErrExecutionEvidenceRequired
	}
	return nil
}
func (r *Repository) readTikTokPrecedingReceipt(ctx context.Context, tx *gorm.DB, org string, ref submission.TikTokReceiptReference) (submission.OfficialReceipt, error) {
	var row struct {
		ActorID, MemberID, IntentKey, StoreID, SubjectID, Kind, BodyHash string
		BodyJSON                                                         []byte
	}
	if tx.WithContext(ctx).Table("listing_submission_official_receipts").Where("organization_id=? AND attempt_id=?", org, ref.ID).Take(&row).Error != nil {
		return submission.OfficialReceipt{}, submission.ErrExecutionEvidenceRequired
	}
	var body submission.OfficialReceipt
	if len(row.BodyJSON) > submission.MaxExecutionPayloadBytes || json.Unmarshal(row.BodyJSON, &body) != nil || submission.ValidateOfficialReceipt(body) != nil || body.Target.Platform != "tiktokshop" || body.ID != ref.ID || body.Owner.OrganizationID != org || body.Owner.ActorID != row.ActorID || body.Owner.MemberID != row.MemberID || body.IntentKey != row.IntentKey || body.Target.StoreID != row.StoreID || body.Target.SubjectID != row.SubjectID || body.Kind != row.Kind || collection.Digest(body) != row.BodyHash || row.BodyHash != ref.Hash {
		return body, submission.ErrExecutionEvidenceRequired
	}
	attempt, found, err := r.findByID(ctx, tx, org, body.ID, false)
	if err != nil || !found || attempt.Status != submission.ExecutionSucceeded || attempt.Target != body.Target || attempt.IntentKey != body.IntentKey || attempt.PayloadFingerprint != body.PayloadFingerprint || attempt.Evidence == nil || attempt.Evidence.Reference != body.ID || attempt.Evidence.Fingerprint != body.ResponseHash || !attempt.Evidence.ObservedAt.Equal(body.ObservedAt) {
		return body, submission.ErrExecutionEvidenceRequired
	}
	return body, nil
}
