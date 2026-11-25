package listingapp

import (
	"context"
	"errors"
	"time"

	storeapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/storecenter"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/preparation"
	record "github.com/henryxu1008/shuomi-ai-commerce/internal/listing/record/target"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

type ShopifyDraftWriter interface {
	Create(context.Context, shopify.Draft) (shopify.ProductObservation, error)
}

type ShopifyExecutionStore interface {
	Binding(context.Context, collection.Scope, string, string) (storecenter.ProductMerchantBinding, error)
	VerifyDraftWrite(context.Context, collection.Scope, storecenter.ProductMerchantBinding) (ShopifyDraftWriter, error)
}

type OfficialShopifyExecutionStore struct {
	Access *storeapp.OfficialProductAccess
}

func shopifyProductSubject(scope collection.Scope, purpose string) storecenter.ProductExecutionSubject {
	return storecenter.ProductExecutionSubject{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, Purpose: purpose}
}

func (s OfficialShopifyExecutionStore) Binding(ctx context.Context, scope collection.Scope, storeID, purpose string) (storecenter.ProductMerchantBinding, error) {
	return s.Access.ShopifyProductBinding(ctx, shopifyProductSubject(scope, purpose), storeID)
}

func (s OfficialShopifyExecutionStore) ShopifyTargetBinding(ctx context.Context, scope collection.Scope, storeID string) (storecenter.ProductMerchantBinding, error) {
	return s.Binding(ctx, scope, storeID, storecenter.ProductPurposeRules)
}

func (s OfficialShopifyExecutionStore) ShopifyPlanDomain(ctx context.Context, scope collection.Scope, expected storecenter.ProductMerchantBinding) (string, error) {
	return s.Access.ShopifyPlanDomain(ctx, shopifyProductSubject(scope, storecenter.ProductPurposeRules), expected)
}

func (s OfficialShopifyExecutionStore) VerifyDraftWrite(ctx context.Context, scope collection.Scope, expected storecenter.ProductMerchantBinding) (ShopifyDraftWriter, error) {
	writer, err := s.Access.VerifyShopifyDraftWrite(ctx, shopifyProductSubject(scope, storecenter.ProductPurposePublish), expected)
	if err != nil {
		return nil, err
	}
	return writer, nil
}

type shopifySubmissionKernel interface {
	PrepareShopifyExecution(context.Context, submission.AcquireExecutionCommand) (submission.TikTokPreparedAcquisition, error)
	BeginShopifySend(context.Context, *submission.TikTokPreparation) (*submission.SendPermit, error)
	StopShopifyBeforeSend(context.Context, *submission.TikTokPreparation, string) (submission.ExecutionAttempt, error)
}

func (s *UploadService) uploadShopify(ctx context.Context, scope collection.Scope, key string, saved record.TargetRecord) (UploadResult, error) {
	stores := s.dependencies.ShopifyStores
	kernel, supported := s.dependencies.Kernel.(shopifySubmissionKernel)
	if stores == nil || !supported {
		return UploadResult{}, record.ErrUnavailable
	}
	binding, err := stores.Binding(ctx, scope, saved.Merchant.StoreID, storecenter.ProductPurposePublish)
	if err != nil || !binding.ValidShopifyApplication() || binding.Shopify.ShopKey != saved.Merchant.Shopify.ShopKey {
		return UploadResult{}, record.ErrForbidden
	}
	target, err := submission.ShopifyDraftTarget(binding, saved.Source.Source.ProductKey)
	if err != nil {
		return UploadResult{}, err
	}
	retained, err := s.dependencies.Receipts.FindOfficialTarget(ctx, scope.OrganizationID, target)
	if err == nil {
		return shopifyUploadResult(saved.ID, retained), nil
	}
	if !errors.Is(err, submission.ErrExecutionNotFound) && !errors.Is(err, submission.ErrExecutionTargetClaimed) {
		return UploadResult{}, err
	}
	if binding != saved.Merchant || !record.PublicationReady(saved) {
		return UploadResult{}, record.ErrNotReady
	}
	proof, _, err := s.shopifyDraftFacts(ctx, scope, saved)
	if err != nil {
		return UploadResult{}, err
	}
	intentProof, raw, err := submission.PrepareShopifyDraftIntent(ctx, proof, saved, key)
	if err != nil {
		return UploadResult{}, err
	}
	draft, _, err := submission.ShopifyDraftPayload(saved)
	if err != nil {
		return UploadResult{}, err
	}
	intent, err := s.dependencies.Intents.PrepareOfficial(ctx, intentProof)
	if err != nil {
		return UploadResult{}, err
	}
	if intent.Target != target || intent.Binding != binding || !submission.ShopifyPayloadMatchesIntent(intent, raw) {
		return UploadResult{}, record.ErrConflict
	}
	acquired, err := kernel.PrepareShopifyExecution(ctx, submission.AcquireExecutionCommand{
		Scope: submission.ExecutionScope{OrganizationID: scope.OrganizationID}, IntentKey: key, Target: target,
		Action: submission.ShopifyDraftAction, Payload: raw, ClaimOwnerID: "supply-worker", Lease: time.Minute,
	})
	if errors.Is(err, submission.ErrExecutionTargetSucceeded) {
		retained, err = s.dependencies.Receipts.FindOfficialTarget(ctx, scope.OrganizationID, target)
		if err == nil {
			return shopifyUploadResult(saved.ID, retained), nil
		}
	}
	if errors.Is(err, submission.ErrExecutionTargetClaimed) || errors.Is(err, submission.ErrExecutionOutcomeUnknown) {
		return unknownUpload(saved.ID, ""), nil
	}
	if err != nil {
		return UploadResult{}, err
	}
	if acquired.Preparation == nil {
		switch acquired.Attempt.Status {
		case submission.ExecutionNotSent:
			return shopifyNotSent(saved.ID, acquired.Attempt.AttemptID), nil
		case submission.ExecutionSucceeded:
			retained, err := s.dependencies.Receipts.ReadOfficial(ctx, scope, acquired.Attempt.AttemptID)
			if err != nil {
				return UploadResult{}, err
			}
			return shopifyUploadResult(saved.ID, retained), nil
		default:
			return s.replay(ctx, scope, saved.ID, acquired.Attempt)
		}
	}
	ctx, cancel := context.WithDeadline(ctx, acquired.Attempt.LeaseExpiresAt)
	defer cancel()
	stop := func(reason string) (UploadResult, error) {
		retention, close := context.WithTimeout(context.WithoutCancel(ctx), submission.ExecutionTimeout)
		defer close()
		attempt, err := kernel.StopShopifyBeforeSend(retention, acquired.Preparation, reason)
		if err != nil {
			return unknownUpload(saved.ID, acquired.Attempt.AttemptID), nil
		}
		return shopifyNotSent(saved.ID, attempt.AttemptID), nil
	}
	_, inventory, err := s.shopifyDraftFacts(ctx, scope, saved)
	if err != nil {
		return stop(submission.NoSendFactsChanged)
	}
	// Reading local approved artifacts can outlast a Shopify writer's short
	// verification window. Obtain it immediately before the first remote write.
	images, err := s.readShopifyDraftImages(ctx, scope, saved, inventory, draft)
	if err != nil {
		return stop(submission.NoSendImageContentChanged)
	}
	if len(images.contents) > 0 {
		if _, _, err := s.shopifyDraftFacts(ctx, scope, saved); err != nil {
			return stop(submission.NoSendFactsChanged)
		}
	}
	writer, err := stores.VerifyDraftWrite(ctx, scope, binding)
	if err != nil || writer == nil {
		return stop(submission.NoSendAccessRecheckFailed)
	}
	if len(images.contents) > 0 {
		if err := images.stage(ctx, writer, &draft); err != nil {
			return stop(submission.NoSendImageContentChanged)
		}
		if _, _, err := s.shopifyDraftFacts(ctx, scope, saved); err != nil {
			return stop(submission.NoSendFactsChanged)
		}
		writer, err = stores.VerifyDraftWrite(ctx, scope, binding)
		if err != nil || writer == nil {
			return stop(submission.NoSendAccessRecheckFailed)
		}
	}
	permit, err := kernel.BeginShopifySend(ctx, acquired.Preparation)
	if err != nil {
		return stop(submission.NoSendPreflightUnavailable)
	}
	claim := uploadClaim(scope, permit)
	outcome, err := writer.Create(ctx, draft)
	if err != nil {
		return s.unknown(ctx, saved.ID, claim), nil
	}
	completion, err := submission.NewShopifyDraftCompletion(scope, saved, acquired.Attempt, claim, raw, outcome)
	if err != nil {
		return s.unknown(ctx, saved.ID, claim), nil
	}
	retained, err = s.complete(ctx, scope, completion, acquired.Attempt.AttemptID)
	if err != nil {
		return s.unknown(ctx, saved.ID, claim), nil
	}
	return shopifyUploadResult(saved.ID, retained), nil
}

func (s *UploadService) shopifyDraftFacts(ctx context.Context, scope collection.Scope, saved record.TargetRecord) (preparation.AuthorizedSource, asset.ApprovedAssetInventory, error) {
	if err := s.authorize(ctx, scope); err != nil {
		return preparation.AuthorizedSource{}, asset.ApprovedAssetInventory{}, err
	}
	if s.dependencies.ReviewGate != nil {
		if err := s.dependencies.ReviewGate(ctx, scope, saved); err != nil {
			return preparation.AuthorizedSource{}, asset.ApprovedAssetInventory{}, err
		}
	}
	head, err := s.dependencies.Records.ReadTargetHead(ctx, scope, saved.TargetID)
	if err != nil || head.ID != saved.ID || collection.Digest(head) != collection.Digest(saved) {
		return preparation.AuthorizedSource{}, asset.ApprovedAssetInventory{}, record.ErrConflict
	}
	proof, err := s.source(ctx, scope, saved)
	if err != nil {
		return preparation.AuthorizedSource{}, asset.ApprovedAssetInventory{}, err
	}
	product, err := s.dependencies.Products.ReadEffectiveTargetProduct(ctx, proof, saved.EffectiveVersion, saved.ApplyReceiptID)
	if err != nil || product.Identity.TenantID != scope.OrganizationID || product.Identity.ProductKey != saved.Source.Source.ProductKey ||
		product.Version != saved.EffectiveVersion || collection.Digest(product.Snapshot) != saved.ProductHash {
		return preparation.AuthorizedSource{}, asset.ApprovedAssetInventory{}, record.ErrConflict
	}
	assetScope := asset.InventoryScope{TenantID: scope.OrganizationID, ProductKey: product.Identity.ProductKey, TargetPlatform: "product", SourceSnapshotVersion: product.Version}
	inventory, err := s.dependencies.Assets.GetApprovedInventory(ctx, assetScope)
	if errors.Is(err, asset.ErrApprovedAssetsNotReady) {
		inventory = asset.ApprovedAssetInventory{Scope: assetScope, Assets: []asset.ApprovedAsset{}}
	} else if err != nil {
		return preparation.AuthorizedSource{}, asset.ApprovedAssetInventory{}, record.ErrUnavailable
	}
	if inventory.Scope != assetScope || collection.Digest(inventory) != saved.InventoryHash {
		return preparation.AuthorizedSource{}, asset.ApprovedAssetInventory{}, record.ErrNotReady
	}
	draft, err := shopify.BuildDraft(*saved.Input.Shopify, product.Snapshot, inventory, saved.Merchant.Shopify.Currency)
	if err != nil || collection.Digest(draft) != collection.Digest(saved.Shopify) {
		return preparation.AuthorizedSource{}, asset.ApprovedAssetInventory{}, record.ErrNotReady
	}
	return proof, inventory, nil
}

func shopifyNotSent(recordID, attemptID string) UploadResult {
	return UploadResult{RecordID: recordID, AttemptID: attemptID, Status: submission.ExecutionNotSent, Message: "尚未发送到 Shopify。请检查店铺授权和商品资料后重新确认。"}
}

func shopifyUploadResult(recordID string, retained submission.OfficialReceipt) UploadResult {
	return UploadResult{
		RecordID: recordID, PublishedRecordID: retained.RecordID, AttemptID: retained.ID,
		Status: submission.ExecutionSucceeded, CurrentRecordUploaded: recordID == retained.RecordID,
		Shopify: retained.Shopify, Message: "已保存到 Shopify 草稿",
	}
}
