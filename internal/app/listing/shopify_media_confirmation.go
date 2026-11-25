package listingapp

import (
	"context"

	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/preparation"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

type ShopifyMediaConfirmationRequest struct {
	ShopifyMediaPreviewRequest
	ConfirmationHash string `json:"confirmationHash"`
}

type ShopifyMediaConfirmationResult struct {
	OperationID string                        `json:"operationId"`
	Status      submission.ShopifyMediaStatus `json:"status"`
}

// ShopifyMediaConfirmation can be composed only for an explicit fixture.
// BuildPlan reads authorized deep evidence rather than the display projection.
type ShopifyMediaConfirmation struct {
	Organization             string
	SourceID, StoreID        string
	Domain, Product, ShopKey string
	Generation               int64
	Sync                     submission.ShopifyMediaSync
	PairedInstallation       *storecenter.ShopifyBinding
	BuildPlan                func(context.Context, submission.ShopifyMediaOperation, preparation.SourceItem) (submission.ShopifyMediaPlan, error)
}

func (a *Application) ShopifyMediaConfirmationEnabled() bool {
	if a == nil || a.ShopifyMediaConfirm == nil || a.Sources == nil || a.Authorization == nil {
		return false
	}
	trial := a.ShopifyMediaConfirm
	if trial.Sync.Journal == nil || !collection.ValidID(trial.SourceID) || !collection.ValidID(trial.StoreID) ||
		!trial.scopeMatches(ShopifyMediaPreviewRequest{SourceID: trial.SourceID, StoreID: trial.StoreID, Product: trial.Product}) {
		return false
	}
	if trial.PairedInstallation == nil {
		return a.ShopifyMedia == nil && trial.Sync.Provider != nil && trial.BuildPlan != nil
	}
	binding := trial.PairedInstallation
	return a.ShopifyMedia != nil && trial.Sync.Provider == nil && trial.BuildPlan == nil &&
		binding.OrganizationID == trial.Organization && binding.StoreID == trial.StoreID &&
		binding.ShopKey == trial.ShopKey && binding.Generation == trial.Generation &&
		binding.Domain == trial.Domain && binding.AppID != "" && binding.ShopID != "" && binding.InstallationID != ""
}

func (trial *ShopifyMediaConfirmation) scopeMatches(input ShopifyMediaPreviewRequest) bool {
	product, err := wire.ParseDraftMediaProduct(input.Product, trial.Domain)
	return err == nil && product == "gid://shopify/Product/"+trial.Product &&
		input.SourceID == trial.SourceID && input.StoreID == trial.StoreID
}

func (trial *ShopifyMediaConfirmation) operation(input ShopifyMediaPreviewRequest, id string, scope collection.Scope, item preparation.SourceItem) (submission.ShopifyMediaOperation, error) {
	if !trial.scopeMatches(input) || scope.OrganizationID != trial.Organization ||
		item.ID != trial.SourceID || item.Source.Kind != "own" {
		return submission.ShopifyMediaOperation{}, preparation.ErrForbidden
	}
	operation := submission.ShopifyMediaOperation{ID: id, Owner: scope, ShopKey: trial.ShopKey,
		Generation: trial.Generation, SourceKey: item.Source.ProductKey, ProductID: "gid://shopify/Product/" + trial.Product}
	if !operation.Valid() {
		return submission.ShopifyMediaOperation{}, preparation.ErrInvalid
	}
	return operation, nil
}

func (a *Application) ConfirmShopifyMedia(ctx context.Context, id string, input ShopifyMediaConfirmationRequest) (ShopifyMediaConfirmationResult, error) {
	if !a.ShopifyMediaConfirmationEnabled() {
		return ShopifyMediaConfirmationResult{}, preparation.ErrUnavailable
	}
	trial := a.ShopifyMediaConfirm
	if !trial.scopeMatches(input.ShopifyMediaPreviewRequest) {
		return ShopifyMediaConfirmationResult{}, preparation.ErrForbidden
	}
	submitter, err := a.Authorization.Authorize(ctx, preparation.PermissionSubmit)
	if err != nil {
		return ShopifyMediaConfirmationResult{}, preparation.ErrForbidden
	}
	manager, err := a.Authorization.Authorize(ctx, preparation.PermissionManage)
	if err != nil || manager != submitter {
		return ShopifyMediaConfirmationResult{}, preparation.ErrForbidden
	}
	selected, err := a.Sources.Select(ctx, input.SourceID)
	if err != nil {
		return ShopifyMediaConfirmationResult{}, err
	}
	scope, item, original, err := selected.Read(ctx)
	if err != nil || scope != submitter {
		return ShopifyMediaConfirmationResult{}, preparation.ErrForbidden
	}
	operation, err := trial.operation(input.ShopifyMediaPreviewRequest, id, scope, item)
	if err != nil {
		return ShopifyMediaConfirmationResult{}, err
	}
	sync := trial.Sync
	if trial.PairedInstallation != nil {
		sync.Provider = a.newShopifyMediaTrialProvider(operation, *trial.PairedInstallation, input.SourceID)
	}
	status, err := sync.ConfirmOperation(ctx, operation, input.ConfirmationHash, func(ctx context.Context) (submission.ShopifyMediaPlan, error) {
		if trial.PairedInstallation != nil {
			_, plan, err := a.readShopifyMediaTrial(ctx, input.ShopifyMediaPreviewRequest, operation, item, original)
			if err == nil && !plan.Valid() {
				err = preparation.ErrConflict
			}
			return plan, err
		}
		plan, err := trial.BuildPlan(ctx, operation, item)
		if err == nil && plan.SourceVersion != item.Source.Version {
			err = preparation.ErrConflict
		}
		return plan, err
	})
	if err != nil {
		return ShopifyMediaConfirmationResult{}, err
	}
	return ShopifyMediaConfirmationResult{OperationID: id, Status: status}, nil
}
