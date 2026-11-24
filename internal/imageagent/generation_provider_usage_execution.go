package imageagent

import (
	"context"
	"errors"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability"
)

type ProviderUsageImageResult struct {
	TaskID, Status, ResultURL, Digest string
}

type ProviderUsageImageTasks interface {
	StartProductImage(context.Context, string, []byte) (string, error)
	LookupProductImage(context.Context, string) (ProviderUsageImageResult, error)
}

func (m GenerationProviderMetadata) ImageQuote() ImageGenerationQuote {
	quality := GenerationRouteQuality(m.Provider, m.Model, m.Protocol)
	if m.BillingMode != "" {
		quality = "auto"
	}
	return ImageGenerationQuote{Billing: m.Billing, Provider: m.Provider, Model: m.Model, Protocol: m.Protocol,
		Resolution: "1024x1024", Quality: quality, PriceVersion: m.PriceVersion, Points: m.Points,
		RouteReference: m.RouteReference, CredentialReference: m.CredentialReference, ConfigurationVersion: m.ConfigurationVersion,
		BillingMode: m.BillingMode, ConnectionID: m.ConnectionID}
}

func quoteProviderUsageImage(p PreparedGenerationProvider, input SlotExecutionInput) (SlotUsageQuote, SlotUsageQuote, error) {
	q := p.Metadata.ImageQuote()
	slot := input.Slot
	if p.ProviderUsage == nil || !q.Valid() || !q.UsesProviderUsage() || input.Artwork != nil || input.ImageSet == nil ||
		!input.ImageSet.Source.OwnProduct || input.ImageSet.Source.ContextKind != ImageSourceListing ||
		input.TargetPlatform != "product" || input.ImageSet.Target.Platform != "product" || input.ImageSet.Route == nil ||
		input.ImageSet.MaxPoints != 0 || slot.Recipe == nil || slot.Recipe.Quote != q ||
		slot.Role != SlotRoleMain || slot.Recipe.Purpose != "product_identity" ||
		slot.Recipe.Placement != (ImagePlacement{Group: "carousel", Order: 1}) ||
		len(slot.SourceAssetIDs) != 1 || len(slot.Recipe.References) != 1 {
		return SlotUsageQuote{}, SlotUsageQuote{}, ErrBudgetQuoteUnavailable
	}
	ref := slot.Recipe.References[0]
	if ref.AssetID != slot.SourceAssetIDs[0] || !validProviderUsageSource(ref) {
		return SlotUsageQuote{}, SlotUsageQuote{}, ErrBudgetQuoteUnavailable
	}
	maximum := UsageVector{Images: 1, ModelCalls: 1}
	fingerprint := generationHash(struct {
		Quote ImageGenerationQuote
		Input string
	}{q, SlotExecutionFingerprint(input)})
	operation := SlotUsageOperation{Name: "render_source_edit", Provider: q.Provider, Model: q.Model,
		Fingerprint: fingerprint, Maximum: maximum, MaximumOutputs: 1}
	quote := SlotUsageQuote{Maximum: maximum, Fingerprint: fingerprint, Operations: []SlotUsageOperation{operation}}
	return quote, quote, nil
}

func validProviderUsageSource(ref ImageSourceObservation) bool {
	return ref.Bytes > 0 && ref.Bytes <= aicapability.GRSaiProductImageMaxSourceBytes &&
		(ref.MediaType == "image/jpeg" || ref.MediaType == "image/png")
}

func (e *GenerationExecution) generateProviderUsage(ctx context.Context, input SlotExecutionInput, intent GenerationIntent, p PreparedGenerationProvider) (SlotGeneratedOutput, error) {
	prompt := input.Slot.Recipe.Prompt
	if input.ImageMethodText != "" {
		var err error
		prompt, err = ComposeImageMethodPrompt(prompt, input.ImageMethodText)
		if err != nil {
			return e.recordProviderUsagePreflightFailure(ctx, intent, err)
		}
	}
	id, err := p.ProviderUsage.StartProductImage(ctx, prompt, input.SourceBytes)
	if err != nil {
		if ProviderDispatchStateOf(err) == ProviderRejectedBeforeEffect {
			return e.recordProviderUsagePreflightFailure(ctx, intent, err)
		}
		return e.markProviderUsageUnknown(ctx, intent, err)
	}
	facts := e.dependencies.Facts.(GenerationProviderTaskRepository)
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	fact, err := facts.RecordGenerationProviderTask(saveCtx, intent, id, time.Now().UTC())
	cancel()
	if err != nil {
		return e.markProviderUsageUnknown(ctx, intent, err)
	}
	for attempt := 0; attempt < 12; attempt++ {
		if attempt > 0 {
			delay := time.NewTimer(5 * time.Second)
			select {
			case <-ctx.Done():
				delay.Stop()
				return e.markProviderUsageUnknown(ctx, intent, ctx.Err())
			case <-delay.C:
			}
		}
		fact, err = e.lookupProviderUsage(ctx, input, fact, p)
		if err != nil {
			return e.markProviderUsageUnknown(ctx, intent, err)
		}
		switch fact.State {
		case GenerationSucceeded:
			output, err := e.ReadGenerationOutput(ctx, input, fact)
			if err != nil {
				return generationUnknown(err)
			}
			output.UsageReceipt = SlotUsageReceipt{Actual: UsageVector{Images: 1, ModelCalls: 1}, ProviderRequestIDs: []string{fact.ProviderTask.ID}, CostBasis: UsageCostUnavailable}
			return output, nil
		case GenerationSupplierFailed:
			return generationUnknown(ErrCommandBlocked)
		}
	}
	return e.markProviderUsageUnknown(ctx, intent, ErrCommandBlocked)
}

func (e *GenerationExecution) recordProviderUsagePreflightFailure(ctx context.Context, intent GenerationIntent, cause error) (SlotGeneratedOutput, error) {
	recorder, ok := e.dependencies.Facts.(GenerationRejectionRecorder)
	if !ok {
		return e.markProviderUsageUnknown(ctx, intent, cause)
	}
	finalCtx, cancel := generationFinalizationContext(ctx)
	defer cancel()
	proof := GenerationRejection{Kind: "source_edit_preflight", Digest: generationHash([]string{intent.Protocol, intent.InputDigest, "preflight"})}
	fact, err := recorder.RecordGenerationRejection(finalCtx, intent, proof)
	if err != nil {
		return generationUnknown(err)
	}
	if err := closeGenerationCapacity(finalCtx, e.dependencies.StorageCapacity, fact); err != nil {
		return generationUnknown(err)
	}
	return SlotGeneratedOutput{}, &ProviderDispatchError{State: ProviderRejectedBeforeEffect, Err: cause}
}

// ReadGenerationOutput shares output inspection between the initial execution
// and GET-only recovery. Record definitive failures before closing unused storage.
func (e *GenerationExecution) ReadGenerationOutput(ctx context.Context, input SlotExecutionInput, fact GenerationFact) (SlotGeneratedOutput, error) {
	if fact.ProviderOutputInvalid {
		return SlotGeneratedOutput{}, ErrInvalidGeneratedOutput
	}
	output, err := e.dependencies.ReadProviderOutput(ctx, input, fact)
	if !fact.Intent.UsesProviderUsage() || !errors.Is(err, ErrInvalidGeneratedOutput) {
		return output, err
	}
	repository, ok := e.dependencies.Facts.(GenerationProviderTaskRepository)
	if !ok {
		return SlotGeneratedOutput{}, ErrCommandBlocked
	}
	finalCtx, cancel := generationFinalizationContext(ctx)
	defer cancel()
	invalid, err := repository.RecordInvalidGenerationProviderOutput(finalCtx, fact.Intent, fact.TerminalProofDigest())
	if err != nil {
		return SlotGeneratedOutput{}, err
	}
	if err := closeGenerationCapacity(finalCtx, e.dependencies.StorageCapacity, invalid); err != nil {
		return SlotGeneratedOutput{}, err
	}
	return SlotGeneratedOutput{}, ErrInvalidGeneratedOutput
}

// RecoverProviderUsage only queries a previously accepted task. The worker
// invokes it after restoring and checking the original live execution identity.
func (e *GenerationExecution) RecoverProviderUsage(ctx context.Context, input SlotExecutionInput, fact GenerationFact) (GenerationFact, error) {
	if e == nil || !fact.Intent.UsesProviderUsage() || fact.ProviderTask.ID == "" || fact.TerminalProofDigest() != "" {
		return fact, nil
	}
	if err := e.authorize(ctx, input); err != nil {
		return fact, err
	}
	source, err := e.dependencies.ReadSourceBytes(ctx, input)
	if err != nil {
		return fact, err
	}
	if err := ValidateImageSourceBytes(input.Slot.Recipe, [][]byte{source}, e.dependencies.MaxSourceBytes); err != nil {
		return fact, err
	}
	p, err := e.dependencies.PrepareProvider(ctx, input, func(context.Context, GenerationSuccess) error { return ErrProviderContractViolation })
	if err != nil {
		return fact, err
	}
	return e.lookupProviderUsage(ctx, input, fact, p)
}

func (e *GenerationExecution) lookupProviderUsage(ctx context.Context, input SlotExecutionInput, fact GenerationFact, p PreparedGenerationProvider) (GenerationFact, error) {
	digest, err := ImageSlotGenerationInputDigest(input)
	if err != nil || fact.Validate() != nil || !fact.Intent.UsesProviderUsage() || fact.Intent.InputDigest != digest ||
		!p.Metadata.ImageQuote().MatchesGenerationIntent(fact.Intent) || p.ProviderUsage == nil ||
		fact.ProviderTask.ID == "" || !time.Now().Before(fact.ProviderTask.AcceptedAt.Add(2*time.Hour)) {
		return fact, ErrRevisionConflict
	}
	if err := e.authorize(ctx, input); err != nil {
		return fact, err
	}
	if err := e.dependencies.RevalidateProvider(ctx, input, p.Metadata); err != nil {
		return fact, err
	}
	result, err := p.ProviderUsage.LookupProductImage(ctx, fact.ProviderTask.ID)
	if err != nil || result.TaskID != fact.ProviderTask.ID {
		return fact, ErrProviderContractViolation
	}
	finalCtx, cancel := generationFinalizationContext(ctx)
	defer cancel()
	switch result.Status {
	case "running":
		return e.dependencies.Facts.ReadGenerationFact(finalCtx, fact.Intent.Identity)
	case "succeeded":
		success := GenerationSuccess{ResponseID: result.TaskID, RequestID: result.TaskID,
			ResultDigest: result.Digest, ProviderImageCount: 1}.WithResultLocator(result.ResultURL, "")
		fact, err = e.dependencies.Facts.RecordGenerationSuccess(finalCtx, fact.Intent, success)
	case "failed", "violation":
		repository, ok := e.dependencies.Facts.(GenerationProviderTaskRepository)
		if !ok {
			return fact, ErrCommandBlocked
		}
		fact, err = repository.RecordGenerationProviderFailure(finalCtx, fact.Intent, result.TaskID, result.Status, result.Digest)
	default:
		return fact, ErrProviderContractViolation
	}
	if err == nil {
		err = closeGenerationCapacity(finalCtx, e.dependencies.StorageCapacity, fact)
	}
	return fact, err
}

func (e *GenerationExecution) markProviderUsageUnknown(ctx context.Context, intent GenerationIntent, cause error) (SlotGeneratedOutput, error) {
	finalCtx, cancel := generationFinalizationContext(ctx)
	defer cancel()
	if _, err := e.dependencies.Facts.MarkGenerationUnknown(finalCtx, intent); err != nil {
		return generationUnknown(err)
	}
	return generationUnknown(cause)
}
