package tools

import (
	"context"
	"task-processor/internal/imageagent"
)

type ImageResultBuilder struct{ profiles ProfileResolver }

func NewAcquisitionMainResultBuilder(profiles ProfileResolver) *ImageResultBuilder {
	return &ImageResultBuilder{profiles: profiles}
}
func NewImageSetResultBuilder() *ImageResultBuilder { return &ImageResultBuilder{} }
func (e *ImageResultBuilder) BuildSlotResult(_ context.Context, input imageagent.SlotExecutionInput, published imageagent.PublishedSlotOutput) (imageagent.SlotExecutionResult, error) {
	resolved, err := resolveSourceInput(input, e.profiles)
	if err != nil {
		return imageagent.SlotExecutionResult{}, err
	}
	if published.SlotID != resolved.slot.ID || published.Attempt != input.Attempt {
		return imageagent.SlotExecutionResult{}, imageagent.ErrRevisionConflict
	}
	manifest, err := imageagent.NormalizeFinalManifest(imageagent.FinalManifest{Assets: published.Assets})
	if err != nil || len(manifest.Assets) == 0 || resolved.slot.Role == imageagent.SlotRoleMain && len(manifest.Assets) != 1 {
		return imageagent.SlotExecutionResult{}, imageagent.ErrValidation
	}
	candidates := make([]imageagent.AssetCandidate, len(manifest.Assets))
	for index, asset := range manifest.Assets {
		if err := imageagent.ValidatePublishedAssetRefForSlot(input, asset, index); err != nil || asset.SourceAssetID != resolved.sourceAssetID {
			return imageagent.SlotExecutionResult{}, imageagent.ErrRevisionConflict
		}
		candidates[index] = imageagent.AssetCandidate{
			AssetID: durableCandidateAssetID(input, resolved.slot, asset, index), SourceAssetID: asset.SourceAssetID,
			Width: asset.Width, Height: asset.Height, Operations: append([]string(nil), asset.Operations...),
			DurableAsset: imageagent.DurableAssetIdentity{ObjectKey: asset.ObjectKey, SHA256: asset.SHA256},
		}
	}
	return imageagent.SlotExecutionResult{SlotID: resolved.slot.ID, Attempt: input.Attempt, Candidates: candidates}, nil
}
