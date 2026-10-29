package temporal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	client "go.temporal.io/sdk/client"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	historypb "go.temporal.io/api/history/v1"
	sdkactivity "go.temporal.io/sdk/activity"
	sdkconverter "go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
	sdkworkflow "go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/encoding/protojson"

	"task-processor/internal/imageagent"
)

func TestImageWorkflowReplaysPreFinalizationHistory(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{TaskQueue: OrganizationTaskQueue})
	env.RegisterWorkflow(preFinalizationHistoryProbe)
	env.RegisterWorkflow(ImageSlotWorkflowV3)
	var observedStartToCloseTimeout time.Duration
	env.SetOnActivityStartedListener(func(info *sdkactivity.Info, _ context.Context, _ sdkconverter.EncodedValues) {
		observedStartToCloseTimeout = info.StartToCloseTimeout
	})
	env.RegisterActivityWithOptions(func(context.Context, ExecuteSlotV3ActivityInput) (imageagent.SlotEffectV3PublishedResult, error) {
		return imageagent.SlotEffectV3PublishedResult{}, nil
	}, sdkactivity.RegisterOptions{Name: activityExecuteSlotV3})
	env.OnGetVersion(externalEffectFinalizationPatch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.DefaultVersion).Once()

	env.ExecuteWorkflow(preFinalizationHistoryProbe, env.Now().Add(2*time.Minute))

	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 2*time.Minute, observedStartToCloseTimeout)
	require.True(t, env.AssertExpectations(t))
}

func preFinalizationHistoryProbe(ctx sdkworkflow.Context, deadline time.Time) error {
	finalizationEnabled := sdkworkflow.GetVersion(ctx, externalEffectFinalizationPatch, sdkworkflow.DefaultVersion, 1) != sdkworkflow.DefaultVersion
	var result SlotWorkflowV3Result
	return sdkworkflow.ExecuteChildWorkflow(ctx, ImageSlotWorkflowV3, SlotWorkflowV3Input{
		RunID: "run-pre-finalization", Identity: imageagent.ExecutionIdentity{BusinessTaskID: "task-" + "run-pre-finalization", MemberID: "member-" + "user-a", ScopeProtocol: imageagent.OrganizationScopeProtocol, RunID: "run-pre-finalization", TenantID: "tenant-a", UserID: "user-a"},
		PlanRevision: 1, Slot: imageagent.Slot{ID: "slot-1", Role: imageagent.SlotRoleScene}, Attempt: 1,
		ExecuteActivityName: activityExecuteSlotV3, BudgetAuthorization: true, DeadlineAt: deadline,
		ExternalEffectFinalization: finalizationEnabled,
	}).Get(ctx, &result)
}

func historyVersionChangeIDs(t *testing.T, history *historypb.History) []string {
	t.Helper()
	var changeIDs []string
	for _, event := range history.Events {
		attributes := event.GetMarkerRecordedEventAttributes()
		if attributes == nil || attributes.GetMarkerName() != "Version" {
			continue
		}
		payloads := attributes.GetDetails()["change-id"]
		if payloads == nil {
			continue
		}
		var changeID string
		require.NoError(t, sdkconverter.GetDefaultDataConverter().FromPayloads(payloads, &changeID))
		changeIDs = append(changeIDs, changeID)
	}
	return changeIDs
}

func scheduledActivityNames(history *historypb.History) []string {
	names := make([]string, 0, len(history.Events))
	for _, event := range history.Events {
		attributes := event.GetActivityTaskScheduledEventAttributes()
		if attributes == nil {
			continue
		}
		names = append(names, attributes.GetActivityType().GetName())
	}
	return names
}

type wireProbeResult struct {
	ExecuteSlot         string
	StartEffectRecovery string
	PublishApproved     string
	ApprovalActionID    string
	ResultDigest        string
}

type approvalProtocolProbeResult struct {
	PublishActivity string
	IdempotencyKey  string
	ResultDigest    string
}

func approvalProtocolProbe(ctx sdkworkflow.Context) (approvalProtocolProbeResult, error) {
	ctx = imageAgentActivityContext(ctx)
	effects := newWorkflowEffectOwner(ctx)
	if !effects.activities.useV3Slot || !effects.activities.useV3Approval {
		return approvalProtocolProbeResult{}, errRetiredImageAgentWire
	}
	plan, results := wireProbePlanAndResults()
	digest, err := resultDigestForWire(plan, results, effects.activities)
	if err != nil {
		return approvalProtocolProbeResult{}, err
	}
	key := approvalPublicationKeyForWire("capture-action", "capture-run", 1, effects.activities)
	err = effects.publishApproved(ctx, PublishApprovedV3ActivityInput{
		RunID: "capture-run", Identity: imageagent.ExecutionIdentity{BusinessTaskID: "task-" + "capture-run", MemberID: "member-" + "user-a", ScopeProtocol: imageagent.OrganizationScopeProtocol, RunID: "capture-run", TenantID: "tenant-a", UserID: "user-a"},
		PlanRevision: 1, CandidateAssetIDs: []string{"candidate-1"}, IdempotencyKey: key,
	})
	if err != nil {
		return approvalProtocolProbeResult{}, err
	}
	return approvalProtocolProbeResult{PublishActivity: effects.activities.publishApproved, IdempotencyKey: key, ResultDigest: digest}, nil
}

func TestApprovalMarkerCombinationsSelectOnlyCompleteProtocols(t *testing.T) {
	for mask := 0; mask < 8; mask++ {
		actionV3 := mask&1 != 0
		publicationV3 := mask&2 != 0
		digestV3 := mask&4 != 0
		name := fmt.Sprintf("action=%t/publication=%t/digest=%t", actionV3, publicationV3, digestV3)
		t.Run(name, func(t *testing.T) {
			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			env.SetStartWorkflowOptions(client.StartWorkflowOptions{TaskQueue: OrganizationTaskQueue})
			env.RegisterWorkflow(approvalProtocolProbe)
			env.RegisterActivityWithOptions(
				func(context.Context, PublishApprovedV3ActivityInput) error { return nil },
				sdkactivity.RegisterOptions{Name: activityPublishApprovedV3},
			)
			activities := 0
			env.SetOnActivityStartedListener(func(*sdkactivity.Info, context.Context, sdkconverter.EncodedValues) { activities++ })
			env.OnGetVersion(slotExecutionWireV3Patch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.Version(1)).Once()
			env.OnGetVersion(approvalActionIDV3Patch, sdkworkflow.DefaultVersion, 1).Return(markerVersion(actionV3)).Once()
			env.OnGetVersion(approvalPublicationWireV3Patch, sdkworkflow.DefaultVersion, 1).Return(markerVersion(publicationV3)).Once()
			env.OnGetVersion(resultDigestV3Patch, sdkworkflow.DefaultVersion, 1).Return(markerVersion(digestV3)).Once()
			env.OnGetVersion(approvalPublicationScopePatch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.DefaultVersion).Once()
			env.OnGetVersion(activityWireV2Patch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.Version(1)).Once()

			allV3 := actionV3 && publicationV3 && digestV3
			wantActivity := activityPublishApprovedV3
			wantKey := "capture-action"
			plan, results := wireProbePlanAndResults()
			wantDigest, err := imageagent.ResultDigestV3(plan, slotProjections(plan, results))
			require.NoError(t, err)
			if allV3 {
				env.OnActivity(wantActivity, mock.Anything, mock.MatchedBy(func(input PublishApprovedV3ActivityInput) bool {
					return input.IdempotencyKey == wantKey
				})).Return(nil).Once()
			}

			env.ExecuteWorkflow(approvalProtocolProbe)
			if !allV3 {
				require.ErrorContains(t, env.GetWorkflowError(), errRetiredImageAgentWire.Error())
				require.Zero(t, activities)
				require.True(t, env.AssertExpectations(t))
				return
			}
			require.NoError(t, env.GetWorkflowError())
			var got approvalProtocolProbeResult
			require.NoError(t, env.GetWorkflowResult(&got))
			require.Equal(t, approvalProtocolProbeResult{PublishActivity: wantActivity, IdempotencyKey: wantKey, ResultDigest: wantDigest}, got)
			require.True(t, env.AssertExpectations(t))
		})
	}
}

func markerVersion(enabled bool) sdkworkflow.Version {
	if enabled {
		return sdkworkflow.Version(1)
	}
	return sdkworkflow.DefaultVersion
}

func workflowWireProbe(ctx sdkworkflow.Context) (wireProbeResult, error) {
	wire := activityWireForWorkflow(ctx)
	if !wire.useV3Slot || !wire.useV3Approval {
		return wireProbeResult{}, errRetiredImageAgentWire
	}
	plan, results := wireProbePlanAndResults()
	digest, err := resultDigestForWire(plan, results, wire)
	if err != nil {
		return wireProbeResult{}, err
	}
	return wireProbeResult{
		ExecuteSlot:         wire.executeSlot,
		StartEffectRecovery: wire.startEffectRecovery,
		PublishApproved:     wire.publishApproved,
		ApprovalActionID:    approvalPublicationKeyForWire("capture-action", "capture-run", 1, wire),
		ResultDigest:        digest,
	}, nil
}

func TestNewWorkflowSelectsExecuteSlotV3(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{TaskQueue: OrganizationTaskQueue})
	env.RegisterWorkflow(workflowWireProbe)
	env.ExecuteWorkflow(workflowWireProbe)
	require.NoError(t, env.GetWorkflowError())

	var got wireProbeResult
	require.NoError(t, env.GetWorkflowResult(&got))
	require.Equal(t, "imageagent.execute_slot.v3", got.ExecuteSlot)
}

func TestNewApprovalScopesActionIDByRunAndRevisionWithV3DigestAndPublishV3(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{TaskQueue: OrganizationTaskQueue})
	env.RegisterWorkflow(workflowWireProbe)
	env.ExecuteWorkflow(workflowWireProbe)
	require.NoError(t, env.GetWorkflowError())

	var got wireProbeResult
	require.NoError(t, env.GetWorkflowResult(&got))
	require.Equal(t, "imageagent.publish_approved.v3", got.PublishApproved)
	require.Equal(t, approvalActionPublicationKey("capture-action", "capture-run", 1), got.ApprovalActionID)
	require.NotEqual(t, got.ApprovalActionID, approvalActionPublicationKey("capture-action", "other-run", 1))
	require.NotEqual(t, got.ApprovalActionID, approvalActionPublicationKey("capture-action", "capture-run", 2))
	plan, results := wireProbePlanAndResults()
	wantDigest, err := imageagent.ResultDigestV3(plan, slotProjections(plan, results))
	require.NoError(t, err)
	require.Equal(t, wantDigest, got.ResultDigest)
}

func TestRecoveryStartMarkerDefaultsOffForLegacyV3Histories(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{TaskQueue: OrganizationTaskQueue})
	env.RegisterWorkflow(workflowWireProbe)
	env.OnGetVersion(slotExecutionWireV3Patch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.Version(1)).Once()
	env.OnGetVersion(approvalActionIDV3Patch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.Version(1)).Once()
	env.OnGetVersion(approvalPublicationWireV3Patch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.Version(1)).Once()
	env.OnGetVersion(resultDigestV3Patch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.Version(1)).Once()
	env.OnGetVersion(approvalPublicationScopePatch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.DefaultVersion).Once()
	env.OnGetVersion(effectRecoveryStartWireV1Patch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.DefaultVersion).Once()
	env.OnGetVersion(activityWireV2Patch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.Version(1)).Once()

	env.ExecuteWorkflow(workflowWireProbe)

	require.NoError(t, env.GetWorkflowError())
	var got wireProbeResult
	require.NoError(t, env.GetWorkflowResult(&got))
	require.Equal(t, activityExecuteSlotV3, got.ExecuteSlot)
	require.Empty(t, got.StartEffectRecovery)
	require.True(t, env.AssertExpectations(t))
}

func TestRecoveryStartMarkerEnablesExternalRecoveryForNewV3Histories(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{TaskQueue: OrganizationTaskQueue})
	env.RegisterWorkflow(workflowWireProbe)
	env.OnGetVersion(slotExecutionWireV3Patch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.Version(1)).Once()
	env.OnGetVersion(approvalActionIDV3Patch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.Version(1)).Once()
	env.OnGetVersion(approvalPublicationWireV3Patch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.Version(1)).Once()
	env.OnGetVersion(resultDigestV3Patch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.Version(1)).Once()
	env.OnGetVersion(approvalPublicationScopePatch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.DefaultVersion).Once()
	env.OnGetVersion(effectRecoveryStartWireV1Patch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.Version(1)).Once()
	env.OnGetVersion(activityWireV2Patch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.Version(1)).Once()

	env.ExecuteWorkflow(workflowWireProbe)

	require.NoError(t, env.GetWorkflowError())
	var got wireProbeResult
	require.NoError(t, env.GetWorkflowResult(&got))
	require.Equal(t, activityStartEffectRecoveryV3, got.StartEffectRecovery)
	require.True(t, env.AssertExpectations(t))
}

func TestRunScopedApprovalPublicationKeyFitsReceiptSchema(t *testing.T) {
	key := approvalActionPublicationKey(strings.Repeat("a", imageagent.MaxActionIDLength), strings.Repeat("r", 64), imageagent.MaxJSONSafePlanRevision)
	require.LessOrEqual(t, len(key), 128)
}

func TestLegacyScopedApprovalHistoryKeepsLegacyPublicationKey(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{TaskQueue: OrganizationTaskQueue})
	env.RegisterWorkflow(workflowWireProbe)
	env.OnGetVersion(slotExecutionWireV3Patch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.Version(1)).Once()
	env.OnGetVersion(approvalActionIDV3Patch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.Version(1)).Once()
	env.OnGetVersion(approvalPublicationWireV3Patch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.Version(1)).Once()
	env.OnGetVersion(resultDigestV3Patch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.Version(1)).Once()
	env.OnGetVersion(approvalPublicationScopePatch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.Version(1)).Once()
	env.OnGetVersion(approvalPublicationKeyLengthPatch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.DefaultVersion).Once()
	env.OnGetVersion(activityWireV2Patch, sdkworkflow.DefaultVersion, 1).Return(sdkworkflow.Version(1)).Once()

	env.ExecuteWorkflow(workflowWireProbe)

	require.NoError(t, env.GetWorkflowError())
	var got wireProbeResult
	require.NoError(t, env.GetWorkflowResult(&got))
	require.Equal(t, legacyApprovalActionPublicationKey("capture-action", "capture-run", 1), got.ApprovalActionID)
	require.True(t, env.AssertExpectations(t))
}

func wireProbePlanAndResults() (imageagent.Plan, []SlotWorkflowResult) {
	const sha = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	plan := imageagent.Plan{Revision: 1, Slots: []imageagent.Slot{{ID: "slot-1", Role: imageagent.SlotRoleScene}}}
	results := []SlotWorkflowResult{{
		Execution: imageagent.SlotExecutionResult{SlotID: "slot-1", Attempt: 1, Candidates: []imageagent.AssetCandidate{{
			AssetID: "candidate-1", DurableAsset: imageagent.DurableAssetIdentity{ObjectKey: "image-agent/public/tenant-a/capture-run/1/slot-1/1/0-" + sha + ".png", SHA256: sha},
		}}},
		Status: imageagent.SlotStatusAccepted,
	}}
	return plan, results
}

func readHistoryFixture(t *testing.T, name string) *historypb.History {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	history := &historypb.History{}
	require.NoError(t, protojson.Unmarshal(raw, history))
	return history
}
