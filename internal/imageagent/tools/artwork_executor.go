package tools

import (
	"context"
	"reflect"

	"task-processor/internal/imageagent"
)

// ArtworkCreator receives only the prompt from a verified immutable plan.
// Provider, credentials, dispatch observation and downloads stay in Integration.
type ArtworkCreator interface {
	CreateArtwork(context.Context, string) (imageagent.GeneratedAsset, error)
}

type ArtworkSlotExecutor struct{ creator ArtworkCreator }

func NewArtworkSlotExecutor(creator ArtworkCreator) *ArtworkSlotExecutor {
	return &ArtworkSlotExecutor{creator: creator}
}

func (e *ArtworkSlotExecutor) QuoteSlot(_ context.Context, input imageagent.SlotExecutionInput, policy imageagent.BudgetPolicy) (imageagent.SlotUsageQuote, error) {
	if e == nil || e.creator == nil || imageagent.ValidateArtworkExecution(input) != nil {
		return imageagent.SlotUsageQuote{}, imageagent.ErrValidation
	}
	// This fixed consumer has a point tariff, not a fabricated money quote.
	if policy.CostMicros.Enabled {
		return imageagent.SlotUsageQuote{}, imageagent.ErrBudgetQuoteUnavailable
	}
	pricing := input.Artwork.Quote
	fingerprint, err := imageagent.ArtworkGenerationInputDigest(input)
	if err != nil {
		return imageagent.SlotUsageQuote{}, err
	}
	maximum := imageagent.UsageVector{Images: 1, AgentSteps: 1, ModelCalls: 1}
	return imageagent.SlotUsageQuote{Maximum: maximum, PricingVersion: pricing.PriceVersion, Fingerprint: fingerprint, Operations: []imageagent.SlotUsageOperation{{Name: "generate_artwork", Provider: pricing.Provider, Model: pricing.Model, PricingVersion: pricing.PriceVersion, Fingerprint: fingerprint, Maximum: maximum, MaximumOutputs: 1}}}, nil
}

func (e *ArtworkSlotExecutor) GenerateQuotedSlot(ctx context.Context, input imageagent.SlotExecutionInput, expected imageagent.SlotUsageQuote) (imageagent.SlotGeneratedOutput, error) {
	quote, err := e.QuoteSlot(ctx, input, imageagent.BudgetPolicy{})
	if err != nil {
		return imageagent.SlotGeneratedOutput{}, err
	}
	if !reflect.DeepEqual(quote, expected) {
		return imageagent.SlotGeneratedOutput{}, imageagent.ErrRevisionConflict
	}
	asset, err := e.creator.CreateArtwork(ctx, input.Artwork.Prompt)
	if err != nil {
		return imageagent.SlotGeneratedOutput{}, err
	}
	if asset.Artwork != nil || asset.SourceURL != "" || asset.URL != "" || len(asset.Bytes) == 0 || !reflect.DeepEqual(asset.Operations, []string{"generate_artwork"}) {
		return imageagent.SlotGeneratedOutput{}, imageagent.ErrProviderContractViolation
	}
	return imageagent.SlotGeneratedOutput{SlotID: input.Slot.ID, Attempt: input.Attempt, Assets: []imageagent.GeneratedAsset{asset}, UsageReceipt: imageagent.SlotUsageReceipt{Actual: quote.Maximum, ProviderRequestIDs: []string{asset.ProviderReceiptID}, CostBasis: imageagent.UsageCostUnavailable}}, nil
}

func (*ArtworkSlotExecutor) GenerateSlot(context.Context, imageagent.SlotExecutionInput) (imageagent.SlotGeneratedOutput, error) {
	return imageagent.SlotGeneratedOutput{}, imageagent.ErrCommandBlocked
}

func (*ArtworkSlotExecutor) BuildSlotResult(_ context.Context, input imageagent.SlotExecutionInput, output imageagent.PublishedSlotOutput) (imageagent.SlotExecutionResult, error) {
	if imageagent.ValidateArtworkExecution(input) != nil || output.SlotID != input.Slot.ID || output.Attempt != input.Attempt || len(output.Assets) != 1 {
		return imageagent.SlotExecutionResult{}, imageagent.ErrValidation
	}
	a := output.Assets[0]
	if imageagent.ValidatePublishedAssetRefForSlot(input, a, 0) != nil {
		return imageagent.SlotExecutionResult{}, imageagent.ErrRevisionConflict
	}
	return imageagent.SlotExecutionResult{SlotID: input.Slot.ID, Attempt: input.Attempt, Candidates: []imageagent.AssetCandidate{{AssetID: durableCandidateAssetID(input, input.Slot, a, 0), Artwork: a.Artwork, DurableAsset: imageagent.DurableAssetIdentity{ObjectKey: a.ObjectKey, SHA256: a.SHA256}, Width: a.Width, Height: a.Height, Operations: append([]string(nil), a.Operations...)}}}, nil
}

var _ imageagent.BudgetedStagedSlotExecutor = (*ArtworkSlotExecutor)(nil)
