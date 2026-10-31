package imageagentworker

import (
	"context"
	"task-processor/internal/authidentity"
	"task-processor/internal/imageagent"
	"task-processor/internal/shared/aiidentity"
	"testing"

	"github.com/stretchr/testify/require"
)

func mainAdmissionContext() context.Context {
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: "org-1", EffectiveOrganizationID: "org-1", EffectiveMemberID: "member-1", UserID: "actor-1"})
	return aiidentity.WithIdentity(ctx, aiidentity.Identity{TenantID: "org-1", UserID: "actor-1", AgentRunID: "run-1", BusinessTaskID: "receipt-1"})
}
func mainAdmissionInput() imageagent.SlotExecutionInput {
	return imageagent.SlotExecutionInput{RunID: "run-1", TenantID: "org-1", UserID: "actor-1", TargetPlatform: "product", ImagePolicyContext: &imageagent.ImagePolicyContext{Country: "zz", Family: "default", SceneCategory: "general"}, PlanRevision: 1, Slot: imageagent.Slot{ID: "main-1", Role: imageagent.SlotRoleMain, SourceAssetIDs: []string{"source-1"}}, Attempt: 1}
}
func mainGenerationQuote() imageagent.SlotUsageQuote {
	return imageagent.SlotUsageQuote{Fingerprint: "main-quote", Operations: []imageagent.SlotUsageOperation{{Name: "render_source_white_background", Fingerprint: "main-operation-quote", MaximumOutputs: 1}}}
}

func TestSourceMainRejectsRetiredInputBeforeGeneration(t *testing.T) {
	e := SourceMainExecutor{}
	input := mainAdmissionInput()
	input.Slot.Role = imageagent.SlotRoleScene
	_, err := e.QuoteSlot(mainAdmissionContext(), input, imageagent.BudgetPolicy{})
	require.Error(t, err)
	_, err = e.GenerateQuotedSlot(mainAdmissionContext(), input, mainGenerationQuote())
	require.Equal(t, imageagent.ProviderRejectedBeforeEffect, imageagent.ProviderDispatchStateOf(err))
	_, err = e.GenerateSlot(mainAdmissionContext(), input)
	require.Equal(t, imageagent.ProviderNotDispatched, imageagent.ProviderDispatchStateOf(err))
}
