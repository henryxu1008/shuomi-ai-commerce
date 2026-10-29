package temporal

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	client "go.temporal.io/sdk/client"

	"github.com/stretchr/testify/require"
	sdkactivity "go.temporal.io/sdk/activity"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"task-processor/internal/imageagent"
	"task-processor/internal/imageagent/store"
)

const v3SHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const testExecuteSlotV3ActivityName = "test.imageagent.execute_slot.v3"

func TestSlotWorkflowPreservesUnknownProviderOutcomeBlockerCode(t *testing.T) {
	err := sdktemporal.NewNonRetryableApplicationError("unknown", slotProviderOutcomeUnknownCode, nil)
	require.Equal(t, "slot_provider_outcome_unknown", slotExecutionV3ErrorCode(err))
	for _, code := range []string{"slot_provider_not_dispatched", slotProviderOutcomeUnknownCode, slotStagingOutcomeUnknownCode, slotPublicationOutcomeUnknownCode} {
		err = sdktemporal.NewNonRetryableApplicationError("unknown", code, nil)
		require.Equal(t, code, slotExecutionV3ErrorCode(err))
	}
	require.Equal(t, "slot_execution_failed", slotExecutionV3ErrorCode(errors.New("transport failed")))
}

func TestImageSlotWorkflowV3PreservesInvalidPersistedPolicyClassification(t *testing.T) {
	for _, tc := range []struct {
		name          string
		activityType  string
		wantBlockCode string
	}{
		{name: "phase", activityType: slotEffectPhaseInvalidCode, wantBlockCode: "slot_effect_phase_invalid"},
		{name: "policy", activityType: slotEffectPolicyInvalidCode, wantBlockCode: "slot_effect_policy_invalid"},
		{name: "future v3 policy", activityType: "imageagent_slot_effect_future_policy_invalid", wantBlockCode: "slot_effect_policy_invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			env.SetStartWorkflowOptions(client.StartWorkflowOptions{TaskQueue: OrganizationTaskQueue})
			env.RegisterWorkflow(ImageSlotWorkflowV3)
			env.RegisterActivityWithOptions(func(context.Context, ExecuteSlotV3ActivityInput) (imageagent.SlotEffectV3PublishedResult, error) {
				return imageagent.SlotEffectV3PublishedResult{}, sdktemporal.NewNonRetryableApplicationError("invalid persisted effect", tc.activityType, nil)
			}, sdkactivity.RegisterOptions{Name: testExecuteSlotV3ActivityName})
			input := SlotWorkflowV3Input{
				RunID: "run-invalid-policy", Identity: imageagent.ExecutionIdentity{ScopeProtocol: imageagent.OrganizationScopeProtocol, RunID: "run-invalid-policy", MemberID: "member-user-a", BusinessTaskID: "task-1", TenantID: "tenant-a", UserID: "user-a"}, PlanRevision: 1,
				Slot:    imageagent.Slot{ID: "scene-1", Role: imageagent.SlotRoleScene, IdempotencyKey: "scene-key", SourceAssetIDs: []string{"source-1"}},
				Attempt: 1, ExecuteActivityName: testExecuteSlotV3ActivityName,
			}

			env.ExecuteWorkflow(ImageSlotWorkflowV3, input)
			require.NoError(t, env.GetWorkflowError())
			var result SlotWorkflowV3Result
			require.NoError(t, env.GetWorkflowResult(&result))
			require.Equal(t, imageagent.SlotStatusBlocked, result.Status)
			require.Equal(t, tc.wantBlockCode, result.ErrorCode)
		})
	}
}

func TestImageSlotWorkflowV3RejectsInvalidMainCandidateCount(t *testing.T) {
	for _, count := range []int{0, 2} {
		t.Run(fmt.Sprintf("count-%d", count), func(t *testing.T) {
			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			env.SetStartWorkflowOptions(client.StartWorkflowOptions{TaskQueue: OrganizationTaskQueue})
			env.RegisterWorkflow(ImageSlotWorkflowV3)
			env.RegisterActivityWithOptions(func(_ context.Context, input ExecuteSlotV3ActivityInput) (imageagent.SlotEffectV3PublishedResult, error) {
				candidates := make([]imageagent.SlotEffectV3AssetCandidate, count)
				for index := range candidates {
					candidates[index] = imageagent.SlotEffectV3AssetCandidate{
						AssetID: fmt.Sprintf("candidate-%d", index), SourceAssetID: "source-1",
						DurableAsset: imageagent.DurableAssetIdentity{ObjectKey: fmt.Sprintf("image-agent/public/tenant-a/fc95297aa4f56781f0decb7d4bf59b1447f09b3611039b80188b1c6beb03ee6a/run-main-count/1/main-1/1/%d-%s.png", index, v3SHA256), SHA256: v3SHA256},
					}
				}
				return imageagent.SlotEffectV3PublishedResult{SlotID: input.Slot.ID, Attempt: input.Attempt, Candidates: candidates}, nil
			}, sdkactivity.RegisterOptions{Name: testExecuteSlotV3ActivityName})
			input := SlotWorkflowV3Input{RunID: "run-main-count", Identity: imageagent.ExecutionIdentity{ScopeProtocol: imageagent.OrganizationScopeProtocol, RunID: "run-main-count", MemberID: "member-user-a", BusinessTaskID: "task-1", TenantID: "tenant-a", UserID: "user-a"}, PlanRevision: 1, Slot: imageagent.Slot{ID: "main-1", Role: imageagent.SlotRoleMain, IdempotencyKey: "main-key", SourceAssetIDs: []string{"source-1"}}, Attempt: 1, ExecuteActivityName: testExecuteSlotV3ActivityName}

			env.ExecuteWorkflow(ImageSlotWorkflowV3, input)
			require.NoError(t, env.GetWorkflowError())
			var result SlotWorkflowV3Result
			require.NoError(t, env.GetWorkflowResult(&result))
			require.Equal(t, imageagent.SlotStatusBlocked, result.Status)
			require.Equal(t, invalidMainCandidateCountCode, result.ErrorCode)
		})
	}
}

func TestImageSlotWorkflowV3ThreadsExternalEffectFinalizationToActivityInput(t *testing.T) {
	for _, tc := range []struct {
		name      string
		flag      bool
		wantPhase imageagent.SlotEffectV3Phase
	}{
		{name: "legacy history", flag: false},
		{name: "new history", flag: true, wantPhase: imageagent.SlotEffectV3PublicationComplete},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			env.SetStartWorkflowOptions(client.StartWorkflowOptions{TaskQueue: OrganizationTaskQueue})
			env.RegisterWorkflow(ImageSlotWorkflowV3)
			observed := false
			env.RegisterActivityWithOptions(func(_ context.Context, input ExecuteSlotV3ActivityInput) (imageagent.SlotEffectV3PublishedResult, error) {
				observed = input.ExternalEffectFinalization
				return imageagent.SlotEffectV3PublishedResult{
					SlotID: input.Slot.ID, Attempt: input.Attempt,
					Candidates: []imageagent.SlotEffectV3AssetCandidate{{
						AssetID: "candidate-0", SourceAssetID: "source-1",
						DurableAsset: imageagent.DurableAssetIdentity{
							ObjectKey: "image-agent/public/tenant-a/fc95297aa4f56781f0decb7d4bf59b1447f09b3611039b80188b1c6beb03ee6a/run-thread-flag/1/scene-1/1/0-" + v3SHA256 + ".png",
							SHA256:    v3SHA256,
						},
					}},
				}, nil
			}, sdkactivity.RegisterOptions{Name: testExecuteSlotV3ActivityName})
			input := SlotWorkflowV3Input{
				RunID: "run-thread-flag", Identity: imageagent.ExecutionIdentity{ScopeProtocol: imageagent.OrganizationScopeProtocol, RunID: "run-thread-flag", MemberID: "member-user-a", BusinessTaskID: "task-1", TenantID: "tenant-a", UserID: "user-a"}, PlanRevision: 1,
				Slot:    imageagent.Slot{ID: "scene-1", Role: imageagent.SlotRoleScene, IdempotencyKey: "scene-key", SourceAssetIDs: []string{"source-1"}},
				Attempt: 1, ExecuteActivityName: testExecuteSlotV3ActivityName, ExternalEffectFinalization: tc.flag,
			}

			env.ExecuteWorkflow(ImageSlotWorkflowV3, input)
			require.NoError(t, env.GetWorkflowError())
			var result SlotWorkflowV3Result
			require.NoError(t, env.GetWorkflowResult(&result))
			require.Equal(t, imageagent.SlotStatusAccepted, result.Status)
			require.Equal(t, tc.wantPhase, result.EffectPhase)
			require.Equal(t, tc.flag, observed)
		})
	}
}

func TestPersistSlotResultV3CannotAcceptInvalidMainCandidateCount(t *testing.T) {
	for _, count := range []int{0, 2} {
		t.Run(fmt.Sprintf("count-%d", count), func(t *testing.T) {
			repository := store.NewMemoryRepository()
			identity := imageagent.ExecutionIdentity{ScopeProtocol: imageagent.OrganizationScopeProtocol, RunID: "run-persist-main-count-v3", MemberID: "member-" + "user-a", BusinessTaskID: "task-main-count", TenantID: "tenant-a", UserID: "user-a"}
			plan := imageagent.Plan{Revision: 1, IdempotencyKey: "plan-main-count", SourceAssetIDs: []string{"source-1"}, CreatedBy: identity.UserID, Slots: []imageagent.Slot{{ID: "main-1", Role: imageagent.SlotRoleMain, SourceAssetIDs: []string{"source-1"}, IdempotencyKey: "main-key"}}}
			run := imageagent.Run{ID: "run-persist-main-count-v3", BusinessTaskID: "task-main-count", TenantID: identity.TenantID, UserID: identity.UserID, Mode: imageagent.RunModeManual, IdempotencyKey: "run-key-main-count", Status: imageagent.RunStatusExecuting, ActivePlanRevision: 1, Version: 1, ScopeProtocol: imageagent.OrganizationScopeProtocol, MemberID: "member-" + identity.UserID}
			initializeActivityProjection(t, repository, run, plan)
			activities, err := NewActivities(ActivityDependencies{PublisherV3: &identityCheckingPublisher{t: t}, ArtifactStore: &recordingArtifactStore{}, StagedSlotExecutor: &recordingStagedExecutor{}, ExecutionAuthorizer: allowGenerationExecution{}, Repository: repository})
			require.NoError(t, err)
			candidates := make([]imageagent.SlotEffectV3AssetCandidate, count)
			for index := range candidates {
				candidates[index] = imageagent.SlotEffectV3AssetCandidate{AssetID: fmt.Sprintf("candidate-%d", index), SourceAssetID: "source-1", DurableAsset: imageagent.DurableAssetIdentity{ObjectKey: fmt.Sprintf("image-agent/public/tenant-a/fc95297aa4f56781f0decb7d4bf59b1447f09b3611039b80188b1c6beb03ee6a/%s/1/main-1/1/%d-%s.png", run.ID, index, v3SHA256), SHA256: v3SHA256}}
			}

			err = activities.PersistSlotResultV3(context.Background(), PersistSlotResultV3ActivityInput{
				RunID: run.ID, Identity: identity, PlanRevision: 1, AttemptKey: "main-key:plan:1:attempt:1",
				Result: SlotWorkflowV3Result{Published: imageagent.SlotEffectV3PublishedResult{SlotID: "main-1", Attempt: 1, Candidates: candidates}, Status: imageagent.SlotStatusAccepted},
			})
			require.NoError(t, err)
			projection, err := repository.GetProjection(context.Background(), imageagent.ScopeForRun(run))
			require.NoError(t, err)
			require.Equal(t, imageagent.SlotStatusBlocked, projection.Slots[0].Slot.Status)
			require.Equal(t, invalidMainCandidateCountCode, projection.Slots[0].ErrorCode)
			require.Empty(t, projection.Slots[0].Candidates)
		})
	}
}

func TestPersistSlotResultV3PersistsProviderBudgetBlockWithoutProjectionConflict(t *testing.T) {
	for _, test := range []struct {
		name string
		new  func(t *testing.T) imageagent.Repository
	}{
		{name: "memory", new: func(*testing.T) imageagent.Repository { return store.NewMemoryRepository() }},
		{name: "gorm sqlite", new: func(t *testing.T) imageagent.Repository {
			db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:image-agent-budget-block-%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
			require.NoError(t, err)
			require.NoError(t, store.AutoMigrate(db))
			return store.NewOrganizationRepository(db)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := test.new(t)
			identity := imageagent.ExecutionIdentity{ScopeProtocol: imageagent.OrganizationScopeProtocol, RunID: "run-budget-block-v3-" + test.name, MemberID: "member-" + "user-a", BusinessTaskID: "task-budget-block", TenantID: "tenant-a", UserID: "user-a"}
			plan := imageagent.Plan{Revision: 1, IdempotencyKey: "plan-budget-block", SourceAssetIDs: []string{"source-1"}, StyleReferenceIDs: []string{}, CreatedBy: identity.UserID, Slots: []imageagent.Slot{{ID: "main-1", Role: imageagent.SlotRoleMain, SourceAssetIDs: []string{"source-1"}, StyleReferenceIDs: []string{}, IdempotencyKey: "main-key"}}}
			run := imageagent.Run{ID: "run-budget-block-v3-" + test.name, BusinessTaskID: "task-budget-block", TenantID: identity.TenantID, UserID: identity.UserID, Mode: imageagent.RunModeManual, IdempotencyKey: "run-budget-block-" + test.name, Status: imageagent.RunStatusExecuting, ActivePlanRevision: 1, Version: 1, ScopeProtocol: imageagent.OrganizationScopeProtocol, MemberID: "member-" + identity.UserID}
			initializeActivityProjection(t, repository, run, plan)
			activities, err := NewActivities(ActivityDependencies{ArtifactStore: &recordingArtifactStore{}, StagedSlotExecutor: &recordingStagedExecutor{}, ExecutionAuthorizer: allowGenerationExecution{}, PublisherV3: &identityCheckingPublisher{t: t}, Repository: repository})
			require.NoError(t, err)

			err = activities.PersistSlotResultV3(context.Background(), PersistSlotResultV3ActivityInput{
				RunID: run.ID, Identity: identity, PlanRevision: 1, AttemptKey: "main-key:plan:1:attempt:1",
				Result: SlotWorkflowV3Result{
					Published: imageagent.SlotEffectV3PublishedResult{SlotID: "main-1", Attempt: 1},
					Status:    imageagent.SlotStatusBlocked, ErrorCode: imageagent.BudgetQuoteUnavailableCode,
				},
			})
			require.NoError(t, err)
			projection, err := repository.GetProjection(context.Background(), imageagent.ScopeForRun(run))
			require.NoError(t, err)
			require.Equal(t, imageagent.SlotStatusBlocked, projection.Slots[0].Slot.Status)
			require.Equal(t, imageagent.BudgetQuoteUnavailableCode, projection.Slots[0].ErrorCode)
		})
	}
}

func initializedSlotEffectActivity(t *testing.T, runID string) (imageagent.Repository, ExecuteSlotV3ActivityInput) {
	t.Helper()
	repository := store.NewMemoryRepository()
	run := imageagent.Run{ScopeProtocol: imageagent.OrganizationScopeProtocol, MemberID: "member-user-a", ID: runID, BusinessTaskID: "task-" + runID, TenantID: "tenant-a", UserID: "user-a", Mode: imageagent.RunModeManual, IdempotencyKey: "run-key-" + runID, Status: imageagent.RunStatusPlanning, CurrentNode: "plan", Version: 1, ActivePlanRevision: 1, MaxConcurrentSlots: 1}
	plan := imageagent.Plan{Revision: 1, IdempotencyKey: "plan-key-1", SourceAssetIDs: []string{"source-1"}, CreatedBy: "user-a", Slots: []imageagent.Slot{{ID: "slot-1", Role: imageagent.SlotRoleMain, SourceAssetIDs: []string{"source-1"}, IdempotencyKey: "slot-key-1"}}}
	catalog, err := imageagent.NormalizeAssetCatalog(imageagent.AssetCatalog{Assets: []imageagent.AuthorizedAsset{{ID: "source-1", Type: imageagent.AuthorizedAssetSource, URL: "https://source.example/source.png"}}})
	require.NoError(t, err)
	_, err = repository.InitializeRun(context.Background(), imageagent.ProjectionInitialization{Scope: imageagent.ScopeForRun(run), Run: run, Plan: plan, Catalog: catalog, Snapshot: imageagent.RunProjection{Run: run, Plan: plan}, CommitID: "start:" + runID, EventType: "run.initialized", EventPayload: []byte(`{}`)})
	require.NoError(t, err)
	return repository, ExecuteSlotV3ActivityInput{RunID: runID, Identity: imageagent.ExecutionIdentity{ScopeProtocol: imageagent.OrganizationScopeProtocol, RunID: runID, MemberID: "member-user-a", BusinessTaskID: "task-" + runID, TenantID: "tenant-a", UserID: "user-a"}, PlanRevision: 1, Slot: plan.Slots[0], Attempt: 1, IdempotencyKey: "slot-key-1:plan:1:attempt:1", AssetCatalog: catalog}
}
