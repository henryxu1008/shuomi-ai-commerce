package temporal

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/agentconfig"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent"
	"github.com/stretchr/testify/require"
	sdkactivity "go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestV3ImageSetInputPreservesFrozenSetAndItsFingerprint(t *testing.T) {
	set := &imageagent.ImageSetPlan{Schema: imageagent.ImageSetSchema, ParametersDigest: "parameters", InputDigest: "approved-input"}
	input := ExecuteSlotV3ActivityInput{RunID: "run", Identity: imageagent.ExecutionIdentity{TenantID: "org", UserID: "actor"}, PlanRevision: 1, Slot: imageagent.Slot{ID: "detail", Recipe: &imageagent.ImageSlotRecipe{Prompt: "approved prompt"}}, Attempt: 1, ImageSet: set}
	execution := slotExecutionInputV3(input)
	require.Equal(t, set, execution.ImageSet)
	before := imageagent.SlotExecutionFingerprint(execution)
	set.ParametersDigest = "changed"
	require.Equal(t, before, imageagent.SlotExecutionFingerprint(execution), "caller mutation must not change the captured set")
	input.Slot.Recipe.Prompt = "changed prompt"
	require.Equal(t, before, imageagent.SlotExecutionFingerprint(execution), "caller mutation must not change the captured recipe")
	changed := slotExecutionInputV3(input)
	require.NotEqual(t, before, imageagent.SlotExecutionFingerprint(changed))
	for _, payload := range []any{SlotWorkflowV3Input{}, ExecuteSlotV3ActivityInput{}, EffectRecoveryWorkflowInput{}} {
		encoded, err := json.Marshal(payload)
		require.NoError(t, err)
		require.NotContains(t, string(encoded), "ImageSet", "historical absent fields must remain absent")
	}
}

func TestSetWorkflowOffersKnownPartialResultsOnlyAfterEveryEffectCloses(t *testing.T) {
	_, repo, input := imageSetPersistenceFixture(t)
	current, err := repo.GetProjection(context.Background(), imageagent.RunScope{TenantID: input.Identity.TenantID, OwnerUserID: input.Identity.UserID, RunID: input.RunID})
	require.NoError(t, err)
	plan := current.Plan
	second := plan.Slots[0]
	second.ID, second.IdempotencyKey = "second", "second"
	second.Recipe = imageagent.CloneImageSlotRecipe(second.Recipe)
	second.Recipe.Placement.Order = 2
	plan.Slots = append(plan.Slots, second)
	plan.Set.MaxPoints = 24
	plan.Set.QuoteDigest, err = imageagent.ImageSetQuoteDigest(plan)
	require.NoError(t, err)
	hash := strings.Repeat("a", 64)
	proof := &imageagent.ImageGenerationProof{UnitVersion: "ai-micropoint-v1", IntentID: hash, Fingerprint: hash, SettlementProofDigest: hash, Points: 12}
	results := []SlotWorkflowResult{
		{Execution: imageagent.SlotExecutionResult{SlotID: plan.Slots[0].ID, Attempt: 1, Candidates: []imageagent.AssetCandidate{{AssetID: "generated", SourceAssetID: "source-1", Width: 1024, Height: 1024, Operations: []string{"render_source_edit"}, DurableAsset: imageagent.DurableAssetIdentity{ObjectKey: "image-agent/public/tenant-a/run-1/1/overview/1/0-" + hash + ".png", SHA256: hash}, GenerationProof: proof}}}, Status: imageagent.SlotStatusAccepted, Closure: &imageagent.ImageSlotClosure{UnitVersion: "ai-micropoint-v1", Kind: "settled", IntentID: hash, Fingerprint: hash, SettlementProofDigest: hash, Points: 12}},
		{Execution: imageagent.SlotExecutionResult{SlotID: second.ID}, Status: imageagent.SlotStatusBlocked, ErrorCode: imageagent.BudgetElapsedCode, Closure: &imageagent.ImageSlotClosure{UnitVersion: "ai-micropoint-v1", Kind: "not_dispatched"}},
	}

	projection := summarizeCurrentResults(plan, results)
	require.Equal(t, imageagent.RunStatusAwaitingFinalApproval, projection.Status)
	require.Nil(t, projection.Block)
	digest, err := projectionResultDigest(plan, results)
	require.NoError(t, err)
	require.NotEmpty(t, digest)
	acceptedClosure := results[0].Closure
	results[0].Closure = nil
	results[0].EffectPhase = imageagent.SlotEffectV3PublicationComplete
	require.Equal(t, imageagent.RunStatusBlocked, summarizeCurrentResults(plan, results).Status, "materialized output without original settlement evidence must stay blocked")
	results[0].Closure = acceptedClosure
	results[1].Closure = nil
	results[1].Execution.Attempt = 1
	results[1].ErrorCode = imageagent.SlotProviderOutcomeUnknownCode
	results[1].EffectPhase = imageagent.SlotEffectV3ProviderUnknown
	unknown := summarizeCurrentResults(plan, results)
	require.Equal(t, []imageagent.RecoverableEffect{{SlotID: second.ID, Attempt: 1, Code: imageagent.SlotProviderOutcomeUnknownCode}}, unknown.RecoverableEffects, "the full set must expose the original UNKNOWN recovery identity")
	projection = summarizeCurrentResults(plan, results)
	require.Equal(t, imageagent.RunStatusBlocked, projection.Status)
	_, err = projectionResultDigest(plan, results)
	require.Error(t, err)
}

type setPlanRepository struct {
	imageagent.Repository
	projection imageagent.RunProjection
}

func (r setPlanRepository) GetProjection(context.Context, imageagent.RunScope) (imageagent.RunProjection, error) {
	return r.projection, nil
}

func TestV3ImageSetExecutionRejectsAChangedOrOmittedPersistedRecipe(t *testing.T) {
	activities, _, activityInput := imageSetPersistenceFixture(t)
	input := slotExecutionInputV3(activityInput)
	set := activityInput.ImageSet
	require.NoError(t, activities.validatePersistedImageSetExecution(context.Background(), input))
	input.ImageSet = nil
	require.ErrorIs(t, activities.validatePersistedImageSetExecution(context.Background(), input), imageagent.ErrRevisionConflict)
	input.ImageSet = imageagent.CloneImageSetPlan(set)
	input.Slot.Recipe = &imageagent.ImageSlotRecipe{Prompt: "changed prompt"}
	require.ErrorIs(t, activities.validatePersistedImageSetExecution(context.Background(), input), imageagent.ErrRevisionConflict)
}

func TestSetRegenerationCannotUseTheOriginalRunsRetryCommand(t *testing.T) {
	input := WorkflowInput{Plan: imageagent.Plan{Revision: 1, Set: &imageagent.ImageSetPlan{Schema: imageagent.ImageSetSchema}, Slots: []imageagent.Slot{{ID: "detail"}}}}
	results := []SlotWorkflowResult{{Status: imageagent.SlotStatusBlocked}}
	projection := WorkflowResult{Status: imageagent.RunStatusBlocked, Block: &imageagent.Block{Code: imageagent.SlotProviderNotDispatchedCode, SlotID: "detail"}}
	state := workflowUpdateState{input: &input, results: &results, projection: &projection}
	err := state.validateRetrySlotBusiness(RetrySlotSignal{PlanRevision: 1, SlotID: "detail"})
	require.Error(t, err, "a new set generation must obtain a fresh plan confirmation and admission")
}

func TestSetRecoveryCompletionCannotDiscardSettlementEvidence(t *testing.T) {
	input := WorkflowInput{RunID: "run", Plan: imageagent.Plan{Revision: 1, Set: &imageagent.ImageSetPlan{Schema: imageagent.ImageSetSchema}, Slots: []imageagent.Slot{{ID: "detail", Recipe: &imageagent.ImageSlotRecipe{Quote: imageagent.ImageGenerationQuote{Points: 12}}}}}}
	results := []SlotWorkflowResult{{Execution: imageagent.SlotExecutionResult{SlotID: "detail", Attempt: 1}, Status: imageagent.SlotStatusBlocked}}
	projection := WorkflowResult{Status: imageagent.RunStatusBlocked, Slots: []imageagent.SlotProjection{{Slot: input.Plan.Slots[0], Attempt: 1}}, RecoverableEffects: []imageagent.RecoverableEffect{{SlotID: "detail", Attempt: 1, Code: imageagent.SlotProviderOutcomeUnknownCode}}}
	state := workflowUpdateState{input: &input, results: &results, projection: &projection}
	signal := EffectRecoveryCompletedSignal{RunID: "run", PlanRevision: 1, SlotID: "detail", Attempt: 1, Result: EffectRecoveryResult{Outcome: EffectRecoveryOutcomePublished, EffectPhase: imageagent.SlotEffectV3PublicationComplete, Published: imageagent.SlotEffectV3PublishedResult{SlotID: "detail", Attempt: 1, Candidates: []imageagent.SlotEffectV3AssetCandidate{{AssetID: "candidate", SourceAssetID: "source", DurableAsset: imageagent.DurableAssetIdentity{ObjectKey: "image-agent/public/tenant-a/run-1/1/detail/1/0-" + v3SHA256 + ".png", SHA256: v3SHA256}, Width: 1024, Height: 1024, Operations: []string{"render_source_edit"}}}}}}
	require.False(t, state.applyEffectRecoveryCompleted(signal), "a set recovery signal needs the original generation and settlement references")
	signal.Result.Closure = &imageagent.ImageSlotClosure{UnitVersion: "ai-micropoint-v1", Kind: "settled", IntentID: v3SHA256, Fingerprint: v3SHA256, SettlementProofDigest: v3SHA256, Points: 12}
	signal.Result.GenerationProof = &imageagent.ImageGenerationProof{UnitVersion: "ai-micropoint-v1", IntentID: v3SHA256, Fingerprint: v3SHA256, SettlementProofDigest: v3SHA256, Points: 13}
	require.False(t, state.applyEffectRecoveryCompleted(signal), "recovery must preserve the exact original point quote")
	signal.Result.GenerationProof.Points = 12
	require.True(t, state.applyEffectRecoveryCompleted(signal))
	require.Equal(t, signal.Result.Closure, projection.Slots[0].Closure)
	require.Equal(t, signal.Result.GenerationProof, projection.Slots[0].Candidates[0].GenerationProof)
}

func TestOwnedSetLateUnknownCannotReopenOriginalTerminalFailure(t *testing.T) {
	input := WorkflowInput{RunID: "run", Plan: imageagent.Plan{Revision: 1,
		Set:   &imageagent.ImageSetPlan{Source: imageagent.ImageSourceBinding{OwnProduct: true}},
		Slots: []imageagent.Slot{{ID: "main"}, {ID: "detail"}}}}
	results := make([]SlotWorkflowResult, 2)
	for index, slot := range input.Plan.Slots {
		results[index] = SlotWorkflowResult{Execution: imageagent.SlotExecutionResult{SlotID: slot.ID, Attempt: 1},
			Status: imageagent.SlotStatusBlocked, ErrorCode: imageagent.SlotProviderOutcomeUnknownCode, EffectPhase: imageagent.SlotEffectV3ProviderUnknown}
	}
	projection := WorkflowResult{Status: imageagent.RunStatusBlocked, Slots: slotProjections(input.Plan, results), RecoverableEffects: imageSetRecoverableEffects(results)}
	state := workflowUpdateState{input: &input, results: &results, projection: &projection}
	closure := &imageagent.ImageSlotClosure{UnitVersion: agentconfig.ProviderImageUsageUnit, Kind: "provider_failed",
		IntentID: v3SHA256, Fingerprint: v3SHA256, ProviderUsageProofDigest: v3SHA256}
	signal := EffectRecoveryCompletedSignal{RunID: input.RunID, PlanRevision: 1, SlotID: "main", Attempt: 1,
		Result: EffectRecoveryResult{Outcome: EffectRecoveryOutcomeProviderUnknown, EffectPhase: imageagent.SlotEffectV3ProviderUnknown,
			BlockedCode: imageagent.ImageProviderFailedCode, Closure: closure}}
	require.True(t, state.applyEffectRecoveryCompleted(signal))

	signal.Result.Closure = nil
	signal.Result.BlockedCode = imageagent.SlotProviderOutcomeUnknownCode
	require.False(t, state.applyEffectRecoveryCompleted(signal))
	require.Equal(t, closure, projection.Slots[0].Closure)
	require.Equal(t, imageagent.ImageProviderFailedCode, projection.Slots[0].ErrorCode)
	require.Equal(t, []imageagent.RecoverableEffect{{SlotID: "detail", Attempt: 1, Code: imageagent.SlotProviderOutcomeUnknownCode}}, projection.RecoverableEffects)
}

func TestOwnedSetCancellationKeepsTheOriginalUnresolvedAttempt(t *testing.T) {
	plan := imageagent.Plan{Revision: 1, Set: &imageagent.ImageSetPlan{Source: imageagent.ImageSourceBinding{OwnProduct: true}},
		Slots: []imageagent.Slot{{ID: "main"}, {ID: "detail"}, {ID: "unstarted"}}}
	proof := imageagent.ImageSlotClosure{UnitVersion: agentconfig.ProviderImageUsageUnit,
		Kind: "provider_usage", IntentID: v3SHA256, Fingerprint: v3SHA256, ProviderUsageProofDigest: v3SHA256}
	results := []SlotWorkflowResult{
		{Execution: imageagent.SlotExecutionResult{SlotID: "main", Attempt: 1,
			Candidates: []imageagent.AssetCandidate{{AssetID: "candidate", GenerationProof: proof.CandidateProof()}}},
			Status: imageagent.SlotStatusAccepted, EffectPhase: imageagent.SlotEffectV3PublicationComplete, Closure: &proof},
		{Execution: imageagent.SlotExecutionResult{SlotID: "detail", Attempt: 1}, Status: imageagent.SlotStatusBlocked,
			ErrorCode: imageagent.SlotProviderOutcomeUnknownCode, EffectPhase: imageagent.SlotEffectV3ProviderUnknown},
		{},
	}
	input := WorkflowInput{RunID: "run", Plan: plan, projectionExecutionID: "execution"}
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	var persisted []WorkflowResult
	var started []EffectRecoveryWorkflowInput
	env.RegisterActivityWithOptions(func(_ context.Context, in PersistRunStateActivityInput) error {
		persisted = append(persisted, in.Projection)
		return nil
	}, sdkactivity.RegisterOptions{Name: activityPersistRunState})
	env.RegisterActivityWithOptions(func(_ context.Context, in PersistSlotResultV3ActivityInput) (imageagent.SlotProjection, error) {
		if in.Result.Published.SlotID != "unstarted" || in.Result.EffectPhase != imageagent.SlotEffectV3ProviderNotDispatched {
			return imageagent.SlotProjection{}, imageagent.ErrRevisionConflict
		}
		slot := plan.Slots[2]
		slot.Status = imageagent.SlotStatusBlocked
		return imageagent.SlotProjection{Slot: slot, ErrorCode: in.Result.ErrorCode,
			Closure: &imageagent.ImageSlotClosure{UnitVersion: "ai-micropoint-v1", Kind: "not_dispatched"}}, nil
	}, sdkactivity.RegisterOptions{Name: activityPersistImageSetSlotResult})
	env.RegisterActivityWithOptions(func(_ context.Context, in EffectRecoveryWorkflowInput) error {
		started = append(started, in)
		return nil
	}, sdkactivity.RegisterOptions{Name: activityStartEffectRecoveryV3})
	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		ctx = imageAgentActivityContext(ctx)
		projection := executingProjection(plan, results)
		state := newWorkflowUpdateState(ctx, &input, &projection, &results, newWorkflowEffectOwner(ctx))
		state.cancelActionFingerprint, state.lastBlockedCancelFingerprint = "cancel-request", "cancel-request"
		state.cancelPending = true
		state.commitPendingCancellation(ctx, results)
		if state.cancelCommitErr != nil {
			return state.cancelCommitErr
		}
		if !state.cancelBlocked {
			return imageagent.ErrRevisionConflict
		}
		failed := proof
		failed.Kind = "provider_failed"
		if !state.applyEffectRecoveryCompleted(EffectRecoveryCompletedSignal{RunID: input.RunID, PlanRevision: 1, SlotID: "detail", Attempt: 1,
			Result: EffectRecoveryResult{Outcome: EffectRecoveryOutcomeProviderUnknown, EffectPhase: imageagent.SlotEffectV3ProviderUnknown,
				BlockedCode: imageagent.ImageProviderFailedCode, Closure: &failed}}) || !state.cancelPending {
			return imageagent.ErrRevisionConflict
		}
		state.commitPendingCancellation(ctx, results)
		return state.cancelCommitErr
	})
	require.NoError(t, env.GetWorkflowError())
	require.Len(t, persisted, 2)
	handoff := persisted[0]
	require.Equal(t, imageagent.RunStatusBlocked, handoff.Status)
	require.Equal(t, []string{"main"}, handoff.CompletedSlotIDs)
	require.Equal(t, []imageagent.RecoverableEffect{{SlotID: "detail", Attempt: 1, Code: recoveryRequestedBlockCode}}, handoff.RecoverableEffects)
	require.Equal(t, "detail", handoff.Block.SlotID)
	require.True(t, handoff.Slots[2].Closure.Valid(0))
	require.Len(t, started, 1)
	require.Equal(t, "detail", started[0].Slot.ID)
	require.Equal(t, 1, started[0].Attempt)
	cancelled := persisted[1]
	require.Equal(t, imageagent.RunStatusCancelled, cancelled.Status)
	require.Empty(t, cancelled.RecoverableEffects)
	require.Equal(t, "provider_failed", cancelled.Slots[1].Closure.Kind)
	require.Equal(t, handoff.Slots[0], cancelled.Slots[0])
}
