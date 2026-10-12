package supplychainapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"time"

	storeapp "task-processor/internal/app/storecenter"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

type TemuExecutionMerchant interface {
	Binding() storecenter.ProductMerchantBinding
	CreateLocalGoods(context.Context, model.LocalGoodsV2) (temugoods.CreatedGoods, error)
	UploadLocalImage(context.Context, model.LocalImageUpload) (model.LocalImageUploaded, string, error)
}
type TemuExecutionStore interface {
	TemuExecutionMerchant(context.Context, collection.Scope, string, string, *storecenter.ProductMerchantBinding) (TemuExecutionMerchant, error)
}
type OfficialTemuExecutionStore struct {
	Access *storeapp.OfficialProductAccess
}

func (s OfficialTemuExecutionStore) TemuExecutionMerchant(ctx context.Context, scope collection.Scope, storeID, purpose string, expected *storecenter.ProductMerchantBinding) (TemuExecutionMerchant, error) {
	if s.Access == nil {
		return nil, record.ErrUnavailable
	}
	return s.Access.AuthorizeTemuMutation(ctx, storecenter.ProductExecutionSubject{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, Purpose: purpose}, storeID, expected)
}

func (s *UploadService) temuChannel(ctx context.Context, scope collection.Scope, saved record.TargetRecord) (UploadResult, error) {
	subject, err := submission.TemuProductSubjectID(saved.Source.Source.ProductKey, saved.Merchant)
	if err != nil {
		return UploadResult{}, err
	}
	receipt, err := s.dependencies.Receipts.FindOfficialTarget(ctx, scope.OrganizationID, submission.ExecutionTarget{Platform: "temu", StoreID: saved.Merchant.StoreID, SubjectID: subject})
	if errors.Is(err, submission.ErrExecutionTargetClaimed) {
		return unknownUpload(saved.ID, ""), nil
	}
	if err != nil {
		return UploadResult{}, err
	}
	if submission.ValidateOfficialReceipt(receipt) != nil || receipt.Kind != "publish" || receipt.TemuProduct == nil || receipt.ProductKey != saved.Source.Source.ProductKey || receipt.Binding.SupplierIdentityHash != saved.Merchant.SupplierIdentityHash {
		return UploadResult{}, record.ErrConflict
	}
	result := UploadResult{RecordID: saved.ID, AttemptID: receipt.ID, Status: submission.ExecutionSucceeded, TemuProduct: receipt.TemuProduct, Message: "保留已确认的原 TEMU 商品；审核与在售状态以店铺同步结果为准"}
	if receipt.Owner == scope {
		result.PublishedRecordID, result.CurrentRecordUploaded = receipt.RecordID, receipt.RecordID == saved.ID
	}
	return result, nil
}

func (s *UploadService) uploadTemu(ctx context.Context, scope collection.Scope, key string, saved record.TargetRecord) (UploadResult, error) {
	if s.dependencies.TemuStores == nil || saved.Input.Temu == nil {
		return UploadResult{}, record.ErrNotReady
	}
	// Authenticate current Store authority even for retained results. Replay does
	// not requalify a changed draft or issue another image/product send.
	if _, err := s.dependencies.TemuStores.TemuExecutionMerchant(ctx, scope, saved.Merchant.StoreID, storecenter.ProductPurposePublish, &saved.Merchant); err != nil {
		return UploadResult{}, err
	}
	if existing, err := s.dependencies.Intents.ReadOfficialIntent(ctx, scope, key); err == nil {
		if existing.Kind != "publish" || existing.Target.Platform != "temu" || existing.RecordID != saved.ID || existing.RecordHash != collection.Digest(saved) {
			return UploadResult{}, submission.ErrExecutionIntentConflict
		}
		attempt, err := s.dependencies.Kernel.ReadIntent(ctx, submission.ExecutionScope{OrganizationID: scope.OrganizationID}, key)
		if err == nil {
			return s.replay(ctx, scope, saved.ID, attempt)
		}
		if !errors.Is(err, submission.ErrExecutionNotFound) {
			return UploadResult{}, err
		}
	} else if !errors.Is(err, submission.ErrExecutionNotFound) {
		return UploadResult{}, err
	}
	if prior, err := s.temuChannel(ctx, scope, saved); err == nil {
		return prior, nil
	} else if !errors.Is(err, submission.ErrExecutionNotFound) {
		return UploadResult{}, err
	}
	observed, err := s.temuFacts(ctx, scope, saved)
	if err != nil {
		return UploadResult{}, err
	}
	if saved.Merchant.MerchantMode != storecenter.MerchantLocal {
		return UploadResult{}, record.ErrNotReady
	}
	receipts := make([]submission.OfficialReceipt, 0, len(observed))
	seen := map[string]bool{}
	for _, image := range observed {
		receipt, pending, err := s.temuImage(ctx, scope, saved, image)
		if err != nil {
			return UploadResult{}, err
		}
		if pending != nil {
			return *pending, nil
		}
		if !seen[receipt.ID] {
			receipts = append(receipts, receipt)
			seen[receipt.ID] = true
		}
	}
	payload, _, err := submission.ResolveTemuPublicationPayload(saved, receipts, time.Now().UTC())
	if err != nil {
		return UploadResult{}, record.ErrNotReady
	}
	if _, err = s.temuFacts(ctx, scope, saved); err != nil {
		return UploadResult{}, err
	}
	if err = s.temuPublicationImages(ctx, scope, saved, receipts, payload); err != nil {
		return UploadResult{}, err
	}
	source, err := s.source(ctx, scope, saved)
	if err != nil {
		return UploadResult{}, err
	}
	proof, err := submission.PrepareTemuPublishIntent(ctx, source, saved, key, payload, receipts)
	if err != nil {
		return UploadResult{}, err
	}
	intent, err := s.dependencies.Intents.PrepareOfficial(ctx, proof)
	if err != nil {
		return UploadResult{}, err
	}
	if _, err = s.temuFacts(ctx, scope, saved); err != nil {
		return UploadResult{}, err
	}
	if err = s.temuPublicationImages(ctx, scope, saved, receipts, payload); err != nil {
		return UploadResult{}, err
	}
	acquired, err := s.dependencies.Kernel.Acquire(ctx, submission.AcquireExecutionCommand{Scope: submission.ExecutionScope{OrganizationID: scope.OrganizationID}, IntentKey: key, Target: intent.Target, Action: submission.TemuPublishAction, Payload: payload, ClaimOwnerID: "supply-worker", Lease: time.Minute})
	if errors.Is(err, submission.ErrExecutionTargetSucceeded) {
		return s.temuChannel(ctx, scope, saved)
	}
	if errors.Is(err, submission.ErrExecutionTargetClaimed) {
		return unknownUpload(saved.ID, ""), nil
	}
	if err != nil {
		return UploadResult{}, err
	}
	if acquired.Permit == nil {
		return s.replay(ctx, scope, saved.ID, acquired.Attempt)
	}
	claim := uploadClaim(scope, acquired.Permit)
	ctx, cancel := context.WithDeadline(ctx, acquired.Permit.LeaseExpiresAt)
	defer cancel()
	// This is the actual-send boundary. Drift after claim retains UNKNOWN in the
	// original fence; it never unlocks the effect or manufactures another permit.
	if _, err = s.temuFacts(ctx, scope, saved); err != nil {
		return s.unknown(ctx, saved.ID, claim), nil
	}
	if err = s.temuPublicationImages(ctx, scope, saved, receipts, payload); err != nil {
		return s.unknown(ctx, saved.ID, claim), nil
	}
	merchant, err := s.dependencies.TemuStores.TemuExecutionMerchant(ctx, scope, saved.Merchant.StoreID, storecenter.ProductPurposePublish, &intent.Binding)
	if err != nil {
		return s.unknown(ctx, saved.ID, claim), nil
	}
	var input model.LocalGoodsV2
	if json.Unmarshal(payload, &input) != nil {
		return s.unknown(ctx, saved.ID, claim), nil
	}
	if !time.Now().Before(acquired.Permit.LeaseExpiresAt) {
		return s.unknown(ctx, saved.ID, claim), nil
	}
	result, err := merchant.CreateLocalGoods(ctx, input)
	if err != nil {
		return s.unknown(ctx, saved.ID, claim), nil
	}
	completion, err := submission.NewTemuPublishCompletion(scope, saved.ID, saved.Source.Source.ProductKey, intent.Binding, acquired.Attempt, claim, payload, result)
	if err != nil {
		return s.unknown(ctx, saved.ID, claim), nil
	}
	receipt, err := s.complete(ctx, scope, completion, acquired.Attempt.AttemptID)
	if err != nil {
		return s.unknown(ctx, saved.ID, claim), nil
	}
	return UploadResult{RecordID: saved.ID, PublishedRecordID: saved.ID, AttemptID: receipt.ID, Status: submission.ExecutionSucceeded, CurrentRecordUploaded: true, TemuProduct: receipt.TemuProduct, Message: "已创建 TEMU 商品；审核与在售状态等待店铺同步"}, nil
}

func (s *UploadService) temuPublicationImages(ctx context.Context, scope collection.Scope, saved record.TargetRecord, retained []submission.OfficialReceipt, payload []byte) error {
	current := make([]submission.OfficialReceipt, 0, len(retained))
	for _, receipt := range retained {
		actual, err := s.dependencies.Receipts.FindOfficialTarget(ctx, scope.OrganizationID, receipt.Target)
		if err != nil || actual.TemuImage == nil || collection.Digest(actual) != collection.Digest(receipt) || !time.Now().UTC().Add(submission.TemuImageSendWindow).Before(actual.TemuImage.UseBefore) {
			return record.ErrNotReady
		}
		current = append(current, actual)
	}
	resolved, _, err := submission.ResolveTemuPublicationPayload(saved, current, time.Now().UTC())
	if err != nil || !bytes.Equal(resolved, payload) {
		return record.ErrConflict
	}
	return nil
}

func (s *UploadService) temuImage(ctx context.Context, scope collection.Scope, saved record.TargetRecord, observed temugoods.ImageObservation) (submission.OfficialReceipt, *UploadResult, error) {
	var zero submission.OfficialReceipt
	subject, err := submission.TemuLocalImageSubjectID(observed.ContentHash, observed.CatID, observed.Usage, saved.Merchant)
	if err != nil {
		return zero, nil, err
	}
	target := submission.ExecutionTarget{Platform: "temu", StoreID: saved.Merchant.StoreID, SubjectID: subject}
	pending := func(id string) (submission.OfficialReceipt, *UploadResult, error) {
		result := unknownUpload(saved.ID, id)
		return zero, &result, nil
	}
	var prior *submission.OfficialReceipt
	if receipt, err := s.dependencies.Receipts.FindOfficialTarget(ctx, scope.OrganizationID, target); err == nil {
		if submission.ValidateOfficialReceipt(receipt) != nil || receipt.TemuImage == nil {
			return zero, nil, record.ErrConflict
		}
		if time.Now().UTC().Before(receipt.TemuImage.UseBefore) {
			return receipt, nil, nil
		}
		prior = &receipt
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
	proof, input, err := submission.PrepareTemuImageIntent(ctx, source, saved, observed, prior)
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
	raw, err := json.Marshal(input)
	if err != nil {
		return zero, nil, record.ErrInvalid
	}
	if _, err = s.temuFacts(ctx, scope, saved); err != nil {
		return zero, nil, err
	}
	acquired, err := s.dependencies.Kernel.Acquire(ctx, submission.AcquireExecutionCommand{Scope: submission.ExecutionScope{OrganizationID: scope.OrganizationID}, IntentKey: intent.Key, Target: intent.Target, Action: submission.TemuLocalImageAction, Payload: raw, ClaimOwnerID: "supply-worker", Lease: time.Minute})
	if errors.Is(err, submission.ErrExecutionTargetSucceeded) {
		receipt, err := s.dependencies.Receipts.FindOfficialTarget(ctx, scope.OrganizationID, target)
		return receipt, nil, err
	}
	if errors.Is(err, submission.ErrExecutionTargetClaimed) {
		return pending("")
	}
	if err != nil {
		return zero, nil, err
	}
	if acquired.Permit == nil {
		if acquired.Attempt.Status == submission.ExecutionSucceeded {
			receipt, err := s.dependencies.Receipts.ReadOfficial(ctx, scope, acquired.Attempt.AttemptID)
			return receipt, nil, err
		}
		result, err := s.replay(ctx, scope, saved.ID, acquired.Attempt)
		return zero, &result, err
	}
	claim := uploadClaim(scope, acquired.Permit)
	ctx, cancel := context.WithDeadline(ctx, acquired.Permit.LeaseExpiresAt)
	defer cancel()
	uncertain := func() (submission.OfficialReceipt, *UploadResult, error) {
		result := s.unknown(ctx, saved.ID, claim)
		return zero, &result, nil
	}
	if _, err = s.temuFacts(ctx, scope, saved); err != nil {
		return uncertain()
	}
	merchant, err := s.dependencies.TemuStores.TemuExecutionMerchant(ctx, scope, saved.Merchant.StoreID, storecenter.ProductPurposeImage, &intent.Binding)
	if err != nil {
		return uncertain()
	}
	if !time.Now().Before(acquired.Permit.LeaseExpiresAt) {
		return uncertain()
	}
	value, hash, err := merchant.UploadLocalImage(ctx, input)
	if err != nil {
		return uncertain()
	}
	completion, err := submission.NewTemuImageCompletion(scope, intent.Binding, acquired.Attempt, claim, input, observed, value, hash)
	if err != nil {
		return uncertain()
	}
	receipt, err := s.complete(ctx, scope, completion, acquired.Attempt.AttemptID)
	if err != nil {
		return uncertain()
	}
	return receipt, nil, nil
}
