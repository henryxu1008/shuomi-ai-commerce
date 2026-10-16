package supplychainapp

import (
	"context"
	"errors"
	"reflect"
	storeapp "task-processor/internal/app/storecenter"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	marketplace "task-processor/internal/marketplace/walmart"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

type WalmartExecutionMerchant interface {
	Binding() storecenter.ProductMerchantBinding
	SubmitFeed(context.Context, marketplace.Draft, string) (marketplace.AcceptedFeed, error)
}
type WalmartExecutionStore interface {
	WalmartExecutionMerchant(context.Context, collection.Scope, string, *storecenter.ProductMerchantBinding) (WalmartExecutionMerchant, error)
}
type OfficialWalmartExecutionStore struct {
	Access *storeapp.OfficialProductAccess
}

func (s OfficialWalmartExecutionStore) WalmartExecutionMerchant(ctx context.Context, scope collection.Scope, id string, expected *storecenter.ProductMerchantBinding) (WalmartExecutionMerchant, error) {
	if s.Access == nil {
		return nil, record.ErrUnavailable
	}
	return s.Access.AuthorizeWalmartMutation(ctx, storecenter.ProductExecutionSubject{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, Purpose: storecenter.ProductPurposePublish}, id, expected)
}
func (s *UploadService) walmartFacts(ctx context.Context, scope collection.Scope, saved record.TargetRecord) error {
	if !record.PublicationReady(saved) || saved.Walmart == nil || s.dependencies.WalmartRules == nil || s.dependencies.WalmartStores == nil || s.dependencies.WalmartReadStores == nil {
		return record.ErrNotReady
	}
	if e := s.authorize(ctx, scope); e != nil {
		return e
	}
	if s.dependencies.ReviewGate != nil {
		if e := s.dependencies.ReviewGate(ctx, scope, saved); e != nil {
			return e
		}
	}
	head, e := s.dependencies.Records.ReadTargetHead(ctx, scope, saved.TargetID)
	if e != nil || head.ID != saved.ID || collection.Digest(head) != collection.Digest(saved) {
		return record.ErrConflict
	}
	proof, e := s.source(ctx, scope, saved)
	if e != nil {
		return e
	}
	p, e := s.dependencies.Products.ReadEffectiveTargetProduct(ctx, proof, saved.EffectiveVersion, saved.ApplyReceiptID)
	if e != nil || p.Identity.TenantID != scope.OrganizationID || p.Identity.ProductKey != saved.Source.Source.ProductKey || p.Version != saved.EffectiveVersion || collection.Digest(p.Snapshot) != saved.ProductHash {
		return record.ErrConflict
	}
	as := asset.InventoryScope{TenantID: scope.OrganizationID, ProductKey: p.Identity.ProductKey, TargetPlatform: "walmart", SourceSnapshotVersion: p.Version}
	inventory, e := s.dependencies.Assets.GetApprovedInventory(ctx, as)
	if errors.Is(e, asset.ErrApprovedAssetsNotReady) {
		inventory = asset.ApprovedAssetInventory{Scope: as, Assets: []asset.ApprovedAsset{}}
	} else if e != nil {
		return record.ErrUnavailable
	}
	if inventory.Scope != as || collection.Digest(inventory) != saved.InventoryHash {
		return record.ErrConflict
	}
	binding, rules, e := s.dependencies.WalmartRules.ReadWalmartTargetRules(ctx, scope, saved.Merchant.StoreID, saved.Input.Walmart.Selection)
	if e != nil {
		return e
	}
	if binding != saved.Merchant || rules.Hash() != saved.RulesHash {
		return record.ErrConflict
	}
	images, e := record.ProbeWalmartTargetImages(ctx, s.dependencies.WalmartImages, *saved.Input.Walmart, inventory)
	if e != nil || len(images) != len(saved.WalmartImageObservations) || len(images) > 0 && !reflect.DeepEqual(images, saved.WalmartImageObservations) {
		return record.ErrNotReady
	}
	draft, e := marketplace.BuildDraft(*saved.Input.Walmart, rules, marketplace.SKUNamespace(binding.SupplierIdentityHash, binding.Walmart.GrantID), p.Identity.ProductKey, record.WalmartSourceSKUKeys(p.Snapshot))
	if e != nil || collection.Digest(draft) != collection.Digest(*saved.Walmart) {
		return record.ErrConflict
	}
	return nil
}
func walmartUploadResult(saved record.TargetRecord, r submission.OfficialReceipt) UploadResult {
	return UploadResult{RecordID: saved.ID, PublishedRecordID: r.RecordID, AttemptID: r.ID, Status: submission.ExecutionSucceeded, CurrentRecordUploaded: saved.ID == r.RecordID, Walmart: r.Walmart, Message: "Walmart 已受理 Feed；逐 SKU 处理与发布状态待查询"}
}
func (s *UploadService) uploadWalmart(ctx context.Context, scope collection.Scope, key string, saved record.TargetRecord) (UploadResult, error) {
	if !record.PublicationReady(saved) || saved.Walmart == nil || s.dependencies.WalmartStores == nil {
		return UploadResult{}, record.ErrNotReady
	}
	if _, e := s.dependencies.WalmartStores.WalmartExecutionMerchant(ctx, scope, saved.Merchant.StoreID, &saved.Merchant); e != nil {
		return UploadResult{}, e
	}
	if intent, e := s.dependencies.Intents.ReadOfficialIntent(ctx, scope, key); e == nil {
		if !submission.WalmartIntentMatchesRecord(intent, saved) {
			return UploadResult{}, submission.ErrExecutionIntentConflict
		}
		a, e := s.dependencies.Kernel.ReadIntent(ctx, submission.ExecutionScope{OrganizationID: scope.OrganizationID}, key)
		if e == nil {
			if a.Status == submission.ExecutionSucceeded {
				r, e := s.dependencies.Receipts.ReadOfficial(ctx, scope, a.AttemptID)
				if e != nil || !submission.WalmartReceiptMatchesIntent(r, intent) {
					return UploadResult{}, record.ErrConflict
				}
				return walmartUploadResult(saved, r), nil
			}
			_, _ = s.dependencies.Kernel.Expire(ctx, submission.ExecutionScope{OrganizationID: scope.OrganizationID}, a.AttemptID)
			return unknownUpload(saved.ID, a.AttemptID), nil
		}
		if !errors.Is(e, submission.ErrExecutionNotFound) {
			return UploadResult{}, e
		}
	} else if !errors.Is(e, submission.ErrExecutionNotFound) {
		return UploadResult{}, e
	}
	subject, e := submission.WalmartSubjectID(saved.Source.Source.ProductKey, saved.Merchant.SupplierIdentityHash)
	if e != nil {
		return UploadResult{}, e
	}
	target := submission.ExecutionTarget{Platform: "walmart", StoreID: saved.Merchant.StoreID, SubjectID: subject}
	if saved.Input.Walmart.RepairFrom == "" {
		prior, e := s.dependencies.Receipts.FindOfficialTarget(ctx, scope.OrganizationID, target)
		if e == nil {
			if prior.Walmart == nil || !submission.WalmartCompatibleBinding(prior.Binding, saved.Merchant) || prior.ProductKey != saved.Source.Source.ProductKey {
				return UploadResult{}, record.ErrConflict
			}
			return walmartUploadResult(saved, prior), nil
		}
		if errors.Is(e, submission.ErrExecutionTargetClaimed) {
			return unknownUpload(saved.ID, ""), nil
		}
		if !errors.Is(e, submission.ErrExecutionNotFound) {
			return UploadResult{}, e
		}
	}
	if e = s.walmartFacts(ctx, scope, saved); e != nil {
		return UploadResult{}, e
	}
	// Probe exact generated seller SKUs before acquiring a mutation claim. This
	// does not infer ownership from matching catalog content or titles.
	reader, e := s.dependencies.WalmartReadStores.WalmartMerchant(ctx, scope, saved.Merchant.StoreID, &saved.Merchant)
	if e != nil {
		return UploadResult{}, e
	}
	for _, sku := range saved.Walmart.SKUs {
		existing, e := reader.ProductBySKU(ctx, sku.SellerSKU)
		if e != nil {
			return UploadResult{}, e
		}
		if existing != nil {
			return UploadResult{}, record.ErrConflict
		}
	}
	source, e := s.source(ctx, scope, saved)
	if e != nil {
		return UploadResult{}, e
	}
	proof, e := submission.PrepareWalmartIntent(ctx, source, saved, key)
	if e != nil {
		return UploadResult{}, e
	}
	intent, e := s.dependencies.Intents.PrepareOfficial(ctx, proof)
	if e != nil {
		return UploadResult{}, e
	}
	if !submission.WalmartIntentMatchesRecord(intent, saved) {
		return UploadResult{}, submission.ErrExecutionIntentConflict
	}
	a, e := s.dependencies.Kernel.Acquire(ctx, submission.AcquireExecutionCommand{Scope: submission.ExecutionScope{OrganizationID: scope.OrganizationID}, IntentKey: key, Target: target, Action: submission.WalmartSubmitAction, Payload: saved.Walmart.Payload, ClaimOwnerID: "supply-worker", Lease: submission.ExecutionTimeout})
	if errors.Is(e, submission.ErrExecutionTargetClaimed) {
		return unknownUpload(saved.ID, ""), nil
	}
	if e != nil {
		return UploadResult{}, e
	}
	if a.Permit == nil {
		if a.Attempt.Status == submission.ExecutionSucceeded {
			r, e := s.dependencies.Receipts.ReadOfficial(ctx, scope, a.Attempt.AttemptID)
			if e != nil {
				return UploadResult{}, e
			}
			return walmartUploadResult(saved, r), nil
		}
		return unknownUpload(saved.ID, a.Attempt.AttemptID), nil
	}
	claim := uploadClaim(scope, a.Permit)
	ctx, cancel := context.WithDeadline(ctx, a.Permit.LeaseExpiresAt)
	defer cancel()
	uncertain := func() (UploadResult, error) { return s.unknown(ctx, saved.ID, claim), nil }
	if e = s.walmartFacts(ctx, scope, saved); e != nil {
		return uncertain()
	}
	merchant, e := s.dependencies.WalmartStores.WalmartExecutionMerchant(ctx, scope, saved.Merchant.StoreID, &intent.Binding)
	if e != nil {
		return uncertain()
	}
	accepted, e := merchant.SubmitFeed(ctx, *saved.Walmart, submission.WalmartCorrelationID(intent))
	if e != nil {
		return uncertain()
	}
	completion, e := submission.NewWalmartCompletion(scope, saved, intent, a.Attempt, claim, accepted)
	if e != nil {
		return uncertain()
	}
	retained, e := s.complete(ctx, scope, completion, a.Attempt.AttemptID)
	if e != nil {
		return uncertain()
	}
	return walmartUploadResult(saved, retained), nil
}
