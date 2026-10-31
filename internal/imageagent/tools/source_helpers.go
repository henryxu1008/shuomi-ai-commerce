package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"task-processor/internal/imageagent"
	imagepolicy "task-processor/internal/marketplace/imagepolicy"
	productimage "task-processor/internal/product/image"
)

type ProfileResolver interface {
	Resolve(imagepolicy.ProfileInput) (imagepolicy.ProductImageProfile, error)
}

// resolvedSlotInput binds current source editing to the exact authorized source
// and the current policy or ImageSet recipe.
type resolvedSlotInput struct {
	otherSources   []productimage.Asset
	providerPrompt string
	slot           imageagent.Slot
	source         productimage.Asset
	product        productimage.ProductContext
	profile        imagepolicy.ProductImageProfile
	sourceAssetID  string
	sourceBytes    []byte
	sourceDigest   string
	sources        []productimage.Asset
	references     [][]byte
}

func resolveSourceInput(input imageagent.SlotExecutionInput, profiles ProfileResolver) (resolvedSlotInput, error) {
	if input.ImageSet != nil || input.Slot.Recipe != nil {
		return resolveImageSetInput(input)
	}
	if profiles == nil {
		return resolvedSlotInput{}, fmt.Errorf("%w: image policy resolver is required", imageagent.ErrValidation)
	}
	if strings.TrimSpace(input.RunID) == "" || strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.UserID) == "" ||
		input.PlanRevision <= 0 || input.Attempt <= 0 || strings.TrimSpace(input.IdempotencyKey) == "" {
		return resolvedSlotInput{}, imageagent.ErrValidation
	}
	slot := cloneSlot(input.Slot)
	if strings.TrimSpace(slot.ID) == "" || strings.TrimSpace(slot.IdempotencyKey) == "" || len(slot.SourceAssetIDs) != 1 {
		return resolvedSlotInput{}, imageagent.ErrValidation
	}
	if input.ImagePolicyContext == nil {
		return resolvedSlotInput{}, imageagent.ErrValidation
	}
	var profile imagepolicy.ProductImageProfile
	if input.ImagePolicyContext != nil && strings.TrimSpace(input.TargetPlatform) != "" {
		profileInput := imagepolicy.ProfileInput{
			Marketplace: strings.TrimSpace(input.TargetPlatform), Country: strings.TrimSpace(input.ImagePolicyContext.Country),
			Family: strings.TrimSpace(input.ImagePolicyContext.Family), SceneCategory: strings.TrimSpace(input.ImagePolicyContext.SceneCategory),
		}
		if err := imagepolicy.ValidateProfileInput(profileInput); err != nil {
			return resolvedSlotInput{}, err
		}
		if profiles == nil {
			return resolvedSlotInput{}, fmt.Errorf("%w: image policy resolver is required", imageagent.ErrValidation)
		}
		var err error
		profile, err = profiles.Resolve(profileInput)
		if err != nil {
			return resolvedSlotInput{}, err
		}
		if profile.Key != imagepolicy.PolicyKey(profileInput) || strings.TrimSpace(profile.PolicyVersion) == "" {
			return resolvedSlotInput{}, imageagent.ErrValidation
		}
	}
	if err := imageagent.ValidateAcquisitionMainExecution(input); err != nil {
		return resolvedSlotInput{}, err
	}
	sourceID := strings.TrimSpace(slot.SourceAssetIDs[0])
	source, err := authorizedProductAsset(input.AssetCatalog, sourceID, imageagent.AuthorizedAssetSource)
	if err != nil {
		return resolvedSlotInput{}, err
	}
	if len(input.SourceBytes) > productimage.MaxInlineArtifactBytes {
		return resolvedSlotInput{}, imageagent.ErrValidation
	}
	product := productimage.ProductContext{
		ProductKey: strings.TrimSpace(input.ProductContext.ProductID), Title: strings.TrimSpace(input.ProductContext.Title),
		ProductType: strings.TrimSpace(input.ProductContext.ProductType), Attributes: cloneStringMap(input.ProductContext.Attributes),
	}
	if product.ProductKey == "" {
		return resolvedSlotInput{}, imageagent.ErrValidation
	}
	return resolvedSlotInput{
		slot: slot, source: source, product: product, profile: profile,
		sourceAssetID: sourceID,
		sourceBytes:   append([]byte(nil), input.SourceBytes...), sourceDigest: input.SourceDigest,
	}, nil
}
func resolveImageSetInput(input imageagent.SlotExecutionInput) (resolvedSlotInput, error) {
	if input.RunID == "" || input.TenantID == "" || input.UserID == "" || input.PlanRevision <= 0 || input.Attempt <= 0 || input.IdempotencyKey == "" || input.Slot.IdempotencyKey == "" {
		return resolvedSlotInput{}, imageagent.ErrValidation
	}
	if err := imageagent.ValidateImageSetExecution(input); err != nil {
		return resolvedSlotInput{}, err
	}
	slot := cloneSlot(input.Slot)
	sources := make([]productimage.Asset, len(slot.SourceAssetIDs))
	for i, id := range slot.SourceAssetIDs {
		source, err := authorizedProductAsset(input.AssetCatalog, id, imageagent.AuthorizedAssetSource)
		if err != nil {
			return resolvedSlotInput{}, err
		}
		observed := slot.Recipe.References[i]
		if source.Width > 0 && (source.Width != observed.Width || source.Height != observed.Height) {
			return resolvedSlotInput{}, imageagent.ErrRevisionConflict
		}
		source.Width = observed.Width
		source.Height = observed.Height
		source.MediaType = observed.MediaType
		sources[i] = source
	}
	references := make([][]byte, len(input.SourceReferences))
	for i, data := range input.SourceReferences {
		references[i] = append([]byte(nil), data...)
	}
	product := productimage.ProductContext{ProductKey: strings.TrimSpace(input.ProductContext.ProductID), Title: strings.TrimSpace(input.ProductContext.Title), ProductType: strings.TrimSpace(input.ProductContext.ProductType), Attributes: cloneStringMap(input.ProductContext.Attributes)}
	if product.ProductKey == "" || product.ProductKey != input.ImageSet.Source.ProductID {
		return resolvedSlotInput{}, imageagent.ErrRevisionConflict
	}
	return resolvedSlotInput{slot: slot, source: sources[0], sources: sources, otherSources: sources[1:], product: product, sourceAssetID: sources[0].SourceAssetID, references: references}, nil
}
func authorizedProductAsset(catalog imageagent.AssetCatalog, id string, kind imageagent.AuthorizedAssetType) (productimage.Asset, error) {
	var matched *imageagent.AuthorizedAsset
	for index := range catalog.Assets {
		candidate := &catalog.Assets[index]
		if strings.TrimSpace(candidate.ID) == id && candidate.Type == kind {
			if matched != nil {
				return productimage.Asset{}, imageagent.ErrValidation
			}
			matched = candidate
		}
	}
	if matched == nil {
		return productimage.Asset{}, imageagent.ErrValidation
	}
	url := strings.TrimSpace(matched.URL)
	if url == "" {
		url = strings.TrimSpace(matched.DisplayURL)
	}
	sourceURL := strings.TrimSpace(matched.SourceURL)
	if sourceURL == "" {
		sourceURL = url
	}
	validatedURL, err := imageagent.ValidateSafeImageURL(url)
	if err != nil {
		return productimage.Asset{}, err
	}
	validatedSourceURL, err := imageagent.ValidateSafeImageURL(sourceURL)
	if err != nil {
		return productimage.Asset{}, err
	}
	return productimage.Asset{
		URL: validatedURL, SourceURL: validatedSourceURL, SourceAssetID: id, Role: productimage.RoleSource,
		Width: matched.Width, Height: matched.Height, Operations: []string{"ingest"},
	}, nil
}
func generatedAsset(candidate productimage.Candidate, source productimage.Asset, otherSources []productimage.Asset) (imageagent.GeneratedAsset, error) {
	asset := candidate.Asset
	if asset.SourceAssetID != source.SourceAssetID || asset.Role == productimage.RoleSource || len(asset.Operations) == 0 ||
		asset.Width <= 0 || asset.Height <= 0 || strings.TrimSpace(asset.SourceURL) == "" {
		return imageagent.GeneratedAsset{}, imageagent.ErrValidation
	}
	if (asset.URL == "") == (asset.Bytes == nil) {
		return imageagent.GeneratedAsset{}, imageagent.ErrValidation
	}
	for _, forbidden := range append([]productimage.Asset{source}, otherSources...) {
		if asset.URL != "" && (strings.TrimSpace(asset.URL) == strings.TrimSpace(forbidden.URL) || strings.TrimSpace(asset.URL) == strings.TrimSpace(forbidden.SourceURL)) {
			return imageagent.GeneratedAsset{}, imageagent.ErrValidation
		}
	}
	for _, operation := range asset.Operations {
		if strings.Contains(operation, "pass_through") || strings.Contains(operation, "placeholder") || operation == "source" {
			return imageagent.GeneratedAsset{}, imageagent.ErrValidation
		}
	}
	if asset.URL != "" {
		if _, err := imageagent.ValidateSafeImageURL(asset.URL); err != nil {
			return imageagent.GeneratedAsset{}, err
		}
	} else if len(asset.Bytes) == 0 || len(asset.Bytes) > productimage.MaxInlineArtifactBytes || strings.TrimSpace(asset.MediaType) == "" {
		return imageagent.GeneratedAsset{}, imageagent.ErrValidation
	}
	metadata := cloneStringMap(candidate.Metadata.Values)
	return imageagent.GeneratedAsset{
		URL: asset.URL, Bytes: append([]byte(nil), asset.Bytes...), ContentType: asset.MediaType,
		SourceURL: asset.SourceURL, Operations: append([]string(nil), asset.Operations...),
		Width: asset.Width, Height: asset.Height, Metadata: metadata,
		ProviderReceiptID: candidate.Metadata.InvocationID,
	}, nil
}
func providerError(state imageagent.ProviderDispatchState, err error) error {
	return &imageagent.ProviderDispatchError{State: state, Err: err}
}
func cloneSlot(slot imageagent.Slot) imageagent.Slot {
	cloned := slot
	cloned.SourceAssetIDs = append([]string(nil), slot.SourceAssetIDs...)
	cloned.StyleReferenceIDs = append([]string(nil), slot.StyleReferenceIDs...)
	cloned.Recipe = imageagent.CloneImageSlotRecipe(slot.Recipe)
	return cloned
}
func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
func durableCandidateAssetID(input imageagent.SlotExecutionInput, slot imageagent.Slot, asset imageagent.PublishedAssetRef, index int) string {
	encoded, _ := json.Marshal(struct {
		TenantID, RunID, SlotID, ObjectKey, SHA256 string
		PlanRevision                               int64
		Attempt, Index                             int
	}{input.TenantID, input.RunID, slot.ID, asset.ObjectKey, asset.SHA256, input.PlanRevision, input.Attempt, index})
	digest := sha256.Sum256(encoded)
	return "imgcand_" + hex.EncodeToString(digest[:16])
}
