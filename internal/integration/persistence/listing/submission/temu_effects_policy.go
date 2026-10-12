package submissionpersistence

import (
	"context"
	"encoding/json"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sort"
	recordstore "task-processor/internal/integration/persistence/listing/record"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	"task-processor/internal/product/collection"
)

func readTemuPolicyIntent(ctx context.Context, tx *gorm.DB, org, key string) (submission.OfficialIntent, error) {
	var row struct {
		ActorID, MemberID, RecordID, Kind, InputHash, BodyHash string
		BodyJSON                                               []byte
	}
	if tx.WithContext(ctx).Table("listing_submission_official_intents").Clauses(clause.Locking{Strength: "SHARE"}).Where("organization_id=? AND intent_key=?", org, key).Take(&row).Error != nil {
		return submission.OfficialIntent{}, submission.ErrExecutionEvidenceRequired
	}
	var body submission.OfficialIntent
	if len(row.BodyJSON) > submission.MaxExecutionPayloadBytes || json.Unmarshal(row.BodyJSON, &body) != nil || submission.ValidateOfficialIntent(body) != nil || body.Target.Platform != "temu" || body.Owner.OrganizationID != org || body.Key != key || body.Owner.ActorID != row.ActorID || body.Owner.MemberID != row.MemberID || body.RecordID != row.RecordID || body.Kind != row.Kind || body.InputHash != row.InputHash || collection.Digest(body) != row.BodyHash {
		return body, submission.ErrExecutionEvidenceRequired
	}
	return body, nil
}

func (r *Repository) requireTemuEffect(ctx context.Context, tx *gorm.DB, a submission.ExecutionAttempt) (submission.OfficialIntent, error) {
	i, err := readTemuPolicyIntent(ctx, tx, a.OrganizationID, a.IntentKey)
	if err != nil || i.Target != a.Target || i.PayloadFingerprint != a.PayloadFingerprint {
		return i, submission.ErrExecutionEvidenceRequired
	}
	if i.Kind == "publish" && a.Action != submission.TemuPublishAction || i.Kind == "image" && a.Action != submission.TemuLocalImageAction {
		return i, submission.ErrExecutionEvidenceRequired
	}
	product, err := submission.TemuProductSubjectID(i.Source.Source.ProductKey, i.Binding)
	if err != nil {
		return i, err
	}
	// Match the image path's lock order: canonical product before image fences.
	if err = advisoryLock(tx, "target", a.OrganizationID, "temu", a.Target.StoreID, product); err != nil {
		return i, err
	}
	var reference struct{ TargetID string }
	if tx.WithContext(ctx).Raw("SELECT target_id FROM listing_target_records WHERE organization_id=? AND actor_id=? AND member_id=? AND id=?", i.Owner.OrganizationID, i.Owner.ActorID, i.Owner.MemberID, i.RecordID).Scan(&reference).Error != nil || reference.TargetID == "" {
		return i, submission.ErrExecutionEvidenceRequired
	}
	if tx.WithContext(ctx).Exec("SELECT 1 FROM listing_preparation_targets WHERE organization_id=? AND actor_id=? AND member_id=? AND id=? FOR SHARE", i.Owner.OrganizationID, i.Owner.ActorID, i.Owner.MemberID, reference.TargetID).Error != nil {
		return i, submission.ErrExecutionUnavailable
	}
	records, err := recordstore.NewRepository(ctx, tx)
	if err != nil {
		return i, submission.ErrExecutionUnavailable
	}
	saved, err := records.ReadTargetHead(ctx, i.Owner, reference.TargetID)
	if err != nil || saved.ID != i.RecordID || collection.Digest(saved) != i.RecordHash || saved.Source != i.Source || saved.Merchant != i.Binding || !record.PublicationReady(saved) {
		return i, submission.ErrExecutionIntentConflict
	}
	// Current inventory lives in the independent AssetDB owner. The original
	// application checks its exact scope/hash/content before claim and send;
	// this Product transaction locks only its immutable approved references.
	if i.Kind == "image" {
		found := false
		for _, image := range saved.TemuImageObservations {
			if collection.Digest(image) == collection.Digest(i.TemuImage) {
				found = true
			}
		}
		if !found {
			return i, submission.ErrExecutionEvidenceRequired
		}
		fence, found, err := r.findTarget(ctx, tx, a.OrganizationID, submission.ExecutionTarget{Platform: "temu", StoreID: a.Target.StoreID, SubjectID: product}, true)
		if err != nil {
			return i, err
		}
		if found && fence.CurrentStatus == string(submission.ExecutionSucceeded) {
			return i, submission.ErrExecutionTargetSucceeded
		}
		if found && (fence.CurrentStatus == string(submission.ExecutionClaimed) || fence.CurrentStatus == string(submission.ExecutionOutcomeUnknown)) {
			return i, submission.ErrExecutionTargetClaimed
		}
	} else if err := r.requireTemuPublicationImages(ctx, tx, i, saved); err != nil {
		return i, err
	}
	return i, nil
}

func (r *Repository) requireTemuPublicationImages(ctx context.Context, tx *gorm.DB, intent submission.OfficialIntent, saved record.TargetRecord) error {
	type reference struct {
		ref         submission.TemuImageReference
		subject     string
		contentHash string
		catID       int64
	}
	refs := make([]reference, 0, len(intent.TemuImages))
	for _, ref := range intent.TemuImages {
		found := false
		for _, image := range saved.TemuImageObservations {
			if image.AssetID != ref.AssetID || image.Usage != ref.Usage {
				continue
			}
			if found {
				return submission.ErrExecutionEvidenceRequired
			}
			found = true
			subject, err := submission.TemuLocalImageSubjectID(image.ContentHash, image.CatID, image.Usage, intent.Binding)
			if err != nil {
				return err
			}
			refs = append(refs, reference{ref: ref, subject: subject, contentHash: image.ContentHash, catID: image.CatID})
		}
		if !found {
			return submission.ErrExecutionEvidenceRequired
		}
	}
	if len(refs) != len(saved.TemuImageObservations) {
		return submission.ErrExecutionEvidenceRequired
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].subject < refs[j].subject })
	for _, entry := range refs {
		target := submission.ExecutionTarget{Platform: "temu", StoreID: intent.Target.StoreID, SubjectID: entry.subject}
		if err := advisoryLock(tx, "target", intent.Owner.OrganizationID, target.Platform, target.StoreID, target.SubjectID); err != nil {
			return err
		}
		fence, found, err := r.findTarget(ctx, tx, intent.Owner.OrganizationID, target, true)
		if err != nil {
			return err
		}
		if !found || fence.CurrentStatus != string(submission.ExecutionSucceeded) || fence.CurrentAttemptID != entry.ref.ReceiptID {
			return submission.ErrExecutionEvidenceRequired
		}
		a, found, err := r.findByID(ctx, tx, intent.Owner.OrganizationID, fence.CurrentAttemptID, false)
		if err != nil {
			return err
		}
		if !found || a.Status != submission.ExecutionSucceeded || a.FenceEpoch != fence.Epoch || a.Target != target || a.Action != submission.TemuLocalImageAction {
			return submission.ErrExecutionEvidenceRequired
		}
		receipt, err := readTemuPolicyReceipt(ctx, tx, a)
		if err != nil {
			return err
		}
		if collection.Digest(receipt) != entry.ref.ReceiptHash || receipt.TemuImage == nil || receipt.Binding.SupplierIdentityHash != intent.Binding.SupplierIdentityHash || receipt.TemuImage.FenceEpoch != fence.Epoch || !receipt.TemuImage.AttemptCreatedAt.Equal(a.CreatedAt) || receipt.TemuImage.Image.ContentHash != entry.contentHash || receipt.TemuImage.Image.CatID != entry.catID || receipt.TemuImage.Image.Usage != entry.ref.Usage {
			return submission.ErrExecutionEvidenceRequired
		}
		if !r.now().UTC().Add(submission.TemuImageSendWindow).Before(receipt.TemuImage.UseBefore) {
			return submission.ErrExecutionEvidenceRequired
		}
	}
	return nil
}

func readTemuPolicyReceipt(ctx context.Context, tx *gorm.DB, a submission.ExecutionAttempt) (submission.OfficialReceipt, error) {
	var row struct {
		ActorID, MemberID, IntentKey, StoreID, SubjectID, Kind, BodyHash string
		BodyJSON                                                         []byte
	}
	if tx.WithContext(ctx).Table("listing_submission_official_receipts").Clauses(clause.Locking{Strength: "SHARE"}).Where("organization_id=? AND attempt_id=?", a.OrganizationID, a.AttemptID).Take(&row).Error != nil {
		return submission.OfficialReceipt{}, submission.ErrExecutionEvidenceRequired
	}
	var body submission.OfficialReceipt
	if len(row.BodyJSON) > submission.MaxExecutionPayloadBytes || json.Unmarshal(row.BodyJSON, &body) != nil || submission.ValidateOfficialReceipt(body) != nil || collection.Digest(body) != row.BodyHash || body.Owner.OrganizationID != a.OrganizationID || body.ID != a.AttemptID || body.Owner.ActorID != row.ActorID || body.Owner.MemberID != row.MemberID || body.IntentKey != row.IntentKey || body.Target.StoreID != row.StoreID || body.Target.SubjectID != row.SubjectID || body.Kind != row.Kind || body.Target != a.Target || body.PayloadFingerprint != a.PayloadFingerprint || a.Evidence == nil || a.Evidence.Reference != body.ID || a.Evidence.Fingerprint != body.ResponseHash || !a.Evidence.ObservedAt.Equal(body.ObservedAt) {
		return body, submission.ErrExecutionEvidenceRequired
	}
	return body, nil
}

func (r *Repository) requireTemuImageRenewal(ctx context.Context, tx *gorm.DB, next submission.OfficialIntent, current submission.ExecutionAttempt) error {
	if next.Kind != "image" || next.TemuImage == nil || next.TemuRenewalFrom != current.AttemptID || current.Status != submission.ExecutionSucceeded || current.Action != submission.TemuLocalImageAction {
		return submission.ErrExecutionEvidenceRequired
	}
	original, err := readTemuPolicyIntent(ctx, tx, current.OrganizationID, current.IntentKey)
	if err != nil {
		return err
	}
	receipt, err := readTemuPolicyReceipt(ctx, tx, current)
	if err != nil {
		return err
	}
	if receipt.TemuImage == nil || original.Kind != "image" || original.TemuImage == nil || original.Target != current.Target || original.PayloadFingerprint != current.PayloadFingerprint || receipt.Owner != original.Owner || receipt.IntentKey != original.Key || receipt.Binding != original.Binding || !receipt.TemuImage.AttemptCreatedAt.Equal(current.CreatedAt) || receipt.TemuImage.FenceEpoch != current.FenceEpoch {
		return submission.ErrExecutionEvidenceRequired
	}
	old := receipt.TemuImage.Image
	old.RemoteURLs = nil
	old.ResponseHash = ""
	if collection.Digest(old) != collection.Digest(original.TemuImage) || next.Binding.SupplierIdentityHash != receipt.Binding.SupplierIdentityHash || next.TemuImage.ContentHash != old.ContentHash || next.TemuImage.CatID != old.CatID || next.TemuImage.Usage != old.Usage || next.TemuImage.Width != old.Width || next.TemuImage.Height != old.Height || next.TemuImage.Bytes != old.Bytes || next.TemuImage.MediaType != old.MediaType {
		return submission.ErrExecutionEvidenceRequired
	}
	if r.now().UTC().Before(receipt.TemuImage.UseBefore) {
		return submission.ErrExecutionTargetSucceeded
	}
	return nil
}
