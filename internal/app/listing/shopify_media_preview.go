package listingapp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"slices"
	"time"

	"github.com/google/uuid"

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
	SourceID         string                       `json:"sourceId"`
	StoreID          string                       `json:"storeId"`
	Title            string                       `json:"title"`
	AdminURL         string                       `json:"adminUrl"`
	GalleryCount     int                          `json:"galleryCount"`
	ConfirmationHash string                       `json:"confirmationHash,omitempty"`
	Images           []ShopifyMediaPreviewImage   `json:"images"`
	Variants         []ShopifyMediaPreviewVariant `json:"variants"`
}

type ShopifyMediaPreviewImage struct {
	ID         string               `json:"id"`
	URL        string               `json:"url"`
	Color      string               `json:"color,omitempty"`
	Role       asset.Role           `json:"role"`
	State      wire.DraftMediaState `json:"state"`
	mediaID    string
	sourceID   string
	uploadable bool
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
	if a.canPreviewShopifyMediaTrial(ctx, input, scope) {
		op, err := a.ShopifyMediaConfirm.operation(input, uuid.NewString(), scope, item)
		if err != nil {
			return ShopifyMediaPreview{}, err
		}
		preview, plan, err := a.readShopifyMediaTrial(ctx, input, op, item, original)
		if err == nil && plan.Valid() && a.canPreviewShopifyMediaTrial(ctx, input, scope) {
			preview.ConfirmationHash = plan.PreviewHash()
		}
		return preview, err
	}
	selectedImages, err := a.readShopifyMediaSource(ctx, scope, item, original)
	if err != nil {
		return ShopifyMediaPreview{}, err
	}
	service := a.ShopifyMedia
	subject := shopifyProductSubject(scope, storecenter.ProductPurposeStatus)
	remote, adminURL, err := service.Store.InspectShopifyDraftMedia(ctx, subject, input.StoreID, input.Product, selectedImages.proofs)
	if err != nil {
		return ShopifyMediaPreview{}, shopifyMediaPreviewError(err)
	}
	images, err := matchShopifyMediaPreviewImages(selectedImages, remote.Images)
	if err != nil {
		return ShopifyMediaPreview{}, err
	}
	variants, matches := previewShopifyVariants(original.Snapshot.Variants, remote.Variants, images, selectedImages.source.Images)
	if !matches {
		return ShopifyMediaPreview{}, preparation.ErrConflict
	}
	if err := a.recheckShopifyMediaSource(ctx, selectedImages); err != nil {
		return ShopifyMediaPreview{}, err
	}
	return ShopifyMediaPreview{SourceID: input.SourceID, StoreID: input.StoreID, Title: remote.Title,
		AdminURL: adminURL, GalleryCount: remote.GalleryCount, Images: images, Variants: variants}, nil
}

type shopifyMediaSourceEvidence struct {
	source         asset.SourceSelection
	reader         SourceImageReader
	selection      asset.SourceSelectionRequest
	inventory      asset.ImageSetInventory
	inventoryScope asset.InventoryScope
	images         []ShopifyMediaPreviewImage
	proofs         []wire.DraftMediaFingerprint
	positions      []int
}

func (a *Application) readShopifyMediaSource(ctx context.Context, scope collection.Scope, item preparation.SourceItem, original catalog.PublishedSnapshot) (shopifyMediaSourceEvidence, error) {
	selection := asset.SourceSelectionRequest{ItemID: item.ID, OriginalPublicationID: original.PublicationID,
		OriginalSnapshotVersion: original.Version, EffectiveCatalogVersion: original.Version, TargetPlatform: "product"}
	reader := SourceImageReader{Sources: a.Sources, Products: a.Products, Authorization: a.Authorization, OriginalOnly: true}
	source, err := reader.ReadSourceSelection(ctx, selection)
	if err != nil {
		return shopifyMediaSourceEvidence{}, err
	}
	inventoryScope := asset.InventoryScope{TenantID: scope.OrganizationID, ProductKey: source.ProductKey,
		SourceSnapshotVersion: source.EffectiveCatalogVersion, TargetPlatform: "product"}
	inventory, err := a.ShopifyMedia.Inventory.ReadImageSetInventory(ctx, inventoryScope)
	if err != nil {
		return shopifyMediaSourceEvidence{}, err
	}
	images, proofs, positions, err := a.ShopifyMedia.previewImages(ctx, source, inventory)
	if err != nil {
		return shopifyMediaSourceEvidence{}, err
	}
	return shopifyMediaSourceEvidence{source: source, reader: reader, selection: selection, inventory: inventory,
		inventoryScope: inventoryScope, images: images, proofs: proofs, positions: positions}, nil
}

func (a *Application) recheckShopifyMediaSource(ctx context.Context, selected shopifyMediaSourceEvidence) error {
	if _, err := selected.reader.ReadSourceSelection(ctx, selected.selection); err != nil {
		return err
	}
	current, err := a.ShopifyMedia.Inventory.ReadImageSetInventory(ctx, selected.inventoryScope)
	if err != nil {
		return err
	}
	if selected.inventory.Head != current.Head || selected.inventory.ApprovalActionID != current.ApprovalActionID {
		return preparation.ErrConflict
	}
	return nil
}

func matchShopifyMediaPreviewImages(selected shopifyMediaSourceEvidence, matches []wire.DraftMediaMatch) ([]ShopifyMediaPreviewImage, error) {
	if len(matches) != len(selected.positions) {
		return nil, preparation.ErrUnavailable
	}
	images := slices.Clone(selected.images)
	for i, match := range matches {
		position := selected.positions[i]
		images[position].State = match.State
		images[position].mediaID = match.MediaID
	}
	return images, nil
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
			if err == nil && len(content) > 0 {
				view.uploadable = bytes.HasPrefix(content, []byte("\x89PNG\r\n\x1a\n"))
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
