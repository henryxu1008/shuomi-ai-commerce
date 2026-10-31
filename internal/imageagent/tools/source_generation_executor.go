package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"task-processor/internal/imageagent"
	imagepolicy "task-processor/internal/marketplace/imagepolicy"
	productimage "task-processor/internal/product/image"
)

type SourceGenerationDependencies struct {
	SourceEditor    productimage.SourceEditor
	WhiteBackground productimage.SourceWhiteBackground
	UsageQuoter     productimage.UsageQuoter
	ProfileResolver ProfileResolver
}
type SourceGenerationExecutor struct{ dependencies SourceGenerationDependencies }

func NewSourceGenerationExecutor(dependencies SourceGenerationDependencies) *SourceGenerationExecutor {
	return &SourceGenerationExecutor{dependencies: dependencies}
}
func (e *SourceGenerationExecutor) resolveInput(input imageagent.SlotExecutionInput) (resolvedSlotInput, error) {
	if e == nil {
		return resolvedSlotInput{}, imageagent.ErrValidation
	}
	return resolveSourceInput(input, e.dependencies.ProfileResolver)
}

type quotedSlotOperation struct {
	public     imageagent.SlotUsageOperation
	capability productimage.UsageQuote
}
type quotedSlotExecution struct {
	quote      imageagent.SlotUsageQuote
	operations []quotedSlotOperation
}

func (e *SourceGenerationExecutor) QuoteSlot(ctx context.Context, input imageagent.SlotExecutionInput, policy imageagent.BudgetPolicy) (imageagent.SlotUsageQuote, error) {
	if input.ImageSet != nil {
		if err := imageagent.ValidateImageMethodDispatch(input.ImageSet.Method); err != nil {
			return imageagent.SlotUsageQuote{}, err
		}
	}
	quoted, err := e.quoteSlot(ctx, input, policy)
	if err != nil {
		return imageagent.SlotUsageQuote{}, err
	}
	return quoted.quote, nil
}
func (e *SourceGenerationExecutor) quoteSlot(ctx context.Context, input imageagent.SlotExecutionInput, policy imageagent.BudgetPolicy) (quotedSlotExecution, error) {
	resolved, err := e.resolveInput(input)
	if err != nil {
		return quotedSlotExecution{}, err
	}
	if e == nil || e.dependencies.UsageQuoter == nil {
		return quotedSlotExecution{}, fmt.Errorf("%w: image usage quoter is required", imageagent.ErrBudgetQuoteUnavailable)
	}
	operations := []string{productimage.SourceWhiteBackgroundOperation}
	if resolved.slot.Recipe != nil {
		operations = []string{productimage.SourceEditOperation}
	}
	inputFingerprint := imageagent.SlotExecutionFingerprint(input)
	quoted := quotedSlotExecution{operations: make([]quotedSlotOperation, 0, len(operations))}
	pricingVersions := make([]string, 0, len(operations))
	for _, operation := range operations {
		request := productimage.UsageQuoteRequest{Operation: operation, InputFingerprint: inputFingerprint, MaximumOutputs: 1}
		capability, quoteErr := e.dependencies.UsageQuoter.QuoteUsage(ctx, request)
		if quoteErr != nil {
			return quotedSlotExecution{}, fmt.Errorf("%w: quote %s: %v", imageagent.ErrBudgetQuoteUnavailable, operation, quoteErr)
		}
		capability, quoteErr = productimage.NormalizeUsageAuthorization(capability, operation)
		if quoteErr != nil || capability.MaximumOutputs != 1 {
			return quotedSlotExecution{}, fmt.Errorf("%w: %s capability returned an invalid quote", imageagent.ErrBudgetQuoteUnavailable, operation)
		}
		if policy.CostMicros.Enabled && !capability.CostUpperBoundKnown {
			return quotedSlotExecution{}, fmt.Errorf("%w: %s capability has no trustworthy cost upper bound", imageagent.ErrBudgetQuoteUnavailable, operation)
		}
		maximum := imageagent.UsageVector{Images: 1, AgentSteps: 1, ModelCalls: capability.MaximumModelCalls}
		if capability.CostUpperBoundKnown {
			maximum.CostMicros = capability.MaximumCostMicros
		}
		public := imageagent.SlotUsageOperation{
			Name: operation, Provider: capability.Provider, Model: capability.Model,
			PricingVersion: capability.PricingVersion, Fingerprint: capability.Fingerprint,
			Maximum: maximum, MaximumOutputs: capability.MaximumOutputs,
		}
		quoted.quote.Maximum, err = imageagent.CheckedAddUsage(quoted.quote.Maximum, maximum)
		if err != nil {
			return quotedSlotExecution{}, err
		}
		quoted.operations = append(quoted.operations, quotedSlotOperation{public: public, capability: capability})
		quoted.quote.Operations = append(quoted.quote.Operations, public)
		pricingVersions = append(pricingVersions, capability.PricingVersion)
	}
	quoted.quote.PricingVersion = strings.Join(pricingVersions, "+")
	encoded, err := json.Marshal(struct {
		InputFingerprint string
		Policy           imagepolicy.ProductImageProfile
		Operations       []imageagent.SlotUsageOperation
		PricingVersion   string
	}{inputFingerprint, resolved.profile, quoted.quote.Operations, quoted.quote.PricingVersion})
	if err != nil {
		return quotedSlotExecution{}, err
	}
	digest := sha256.Sum256(encoded)
	quoted.quote.Fingerprint = hex.EncodeToString(digest[:])
	if err := imageagent.ValidateSlotUsageQuote(quoted.quote); err != nil {
		return quotedSlotExecution{}, err
	}
	return quoted, nil
}
func (e *SourceGenerationExecutor) GenerateQuotedSlot(ctx context.Context, input imageagent.SlotExecutionInput, expected imageagent.SlotUsageQuote) (imageagent.SlotGeneratedOutput, error) {
	if err := imageagent.ValidateSlotUsageQuote(expected); err != nil {
		return imageagent.SlotGeneratedOutput{}, providerError(imageagent.ProviderNotDispatched, err)
	}
	quoted, err := e.quoteSlot(ctx, input, imageagent.BudgetPolicy{})
	if err != nil {
		return imageagent.SlotGeneratedOutput{}, providerError(imageagent.ProviderNotDispatched, err)
	}
	if quoted.quote.Fingerprint != expected.Fingerprint {
		return imageagent.SlotGeneratedOutput{}, providerError(imageagent.ProviderNotDispatched, imageagent.ErrRevisionConflict)
	}
	return e.generateSlot(ctx, input, &quoted)
}
func (e *SourceGenerationExecutor) GenerateSlot(_ context.Context, input imageagent.SlotExecutionInput) (imageagent.SlotGeneratedOutput, error) {
	if _, err := e.resolveInput(input); err != nil {
		return imageagent.SlotGeneratedOutput{}, providerError(imageagent.ProviderRejectedBeforeEffect, err)
	}
	return imageagent.SlotGeneratedOutput{}, providerError(imageagent.ProviderNotDispatched, imageagent.ErrBudgetQuoteUnavailable)
}
func (e *SourceGenerationExecutor) generateSlot(ctx context.Context, input imageagent.SlotExecutionInput, quoted *quotedSlotExecution) (imageagent.SlotGeneratedOutput, error) {
	if input.ImageSet != nil {
		if err := imageagent.ValidateImageMethodDispatch(input.ImageSet.Method); err != nil {
			return imageagent.SlotGeneratedOutput{}, providerError(imageagent.ProviderRejectedBeforeEffect, err)
		}
	}
	resolved, err := e.resolveInput(input)
	if err != nil {
		return imageagent.SlotGeneratedOutput{}, err
	}
	var candidates []productimage.Candidate
	var receipt imageagent.SlotUsageReceipt
	if resolved.slot.Recipe != nil {
		if input.ImageSet != nil && input.ImageSet.Method != nil {
			resolved.providerPrompt, err = imageagent.ComposeImageMethodPrompt(resolved.slot.Recipe.Prompt, input.ImageMethodText)
			if err != nil {
				return imageagent.SlotGeneratedOutput{}, providerError(imageagent.ProviderRejectedBeforeEffect, err)
			}
		}
		candidates, receipt, err = e.generateSourceSet(ctx, resolved, quoted)
	} else {
		candidates, receipt, err = e.generateMain(ctx, resolved, quoted)
	}

	if err != nil {
		return imageagent.SlotGeneratedOutput{}, fmt.Errorf("execute slot %q: %w", resolved.slot.ID, err)
	}
	assets := make([]imageagent.GeneratedAsset, len(candidates))
	for index, candidate := range candidates {
		asset, mapErr := generatedAsset(candidate, resolved.source, resolved.otherSources)
		if mapErr != nil {
			return imageagent.SlotGeneratedOutput{}, mapErr
		}
		assets[index] = asset
	}
	output := imageagent.SlotGeneratedOutput{
		SlotID: resolved.slot.ID, Attempt: input.Attempt, SourceAssetID: resolved.sourceAssetID,
		Assets: assets, UsageReceipt: receipt,
	}
	return output, nil
}
func (e *SourceGenerationExecutor) generateMain(ctx context.Context, input resolvedSlotInput, quoted *quotedSlotExecution) ([]productimage.Candidate, imageagent.SlotUsageReceipt, error) {
	if len(input.sourceBytes) == 0 || len(input.sourceBytes) > productimage.MaxInlineArtifactBytes {
		return nil, imageagent.SlotUsageReceipt{}, providerError(imageagent.ProviderRejectedBeforeEffect, imageagent.ErrValidation)
	}
	digest := sha256.Sum256(input.sourceBytes)
	if hex.EncodeToString(digest[:]) != input.sourceDigest {
		return nil, imageagent.SlotUsageReceipt{}, providerError(imageagent.ProviderRejectedBeforeEffect, imageagent.ErrRevisionConflict)
	}
	if e.dependencies.WhiteBackground == nil {
		return nil, imageagent.SlotUsageReceipt{}, imageagent.ErrValidation
	}
	white, err := e.dependencies.WhiteBackground.EditSourceWhiteBackground(ctx, productimage.SourceWhiteBackgroundRequest{
		Source: input.source, SourceBytes: append([]byte(nil), input.sourceBytes...), Product: input.product,
		Authorization: capabilityAuthorization(quoted, productimage.SourceWhiteBackgroundOperation),
	})
	if err != nil {
		// An ordinary provider error is not proof of no side effect.
		return nil, imageagent.SlotUsageReceipt{}, providerError(imageagent.ProviderDispatchedUnknown, err)
	}
	return []productimage.Candidate{white}, receiptForQuote(quoted, 1, []productimage.Candidate{white}), nil
}
func (e *SourceGenerationExecutor) generateSourceSet(ctx context.Context, input resolvedSlotInput, quoted *quotedSlotExecution) ([]productimage.Candidate, imageagent.SlotUsageReceipt, error) {
	if e.dependencies.SourceEditor == nil || quoted == nil {
		return nil, imageagent.SlotUsageReceipt{}, providerError(imageagent.ProviderRejectedBeforeEffect, imageagent.ErrValidation)
	}
	if err := imageagent.ValidateImageSourceBytes(input.slot.Recipe, input.references, productimage.MaxInlineArtifactBytes); err != nil {
		return nil, imageagent.SlotUsageReceipt{}, providerError(imageagent.ProviderRejectedBeforeEffect, err)
	}
	prompt := input.slot.Recipe.Prompt
	if input.providerPrompt != "" {
		prompt = input.providerPrompt
	}
	candidate, err := e.dependencies.SourceEditor.EditSources(ctx, productimage.SourceEditRequest{Sources: input.sources, ReferenceBytes: input.references, Product: input.product, Prompt: prompt, PromptVersion: input.slot.Recipe.PromptVersion, Authorization: capabilityAuthorization(quoted, productimage.SourceEditOperation)})
	if err != nil {
		return nil, imageagent.SlotUsageReceipt{}, providerError(imageagent.ProviderDispatchedUnknown, err)
	}
	candidates := []productimage.Candidate{candidate}
	return candidates, receiptForQuote(quoted, 1, candidates), nil
}
func capabilityAuthorization(quoted *quotedSlotExecution, operation string) *productimage.UsageQuote {
	if quoted == nil {
		return nil
	}
	for _, candidate := range quoted.operations {
		if candidate.capability.Operation == operation {
			authorization := candidate.capability
			return &authorization
		}
	}
	return nil
}
func receiptForQuote(quoted *quotedSlotExecution, actualImages int64, candidates []productimage.Candidate) imageagent.SlotUsageReceipt {
	if quoted == nil {
		return imageagent.SlotUsageReceipt{}
	}
	receipt := imageagent.SlotUsageReceipt{Actual: quoted.quote.Maximum, CostBasis: imageagent.UsageCostReservedUpperBound}
	receipt.Actual.Images = actualImages
	for _, operation := range quoted.operations {
		if !operation.capability.CostUpperBoundKnown {
			receipt.Actual.CostMicros = 0
			receipt.CostBasis = imageagent.UsageCostUnavailable
			break
		}
	}
	for _, candidate := range candidates {
		if id := strings.TrimSpace(candidate.Metadata.InvocationID); id != "" {
			receipt.ProviderRequestIDs = append(receipt.ProviderRequestIDs, id)
		}
	}
	return receipt
}

func (e *SourceGenerationExecutor) BuildSlotResult(ctx context.Context, input imageagent.SlotExecutionInput, published imageagent.PublishedSlotOutput) (imageagent.SlotExecutionResult, error) {
	if e == nil {
		return imageagent.SlotExecutionResult{}, imageagent.ErrValidation
	}
	if input.ImageSet != nil {
		return NewImageSetResultBuilder().BuildSlotResult(ctx, input, published)
	}
	return NewAcquisitionMainResultBuilder(e.dependencies.ProfileResolver).BuildSlotResult(ctx, input, published)
}
