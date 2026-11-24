package listingapp

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	record "github.com/henryxu1008/shuomi-ai-commerce/internal/listing/record/target"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/shein/goods"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	"github.com/stretchr/testify/require"
)

type shopifyFixtureKernel struct {
	*uploadKernel
	*submission.ExecutionKernel
}

func (k shopifyFixtureKernel) Acquire(ctx context.Context, c submission.AcquireExecutionCommand) (submission.ExecutionAcquisition, error) {
	return k.uploadKernel.Acquire(ctx, c)
}
func (k shopifyFixtureKernel) ReadIntent(ctx context.Context, scope submission.ExecutionScope, key string) (submission.ExecutionAttempt, error) {
	return k.uploadKernel.ReadIntent(ctx, scope, key)
}
func (k shopifyFixtureKernel) Complete(ctx context.Context, claim submission.ExecutionClaim, evidence submission.ExecutionEvidence) (submission.ExecutionAttempt, error) {
	return k.uploadKernel.Complete(ctx, claim, evidence)
}
func (k shopifyFixtureKernel) MarkUnknown(ctx context.Context, claim submission.ExecutionClaim, reason submission.UnknownReason) (submission.ExecutionAttempt, error) {
	return k.uploadKernel.MarkUnknown(ctx, claim, reason)
}
func (k shopifyFixtureKernel) Expire(ctx context.Context, scope submission.ExecutionScope, id string) (submission.ExecutionAttempt, error) {
	return k.uploadKernel.Expire(ctx, scope, id)
}

type shopifyDraftFixture struct {
	binding         storecenter.ProductMerchantBinding
	writes          int
	preflight, send error
}

func (f *shopifyDraftFixture) Binding(context.Context, collection.Scope, string, string) (storecenter.ProductMerchantBinding, error) {
	return f.binding, nil
}
func (f *shopifyDraftFixture) VerifyDraftWrite(_ context.Context, _ collection.Scope, expected storecenter.ProductMerchantBinding) (ShopifyDraftWriter, error) {
	if expected != f.binding {
		return nil, record.ErrConflict
	}
	return f, f.preflight
}
func (f *shopifyDraftFixture) Create(_ context.Context, draft shopify.Draft) (shopify.ProductObservation, error) {
	if draft.SourceKey == "" || draft.Currency != "USD" || draft.Variants[0].Price != "12.50" {
		return shopify.ProductObservation{}, errors.New("incorrect Shopify draft")
	}
	f.writes++
	return shopify.ProductObservation{ID: "gid://shopify/Product/57", Status: "DRAFT", Media: []string{"PROCESSING"}}, f.send
}

func TestShopifyUploadRetainsDraftAndDoesNotResendUnknown(t *testing.T) {
	for _, lost := range []bool{false, true} {
		t.Run(map[bool]string{false: "platform draft", true: "response lost"}[lost], func(t *testing.T) {
			ctx := context.Background()
			uploader, scope, records, legacy, ledger, receipts, _ := uploadFixture(t)
			kernel, err := submission.NewExecutionKernel(tiktokUploadKernelRepository{kernel: ledger})
			require.NoError(t, err)
			uploader.dependencies.Kernel = shopifyFixtureKernel{ledger, kernel}
			saved := &records.saved
			saved.Merchant.Platform, saved.Merchant.Site, saved.Merchant.ApplicationType = storecenter.PlatformShopify, "shopify", ""
			saved.Merchant.ApplicationID, saved.Merchant.ApplicationRevision = "gid://shopify/App/32", "shopify-admin-2026-10"
			saved.Merchant.Shopify = storecenter.ShopifyMerchantBinding{ShopKey: uuid.NewString(), Currency: "USD"}
			saved.Input.Draft, saved.Result, saved.ImageObservations = goods.OfficialDraftInput{}, goods.OfficialDraft{}, nil
			saved.Input.Shopify = &shopify.Input{Currency: "USD", Prices: []string{"12.50"}}
			inventory := uploader.dependencies.Assets.(uploadAssets).inventory
			inventory.Scope.TargetPlatform = "product"
			inventory.Assets = []asset.ApprovedAsset{}
			uploader.dependencies.Assets = uploadAssets{inventory}
			draft, err := shopify.BuildDraft(*saved.Input.Shopify, catalog.ProductSnapshot{Title: "Original"}, inventory, "USD")
			require.NoError(t, err)
			saved.Shopify, saved.InventoryHash, saved.RulesHash = &draft, collection.Digest(inventory), collection.Digest(saved.Merchant.Shopify)
			require.True(t, record.PublicationReady(*saved))

			store := &shopifyDraftFixture{binding: saved.Merchant, preflight: errors.New("authorization changed")}
			uploader.dependencies.ShopifyStores = store
			failed, err := uploader.Upload(ctx, scope, uuid.NewString(), saved.ID)
			require.NoError(t, err)
			require.Equal(t, submission.ExecutionNotSent, failed.Status)
			require.Zero(t, store.writes)
			store.preflight = nil
			if lost {
				store.send = errors.New("response lost")
			}
			result, err := uploader.Upload(ctx, scope, uuid.NewString(), saved.ID)
			require.NoError(t, err)
			if lost {
				require.Equal(t, submission.ExecutionOutcomeUnknown, result.Status)
				require.Empty(t, receipts.receipts)
			} else {
				require.Equal(t, submission.ExecutionSucceeded, result.Status)
				require.Equal(t, "gid://shopify/Product/57", result.Shopify.ID)
				require.Equal(t, "DRAFT", result.Shopify.Status)
			}
			again, err := uploader.Upload(ctx, scope, uuid.NewString(), saved.ID)
			require.NoError(t, err)
			require.Equal(t, result.Status, again.Status)
			require.Equal(t, 1, store.writes)
			require.Zero(t, legacy.publishes)
		})
	}
}
