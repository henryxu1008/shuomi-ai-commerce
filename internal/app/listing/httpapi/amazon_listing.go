package listinghttp

import (
	"context"
	"net/http"
	"time"

	listingapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/listing"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/preparation"
	record "github.com/henryxu1008/shuomi-ai-commerce/internal/listing/record/target"
	amazonmarket "github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/amazon"
)

type amazonReviewInput struct {
	SourceID         string                  `json:"sourceId"`
	StoreID          string                  `json:"storeId"`
	EffectiveVersion uint64                  `json:"effectiveVersion,string"`
	ExpectedRevision int64                   `json:"expectedRevision"`
	Details          amazonmarket.DraftInput `json:"details"`
}

type amazonReviewView struct {
	SourceID   string                  `json:"sourceId"`
	StoreID    string                  `json:"storeId"`
	RecordID   string                  `json:"recordId"`
	Revision   int64                   `json:"revision"`
	SKU        string                  `json:"sku"`
	WireHash   string                  `json:"wireHash"`
	ValidUntil time.Time               `json:"validUntil"`
	Details    amazonmarket.DraftInput `json:"details"`
}

func handleAmazonListing(ctx context.Context, app *listingapp.Application, action string, request *http.Request) (any, error) {
	if app.Amazon == nil || app.Targets == nil || app.Authorization == nil {
		return nil, preparation.ErrUnavailable
	}
	switch action {
	case "amazon-review":
		var input amazonReviewInput
		key, err := listingBodyLimit(request, &input, true, 16<<10)
		if err != nil {
			return nil, err
		}
		saved, err := app.Targets.Create(ctx, key, record.TargetInput{
			SourceID: input.SourceID, StoreID: input.StoreID, EffectiveVersion: input.EffectiveVersion,
			ExpectedRevision: input.ExpectedRevision, Amazon: &input.Details,
		})
		if err != nil {
			return nil, err
		}
		if saved.Record.Amazon == nil {
			return nil, record.ErrConflict
		}
		return projectAmazonReview(saved.Record), nil
	case "amazon-confirm":
		var input struct {
			RecordID string `json:"recordId"`
			WireHash string `json:"wireHash"`
		}
		key, err := listingBodyLimit(request, &input, true, 1024)
		if err != nil {
			return nil, err
		}
		scope, err := app.Authorization.Authorize(ctx, preparation.PermissionSubmit)
		if err != nil {
			return nil, err
		}
		return app.Amazon.Confirm(ctx, scope, key, input.RecordID, input.WireHash)
	default:
		var input struct {
			SourceID string `json:"sourceId"`
			StoreID  string `json:"storeId"`
		}
		if _, err := listingBodyLimit(request, &input, false, 1024); err != nil {
			return nil, err
		}
		scope, err := app.Authorization.Authorize(ctx, preparation.PermissionRead)
		if err != nil {
			return nil, err
		}
		if action == "amazon-details" {
			saved, err := app.ReadTarget(ctx, input.SourceID, input.StoreID)
			if err != nil {
				return nil, err
			}
			if saved.Amazon == nil || saved.Input.Amazon == nil {
				return nil, record.ErrNotFound
			}
			return projectAmazonReview(saved), nil
		}
		return app.Amazon.Read(ctx, scope, input.SourceID, input.StoreID, action == "amazon-check")
	}
}

func projectAmazonReview(saved record.TargetRecord) amazonReviewView {
	return amazonReviewView{SourceID: saved.Source.ID, StoreID: saved.Merchant.StoreID, RecordID: saved.ID, Revision: saved.Revision,
		SKU: saved.Amazon.Draft.SKU, WireHash: saved.Amazon.Draft.WireHash, ValidUntil: saved.Amazon.ValidUntil, Details: *saved.Input.Amazon}
}
