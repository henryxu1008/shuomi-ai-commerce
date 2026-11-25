package listingapp

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/preparation"
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

type shopifyUploadFixtureRepository struct {
	tiktokUploadKernelRepository
	rejectSend bool
}

func (r *shopifyUploadFixtureRepository) VerifyShopifySend(ctx context.Context, claim submission.ExecutionClaim) error {
	value, err := r.Get(ctx, claim.Scope, claim.AttemptID)
	if err != nil {
		return err
	}
	if r.rejectSend || value.Status != submission.ExecutionClaimed || value.FenceEpoch != claim.FenceEpoch || value.ClaimOwnerID != claim.OwnerID {
		return submission.ErrExecutionClaimRejected
	}
	return nil
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
			repository := &shopifyUploadFixtureRepository{tiktokUploadKernelRepository: tiktokUploadKernelRepository{kernel: ledger}}
			kernel, err := submission.NewExecutionKernel(repository)
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
			repository.rejectSend = true
			refused, err := uploader.Upload(ctx, scope, uuid.NewString(), saved.ID)
			require.NoError(t, err)
			require.Equal(t, submission.ExecutionNotSent, refused.Status)
			require.Zero(t, store.writes, "a legacy in-memory permit cannot override the Product send check")
			repository.rejectSend = false
			if lost {
				store.send = errors.New("response lost")
			}
			intentKey := uuid.NewString()
			result, err := uploader.Upload(ctx, scope, intentKey, saved.ID)
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
			if lost {
				operation := preparation.Operation{Owner: scope, Input: preparation.OperationInput{
					Action: preparation.OperationUpload, StoreID: saved.Merchant.StoreID,
				}}
				item := preparation.OperationItem{SourceID: saved.Source.ID, RecordID: saved.ID,
					RecordRevision: saved.Revision, Status: preparation.ItemUnknown, ResultReference: result.AttemptID}
				merchants := &rulesFixture{binding: saved.Merchant}
				app := Application{Records: records, PublicationReceipts: receipts, TargetMerchants: merchants}
				before, err := app.operationItemView(ctx, operation, item)
				require.NoError(t, err)
				require.Equal(t, item, before.OperationItem)
				observation := shopify.ProductObservation{ID: "gid://shopify/Product/57", Status: "DRAFT", Media: []string{"PROCESSING"}}
				proof, err := submission.NewShopifyDraftResolution(receipts.intents[intentKey], *saved, ledger.attempts[intentKey], observation)
				require.NoError(t, err)
				_, err = receipts.ResolveOfficial(ctx, proof)
				require.NoError(t, err)
				for range 2 {
					view, err := app.operationItemView(ctx, operation, item)
					require.NoError(t, err)
					require.Equal(t, item, view.OperationItem)
					require.Nil(t, view.ConfirmedProduct)
					require.Equal(t, &observation, view.ConfirmedShopify)
				}
				merchants.binding.Shopify.ShopKey = uuid.NewString()
				view, err := app.operationItemView(ctx, operation, item)
				require.ErrorIs(t, err, record.ErrForbidden)
				require.Nil(t, view.ConfirmedShopify)
			}
			require.Equal(t, 1, store.writes)
			require.Zero(t, legacy.publishes)
		})
	}
}

type stagingDraftFixture struct {
	shopifyDraftFixture
	stageError    error
	accessRevoked bool
	calls         []string
	sent          shopify.Draft
}

func (f *stagingDraftFixture) VerifyDraftWrite(ctx context.Context, scope collection.Scope, expected storecenter.ProductMerchantBinding) (ShopifyDraftWriter, error) {
	f.calls = append(f.calls, "verify")
	_, err := f.shopifyDraftFixture.VerifyDraftWrite(ctx, scope, expected)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (f *stagingDraftFixture) StageImages(_ context.Context, images [][]byte) ([]string, error) {
	f.calls = append(f.calls, "stage")
	if len(images) != 1 || string(images[0]) != "approved bytes" {
		return nil, errors.New("wrong staged content")
	}
	if f.accessRevoked {
		f.preflight = errors.New("store authorization changed while staging")
	}
	return []string{"https://shopify-staged-uploads.storage.googleapis.com/tmp/123/products/abc/image.png"}, f.stageError
}

func (f *stagingDraftFixture) Create(ctx context.Context, draft shopify.Draft) (shopify.ProductObservation, error) {
	f.calls = append(f.calls, "create")
	f.sent = draft
	return f.shopifyDraftFixture.Create(ctx, draft)
}

func TestShopifyUploadRechecksAccessAndPreservesFrozenGalleryAfterStaging(t *testing.T) {
	ctx := context.Background()
	uploader, scope, records, _, ledger, receipts, _ := uploadFixture(t)
	kernel, err := submission.NewExecutionKernel(&shopifyUploadFixtureRepository{tiktokUploadKernelRepository: tiktokUploadKernelRepository{kernel: ledger}})
	require.NoError(t, err)
	uploader.dependencies.Kernel = shopifyFixtureKernel{ledger, kernel}
	saved := &records.saved
	saved.Merchant.Platform, saved.Merchant.Site, saved.Merchant.ApplicationType = storecenter.PlatformShopify, "shopify", ""
	saved.Merchant.ApplicationID, saved.Merchant.ApplicationRevision = "gid://shopify/App/32", "shopify-admin-2026-10"
	saved.Merchant.Shopify = storecenter.ShopifyMerchantBinding{ShopKey: uuid.NewString(), Currency: "USD"}
	saved.Input.Draft, saved.Result, saved.ImageObservations = goods.OfficialDraftInput{}, goods.OfficialDraft{}, nil
	const generated = "https://localhost:36522/approved.png"
	const public = "https://example.com/gallery.jpg"
	saved.Input.Shopify = &shopify.Input{Currency: "USD", Prices: []string{"12.50"}, ImageIDs: []string{"public", "generated"}, VariantImageIDs: []string{"generated"}}
	inventory := uploader.dependencies.Assets.(uploadAssets).inventory
	inventory.Scope.TargetPlatform = "product"
	inventory.Assets = []asset.ApprovedAsset{
		{ID: "generated", URL: generated, Role: asset.RoleMain, GenerationEvidence: &asset.GenerationEvidence{}},
		{ID: "public", URL: public, Role: asset.RoleGallery},
	}
	uploader.dependencies.Assets = uploadAssets{inventory}
	store := &stagingDraftFixture{shopifyDraftFixture: shopifyDraftFixture{binding: saved.Merchant}, stageError: errors.New("temporary stage failed")}
	var revokeWhileReading bool
	uploader.dependencies.ShopifyGeneratedImages = func(_ context.Context, selection asset.SourceSelection, selected asset.ApprovedAsset, limit int64) ([]byte, error) {
		store.calls = append(store.calls, "read")
		require.Equal(t, scope.OrganizationID, selection.TenantID)
		require.Equal(t, inventory.Scope.ProductKey, selection.ProductKey)
		require.Equal(t, inventory.Scope.SourceSnapshotVersion, selection.EffectiveCatalogVersion)
		require.Equal(t, inventory.Assets[0], selected)
		require.Positive(t, limit)
		if revokeWhileReading {
			store.preflight = errors.New("store authorization changed while reading approved images")
		}
		return []byte("approved bytes"), nil
	}
	draft, err := shopify.BuildDraft(*saved.Input.Shopify, catalog.ProductSnapshot{Title: "Original"}, inventory, "USD")
	require.NoError(t, err)
	saved.Shopify, saved.InventoryHash, saved.RulesHash = &draft, collection.Digest(inventory), collection.Digest(saved.Merchant.Shopify)
	uploader.dependencies.ShopifyStores = store

	failed, err := uploader.Upload(ctx, scope, uuid.NewString(), saved.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionNotSent, failed.Status)
	require.Equal(t, []string{"read", "verify", "stage"}, store.calls)
	require.Zero(t, store.writes)

	store.calls, store.stageError, store.accessRevoked = nil, nil, true
	revoked, err := uploader.Upload(ctx, scope, uuid.NewString(), saved.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionNotSent, revoked.Status)
	require.Equal(t, []string{"read", "verify", "stage", "verify"}, store.calls)
	require.Zero(t, store.writes)

	store.calls, store.preflight, store.accessRevoked, revokeWhileReading = nil, nil, false, true
	revoked, err = uploader.Upload(ctx, scope, uuid.NewString(), saved.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionNotSent, revoked.Status)
	require.Equal(t, []string{"read", "verify"}, store.calls)
	require.Zero(t, store.writes)

	store.calls, store.preflight, revokeWhileReading = nil, nil, false
	result, err := uploader.Upload(ctx, scope, uuid.NewString(), saved.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionSucceeded, result.Status)
	require.Equal(t, []string{"read", "verify", "stage", "verify", "create"}, store.calls)
	require.Len(t, receipts.receipts, 1)
	require.Equal(t, public, store.sent.Images[0].URL)
	require.Contains(t, store.sent.Images[1].URL, "shopify-staged-uploads.storage.googleapis.com")
	require.Equal(t, 2, store.sent.Variants[0].ImageIndex)
	require.Equal(t, generated, saved.Shopify.Images[1].URL)

	replay, err := uploader.Upload(ctx, scope, uuid.NewString(), saved.ID)
	require.NoError(t, err)
	require.Equal(t, result, replay)
	require.Equal(t, 1, store.writes)
}
