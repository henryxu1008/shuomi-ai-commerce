package imageagentworker

import (
	"context"
	d "task-processor/internal/aicapability/routeconfig"
	appruntime "task-processor/internal/app/runtime"
	"task-processor/internal/core/config"
	"task-processor/internal/imageagent"
	imagestore "task-processor/internal/imageagent/store"
	imagetools "task-processor/internal/imageagent/tools"
	productasset "task-processor/internal/product/asset"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// The current application supplies its existing source/rule and Asset owners.
// This worker consumes only the set protocol and never assembles retired
// Extract/Render/Review capabilities or another approval inventory.
func NewImageSetTemporalDependencies(cfg *config.Config, imageDB, resourceDB *gorm.DB, authorizer imageagent.ExecutionAuthorizer, contexts imageagent.ImageSetContextReader, publisher imageagent.ApprovedImageSetPublisher, approvals productasset.ApprovalCommitReader, logger *logrus.Logger, methods ...imageagent.ImageMethodContext) (appruntime.ImageAgentTemporalDependencies, error) {
	return NewManagedImageSetTemporalDependencies(cfg, imageDB, resourceDB, authorizer, contexts, publisher, approvals, logger, nil, methods...)
}
func NewManagedImageSetTemporalDependencies(cfg *config.Config, imageDB, resourceDB *gorm.DB, authorizer imageagent.ExecutionAuthorizer, contexts imageagent.ImageSetContextReader, publisher imageagent.ApprovedImageSetPublisher, approvals productasset.ApprovalCommitReader, logger *logrus.Logger, current d.CurrentReader, methods ...imageagent.ImageMethodContext) (appruntime.ImageAgentTemporalDependencies, error) {
	var empty appruntime.ImageAgentTemporalDependencies
	if len(methods) > 1 || cfg == nil || imageDB == nil || resourceDB == nil || authorizer == nil || contexts == nil || publisher == nil || approvals == nil || !cfg.ImageAgent.Generation.Configured() {
		return empty, imageagent.ErrValidation
	}
	repository := imagestore.NewOrganizationRepository(imageDB)
	executor, recovery, err := buildOrganizationGenerationWithCurrent(cfg.ImageAgent.Generation, imageDB, resourceDB, repository, authorizer, logger, current, contexts)
	if err != nil {
		return empty, err
	}
	if len(methods) == 1 {
		executor.WithImageMethodContext(methods[0])
	}
	artifacts, err := buildImageAgentDurableArtifactStore(cfg, defaultImageAgentArtifactTiming, logger)
	if err != nil {
		return empty, err
	}
	if err = configureStorageCapacity(cfg, resourceDB, repository, artifacts, executor, recovery); err != nil {
		return empty, err
	}
	slots := imageSetSlotExecutor{generation: executor, builder: imagetools.NewImageSetResultBuilder()}
	return appruntime.ImageAgentTemporalDependencies{Repository: repository, ExecutionAuthorizer: authorizer, SlotExecutor: slots, StagedSlotExecutor: slots, ArtifactStore: artifacts, ImageSetPublisher: publisher, ImageSetApprovals: approvals, GenerationRecovery: recovery, GenerationOutputRecovery: generationOutputRecovery(nil, logger), PublicationLeaseDuration: defaultImageAgentArtifactTiming.PublicationLeaseDuration}, nil
}

type imageSetSlotExecutor struct {
	generation *imageagent.GenerationExecution
	builder    *imagetools.ProductImageSlotExecutor
}

func (e imageSetSlotExecutor) ExecuteSlot(context.Context, imageagent.SlotExecutionInput) (imageagent.SlotExecutionResult, error) {
	return imageagent.SlotExecutionResult{}, imageagent.ErrCommandBlocked
}
func (e imageSetSlotExecutor) GenerateSlot(context.Context, imageagent.SlotExecutionInput) (imageagent.SlotGeneratedOutput, error) {
	return imageagent.SlotGeneratedOutput{}, imageagent.ErrCommandBlocked
}
func (e imageSetSlotExecutor) QuoteSlot(ctx context.Context, input imageagent.SlotExecutionInput, policy imageagent.BudgetPolicy) (imageagent.SlotUsageQuote, error) {
	if input.ImageSet == nil || e.generation == nil {
		if input.Artwork == nil || imageagent.ValidateArtworkExecution(input) != nil || e.generation == nil {
			return imageagent.SlotUsageQuote{}, imageagent.ErrCommandBlocked
		}
	}
	return e.generation.QuoteSlot(ctx, input, policy)
}
func (e imageSetSlotExecutor) GenerateQuotedSlot(ctx context.Context, input imageagent.SlotExecutionInput, quote imageagent.SlotUsageQuote) (imageagent.SlotGeneratedOutput, error) {
	if input.ImageSet == nil || e.generation == nil {
		if input.Artwork == nil || imageagent.ValidateArtworkExecution(input) != nil || e.generation == nil {
			return imageagent.SlotGeneratedOutput{}, imageagent.ErrCommandBlocked
		}
	}
	return e.generation.GenerateQuotedSlot(ctx, input, quote)
}
func (e imageSetSlotExecutor) BuildSlotResult(ctx context.Context, input imageagent.SlotExecutionInput, output imageagent.PublishedSlotOutput) (imageagent.SlotExecutionResult, error) {
	if input.Artwork != nil {
		return imagetools.NewArtworkSlotExecutor(nil).BuildSlotResult(ctx, input, output)
	}
	if input.ImageSet == nil || e.builder == nil {
		return imageagent.SlotExecutionResult{}, imageagent.ErrCommandBlocked
	}
	return e.builder.BuildSlotResult(ctx, input, output)
}
