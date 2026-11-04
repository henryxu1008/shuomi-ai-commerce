package imageagentworker

import (
	"context"
	"task-processor/internal/aicapability"
	"task-processor/internal/imageagent"
	"task-processor/internal/imageagent/objectstore"
	"task-processor/internal/integration/httpimage"
	"task-processor/internal/integration/openai"
	productimage "task-processor/internal/product/image"
)

// Preserve the already returned bytes through the original immutable bundle.
// A storage failure never erases trustworthy usage or fabricates no-send proof.
func (f generationProviderFactory) observeFlare(ctx context.Context, input imageagent.SlotExecutionInput, o openai.FlareImageObservation, observe func(context.Context, imageagent.GenerationSuccess) error) error {
	t := o.Usage
	if t.Validate() != nil {
		return imageagent.ErrProviderContractViolation
	}
	u := aicapability.BillableModelUsage{MappingVersion: aicapability.ReferenceImagesTotalsV1, Complete: true, InputText: t.InputTextTokens, InputImage: t.InputImageTokens, OutputImage: t.OutputTokens, ReferenceImageTotals: aicapability.ReferenceImageTotals{Known: true, Input: t.InputTokens, Output: t.OutputTokens, Total: t.TotalTokens, InputTextKnown: t.InputTextKnown, InputImageKnown: t.InputImageKnown}}
	if u.Validate() != nil {
		return imageagent.ErrProviderContractViolation
	}
	proof := imageagent.GenerationSuccess{BillableUsage: u, UsageKnown: true, InputTokens: t.InputTokens, OutputTokens: t.OutputTokens, TotalTokens: t.TotalTokens, ResponseID: o.ResponseID, RequestID: o.RequestID, ResultDigest: o.ResultDigest, ResultUnavailable: o.ResultUnavailable}
	var preserveErr error
	if o.ResultUnavailable == "" {
		mime, w, h, err := httpimage.InspectGeneratedArtifact(o.ImageBytes)
		if err != nil || len(o.ImageBytes) == 0 || len(o.ImageBytes) > productimage.MaxInlineArtifactBytes || (mime != "image/png" && mime != "image/jpeg") || (input.Artwork != nil || input.ImageSet != nil) && (w != 1024 || h != 1024) || input.Artwork != nil && len(o.ImageBytes) > 3<<20 {
			proof.ResultUnavailable = "invalid_result"
		} else {
			proof.ResultLocatorKind = "artifact_bundle"
			id := imageagent.SlotExternalEffectIdentity{RunScope: imageagent.RunScope{TenantID: input.TenantID, OwnerUserID: input.UserID, RunID: input.RunID}, PlanRevision: input.PlanRevision, SlotID: input.Slot.ID, Attempt: input.Attempt}
			asset := objectstore.ArtifactInput{Bytes: o.ImageBytes, ContentType: mime, Width: w, Height: h, ProviderReceiptID: o.ResultDigest, Operations: []string{productimage.SourceWhiteBackgroundOperation}}
			if input.Artwork != nil {
				var fact imageagent.GenerationFact
				fact, preserveErr = f.facts.ReadGenerationFact(ctx, id)
				if preserveErr == nil && (fact.Validate() != nil || fact.Intent.Identity != id || fact.State != imageagent.GenerationDispatchStarted && fact.State != imageagent.GenerationUnknown && fact.State != imageagent.GenerationSucceeded) {
					preserveErr = imageagent.ErrRevisionConflict
				}
				if preserveErr == nil {
					var provenance imageagent.ArtworkProvenance
					provenance, preserveErr = imageagent.NewArtworkProvenance(input, fact)
					if preserveErr == nil {
						asset.Artwork = &provenance
					}
				}
				asset.Operations = []string{"generate_artwork"}
			} else if len(input.Slot.SourceAssetIDs) > 0 {
				asset.SourceAssetID = input.Slot.SourceAssetIDs[0]
				if input.ImageSet != nil {
					asset.Operations = []string{productimage.SourceEditOperation}
				}
			} else {
				preserveErr = imageagent.ErrValidation
			}
			if preserveErr == nil {
				var prepared objectstore.PreparedSlotArtifacts
				prepared, preserveErr = f.artifacts.PrepareSlotArtifacts(objectstore.PrepareSlotArtifactsInput{Identity: id, Assets: []objectstore.ArtifactInput{asset}})
				// Pure artifact validation failure is an unusable returned image,
				// unlike a failed/ambiguous storage PUT after preparation.
				if preserveErr != nil {
					proof.ResultLocatorKind = ""
					proof.ResultUnavailable = "invalid_result"
				}
				if preserveErr == nil {
					preserveErr = f.artifacts.PreserveSlotArtifacts(ctx, id, prepared)
				}
				if preserveErr == nil {
					preserveErr = f.artifacts.EnsureStaged(ctx, prepared)
				}
			}
		}
	}
	if err := observe(ctx, proof); err != nil {
		return err
	}
	if proof.ResultUnavailable != "" {
		return imageagent.ErrInvalidGeneratedOutput
	}
	return preserveErr
}

func flareInputLimits(e aicapability.ReferenceImageEstimate) openai.FlareImageInputLimits {
	return openai.FlareImageInputLimits{Size: e.Size, Quality: e.Quality, AllowAutoSize: e.AllowAutoSize, MaximumPromptBytes: e.MaximumPromptBytes, MaximumReferenceImages: e.MaximumReferenceImages, MaximumReferenceBytes: e.MaximumReferenceBytes, MaximumReferencePixels: e.MaximumReferencePixels}
}
