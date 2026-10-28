package imageagentworker

import (
	"context"
	"testing"

	"task-processor/internal/authidentity"
	"task-processor/internal/imageagent"
	"task-processor/internal/shared/aiidentity"

	"github.com/stretchr/testify/require"
)

func TestOrganizationMainRetiredReviewAdmissionCannotReserveAcrossGeneration(t *testing.T) {
	for _, platform := range []string{"product", "shein", ""} {
		t.Run(platform, func(t *testing.T) {
			delegate := &recordingMainExecutor{}
			executor := organizationMainSlotExecutor{delegate: delegate}
			input := mainAdmissionInput()
			input.TargetPlatform = platform
			input.ImagePolicyContext = &imageagent.ImagePolicyContext{Country: "zz", Family: "default", SceneCategory: "general"}
			if platform == "shein" {
				input.ImagePolicyContext = &imageagent.ImagePolicyContext{Country: "us", Family: "default", SceneCategory: "shoes"}
			}
			ctx, cancel := context.WithCancel(mainAdmissionContext())
			for _, cancelled := range []bool{false, true} {
				if cancelled {
					cancel()
				}
				_, err := executor.GenerateQuotedSlot(ctx, input, mainGenerationQuote())
				require.ErrorIs(t, err, imageagent.ErrBudgetQuoteUnavailable)
				require.Equal(t, imageagent.ProviderNotDispatched, imageagent.ProviderDispatchStateOf(err))
				_, err = executor.GenerateSlot(ctx, input)
				require.ErrorIs(t, err, imageagent.ErrBudgetQuoteUnavailable)
				_, err = executor.QuoteStagedReview(ctx, input, imageagent.BudgetPolicy{})
				require.ErrorIs(t, err, imageagent.ErrBudgetQuoteUnavailable)
				_, err = executor.ReviewStagedSlotQuoted(ctx, input, imageagent.SlotGeneratedOutput{}, mainGenerationQuote())
				require.ErrorIs(t, err, imageagent.ErrBudgetQuoteUnavailable)
				require.ErrorIs(t, executor.ReviewStagedSlot(ctx, input, imageagent.SlotGeneratedOutput{}), imageagent.ErrBudgetQuoteUnavailable)
			}
			cancel()
			require.Zero(t, delegate.generateCalls)
			require.Zero(t, delegate.reviewCalls)
		})
	}
}

func mainAdmissionContext() context.Context {
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: "org-1", EffectiveOrganizationID: "org-1", EffectiveMemberID: "member-1", UserID: "actor-1"})
	return aiidentity.WithIdentity(ctx, aiidentity.Identity{TenantID: "org-1", UserID: "actor-1", AgentRunID: "run-1", BusinessTaskID: "receipt-1"})
}
func mainAdmissionInput() imageagent.SlotExecutionInput {
	return imageagent.SlotExecutionInput{RunID: "run-1", TenantID: "org-1", UserID: "actor-1", PlanRevision: 1, Slot: imageagent.Slot{ID: "main-1", Role: imageagent.SlotRoleMain}, Attempt: 1}
}
func mainGenerationQuote() imageagent.SlotUsageQuote {
	return imageagent.SlotUsageQuote{Fingerprint: "old-slot-quote", Operations: []imageagent.SlotUsageOperation{{Name: "review", Fingerprint: "old-review-quote", MaximumOutputs: 1}}}
}

type recordingMainExecutor struct{ generateCalls, reviewCalls int }

func (e *recordingMainExecutor) GenerateQuotedSlot(context.Context, imageagent.SlotExecutionInput, imageagent.SlotUsageQuote) (imageagent.SlotGeneratedOutput, error) {
	e.generateCalls++
	return imageagent.SlotGeneratedOutput{}, nil
}
func (*recordingMainExecutor) QuoteSlot(context.Context, imageagent.SlotExecutionInput, imageagent.BudgetPolicy) (imageagent.SlotUsageQuote, error) {
	return mainGenerationQuote(), nil
}
func (e *recordingMainExecutor) GenerateSlot(context.Context, imageagent.SlotExecutionInput) (imageagent.SlotGeneratedOutput, error) {
	e.generateCalls++
	return imageagent.SlotGeneratedOutput{}, nil
}
func (*recordingMainExecutor) BuildSlotResult(context.Context, imageagent.SlotExecutionInput, imageagent.PublishedSlotOutput) (imageagent.SlotExecutionResult, error) {
	return imageagent.SlotExecutionResult{}, nil
}
func (e *recordingMainExecutor) ReviewStagedSlot(context.Context, imageagent.SlotExecutionInput, imageagent.SlotGeneratedOutput) error {
	e.reviewCalls++
	return nil
}
func (e *recordingMainExecutor) QuoteStagedReview(context.Context, imageagent.SlotExecutionInput, imageagent.BudgetPolicy) (imageagent.SlotUsageQuote, error) {
	e.reviewCalls++
	return imageagent.SlotUsageQuote{}, nil
}
func (e *recordingMainExecutor) ReviewStagedSlotQuoted(context.Context, imageagent.SlotExecutionInput, imageagent.SlotGeneratedOutput, imageagent.SlotUsageQuote) (imageagent.SlotUsageReceipt, error) {
	e.reviewCalls++
	return imageagent.SlotUsageReceipt{}, nil
}
