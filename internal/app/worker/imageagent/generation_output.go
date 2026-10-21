package imageagentworker

import (
	"context"
	"errors"
	"net/http"
	"task-processor/internal/imageagent"
	"task-processor/internal/integration/httpimage"
	"task-processor/internal/pkg/imagex"
	productimage "task-processor/internal/product/image"

	"github.com/sirupsen/logrus"
)

// No credential resolver, current price, provider client or source fetch is
// reachable from this GET-only recovery closure.
func generationOutputRecovery(fetch func(context.Context, string) ([]byte, error), logger *logrus.Logger) imageagent.GenerationOutputRecovery {
	if fetch == nil {
		fetch = func(ctx context.Context, raw string) ([]byte, error) {
			return httpimage.Download(ctx, httpimage.NewPublicImageHTTPClient(), raw, productimage.MaxInlineArtifactBytes)
		}
	}
	return func(ctx context.Context, input imageagent.SlotExecutionInput, fact imageagent.GenerationFact) (imageagent.SlotGeneratedOutput, error) {
		bad := func() (imageagent.SlotGeneratedOutput, error) {
			return imageagent.SlotGeneratedOutput{}, imageagent.ErrValidation
		}
		id := imageagent.SlotExternalEffectIdentity{RunScope: imageagent.RunScope{TenantID: input.TenantID, OwnerUserID: input.UserID, RunID: input.RunID}, PlanRevision: input.PlanRevision, SlotID: input.Slot.ID, Attempt: input.Attempt}
		if fact.Validate() != nil || fact.State != imageagent.GenerationSucceeded || fact.Intent.Identity != id || fact.Intent.MemberID != input.OrganizationIdentity.MemberID || fact.Intent.CatalogHash != input.AssetCatalog.Manifest.Hash || input.Artwork == nil && len(input.Slot.SourceAssetIDs) < 1 {
			return bad()
		}
		operation := productimage.SourceWhiteBackgroundOperation
		var artwork *imageagent.ArtworkProvenance
		if input.Artwork != nil {
			provenance, err := imageagent.NewArtworkProvenance(input, fact)
			if err != nil {
				return bad()
			}
			artwork = &provenance
			operation = "generate_artwork"
		} else if input.ImageSet != nil || input.Slot.Recipe != nil || fact.Intent.InputProtocol != "" {
			digest, err := imageagent.ImageSlotGenerationInputDigest(input)
			if err != nil || fact.Intent.InputProtocol != imageagent.ImageSetSchema || fact.Intent.InputDigest != digest || fact.Intent.SourceDigest != imageagent.ImageSourceBundleDigest(input.Slot.Recipe.References) || fact.Intent.PromptVersion != input.Slot.Recipe.PromptVersion {
				return bad()
			}
			operation = productimage.SourceEditOperation
		} else if input.Slot.Role != imageagent.SlotRoleMain || len(input.Slot.SourceAssetIDs) != 1 {
			return bad()
		}
		var sourceURL, sourceID string
		if artwork == nil {
			sourceID = input.Slot.SourceAssetIDs[0]
			for _, asset := range input.AssetCatalog.Assets {
				if asset.ID == input.Slot.SourceAssetIDs[0] && asset.Type == imageagent.AuthorizedAssetSource {
					sourceURL = asset.SourceURL
					if sourceURL == "" {
						sourceURL = asset.URL
					}
				}
			}
			if _, err := imageagent.ValidateSafeImageURL(sourceURL); err != nil {
				return bad()
			}
		}
		if _, err := imageagent.ValidateSafeImageURL(fact.Success.ResultURL); err != nil || fact.Success.ResultUnavailable != "" {
			// Only after binding the immutable success to this original input
			// may an unusable locator close the charged image-set failure.
			if input.ImageSet != nil || input.Artwork != nil {
				return imageagent.SlotGeneratedOutput{}, imageagent.ErrInvalidGeneratedOutput
			}
			return bad()
		}
		data, err := fetch(ctx, fact.Success.ResultURL)
		if err != nil {
			var response *httpimage.HTTPStatusError
			// This is the original success's fixed public URL; no alternate
			// credentials or replacement locator are part of this recovery.
			if errors.Is(err, httpimage.ErrBodyTooLarge) || errors.As(err, &response) && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusGone) {
				return imageagent.SlotGeneratedOutput{}, imageagent.ErrInvalidGeneratedOutput
			}
			return bad()
		}
		if len(data) == 0 || len(data) > productimage.MaxInlineArtifactBytes {
			return imageagent.SlotGeneratedOutput{}, imageagent.ErrInvalidGeneratedOutput
		}
		contentType, width, height, err := httpimage.InspectGeneratedArtifact(data)
		if err != nil {
			return imageagent.SlotGeneratedOutput{}, imageagent.ErrInvalidGeneratedOutput
		}
		if (input.ImageSet != nil || input.Artwork != nil || fact.Intent.Provider == "grsai" && fact.Intent.Model == "gpt-image-2-vip") && (width != 1024 || height != 1024) {
			if logger != nil {
				imageDimensionsDiagnostic(logrus.NewEntry(logger), fact.Intent.Provider, fact.Intent.Model)(1024, 1024, width, height)
			}
			return imageagent.SlotGeneratedOutput{}, imageagent.ErrInvalidGeneratedOutput
		}
		info, err := imagex.Inspect(data)
		if err != nil || info.Width != width || info.Height != height {
			return imageagent.SlotGeneratedOutput{}, imageagent.ErrInvalidGeneratedOutput
		}
		return imageagent.SlotGeneratedOutput{SlotID: input.Slot.ID, Attempt: input.Attempt, SourceAssetID: sourceID, Assets: []imageagent.GeneratedAsset{{Bytes: append([]byte(nil), data...), ContentType: contentType, Width: width, Height: height, SourceURL: sourceURL, Operations: []string{operation}, ProviderReceiptID: fact.Success.RequestID, Artwork: artwork}}}, nil
	}
}

// Only immutable route identity and measured dimensions reach the log. Neither
// prompts nor provider result URLs belong in a dimension diagnostic.
func imageDimensionsDiagnostic(logger *logrus.Entry, provider, model string) func(int, int, int, int) {
	if logger == nil {
		return nil
	}
	return func(expectedWidth, expectedHeight, actualWidth, actualHeight int) {
		logger.WithFields(logrus.Fields{"provider": provider, "model": model, "expected_width": expectedWidth, "expected_height": expectedHeight, "actual_width": actualWidth, "actual_height": actualHeight}).Warn("Generated image output dimensions mismatch")
	}
}
