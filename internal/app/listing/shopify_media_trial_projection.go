package listingapp

import (
	"context"
	"encoding/hex"
	"errors"
	"slices"
	"strings"
	"time"

	storeapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/storecenter"
	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/preparation"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

func (a *Application) canPreviewShopifyMediaTrial(ctx context.Context, input ShopifyMediaPreviewRequest, expected collection.Scope) bool {
	if !a.ShopifyMediaConfirmationEnabled() {
		return false
	}
	trial := a.ShopifyMediaConfirm
	if expected.OrganizationID != trial.Organization || !trial.scopeMatches(input) {
		return false
	}
	submitter, err := a.Authorization.Authorize(ctx, preparation.PermissionSubmit)
	return err == nil && submitter == expected
}

// The display and consent digest both come from this read. Confirmation uses
// the same projection only to create a new operation.
func (a *Application) readShopifyMediaTrial(ctx context.Context, input ShopifyMediaPreviewRequest, operation submission.ShopifyMediaOperation, item preparation.SourceItem, original catalog.PublishedSnapshot) (ShopifyMediaPreview, submission.ShopifyMediaPlan, error) {
	service := a.ShopifyMedia
	if service == nil || service.Store == nil || service.Inventory == nil || service.ReadImage == nil {
		return ShopifyMediaPreview{}, submission.ShopifyMediaPlan{}, preparation.ErrUnavailable
	}
	selected, err := a.readShopifyMediaSource(ctx, operation.Owner, item, original)
	if err != nil {
		return ShopifyMediaPreview{}, submission.ShopifyMediaPlan{}, err
	}
	subject := shopifyProductSubject(operation.Owner, storecenter.ProductPurposeStatus)
	binding, err := service.Store.ShopifyProductBinding(ctx, subject, input.StoreID)
	if err != nil || binding.Shopify.ShopKey != operation.ShopKey || binding.ConnectionRevision != operation.Generation {
		return ShopifyMediaPreview{}, submission.ShopifyMediaPlan{}, preparation.ErrForbidden
	}
	remote, adminURL, err := service.Store.ReadShopifyDraftMediaEvidence(ctx, subject, binding, operation.ProductID)
	if err != nil {
		return ShopifyMediaPreview{}, submission.ShopifyMediaPlan{}, shopifyMediaPreviewError(err)
	}
	if err := a.recheckShopifyMediaSource(ctx, selected); err != nil {
		return ShopifyMediaPreview{}, submission.ShopifyMediaPlan{}, err
	}
	return projectShopifyMediaEvidence(input, operation, selected, original.Snapshot.Variants, original.Version, remote, adminURL, time.Now().UTC().Add(5*time.Minute))
}

func shopifyMediaPreviewError(err error) error {
	switch {
	case errors.Is(err, wire.ErrInvalidRequest):
		return preparation.ErrInvalid
	case errors.Is(err, storeapp.ErrProductAccessChanged):
		return preparation.ErrForbidden
	case errors.Is(err, wire.ErrDraftUnverified):
		return preparation.ErrConflict
	default:
		return preparation.ErrUnavailable
	}
}

func projectShopifyMediaEvidence(input ShopifyMediaPreviewRequest, operation submission.ShopifyMediaOperation, selected shopifyMediaSourceEvidence, local []catalog.Variant, version uint64, remote wire.DraftMediaEvidence, adminURL string, expires time.Time) (ShopifyMediaPreview, submission.ShopifyMediaPlan, error) {
	var plan submission.ShopifyMediaPlan
	if remote.ID != operation.ProductID {
		return ShopifyMediaPreview{}, plan, preparation.ErrConflict
	}
	images, err := matchShopifyMediaPreviewImages(selected, remote.MatchOriginals(selected.proofs))
	if err != nil {
		return ShopifyMediaPreview{}, plan, err
	}
	variants, matched := previewShopifyVariants(local, remote.Variants, images, selected.source.Images)
	if !matched {
		return ShopifyMediaPreview{}, plan, preparation.ErrConflict
	}
	preview := ShopifyMediaPreview{SourceID: input.SourceID, StoreID: input.StoreID, Title: remote.Title,
		AdminURL: adminURL, GalleryCount: len(remote.Gallery), Images: images, Variants: variants}
	if len(variants) != 6 || len(remote.Gallery) == 0 || len(selected.positions) != len(images) ||
		selected.inventory.Head == (asset.ImageInventoryHead{}) || selected.inventory.ApprovalActionID == "" {
		return preview, plan, nil
	}
	gallery, valid := shopifyTrialGallery(remote.Gallery)
	if !valid {
		return preview, plan, nil
	}
	approved := make(map[string]string, len(images))
	hashes := make(map[string]bool, len(images))
	var additions []string
	for i, proof := range selected.proofs {
		image := images[selected.positions[i]]
		hash := hex.EncodeToString(proof.Hash[:])
		if proof.Size < 1 || proof.Hash == ([32]byte{}) || image.ID == "" || approved[image.ID] != "" || hashes[hash] ||
			image.State == wire.DraftMediaUnverified || image.State == wire.DraftMediaMissing && !image.uploadable {
			return preview, plan, nil
		}
		approved[image.ID], hashes[hash] = hash, true
		if image.State == wire.DraftMediaMissing {
			additions = append(additions, hash)
		}
	}
	covers, needsCover, valid := shopifyTrialCovers(variants, remote.Variants, approved)
	if !valid || len(additions) == 0 && !needsCover {
		return preview, plan, nil
	}
	approvalHash := collection.Digest(struct {
		Head           asset.ImageInventoryHead
		Action, Origin string
		Digests        map[string]string
	}{selected.inventory.Head, selected.inventory.ApprovalActionID, selected.selection.OriginalPublicationID, approved})
	plan = submission.ShopifyMediaPlan{ID: operation.ID, Owner: operation.Owner, ShopKey: operation.ShopKey, Generation: operation.Generation,
		SourceKey: operation.SourceKey, ProductID: operation.ProductID, SourceVersion: version,
		ApprovalHash: approvalHash, BeforeHash: shopifyTrialBeforeHash(remote), ExpiresAt: expires,
		GalleryIDs: gallery, Images: additions, Covers: covers}
	if !plan.Valid() {
		return preview, submission.ShopifyMediaPlan{}, nil
	}
	return preview, plan, nil
}

func shopifyTrialGallery(media []wire.DraftGalleryEvidence) ([]string, bool) {
	ids := make([]string, 0, len(media))
	byHash := make(map[[32]byte]bool, len(media))
	for _, image := range media {
		if image.Type != "IMAGE" || image.Status != "READY" || image.Size < 1 || image.Hash == ([32]byte{}) || byHash[image.Hash] {
			return nil, false
		}
		byHash[image.Hash] = true
		ids = append(ids, image.ID)
	}
	return ids, true
}

func shopifyTrialCovers(views []ShopifyMediaPreviewVariant, variants []wire.DraftMediaVariant, proofs map[string]string) ([]submission.ShopifyMediaCover, bool, bool) {
	bySKU := make(map[string]wire.DraftMediaVariant, len(variants))
	for _, variant := range variants {
		bySKU[variant.SKU] = variant
	}
	covers := make([]submission.ShopifyMediaCover, 0, len(views))
	var change bool
	for _, view := range views {
		variant := bySKU[view.SKU]
		hash := proofs[view.AssetID]
		if hash == "" || (view.State != "same" && (view.State != "different" || len(variant.MediaIDs) != 0)) {
			return nil, false, false
		}
		cover := submission.ShopifyMediaCover{VariantID: variant.ID, ImageHash: hash}
		if view.State == "same" {
			cover.CurrentMediaID = variant.MediaIDs[0]
		} else {
			change = true
		}
		covers = append(covers, cover)
	}
	return covers, change, true
}

func shopifyTrialBeforeHash(remote wire.DraftMediaEvidence) string {
	remote.Variants = slices.Clone(remote.Variants)
	slices.SortFunc(remote.Variants, func(a, b wire.DraftMediaVariant) int { return strings.Compare(a.ID, b.ID) })
	for i, variant := range remote.Variants {
		options := slices.Clone(variant.Options)
		slices.SortFunc(options, func(a, b wire.DraftMediaOption) int { return strings.Compare(a.Name, b.Name) })
		remote.Variants[i].Options = options
	}
	return collection.Digest(remote)
}
