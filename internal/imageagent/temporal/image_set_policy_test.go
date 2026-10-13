package temporal

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	sdkactivity "go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
	"strings"
	"task-processor/internal/agent"
	"task-processor/internal/imageagent"
	"task-processor/internal/product/asset"
	"testing"
)

func nativePolicyAdoptionWorkflow(ctx workflow.Context, input WorkflowInput, selection asset.ImageSetCommand) (imageagent.RunStatus, error) {
	ctx = imageAgentActivityContext(ctx)
	input.externalEffectFinalization = true
	projection := WorkflowResult{Status: imageagent.RunStatusAwaitingFinalApproval, Plan: input.Plan, ResultDigest: selection.ApprovingResult.ResultDigest}
	results := []SlotWorkflowResult{}
	state := newWorkflowUpdateState(ctx, &input, &projection, &results, newWorkflowEffectOwner(ctx))
	_, err := state.handleResultAdoption(ctx, resultAdoptionIntent{RunID: input.RunID, PlanRevision: input.Plan.Revision, ResultDigest: projection.ResultDigest, ActorID: input.Identity.UserID, ActionID: selection.ActionID, Selection: &selection})
	return projection.Status, err
}
func TestNativeImagePolicyUsesOriginalActionOwnerWithoutHumanSignal(t *testing.T) {
	_, repo, slot := imageSetPersistenceFixture(t)
	p, err := repo.GetProjection(context.Background(), imageagent.RunScope{TenantID: slot.Identity.TenantID, OwnerUserID: slot.Identity.UserID, RunID: slot.RunID})
	require.NoError(t, err)
	input := WorkflowInput{RunID: p.Run.ID, Identity: slot.Identity, Plan: p.Plan}
	resultDigest := strings.Repeat("a", 64)
	source := p.Plan.Set.Source
	selection := asset.ImageSetCommand{ActionID: uuid.NewString(), ApprovingResult: asset.ImageSetResultBinding{RunID: input.RunID, PlanRevision: 1, ResultDigest: resultDigest}, Source: asset.SourceSelectionRequest{ContextKind: "acquisition", ItemID: source.OperationID, OriginalPublicationID: source.OriginalPublicationID, OriginalSnapshotVersion: 1, EffectiveCatalogVersion: 1, TargetPlatform: "product"}, SelectionDigest: strings.Repeat("b", 64), Choices: []asset.ImageSetChoice{{Kind: "source", SourceID: "source-1", Presentation: asset.ImagePresentation{Group: "detail", Order: 1}}}}
	selection.PolicyAdoption = &asset.ImageSetPolicyProvenance{Identity: agent.ExecutionIdentity{OrganizationID: slot.Identity.TenantID, ActorID: slot.Identity.UserID, MemberID: slot.Identity.MemberID}, Mode: agent.ModeFullAccess, ModeRevision: "1", ModeSnapshotRef: agent.ModeSnapshotRef{Kind: agent.ExecutionModeSnapshotKind, ID: uuid.NewString(), Digest: strings.Repeat("c", 64)}, ConfigurationSnapshotRef: p.Plan.Set.Configuration, RunID: p.Run.ID, PlanRevision: 1, ResultDigest: resultDigest, PolicyVersion: agent.ExecutionModePolicyVersion, ActionVersion: asset.ImageSetPolicyAdoptionVersion}
	selection.Choices = []asset.ImageSetChoice{{Kind: "generated", RunID: p.Run.ID, PlanRevision: 1, SlotID: p.Plan.Slots[0].ID, Attempt: 1, AssetID: "generated-asset", ResultDigest: resultDigest, Presentation: asset.ImagePresentation{Group: "detail", Order: 1}}}
	require.True(t, selection.PolicyAdoption.Valid())
	env := newWorkflowEnv(t)
	env.RegisterWorkflow(nativePolicyAdoptionWorkflow)
	env.RegisterActivityWithOptions(func(context.Context, PublishImageSetActivityInput) error { return nil }, sdkactivity.RegisterOptions{Name: activityPublishApprovedImageSet})
	env.OnActivity(activityPersistPendingCommand, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(activityPublishApprovedImageSet, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		actual := activityInputFromArgs[PublishImageSetActivityInput](t, args)
		require.Equal(t, selection, actual.Selection)
		require.NotNil(t, actual.Selection.PolicyAdoption)
	}).Return(nil).Once()
	env.OnActivity(activityPersistRunState, mock.Anything, mock.Anything).Return(nil).Once()
	env.ExecuteWorkflow(nativePolicyAdoptionWorkflow, input, selection)
	require.NoError(t, env.GetWorkflowError())
	var status imageagent.RunStatus
	require.NoError(t, env.GetWorkflowResult(&status))
	require.Equal(t, imageagent.RunStatusCompleted, status)
	env.AssertExpectations(t)
}
