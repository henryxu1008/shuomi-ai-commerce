package supplychainapp

import (
	"context"
	"errors"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	model "task-processor/internal/marketplace/shein/model"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"time"
)

// PublicationView retains confirmed creation facts. A separately timestamped
// TikTokStatus may contain a manual read; neither grants authority to send.
type PublicationView struct {
	binding      storecenter.ProductMerchantBinding
	SourceID     string                          `json:"sourceId"`
	StoreID      string                          `json:"storeId"`
	RecordID     string                          `json:"recordId,omitempty"`
	ReceiptID    string                          `json:"receiptId,omitempty"`
	Product      *model.PublishResult            `json:"product,omitempty"`
	TemuProduct  *temugoods.CreatedGoods         `json:"temuProduct,omitempty"`
	TikTok       *submission.TikTokEffectReceipt `json:"tiktok,omitempty"`
	TikTokStatus *TikTokStatusObservation        `json:"tiktokStatus,omitempty"`
	ObservedAt   *time.Time                      `json:"observedAt,omitempty"`
}

func (a *Application) Publication(ctx context.Context, sourceID, storeID string) (PublicationView, error) {
	value := PublicationView{SourceID: sourceID, StoreID: storeID}
	if ctx == nil || !collection.ValidID(sourceID) || !collection.ValidID(storeID) {
		return value, preparation.ErrInvalid
	}
	if a == nil || a.PublicationReceipts == nil || a.TargetMerchants == nil {
		return value, record.ErrUnavailable
	}
	scope, err := a.Authorization.Authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return value, err
	}
	proof, err := a.Sources.Select(ctx, sourceID)
	if err != nil {
		return value, err
	}
	owner, source, _, err := proof.Read(ctx)
	if err != nil || owner != scope {
		return value, preparation.ErrForbidden
	}
	binding, err := a.TargetMerchants.ReadTargetMerchant(ctx, scope, storeID)
	if err != nil {
		return value, record.ErrForbidden
	}
	if !validTargetMerchant(scope, storeID, binding) {
		return value, record.ErrConflict
	}
	subject, err := submission.ProductSubjectID(source.Source.ProductKey, binding.Site)
	platform := "shein"
	if binding.Platform == storecenter.PlatformTemu {
		platform = "temu"
		subject, err = submission.TemuProductSubjectID(source.Source.ProductKey, binding)
	}
	if binding.Platform == storecenter.PlatformTikTokShop {
		platform = "tiktokshop"
		subject, err = submission.TikTokProductSubjectID(source.Source.ProductKey, submission.TikTokPublishStage)
	}
	if err != nil {
		return value, err
	}
	receipt, err := a.PublicationReceipts.FindOfficialTarget(ctx, scope.OrganizationID, submission.ExecutionTarget{Platform: platform, StoreID: storeID, SubjectID: subject})
	if errors.Is(err, submission.ErrExecutionNotFound) || errors.Is(err, submission.ErrExecutionTargetClaimed) {
		return value, nil
	}
	if err != nil {
		return value, err
	}
	if submission.ValidateOfficialReceipt(receipt) != nil || receipt.Kind != "publish" || receipt.ProductKey != source.Source.ProductKey || receipt.Binding.SupplierIdentityHash != binding.SupplierIdentityHash || receipt.Binding.StoreID != storeID || receipt.Target.SubjectID != subject {
		return value, record.ErrConflict
	}
	if _, _, _, err = proof.Read(ctx); err != nil {
		return value, err
	}
	current, err := a.TargetMerchants.ReadTargetMerchant(ctx, scope, storeID)
	if err != nil || current != binding {
		return value, record.ErrForbidden
	}
	if receipt.Target.Platform == "tiktokshop" {
		if !submission.TikTokCompatibleBinding(receipt.Binding, binding) {
			return value, record.ErrConflict
		}
		value.TikTok = receipt.TikTok
	} else if receipt.Target.Platform == "temu" {
		value.TemuProduct = receipt.TemuProduct
	} else {
		value.Product = &receipt.Product
	}
	value.ReceiptID = receipt.ID
	value.binding = binding
	value.ObservedAt = &receipt.ObservedAt
	if receipt.Owner == scope {
		value.RecordID = receipt.RecordID
	}
	return value, nil
}
