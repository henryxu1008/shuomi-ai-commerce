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
)

// OperationItemView preserves the original execution history and separately
// projects confirmed provider facts retained by the Submission owner.
type OperationItemView struct {
	preparation.OperationItem
	ConfirmedProduct     *model.PublishResult            `json:"confirmedProduct,omitempty"`
	ConfirmedTemuProduct *temugoods.CreatedGoods         `json:"confirmedTemuProduct,omitempty"`
	ConfirmedTikTok      *submission.TikTokEffectReceipt `json:"confirmedTikTok,omitempty"`
	ConfirmedWalmart     *submission.WalmartFeedReceipt  `json:"confirmedWalmart,omitempty"`
}

func (a *Application) operationItemView(ctx context.Context, operation preparation.Operation, item preparation.OperationItem) (OperationItemView, error) {
	view := OperationItemView{OperationItem: item}
	if operation.Input.Action != preparation.OperationUpload || item.Status != preparation.ItemUnknown || item.RecordID == "" || item.ResultReference == "" {
		return view, nil
	}
	if a.PublicationReceipts == nil || a.Records == nil {
		return view, record.ErrUnavailable
	}
	receipt, err := a.PublicationReceipts.ReadOfficial(ctx, operation.Owner, item.ResultReference)
	if errors.Is(err, submission.ErrExecutionNotFound) {
		return view, nil
	}
	if err != nil {
		return view, err
	}
	if submission.ValidateOfficialReceipt(receipt) != nil || receipt.Owner != operation.Owner || receipt.ID != item.ResultReference || receipt.Kind != "publish" && receipt.Target.Platform != "tiktokshop" || receipt.RecordID != item.RecordID || receipt.Target.StoreID != operation.Input.StoreID {
		return view, record.ErrConflict
	}
	saved, err := a.Records.ReadTargetRecord(ctx, operation.Owner, item.RecordID)
	if err != nil {
		return view, err
	}
	if saved.ID != item.RecordID || saved.Revision != item.RecordRevision || saved.Source.ID != item.SourceID || saved.Source.Source.ProductKey != receipt.ProductKey || saved.Merchant != receipt.Binding {
		return view, record.ErrConflict
	}
	if receipt.Walmart != nil {
		view.ConfirmedWalmart = receipt.Walmart
	} else if receipt.TikTok != nil {
		view.ConfirmedTikTok = receipt.TikTok
	} else if receipt.TemuProduct != nil {
		view.ConfirmedTemuProduct = receipt.TemuProduct
	} else {
		view.ConfirmedProduct = &receipt.Product
	}
	return view, nil
}

func (a *Application) OperationItems(ctx context.Context, id string, q collection.Query) (collection.Page[OperationItemView], error) {
	result := collection.Page[OperationItemView]{Items: []OperationItemView{}}
	ctx, cancel := context.WithTimeout(ctx, preparation.Timeout)
	defer cancel()
	operation, err := a.Operations.Read(ctx, id)
	if err != nil {
		return result, err
	}
	page, err := a.Operations.ListItems(ctx, id, q)
	if err != nil {
		return result, err
	}
	confirmed := false
	for _, item := range page.Items {
		value, err := a.operationItemView(ctx, operation, item)
		if err != nil {
			return result, err
		}
		result.Items = append(result.Items, value)
		confirmed = confirmed || value.ConfirmedProduct != nil || value.ConfirmedTemuProduct != nil || value.ConfirmedTikTok != nil || value.ConfirmedWalmart != nil
	}
	if confirmed {
		if a.TargetMerchants == nil {
			return result, record.ErrUnavailable
		}
		binding, err := a.TargetMerchants.ReadTargetMerchant(ctx, operation.Owner, operation.Input.StoreID)
		if err != nil {
			return result, record.ErrForbidden
		}
		if !validTargetMerchant(operation.Owner, operation.Input.StoreID, binding) {
			return result, record.ErrConflict
		}
	}
	scope, err := a.Authorization.Authorize(ctx, preparation.PermissionRead)
	if err != nil || scope != operation.Owner {
		return result, preparation.ErrForbidden
	}
	result.Total, result.NextCursor = page.Total, page.NextCursor
	return result, nil
}
