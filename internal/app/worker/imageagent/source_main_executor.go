package imageagentworker

import (
	"context"

	"task-processor/internal/imageagent"
	imagetools "task-processor/internal/imageagent/tools"
	imagepolicy "task-processor/internal/marketplace/imagepolicy"
)

// SourceMainExecutor is the current #487 generation admission seam.
// The cancelled Extract/Render/Review flow has no production consumer. In
// particular, this executor must never reserve tokens for a future Review.
// Only explicit point-priced source generation can replace the closed seam.
type SourceMainExecutor struct {
	builder    *imagetools.ImageResultBuilder
	generation *imageagent.GenerationExecution
}

func (e SourceMainExecutor) QuoteSlot(ctx context.Context, input imageagent.SlotExecutionInput, policy imageagent.BudgetPolicy) (imageagent.SlotUsageQuote, error) {
	if err := imageagent.ValidateAcquisitionMainExecution(input); err != nil {
		return imageagent.SlotUsageQuote{}, err
	}
	if e.generation != nil {
		return e.generation.QuoteSlot(ctx, input, policy)
	}
	return imageagent.SlotUsageQuote{}, imageagent.ErrBudgetQuoteUnavailable
}

func (e SourceMainExecutor) GenerateQuotedSlot(ctx context.Context, input imageagent.SlotExecutionInput, quote imageagent.SlotUsageQuote) (imageagent.SlotGeneratedOutput, error) {
	if err := imageagent.ValidateAcquisitionMainExecution(input); err != nil {
		return imageagent.SlotGeneratedOutput{}, &imageagent.ProviderDispatchError{State: imageagent.ProviderRejectedBeforeEffect, Err: err}
	}
	if e.generation != nil {
		return e.generation.GenerateQuotedSlot(ctx, input, quote)
	}
	return imageagent.SlotGeneratedOutput{}, &imageagent.ProviderDispatchError{State: imageagent.ProviderNotDispatched, Err: imageagent.ErrBudgetQuoteUnavailable}
}

func (SourceMainExecutor) GenerateSlot(context.Context, imageagent.SlotExecutionInput) (imageagent.SlotGeneratedOutput, error) {
	return imageagent.SlotGeneratedOutput{}, &imageagent.ProviderDispatchError{State: imageagent.ProviderNotDispatched, Err: imageagent.ErrBudgetQuoteUnavailable}
}

func (e SourceMainExecutor) BuildSlotResult(ctx context.Context, input imageagent.SlotExecutionInput, published imageagent.PublishedSlotOutput) (imageagent.SlotExecutionResult, error) {
	if e.builder == nil {
		return imageagent.SlotExecutionResult{}, imageagent.ErrValidation
	}
	return e.builder.BuildSlotResult(ctx, input, published)
}

// ProfileResolver consumes only the current explicit image-policy key.
type ProfileResolver interface {
	Resolve(imagepolicy.ProfileInput) (imagepolicy.ProductImageProfile, error)
}

const imageAgentOpenAIClientName = "image_gpt_image_2"
