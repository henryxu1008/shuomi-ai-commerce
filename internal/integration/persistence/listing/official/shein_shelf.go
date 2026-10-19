package officialpersistence

import (
	"context"
	"encoding/json"
	"errors"
	recordstore "task-processor/internal/integration/persistence/listing/record"
	"task-processor/internal/listing/submission"
	"task-processor/internal/product/collection"
)

func (r *OfficialRepository) PrepareShelfIntent(ctx context.Context, proof submission.ShelfIntentCommit) (submission.OfficialIntent, error) {
	body, e := proof.Read(ctx)
	if e != nil || body.Shelf == nil {
		return submission.OfficialIntent{}, submission.ErrExecutionEvidenceRequired
	}
	original, e := r.ReadOfficial(ctx, body.Owner, body.Shelf.OriginalReceiptID)
	if e != nil || collection.Digest(original) != body.Shelf.OriginalReceiptHash {
		return submission.OfficialIntent{}, submission.ErrExecutionEvidenceRequired
	}
	ctx, cancel := context.WithTimeout(ctx, submission.ExecutionTimeout)
	defer cancel()
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return submission.OfficialIntent{}, submission.ErrExecutionUnavailable
	}
	defer tx.Rollback()
	if tx.Exec("SET LOCAL synchronous_commit=on").Error != nil || advisoryLock(tx, "official-intent", body.Owner.OrganizationID, body.Key) != nil {
		return submission.OfficialIntent{}, submission.ErrExecutionUnavailable
	}
	if prior, e := readOfficialIntent(tx, body.Owner.OrganizationID, body.Key, true); e == nil {
		if prior.InputHash != body.InputHash {
			return prior, submission.ErrExecutionIntentConflict
		}
		return prior, nil
	} else if !errors.Is(e, submission.ErrExecutionNotFound) {
		return body, e
	}
	retained, e := readOfficialReceipt(tx, body.Owner.OrganizationID, original.ID)
	if e != nil || collection.Digest(retained) != body.Shelf.OriginalReceiptHash {
		return body, submission.ErrExecutionEvidenceRequired
	}
	originalIntent, e := readOfficialIntent(tx, body.Owner.OrganizationID, original.IntentKey, false)
	if e != nil || originalIntent.Kind != "publish" || originalIntent.RecordID != body.RecordID || originalIntent.RecordHash != body.RecordHash || !receiptMatchesIntent(original, originalIntent) {
		return body, submission.ErrExecutionEvidenceRequired
	}
	records, e := recordstore.NewRepository(ctx, tx)
	if e != nil {
		return body, submission.ErrExecutionUnavailable
	}
	saved, e := records.ReadTargetRecord(ctx, body.Owner, body.RecordID)
	if e != nil || collection.Digest(saved) != body.RecordHash || saved.Source != body.Source || !submission.CompatibleShelfBinding(original.Binding, body.Binding) {
		return body, submission.ErrExecutionEvidenceRequired
	}
	if _, e = submission.OriginalShelfPlan(saved, original, body.Shelf.SKC); e != nil {
		return body, e
	}
	raw, e := json.Marshal(body)
	if e != nil || len(raw) > submission.MaxExecutionPayloadBytes {
		return body, submission.ErrExecutionInvalid
	}
	row := officialIntentRow{OrganizationID: body.Owner.OrganizationID, ActorID: body.Owner.ActorID, MemberID: body.Owner.MemberID, IntentKey: body.Key, RecordID: body.RecordID, Kind: body.Kind, InputHash: body.InputHash, BodyHash: collection.Digest(body), BodyJSON: raw}
	if result := tx.Create(&row); result.Error != nil || result.RowsAffected != 1 {
		return body, submission.ErrExecutionUnavailable
	}
	if _, e = proof.Read(ctx); e != nil {
		return body, e
	}
	if tx.Commit().Error != nil {
		return body, submission.ErrExecutionOutcomeUnknown
	}
	return body, nil
}
