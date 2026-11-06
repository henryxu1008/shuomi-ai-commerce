package listingapp

import (
	"context"
	"errors"
	"fmt"

	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	tiktok "task-processor/internal/marketplace/tiktokshop"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/review"
	"task-processor/internal/storecenter"
)

// Projection/authorization adapter; sequencing and durable commands remain in
// preparation, while this adapter reads facts from the current owners.
type UploadPlanInputs struct {
	Application     *Application
	Operations      preparation.UploadPlanOperationReader
	WalmartAttempts interface {
		ReadWalmartAttempt(context.Context, submission.ExecutionScope, submission.ExecutionTarget) (submission.ExecutionAttempt, error)
	}
}

func (r UploadPlanInputs) ReadUploadPlanBinding(ctx context.Context, scope collection.Scope, in preparation.UploadPlanInput) (string, error) {
	a := r.Application
	if a == nil || a.TargetMerchants == nil {
		return "", preparation.ErrUnavailable
	}
	b, err := a.TargetMerchants.ReadTargetMerchant(ctx, scope, in.StoreID)
	if err != nil {
		return "", err
	}
	if !validTargetMerchant(scope, in.StoreID, b) {
		return "", record.ErrForbidden
	}
	if in.PublicationMode == "platform_draft" && b.Platform != storecenter.PlatformTikTokShop {
		return "", record.ErrNotReady
	}
	return collection.Digest(b), nil
}

func (r UploadPlanInputs) AuthorizeUploadPlanStep(ctx context.Context, _ preparation.UploadPlan, in preparation.OperationInput) error {
	if r.Application == nil {
		return preparation.ErrUnavailable
	}
	if in.Action == preparation.OperationOptimize {
		if r.Application.AuthorizeOptimization == nil {
			return record.ErrNotReady
		}
		return r.Application.AuthorizeOptimization(ctx, in)
	}
	return nil
}

func (r UploadPlanInputs) ReadUploadPlanFacts(ctx context.Context, plan preparation.UploadPlan, item preparation.UploadPlanItem) (preparation.UploadPlanFacts, error) {
	facts := preparation.UploadPlanFacts{}
	a := r.Application
	if a == nil || a.Sources == nil || a.Records == nil || r.Operations == nil {
		return facts, preparation.ErrUnavailable
	}
	selected, err := a.Sources.Select(ctx, item.SourceID)
	if err != nil {
		return facts, err
	}
	scope, source, _, err := selected.Read(ctx)
	if err != nil {
		return facts, err
	}
	if scope != plan.Owner || source.PreparationID != plan.Input.PreparationID || source.Source.Version != item.SourceVersion {
		return facts, record.ErrConflict
	}
	if collection.IsResearchFixtureProduct(source.Source.ProductKey) {
		return facts, preparation.ErrForbidden
	}
	saved, err := a.Records.ReadTargetHead(ctx, scope, record.TargetIdentity(scope, item.SourceID, plan.Input.StoreID))
	if err != nil && !errors.Is(err, record.ErrNotFound) {
		return facts, err
	}
	if err == nil {
		if saved.Source.ID != source.ID || saved.Source.PreparationID != source.PreparationID || saved.Source.Source != source.Source || saved.Input.StoreID != plan.Input.StoreID || saved.TargetID != record.TargetIdentity(scope, source.ID, plan.Input.StoreID) {
			return facts, record.ErrConflict
		}
		facts.PublicationSummary = uploadPlanPublicationSummary(saved)
		facts.HasTarget = true
		facts.Ready = record.PublicationReady(saved)
		facts.Target = preparation.TargetApproval{SourceID: source.ID, RecordID: saved.ID, Revision: saved.Revision}
		facts.PublicationMode = "platform"
		if saved.Input.TikTok != nil && saved.Input.TikTok.SaveMode == "AS_DRAFT" {
			facts.PublicationMode = "platform_draft"
		}
		if saved.Walmart != nil {
			facts.WalmartApproval = &preparation.WalmartUploadApproval{SourceID: source.ID, RecordID: saved.ID, Revision: saved.Revision, WireHash: saved.Walmart.WireHash}
		}
	}
	if !facts.HasTarget {
		if a.TargetMerchants == nil {
			return facts, preparation.ErrUnavailable
		}
		binding, e := a.TargetMerchants.ReadTargetMerchant(ctx, scope, plan.Input.StoreID)
		if e != nil {
			return facts, e
		}
		if !validTargetMerchant(scope, plan.Input.StoreID, binding) {
			return facts, record.ErrForbidden
		}
		facts.ManualTargetRequired = binding.Platform == storecenter.PlatformTikTokShop || binding.Platform == storecenter.PlatformWalmart
	}
	publication, e := a.Publication(ctx, source.ID, plan.Input.StoreID)
	if e != nil {
		return facts, e
	}
	facts.ExistingPublicationRecordID = publication.RecordID
	if publication.RecordID != "" {
		if publication.TikTok != nil {
			facts.PublicationOutcome = uploadPlanTikTokOutcome(publication.TikTok)
		} else {
			facts.PublicationOutcome = "accepted"
		}
	}
	if facts.Ready && facts.WalmartApproval != nil && saved.Input.Walmart.RepairFrom != "" && publication.Walmart != nil && publication.RecordID != "" && submission.WalmartCompatibleBinding(saved.Merchant, publication.binding) {
		if observation := walmartRepairObservation(saved, publication); observation != nil {
			facts.WalmartRepairObservationID = observation.ID
			facts.PublicationSummary = append(facts.PublicationSummary, fmt.Sprintf("修复原 Walmart Feed %s · %d 个已选失败 SKU", observation.Observation.FeedID, len(saved.Walmart.SKUs)))
		} else if len(plan.Input.SourceIDs) == 1 && plan.Input.SourceIDs[0] == item.SourceID && item.UploadOperationID == "" {
			id, e := r.walmartRejectedRepair(ctx, selected, scope, saved)
			if e != nil {
				return facts, e
			}
			if id != "" {
				facts.WalmartRejectedAttemptID = id
				facts.PublicationSummary = append(facts.PublicationSummary, "Walmart 明确拒绝 · 原尝试 "+id+" · 同载荷重试")
			}
		}
	}

	for _, ref := range []struct{ id, action string }{{item.AdaptOperationID, preparation.OperationAdapt}, {item.OptimizeOperationID, preparation.OperationOptimize}, {item.UploadOperationID, preparation.OperationUpload}} {
		if ref.id == "" {
			continue
		}
		op, original, e := r.Operations.ReadUploadPlanOperation(ctx, scope, ref.id, source.ID)
		if e != nil {
			return facts, e
		}
		if op.Owner != scope || op.Input.PlanID != plan.ID || op.Input.PreparationID != plan.Input.PreparationID || op.Input.StoreID != plan.Input.StoreID || op.Input.Action != ref.action || original.SourceID != source.ID {
			return facts, record.ErrConflict
		}
		status := original.Status
		if op.Status == preparation.OperationCancelled && (status == preparation.ItemPending || status == preparation.ItemRunning) {
			status = preparation.ItemCancelled
		}
		switch ref.action {
		case preparation.OperationAdapt:
			facts.AdaptStatus = status
		case preparation.OperationUpload:
			facts.UploadStatus = status
			view, e := a.operationItemView(ctx, op, original)
			if e != nil {
				return facts, e
			}
			facts.UploadConfirmed = view.ConfirmedProduct != nil || view.ConfirmedTemuProduct != nil || view.ConfirmedWalmart != nil
			if view.ConfirmedTikTok != nil {
				facts.UploadResultOutcome = uploadPlanTikTokOutcome(view.ConfirmedTikTok)
				facts.UploadConfirmed = facts.UploadResultOutcome == "accepted"
			}
		case preparation.OperationOptimize:
			facts.OptimizationStatus = status
			if status != preparation.ItemReview {
				continue
			}
			if a.StageProjection.Reviews == nil {
				return facts, record.ErrUnavailable
			}
			originalTarget, e := a.Records.ReadTargetRecord(ctx, scope, original.RecordID)
			if e != nil {
				return facts, e
			}
			if originalTarget.ID != original.RecordID || originalTarget.Revision != original.RecordRevision || originalTarget.Source.ID != source.ID || originalTarget.Input.StoreID != plan.Input.StoreID {
				return facts, record.ErrConflict
			}
			v, e := a.StageProjection.Reviews.Read(ctx, review.Scope{Org: scope.OrganizationID, Actor: scope.ActorID}, original.ResultReference)
			if e != nil {
				return facts, e
			}
			facts.ReviewRequired, facts.ReviewRejected, e = uploadPlanReviewState(scope, original.ResultReference, originalTarget, saved, v)
			if e != nil {
				return facts, e
			}
		}
	}
	if facts.PublicationOutcome == "failed" || facts.UploadResultOutcome == "failed" {
		facts.PublicationSummary = append(facts.PublicationSummary, "TikTok 美国站同步未成功，全球商品及原业务失败回执已保留；不会自动重发")
	} else if facts.PublicationOutcome == "asset_retained" || facts.UploadResultOutcome == "asset_retained" {
		facts.PublicationSummary = append(facts.PublicationSummary, "仅远端资产已保留，原刊登结果未确认；沿原操作核实，不重复创建")
	}
	return facts, nil
}

// Use the existing result owner's projection for preview only. Submission
// still validates current evidence, SKU ownership and the approved draft at Acquire.
func walmartRepairObservation(saved record.TargetRecord, publication PublicationView) *submission.WalmartResultRecord {
	repairFrom := saved.Input.Walmart.RepairFrom
	for index := range publication.WalmartResults {
		result := &publication.WalmartResults[index]
		if result.ReceiptID != repairFrom || result.Limited || result.Observation.Partial {
			continue
		}
		eligible := make(map[string]bool, len(result.RepairableSKUs))
		for _, sku := range result.RepairableSKUs {
			eligible[sku] = true
		}
		for _, sku := range saved.Walmart.SKUs {
			if !eligible[sku.SellerSKU] {
				return nil
			}
		}
		return result
	}
	return nil
}

// The preview is only a read of the current rejection. Continue binds it to
// the preview hash; the original Submission acquire still owns send admission.
func (r UploadPlanInputs) walmartRejectedRepair(ctx context.Context, selected preparation.AuthorizedSource, scope collection.Scope, saved record.TargetRecord) (string, error) {
	if r.WalmartAttempts == nil || r.Application.Uploader == nil {
		return "", record.ErrUnavailable
	}
	subject, err := submission.WalmartSubjectID(saved.Source.Source.ProductKey, saved.Merchant.SupplierIdentityHash)
	if err != nil {
		return "", err
	}
	target := submission.ExecutionTarget{Platform: "walmart", StoreID: saved.Merchant.StoreID, SubjectID: subject}
	attempt, err := r.WalmartAttempts.ReadWalmartAttempt(ctx, submission.ExecutionScope{OrganizationID: scope.OrganizationID}, target)
	if errors.Is(err, submission.ErrExecutionNotFound) {
		return "", nil
	}
	if err != nil || attempt.Status != submission.ExecutionFailedDefinitive {
		return "", err
	}
	rejected, err := r.Application.Uploader.dependencies.Intents.ReadOfficialIntent(ctx, scope, attempt.IntentKey)
	if errors.Is(err, submission.ErrExecutionNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	// Use the old key solely to reconstruct the comparison in memory.
	proof, err := submission.PrepareWalmartIntent(ctx, selected, saved, rejected.Key)
	if err != nil {
		return "", err
	}
	candidate, err := proof.Read(ctx)
	if err != nil || !submission.WalmartRejectedRepairMatches(candidate, rejected, attempt) {
		return "", err
	}
	return attempt.AttemptID, nil
}

func uploadPlanReviewState(scope collection.Scope, id string, original, saved record.TargetRecord, v review.Record) (bool, bool, error) {
	if v.ID != id || v.Org != scope.OrganizationID || v.Owner != scope.ActorID || v.Input.ProductKey != original.Source.Source.ProductKey || v.Input.BaseVersion != original.EffectiveVersion {
		return true, false, record.ErrConflict
	}
	if v.State == "rejected" {
		return true, true, nil
	}
	if v.State != "applied" || v.Receipt == nil || v.Receipt.ProposalID != id || v.Receipt.Actor != scope.ActorID || v.Receipt.ProductVersion != saved.EffectiveVersion || saved.ApplyReceiptID != id {
		return true, false, nil
	}
	return false, false, nil
}

var _ preparation.UploadPlanInputs = UploadPlanInputs{}

func uploadPlanPublicationSummary(saved record.TargetRecord) []string {
	if saved.Input.TikTok != nil {
		return []string{fmt.Sprintf("TikTok %s · %s", saved.Input.TikTok.Selection.Method, saved.Input.TikTok.SaveMode)}
	}
	if saved.Temu != nil {
		return []string{"TEMU · " + saved.Temu.Mode}
	}
	if saved.Walmart != nil {
		return []string{fmt.Sprintf("Walmart · %s · %d SKU · %s", saved.Walmart.Selection.FeedType, len(saved.Walmart.SKUs), saved.Walmart.WireHash)}
	}
	out := []string{}
	for i, skc := range saved.Result.Product.SKCs {
		plan := "原上架方式未填写"
		if skc.ShelfWay == "2" {
			plan = "预约上架（北京时间） " + skc.HopeOnSaleDate
		} else if skc.ShelfWay == "1" {
			plan = "审核后自动上架"
		}
		out = append(out, fmt.Sprintf("SHEIN 款式 %d · %s · shelf_require %s", i+1, plan, skc.ShelfRequire))
	}
	return out
}

// A retained image/global asset proves only that stage. A failed US publication
// is a durable business outcome, never a successful channel submission.
func uploadPlanTikTokOutcome(v *submission.TikTokEffectReceipt) string {
	if v == nil {
		return ""
	}
	if v.Publication != nil {
		switch v.Publication.State {
		case tiktok.PublicationFailed:
			return "failed"
		case tiktok.SubmittedForReview, tiktok.RemoteDraft:
			return "accepted"
		}
	}
	if v.Local != nil {
		return "accepted"
	}
	if v.Global != nil || v.Uploaded != nil {
		return "asset_retained"
	}
	return ""
}
