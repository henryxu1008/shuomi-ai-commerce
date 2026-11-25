package listingapp

import (
	"context"
	"crypto/sha256"
	"errors"
	"slices"
	"time"

	storeapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/storecenter"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent"
	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/preparation"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

const maxShopifyPreviewImages = 30

type ShopifyMediaPreviewService struct {
	Store     *storeapp.OfficialProductAccess
	Inventory asset.ImageSetInventoryReader
	ReadImage func(context.Context, asset.SourceSelection, asset.ApprovedAsset, int64) ([]byte, error)
}

type ShopifyMediaPreviewRequest struct {
	SourceID string `json:"sourceId"`
	StoreID  string `json:"storeId"`
	Product  string `json:"product"`
}

type ShopifyMediaPreview struct {
	SourceID     string                       `json:"sourceId"`
	StoreID      string                       `json:"storeId"`
	Title        string                       `json:"title"`
	AdminURL     string                       `json:"adminUrl"`
	GalleryCount int                          `json:"galleryCount"`
	Images       []ShopifyMediaPreviewImage   `json:"images"`
	Variants     []ShopifyMediaPreviewVariant `json:"variants"`
}

type ShopifyMediaPreviewImage struct {
	ID       string               `json:"id"`
	URL      string               `json:"url"`
	Color    string               `json:"color,omitempty"`
	Role     asset.Role           `json:"role"`
	State    wire.DraftMediaState `json:"state"`
	mediaID  string
	sourceID string
}

type ShopifyMediaPreviewVariant struct {
	SKU            string `json:"sku"`
	AssetID        string `json:"assetId,omitempty"`
	CurrentAssetID string `json:"currentAssetId,omitempty"`
	State          string `json:"state"`
}

func (a *Application) PreviewShopifyMedia(ctx context.Context, input ShopifyMediaPreviewRequest) (ShopifyMediaPreview, error) {
	if a == nil || a.ShopifyMedia == nil {
		return ShopifyMediaPreview{}, preparation.ErrUnavailable
	}
	if !collection.ValidID(input.SourceID) || !collection.ValidID(input.StoreID) || input.Product == "" || len(input.Product) > 512 {
		return ShopifyMediaPreview{}, preparation.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	selected, err := a.Sources.Select(ctx, input.SourceID)
	if err != nil {
		return ShopifyMediaPreview{}, err
	}
	scope, item, original, err := selected.Read(ctx)
	if err != nil {
		return ShopifyMediaPreview{}, err
	}
	if item.Source.Kind != "own" {
		return ShopifyMediaPreview{}, preparation.ErrConflict
	}
	selectionInput := asset.SourceSelectionRequest{ItemID: item.ID, OriginalPublicationID: original.PublicationID,
		OriginalSnapshotVersion: original.Version, EffectiveCatalogVersion: original.Version, TargetPlatform: "product"}
	sourceReader := SourceImageReader{Sources: a.Sources, Products: a.Products, Authorization: a.Authorization, OriginalOnly: true}
	source, err := sourceReader.ReadSourceSelection(ctx, selectionInput)
	if err != nil {
		return ShopifyMediaPreview{}, err
	}
	service := a.ShopifyMedia
	inventoryScope := asset.InventoryScope{TenantID: scope.OrganizationID, ProductKey: source.ProductKey, SourceSnapshotVersion: source.EffectiveCatalogVersion, TargetPlatform: "product"}
	inventory, err := service.Inventory.ReadImageSetInventory(ctx, inventoryScope)
	if err != nil {
		return ShopifyMediaPreview{}, err
	}
	images, proofs, proofPositions, err := service.previewImages(ctx, source, inventory)
	if err != nil {
		return ShopifyMediaPreview{}, err
	}
	subject := shopifyProductSubject(scope, storecenter.ProductPurposeStatus)
	remote, adminURL, err := service.Store.InspectShopifyDraftMedia(ctx, subject, input.StoreID, input.Product, proofs)
	switch {
	case errors.Is(err, wire.ErrInvalidRequest):
		return ShopifyMediaPreview{}, preparation.ErrInvalid
	case errors.Is(err, storeapp.ErrProductAccessChanged):
		return ShopifyMediaPreview{}, preparation.ErrForbidden
	case errors.Is(err, wire.ErrDraftUnverified):
		return ShopifyMediaPreview{}, preparation.ErrConflict
	case err != nil:
		return ShopifyMediaPreview{}, preparation.ErrUnavailable
	}
	if len(remote.Images) != len(proofPositions) {
		return ShopifyMediaPreview{}, preparation.ErrUnavailable
	}
	for i, match := range remote.Images {
		position := proofPositions[i]
		images[position].State = match.State
		images[position].mediaID = match.MediaID
	}
	variants, matches := previewShopifyVariants(original.Snapshot.Variants, remote.Variants, images, source.Images)
	if !matches {
		return ShopifyMediaPreview{}, preparation.ErrConflict
	}
	if _, err := sourceReader.ReadSourceSelection(ctx, selectionInput); err != nil {
		return ShopifyMediaPreview{}, err
	}
	current, err := service.Inventory.ReadImageSetInventory(ctx, inventoryScope)
	if err != nil {
		return ShopifyMediaPreview{}, err
	}
	if inventory.Head != current.Head || inventory.ApprovalActionID != current.ApprovalActionID {
		return ShopifyMediaPreview{}, preparation.ErrConflict
	}
	return ShopifyMediaPreview{SourceID: input.SourceID, StoreID: input.StoreID, Title: remote.Title,
		AdminURL: adminURL, GalleryCount: remote.GalleryCount, Images: images, Variants: variants}, nil
}

func (s *ShopifyMediaPreviewService) previewImages(ctx context.Context, source asset.SourceSelection, inventory asset.ImageSetInventory) ([]ShopifyMediaPreviewImage, []wire.DraftMediaFingerprint, []int, error) {
	images := make([]ShopifyMediaPreviewImage, 0, len(inventory.Assets))
	var proofs []wire.DraftMediaFingerprint
	var positions []int
	for _, approved := range inventory.Assets {
		if approved.Role == asset.RoleDesign {
			continue
		}
		if len(images) == maxShopifyPreviewImages {
			return nil, nil, nil, preparation.ErrConflict
		}
		view := ShopifyMediaPreviewImage{ID: approved.ID, URL: approved.URL, Role: approved.Role, State: wire.DraftMediaUnverified, sourceID: approved.SourceAssetID}
		if approved.Presentation != nil {
			view.Color = approved.Presentation.Color
			if view.Color != "" && approved.Presentation.Group == "carousel" && approved.Presentation.Order == 1 {
				view.Role = asset.RoleMain
			}
		}
		if approved.GenerationEvidence != nil {
			content, err := s.ReadImage(ctx, source, approved, wire.MaxStagedDraftImageBytes)
			if err == nil {
				proofs = append(proofs, wire.DraftMediaFingerprint{Size: int64(len(content)), Hash: sha256.Sum256(content)})
				positions = append(positions, len(images))
			}
		}
		images = append(images, view)
	}
	return images, proofs, positions, nil
}

func previewShopifyVariants(local []catalog.Variant, remote []wire.DraftMediaVariant, images []ShopifyMediaPreviewImage, sources []asset.SourceImage) ([]ShopifyMediaPreviewVariant, bool) {
	if len(local) == 0 || len(local) != len(remote) {
		return nil, false
	}
	bySKU := make(map[string]wire.DraftMediaVariant, len(remote))
	for _, variant := range remote {
		if _, duplicate := bySKU[variant.SKU]; duplicate || variant.SKU == "" {
			return nil, false
		}
		bySKU[variant.SKU] = variant
	}
	result := make([]ShopifyMediaPreviewVariant, 0, len(local))
	colorOption := previewColorOption(local, images, sources)
	for _, variant := range local {
		other, found := bySKU[variant.SKU]
		if !found || !equalShopifyOptions(variant.Attributes, other.Options) {
			return nil, false
		}
		delete(bySKU, variant.SKU)
		result = append(result, shopifyVariantImagePreview(variant, other, images, colorOption))
	}
	return result, true
}

func equalShopifyOptions(attributes []catalog.Attribute, options []wire.DraftMediaOption) bool {
	if len(attributes) != len(options) {
		return false
	}
	matched := make(map[string]string, len(options))
	for _, option := range options {
		if _, duplicate := matched[option.Name]; duplicate || option.Name == "" || option.Value == "" {
			return false
		}
		matched[option.Name] = option.Value
	}
	for _, attribute := range attributes {
		if matched[attribute.Name] != attribute.Value || attribute.Value == "" {
			return false
		}
		delete(matched, attribute.Name)
	}
	return true
}

func previewColorOption(variants []catalog.Variant, images []ShopifyMediaPreviewImage, sources []asset.SourceImage) imageagent.ImageColorOption {
	byURL := make(map[string]string, len(sources))
	for _, source := range sources {
		byURL[source.URL] = source.ID
	}
	var result imageagent.ImageColorOption
	for _, option := range variants[0].Attributes {
		candidate, ok := listingColorOption(option.Name, variants, byURL)
		if !ok {
			continue
		}
		if slices.ContainsFunc(candidate.Groups, func(group imageagent.ImageColorGroup) bool {
			return !slices.ContainsFunc(images, func(image ShopifyMediaPreviewImage) bool {
				return isShopifyGroupMain(image, group)
			})
		}) {
			continue
		}
		if result.Name != "" {
			return imageagent.ImageColorOption{}
		}
		result = candidate
	}
	return result
}

func isShopifyGroupMain(image ShopifyMediaPreviewImage, group imageagent.ImageColorGroup) bool {
	return group.Value != "" && image.Role == asset.RoleMain && image.Color == group.Value &&
		slices.Contains(group.OriginalIDs, image.sourceID)
}

func shopifyVariantImagePreview(variant catalog.Variant, remote wire.DraftMediaVariant, images []ShopifyMediaPreviewImage, colorOption imageagent.ImageColorOption) ShopifyMediaPreviewVariant {
	result := ShopifyMediaPreviewVariant{SKU: variant.SKU, State: "unverified"}
	var group imageagent.ImageColorGroup
	for _, value := range variant.Attributes {
		if value.Name == colorOption.Name {
			index := slices.IndexFunc(colorOption.Groups, func(candidate imageagent.ImageColorGroup) bool { return candidate.Value == value.Value })
			if index >= 0 {
				group = colorOption.Groups[index]
			}
		}
	}
	var main *ShopifyMediaPreviewImage
	var ambiguous bool
	for i, image := range images {
		if image.mediaID != "" && len(remote.MediaIDs) == 1 && remote.MediaIDs[0] == image.mediaID {
			result.CurrentAssetID = image.ID
		}
		if !isShopifyGroupMain(image, group) {
			continue
		}
		ambiguous = main != nil
		main = &images[i]
	}
	if main == nil || ambiguous {
		return result
	}
	result.AssetID = main.ID
	if len(remote.MediaIDs) > 1 {
		return result
	}
	switch main.State {
	case wire.DraftMediaPresent:
		result.State = "different"
		if len(remote.MediaIDs) == 1 && remote.MediaIDs[0] == main.mediaID {
			result.State = "same"
		}
	case wire.DraftMediaMissing:
		result.State = "different"
	}
	return result
}
