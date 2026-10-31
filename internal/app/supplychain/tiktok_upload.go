package supplychainapp

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	storeapp "task-processor/internal/app/storecenter"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	marketplace "task-processor/internal/marketplace/tiktokshop"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

type TikTokExecutionMerchant interface {
	Binding() storecenter.ProductMerchantBinding
	UploadImage(context.Context, string, string, []byte) (marketplace.UploadedImage, error)
	CreateGlobal(context.Context, marketplace.CreateGlobalRequest) (marketplace.CreatedGlobalProduct, error)
	CreateLocal(context.Context, marketplace.CreateLocalRequest) (marketplace.CreatedLocalProduct, error)
	PublishUS(context.Context, string, marketplace.PublishGlobalRequest, marketplace.GlobalPublishExpectation) (marketplace.GlobalPublishResult, error)
}
type TikTokExecutionStore interface {
	TikTokExecutionMerchant(context.Context, collection.Scope, string, string, *storecenter.ProductMerchantBinding) (TikTokExecutionMerchant, error)
}
type OfficialTikTokExecutionStore struct {
	Access *storeapp.OfficialProductAccess
}

func (s OfficialTikTokExecutionStore) TikTokExecutionMerchant(ctx context.Context, scope collection.Scope, storeID, purpose string, expected *storecenter.ProductMerchantBinding) (TikTokExecutionMerchant, error) {
	if s.Access == nil {
		return nil, record.ErrUnavailable
	}
	return s.Access.AuthorizeTikTokMutation(ctx, storecenter.ProductExecutionSubject{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, Purpose: purpose}, storeID, expected)
}

type TikTokImageContentReader interface {
	record.TargetTikTokImageReader
	ReadTikTokContent(context.Context, asset.ApprovedAsset, string) (marketplace.ImageObservation, []byte, error)
}

type tikTokPreparationKernel interface {
	PrepareTikTokExecution(context.Context, submission.AcquireExecutionCommand) (submission.TikTokPreparedAcquisition, error)
	BeginTikTokSend(context.Context, *submission.TikTokPreparation) (*submission.SendPermit, error)
	StopTikTokBeforeSend(context.Context, *submission.TikTokPreparation, string) (submission.ExecutionAttempt, error)
}

func tiktokStageKey(scope collection.Scope, parent, recordID, stage string, target submission.ExecutionTarget) string {
	return collection.StableID(scope.OrganizationID, "tiktok-stage", parent, stage, target.StoreID, target.SubjectID, recordID)
}

func tiktokNotSent(recordID, attemptID string) UploadResult {
	return UploadResult{RecordID: recordID, AttemptID: attemptID, Status: submission.ExecutionNotSent, Message: "本次未发送，处理缺项并核对资料后可重新确认上传；已成功步骤不会重复执行"}
}

func (s *UploadService) tiktokOriginalStep(ctx context.Context, scope collection.Scope, saved record.TargetRecord, key, stage string, target submission.ExecutionTarget) (*UploadResult, error) {
	a, err := s.dependencies.Kernel.ReadIntent(ctx, submission.ExecutionScope{OrganizationID: scope.OrganizationID}, key)
	if errors.Is(err, submission.ErrExecutionNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	intent, err := s.dependencies.Intents.ReadOfficialIntent(ctx, scope, key)
	if err != nil || intent.Owner != scope || intent.RecordID != saved.ID || intent.RecordHash != collection.Digest(saved) || intent.Kind != stage || intent.Target != target || intent.PayloadFingerprint != a.PayloadFingerprint || a.IntentKey != key || a.OrganizationID != scope.OrganizationID || a.Target != target || a.Action != submission.TikTokStageAction(stage) || submission.ValidatePersistedExecutionAttempt(a) != nil {
		return nil, submission.ErrExecutionIntentConflict
	}
	switch a.Status {
	case submission.ExecutionNotSent:
		result := tiktokNotSent(saved.ID, a.AttemptID)
		return &result, nil
	case submission.ExecutionClaimed, submission.ExecutionOutcomeUnknown:
		_, _ = s.dependencies.Kernel.Expire(ctx, submission.ExecutionScope{OrganizationID: scope.OrganizationID}, a.AttemptID)
		result := unknownUpload(saved.ID, a.AttemptID)
		return &result, nil
	}
	return nil, nil
}

func (s *UploadService) tiktokOriginalCommand(ctx context.Context, scope collection.Scope, saved record.TargetRecord, key string) (*UploadResult, error) {
	for _, image := range saved.TikTokImageObservations {
		target, err := tiktokTarget(saved, submission.TikTokImageStage, &image)
		if err != nil {
			return nil, err
		}
		stepKey := tiktokStageKey(scope, key, saved.ID, submission.TikTokImageStage, target)
		if result, err := s.tiktokOriginalStep(ctx, scope, saved, stepKey, submission.TikTokImageStage, target); result != nil || err != nil {
			return result, err
		}
	}
	if saved.Input.TikTok.Selection.Method == marketplace.GlobalPublishing {
		target, err := tiktokTarget(saved, submission.TikTokGlobalCreateStage, nil)
		if err != nil {
			return nil, err
		}
		stepKey := tiktokStageKey(scope, key, saved.ID, submission.TikTokGlobalCreateStage, target)
		if result, err := s.tiktokOriginalStep(ctx, scope, saved, stepKey, submission.TikTokGlobalCreateStage, target); result != nil || err != nil {
			return result, err
		}
	}
	target, err := tiktokTarget(saved, submission.TikTokPublishStage, nil)
	if err != nil {
		return nil, err
	}
	return s.tiktokOriginalStep(ctx, scope, saved, key, submission.TikTokPublishStage, target)
}

func (s *UploadService) tiktokFacts(ctx context.Context, scope collection.Scope, saved record.TargetRecord) (asset.ApprovedAssetInventory, error) {
	var inventory asset.ApprovedAssetInventory
	if !record.PublicationReady(saved) || s.dependencies.TikTokRules == nil || s.dependencies.TikTokStores == nil || s.dependencies.TikTokImages == nil {
		return inventory, record.ErrNotReady
	}
	if err := s.authorize(ctx, scope); err != nil {
		return inventory, err
	}
	if s.dependencies.ReviewGate != nil {
		if err := s.dependencies.ReviewGate(ctx, scope, saved); err != nil {
			return inventory, err
		}
	}
	head, err := s.dependencies.Records.ReadTargetHead(ctx, scope, saved.TargetID)
	if err != nil || head.ID != saved.ID || collection.Digest(head) != collection.Digest(saved) {
		return inventory, record.ErrConflict
	}
	proof, err := s.source(ctx, scope, saved)
	if err != nil {
		return inventory, err
	}
	product, err := s.dependencies.Products.ReadEffectiveTargetProduct(ctx, proof, saved.EffectiveVersion, saved.ApplyReceiptID)
	if err != nil || product.Identity.TenantID != scope.OrganizationID || product.Identity.ProductKey != saved.Source.Source.ProductKey || product.Version != saved.EffectiveVersion || collection.Digest(product.Snapshot) != saved.ProductHash {
		return inventory, record.ErrConflict
	}
	assetScope := asset.InventoryScope{TenantID: scope.OrganizationID, ProductKey: product.Identity.ProductKey, TargetPlatform: "tiktokshop", SourceSnapshotVersion: product.Version}
	inventory, err = s.dependencies.Assets.GetApprovedInventory(ctx, assetScope)
	if err != nil || inventory.Scope != assetScope || collection.Digest(inventory) != saved.InventoryHash {
		return inventory, record.ErrNotReady
	}
	binding, rules, err := s.dependencies.TikTokRules.ReadTikTokTargetRules(ctx, scope, saved.Merchant.StoreID, saved.Input.TikTok.Selection)
	if err != nil {
		return inventory, err
	}
	if binding != saved.Merchant || marketplace.RuleHash(rules) != saved.RulesHash {
		return inventory, record.ErrConflict
	}
	observations := []marketplace.ImageObservation{}
	for _, selected := range saved.TikTokImageObservations {
		found := false
		for _, approved := range inventory.Assets {
			if approved.ID == selected.AssetID {
				current, err := s.dependencies.TikTokImages.ProbeTikTok(ctx, approved, selected.UseCase)
				if err != nil || current != selected {
					return inventory, record.ErrNotReady
				}
				observations = append(observations, current)
				found = true
				break
			}
		}
		if !found {
			return inventory, record.ErrNotReady
		}
	}
	if !marketplace.BuildDraft(*saved.Input.TikTok, rules, inventory, observations).ReadyForUpload {
		return inventory, record.ErrNotReady
	}
	return inventory, nil
}
func tiktokTarget(saved record.TargetRecord, stage string, image *marketplace.ImageObservation) (submission.ExecutionTarget, error) {
	subject, err := submission.TikTokProductSubjectID(saved.Source.Source.ProductKey, stage)
	if stage == submission.TikTokImageStage {
		if image == nil {
			return submission.ExecutionTarget{}, record.ErrInvalid
		}
		subject, err = submission.TikTokImageSubjectID(*image, saved.Merchant)
	}
	return submission.ExecutionTarget{Platform: "tiktokshop", StoreID: saved.Merchant.StoreID, SubjectID: subject}, err
}
func matchingTikTokReceipt(receipt submission.OfficialReceipt, saved record.TargetRecord, stage string, image *marketplace.ImageObservation) bool {
	if submission.ValidateOfficialReceipt(receipt) != nil || receipt.Kind != stage || !submission.TikTokCompatibleBinding(receipt.Binding, saved.Merchant) {
		return false
	}
	if stage == submission.TikTokImageStage {
		return image != nil && receipt.TikTok.Image != nil && submission.TikTokSameImageContent(*receipt.TikTok.Image, *image)
	}
	return receipt.ProductKey == saved.Source.Source.ProductKey && receipt.TikTok.DraftHash == submission.TikTokDraftHash(saved)
}
func tiktokUploadResult(saved record.TargetRecord, receipt submission.OfficialReceipt) UploadResult {
	result := UploadResult{RecordID: saved.ID, AttemptID: receipt.ID, Status: submission.ExecutionSucceeded, TikTok: receipt.TikTok}
	if receipt.Owner.OrganizationID == saved.Merchant.OrganizationID {
		result.PublishedRecordID = receipt.RecordID
		result.CurrentRecordUploaded = receipt.RecordID == saved.ID && receipt.Kind == submission.TikTokPublishStage
	}
	result.Message = "已保存 TikTok Shop 的实际创建回执；审核及在售状态待平台确认"
	if receipt.TikTok.LocalSaveMode == "AS_DRAFT" {
		result.Message = "已创建 TikTok Shop 平台草稿"
	}
	if publication := receipt.TikTok.Publication; publication != nil {
		switch publication.State {
		case marketplace.SubmittedForReview:
			result.Message = "已提交 TikTok Shop 美国站审核"
		case marketplace.RemoteDraft:
			result.Message = "平台校验未通过，已保留实际创建的草稿商品；不会重复创建"
		case marketplace.PublicationFailed:
			result.Message = "美国站同步未成功，已保留全球商品及原结果；不会自动重发"
		}
	}
	return result
}
func (s *UploadService) uploadTikTok(ctx context.Context, scope collection.Scope, key string, saved record.TargetRecord) (UploadResult, error) {
	if !record.PublicationReady(saved) || s.dependencies.TikTokStores == nil {
		return UploadResult{}, record.ErrNotReady
	}
	// Fresh original-member and Shop access applies even when reading a retained result.
	if _, err := s.dependencies.TikTokStores.TikTokExecutionMerchant(ctx, scope, saved.Merchant.StoreID, storecenter.ProductPurposePublish, &saved.Merchant); err != nil {
		return UploadResult{}, err
	}
	if original, err := s.tiktokOriginalCommand(ctx, scope, saved, key); original != nil || err != nil {
		if err != nil {
			return UploadResult{}, err
		}
		return *original, nil
	}
	target, err := tiktokTarget(saved, submission.TikTokPublishStage, nil)
	if err != nil {
		return UploadResult{}, err
	}
	if prior, err := s.dependencies.Receipts.FindOfficialTarget(ctx, scope.OrganizationID, target); err == nil {
		if submission.ValidateOfficialReceipt(prior) != nil || prior.Kind != submission.TikTokPublishStage || prior.ProductKey != saved.Source.Source.ProductKey || !submission.TikTokCompatibleBinding(prior.Binding, saved.Merchant) {
			return UploadResult{}, record.ErrConflict
		}
		return tiktokUploadResult(saved, prior), nil
	} else if errors.Is(err, submission.ErrExecutionTargetClaimed) {
		return unknownUpload(saved.ID, ""), nil
	} else if !errors.Is(err, submission.ErrExecutionNotFound) {
		return UploadResult{}, err
	}
	if _, err = s.tiktokFacts(ctx, scope, saved); err != nil {
		return UploadResult{}, err
	}
	var global *submission.OfficialReceipt
	if saved.Input.TikTok.Selection.Method == marketplace.GlobalPublishing {
		globalTarget, e := tiktokTarget(saved, submission.TikTokGlobalCreateStage, nil)
		if e != nil {
			return UploadResult{}, e
		}
		if prior, e := s.dependencies.Receipts.FindOfficialTarget(ctx, scope.OrganizationID, globalTarget); e == nil {
			if !matchingTikTokReceipt(prior, saved, submission.TikTokGlobalCreateStage, nil) {
				return UploadResult{}, record.ErrConflict
			}
			global = &prior
		} else if errors.Is(e, submission.ErrExecutionTargetClaimed) {
			return unknownUpload(saved.ID, ""), nil
		} else if !errors.Is(e, submission.ErrExecutionNotFound) {
			return UploadResult{}, e
		}
	}
	images := []submission.OfficialReceipt{}
	if global == nil {
		for _, image := range saved.TikTokImageObservations {
			target, e := tiktokTarget(saved, submission.TikTokImageStage, &image)
			if e != nil {
				return UploadResult{}, e
			}
			imageKey := tiktokStageKey(scope, key, saved.ID, submission.TikTokImageStage, target)
			receipt, pending, e := s.tiktokStage(ctx, scope, saved, imageKey, submission.TikTokImageStage, &image, nil, nil)
			if e != nil {
				return UploadResult{}, e
			}
			if pending != nil {
				return *pending, nil
			}
			images = append(images, receipt)
		}
		if saved.Input.TikTok.Selection.Method == marketplace.GlobalPublishing {
			globalTarget, e := tiktokTarget(saved, submission.TikTokGlobalCreateStage, nil)
			if e != nil {
				return UploadResult{}, e
			}
			globalKey := tiktokStageKey(scope, key, saved.ID, submission.TikTokGlobalCreateStage, globalTarget)
			receipt, pending, e := s.tiktokStage(ctx, scope, saved, globalKey, submission.TikTokGlobalCreateStage, nil, images, nil)
			if e != nil {
				return UploadResult{}, e
			}
			if pending != nil {
				return *pending, nil
			}
			global = &receipt
			images = nil
		}
	}
	receipt, pending, err := s.tiktokStage(ctx, scope, saved, key, submission.TikTokPublishStage, nil, images, global)
	if err != nil {
		return UploadResult{}, err
	}
	if pending != nil {
		return *pending, nil
	}
	return tiktokUploadResult(saved, receipt), nil
}
func (s *UploadService) tiktokStage(ctx context.Context, scope collection.Scope, saved record.TargetRecord, key, stage string, image *marketplace.ImageObservation, images []submission.OfficialReceipt, global *submission.OfficialReceipt) (submission.OfficialReceipt, *UploadResult, error) {
	var zero submission.OfficialReceipt
	kernel, ok := s.dependencies.Kernel.(tikTokPreparationKernel)
	if !ok {
		return zero, nil, record.ErrUnavailable
	}
	target, err := tiktokTarget(saved, stage, image)
	if err != nil {
		return zero, nil, err
	}
	pending := func(id string) (submission.OfficialReceipt, *UploadResult, error) {
		result := unknownUpload(saved.ID, id)
		return zero, &result, nil
	}
	if original, err := s.tiktokOriginalStep(ctx, scope, saved, key, stage, target); original != nil || err != nil {
		return zero, original, err
	}
	if prior, err := s.dependencies.Receipts.FindOfficialTarget(ctx, scope.OrganizationID, target); err == nil {
		if !matchingTikTokReceipt(prior, saved, stage, image) {
			return zero, nil, record.ErrConflict
		}
		return prior, nil, nil
	} else if errors.Is(err, submission.ErrExecutionTargetClaimed) {
		return pending("")
	} else if !errors.Is(err, submission.ErrExecutionNotFound) {
		return zero, nil, err
	}
	if _, err = s.tiktokFacts(ctx, scope, saved); err != nil {
		return zero, nil, err
	}
	source, err := s.source(ctx, scope, saved)
	if err != nil {
		return zero, nil, err
	}
	proof, payload, err := submission.PrepareTikTokIntent(ctx, source, saved, key, stage, image, images, global)
	if err != nil {
		return zero, nil, err
	}
	intent, err := s.dependencies.Intents.PrepareOfficial(ctx, proof)
	if err != nil {
		return zero, nil, err
	}
	if !submission.TikTokPayloadMatchesIntent(intent, payload) {
		return zero, nil, submission.ErrExecutionIntentConflict
	}
	acquired, err := kernel.PrepareTikTokExecution(ctx, submission.AcquireExecutionCommand{Scope: submission.ExecutionScope{OrganizationID: scope.OrganizationID}, IntentKey: key, Target: target, Action: submission.TikTokStageAction(stage), Payload: payload, ClaimOwnerID: "supply-worker", Lease: time.Minute})
	if errors.Is(err, submission.ErrExecutionTargetSucceeded) {
		receipt, e := s.dependencies.Receipts.FindOfficialTarget(ctx, scope.OrganizationID, target)
		if e != nil {
			return zero, nil, e
		}
		if !matchingTikTokReceipt(receipt, saved, stage, image) {
			return zero, nil, record.ErrConflict
		}
		return receipt, nil, nil
	}
	if errors.Is(err, submission.ErrExecutionTargetClaimed) {
		return pending("")
	}
	if err != nil {
		return zero, nil, err
	}
	if acquired.Preparation == nil {
		if acquired.Attempt.Status == submission.ExecutionNotSent {
			result := tiktokNotSent(saved.ID, acquired.Attempt.AttemptID)
			return zero, &result, nil
		}
		if acquired.Attempt.Status == submission.ExecutionSucceeded {
			receipt, e := s.dependencies.Receipts.ReadOfficial(ctx, scope, acquired.Attempt.AttemptID)
			if e != nil {
				return zero, nil, e
			}
			if !matchingTikTokReceipt(receipt, saved, stage, image) {
				return zero, nil, record.ErrConflict
			}
			return receipt, nil, nil
		}
		_, _ = s.dependencies.Kernel.Expire(ctx, submission.ExecutionScope{OrganizationID: scope.OrganizationID}, acquired.Attempt.AttemptID)
		return pending(acquired.Attempt.AttemptID)
	}
	var claim submission.ExecutionClaim
	ctx, cancel := context.WithDeadline(ctx, acquired.Attempt.LeaseExpiresAt)
	defer cancel()
	noSend := func(reason string) (submission.OfficialReceipt, *UploadResult, error) {
		writeCtx, close := context.WithTimeout(context.WithoutCancel(ctx), submission.ExecutionTimeout)
		defer close()
		if a, err := kernel.StopTikTokBeforeSend(writeCtx, acquired.Preparation, reason); err == nil {
			result := tiktokNotSent(saved.ID, a.AttemptID)
			return zero, &result, nil
		}
		return pending(acquired.Attempt.AttemptID)
	}
	begin := func() error {
		permit, err := kernel.BeginTikTokSend(ctx, acquired.Preparation)
		if err == nil {
			claim = uploadClaim(scope, permit)
		}
		return err
	}
	uncertain := func() (submission.OfficialReceipt, *UploadResult, error) {
		result := s.unknown(ctx, saved.ID, claim)
		return zero, &result, nil
	}
	inventory, err := s.tiktokFacts(ctx, scope, saved)
	if err != nil {
		return noSend(submission.NoSendPreflightUnavailable)
	}
	purpose := storecenter.ProductPurposePublish
	if stage == submission.TikTokImageStage {
		purpose = storecenter.ProductPurposeImage
	}
	merchant, err := s.dependencies.TikTokStores.TikTokExecutionMerchant(ctx, scope, saved.Merchant.StoreID, purpose, &intent.Binding)
	if err != nil {
		return noSend(submission.NoSendAccessRecheckFailed)
	}
	effect := submission.TikTokEffectReceipt{DraftHash: intent.TikTok.DraftHash}
	switch stage {
	case submission.TikTokImageStage:
		var approved asset.ApprovedAsset
		for _, a := range inventory.Assets {
			if a.ID == image.AssetID {
				approved = a
				break
			}
		}
		observed, content, e := s.dependencies.TikTokImages.ReadTikTokContent(ctx, approved, image.UseCase)
		defer clear(content)
		if e != nil {
			return noSend(submission.NoSendPreflightUnavailable)
		}
		if observed != *image {
			return noSend(submission.NoSendImageContentChanged)
		}
		if err := begin(); err != nil {
			return noSend(submission.NoSendPreflightUnavailable)
		}
		uploaded, e := merchant.UploadImage(ctx, image.UseCase, observed.MediaType, content)
		if e != nil {
			return uncertain()
		}
		effect.Image = image
		effect.Uploaded = &uploaded
	case submission.TikTokGlobalCreateStage:
		var input marketplace.CreateGlobalRequest
		if json.Unmarshal(payload, &input) != nil {
			return noSend(submission.NoSendPayloadInvalid)
		}
		if err := begin(); err != nil {
			return noSend(submission.NoSendPreflightUnavailable)
		}
		created, e := merchant.CreateGlobal(ctx, input)
		if e != nil {
			return uncertain()
		}
		effect.Global = &created
	case submission.TikTokPublishStage:
		if global == nil {
			var input marketplace.CreateLocalRequest
			if json.Unmarshal(payload, &input) != nil {
				return noSend(submission.NoSendPayloadInvalid)
			}
			if err := begin(); err != nil {
				return noSend(submission.NoSendPreflightUnavailable)
			}
			created, e := merchant.CreateLocal(ctx, input)
			if e != nil {
				return uncertain()
			}
			effect.Local = &created
		} else {
			var input marketplace.PublishGlobalRequest
			if json.Unmarshal(payload, &input) != nil {
				return noSend(submission.NoSendPayloadInvalid)
			}
			expected := marketplace.GlobalPublishExpectation{ShopID: intent.Binding.TikTok.ShopID}
			for _, sku := range global.TikTok.Global.SKUs {
				expected.GlobalSKUIDs = append(expected.GlobalSKUIDs, sku.ID)
			}
			if err := begin(); err != nil {
				return noSend(submission.NoSendPreflightUnavailable)
			}
			result, e := merchant.PublishUS(ctx, global.TikTok.Global.ID, input, expected)
			if e != nil {
				return uncertain()
			}
			effect.Publication = &result
		}
	default:
		return noSend(submission.NoSendPayloadInvalid)
	}
	completion, err := submission.NewTikTokCompletion(scope, saved, intent, acquired.Attempt, claim, payload, effect)
	if err != nil {
		return uncertain()
	}
	receipt, err := s.complete(ctx, scope, completion, acquired.Attempt.AttemptID)
	if err != nil {
		return uncertain()
	}
	return receipt, nil, nil
}
