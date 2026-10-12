package supplychainapp

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"time"
)

type TemuSupplierExecutionMerchant interface {
	CreateSupplierGoods(context.Context, model.SupplierGoods) (temugoods.CreatedGoods, error)
	UploadSupplierImage(context.Context, model.SupplierImageUpload) (model.SupplierImageUploaded, string, error)
}

func (s *UploadService) temuSupplierImage(ctx context.Context, scope collection.Scope, saved record.TargetRecord, observed temugoods.ImageObservation) (submission.OfficialReceipt, *UploadResult, error) {
	var zero submission.OfficialReceipt
	reader, ok := s.dependencies.TemuImages.(TemuImageContentReader)
	if !ok {
		return zero, nil, record.ErrNotReady
	}
	subject, err := submission.TemuImageSubjectID(observed, saved.Merchant)
	if err != nil {
		return zero, nil, err
	}
	target := submission.ExecutionTarget{Platform: "temu", StoreID: saved.Merchant.StoreID, SubjectID: subject}
	pending := func(id string) (submission.OfficialReceipt, *UploadResult, error) {
		r := unknownUpload(saved.ID, id)
		return zero, &r, nil
	}
	if retained, err := s.dependencies.Receipts.FindOfficialTarget(ctx, scope.OrganizationID, target); err == nil {
		if submission.ValidateOfficialReceipt(retained) != nil || retained.TemuImage == nil || retained.TemuImage.SupplierArtifact == nil {
			return zero, nil, record.ErrConflict
		}
		return retained, nil, nil
	} else if errors.Is(err, submission.ErrExecutionTargetClaimed) {
		return pending("")
	} else if !errors.Is(err, submission.ErrExecutionNotFound) {
		return zero, nil, err
	}
	if _, err = s.temuFacts(ctx, scope, saved); err != nil {
		return zero, nil, err
	}
	source, err := s.source(ctx, scope, saved)
	if err != nil {
		return zero, nil, err
	}
	proof, manifest, err := submission.PrepareTemuSupplierImageIntent(ctx, source, saved, observed)
	if err != nil {
		return zero, nil, err
	}
	intent, err := s.dependencies.Intents.PrepareOfficial(ctx, proof)
	if err != nil {
		return zero, nil, err
	}
	if intent.Owner != scope {
		return pending("")
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		return zero, nil, record.ErrInvalid
	}
	if _, err = s.temuFacts(ctx, scope, saved); err != nil {
		return zero, nil, err
	}
	acquired, err := s.dependencies.Kernel.Acquire(ctx, submission.AcquireExecutionCommand{Scope: submission.ExecutionScope{OrganizationID: scope.OrganizationID}, IntentKey: intent.Key, Target: intent.Target, Action: submission.TemuSupplierImageAction, Payload: raw, ClaimOwnerID: "supply-worker", Lease: time.Minute})
	if errors.Is(err, submission.ErrExecutionTargetSucceeded) {
		r, e := s.dependencies.Receipts.FindOfficialTarget(ctx, scope.OrganizationID, target)
		return r, nil, e
	}
	if errors.Is(err, submission.ErrExecutionTargetClaimed) {
		return pending("")
	}
	if err != nil {
		return zero, nil, err
	}
	if acquired.Permit == nil {
		if acquired.Attempt.Status == submission.ExecutionSucceeded {
			r, e := s.dependencies.Receipts.ReadOfficial(ctx, scope, acquired.Attempt.AttemptID)
			return r, nil, e
		}
		r, e := s.replay(ctx, scope, saved.ID, acquired.Attempt)
		return zero, &r, e
	}
	claim := uploadClaim(scope, acquired.Permit)
	ctx, cancel := context.WithDeadline(ctx, acquired.Permit.LeaseExpiresAt)
	defer cancel()
	uncertain := func() (submission.OfficialReceipt, *UploadResult, error) {
		r := s.unknown(ctx, saved.ID, claim)
		return zero, &r, nil
	}
	if _, err = s.temuFacts(ctx, scope, saved); err != nil {
		return uncertain()
	}
	content, err := reader.ReadTemuImage(ctx, manifest.Image, string(intent.Binding.MerchantMode))
	if err != nil {
		return uncertain()
	}
	defer clear(content)
	digest := sha256.Sum256(content)
	if int64(len(content)) != manifest.Image.Bytes || len(content) > 2<<20 || hex.EncodeToString(digest[:]) != manifest.Image.ContentHash {
		return uncertain()
	}
	input := model.SupplierImageUpload{Image: base64.StdEncoding.EncodeToString(content), BusinessType: manifest.BusinessType, Options: manifest.Options}
	// Downloading can outlast source authority or allow current heads to change.
	if _, err = s.temuFacts(ctx, scope, saved); err != nil {
		return uncertain()
	}
	handle, err := s.dependencies.TemuStores.TemuExecutionMerchant(ctx, scope, saved.Merchant.StoreID, storecenter.ProductPurposeImage, &intent.Binding)
	if err != nil {
		return uncertain()
	}
	merchant, ok := handle.(TemuSupplierExecutionMerchant)
	if !ok || !time.Now().Before(acquired.Permit.LeaseExpiresAt) {
		return uncertain()
	}
	result, hash, err := merchant.UploadSupplierImage(ctx, input)
	input.Image = ""
	if err != nil {
		return uncertain()
	}
	url, err := submission.SupplierUploadedImageURL(result)
	if err != nil {
		return uncertain()
	}
	artifact, err := reader.ProbeTemuArtifact(ctx, url)
	if err != nil {
		return uncertain()
	}
	completion, err := submission.NewTemuSupplierImageCompletion(scope, intent.Binding, acquired.Attempt, claim, manifest, result, hash, artifact)
	if err != nil {
		return uncertain()
	}
	receipt, err := s.complete(ctx, scope, completion, acquired.Attempt.AttemptID)
	if err != nil {
		return uncertain()
	}
	return receipt, nil, nil
}
