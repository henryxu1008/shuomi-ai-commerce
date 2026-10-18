package temporal

import (
	"context"
	"testing"

	"task-processor/internal/imageagent"

	"github.com/stretchr/testify/require"
)

type recordingGenerationFinalizer struct {
	calls    int
	identity imageagent.SlotExternalEffectIdentity
	err      error
	state    imageagent.GenerationState
}

type generationScopedProjectionRepository struct {
	imageagent.Repository
	identity imageagent.ExecutionIdentity
}

func (r generationScopedProjectionRepository) GetProjection(ctx context.Context, scope imageagent.RunScope) (imageagent.RunProjection, error) {
	p, err := r.Repository.GetProjection(ctx, scope)
	if err != nil {
		return p, err
	}
	p.Run.ScopeProtocol = r.identity.ScopeProtocol
	p.Run.MemberID = r.identity.MemberID
	p.Run.BusinessTaskID = r.identity.BusinessTaskID
	return p, nil
}

type revokedGenerationAuthorizer struct{ calls int }

func (a *revokedGenerationAuthorizer) AuthorizeExecution(context.Context, imageagent.ExecutionIdentity) error {
	a.calls++
	return imageagent.ErrIdentityRequired
}

type allowGenerationExecution struct{}

func (allowGenerationExecution) AuthorizeExecution(context.Context, imageagent.ExecutionIdentity) error {
	return nil
}

func TestGenerationFinalizationPrecedesV3TerminalPhaseShortcuts(t *testing.T) {
	for _, phase := range []string{"published", "provider_unknown"} {
		t.Run(phase, func(t *testing.T) {
			repo, input := initializedSlotEffectV3Activity(t, "run-generation-"+phase)
			effects := repo.(imageagent.SlotExternalEffectV3Repository)
			executor := &recordingStagedExecutor{generated: generatedV3Output(input, writeTinyPNG(t))}
			artifacts := &recordingArtifactStore{}
			a := newV3Activities(t, repo, effects, executor, artifacts)
			if phase == "provider_unknown" {
				_, _, err := effects.ReserveSlotProviderV3(context.Background(), v3Reservation(input))
				require.NoError(t, err)
			}
			_, err := a.ExecuteSlotV3(context.Background(), input)
			if phase == "published" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			before := executor.GenerateCalls()
			input.Identity.ScopeProtocol = imageagent.OrganizationScopeProtocol
			input.Identity.RunID = input.RunID
			input.Identity.MemberID = "grant-1"
			input.Identity.BusinessTaskID = "acquisition-1"
			a.repository = generationScopedProjectionRepository{Repository: repo, identity: input.Identity}
			a.executionAuthorizer = allowGenerationExecution{}
			finalizer := &recordingGenerationFinalizer{}
			a.generationRecovery = finalizer
			_, err = a.RecoverEffectV3(context.Background(), effectRecoveryWorkflowInput(input))
			require.NoError(t, err)
			require.Equal(t, 1, finalizer.calls)
			require.Equal(t, before, executor.GenerateCalls())
		})
	}
}

func (r *recordingGenerationFinalizer) ReconcileExistingGeneration(ctx context.Context, id imageagent.SlotExternalEffectIdentity) (imageagent.GenerationState, error) {
	r.calls++
	r.identity = id
	if r.state != "" {
		return r.state, r.err
	}
	return imageagent.GenerationSucceeded, r.err
}

type noGenerationReserveCounter struct {
	imageagent.SlotExternalEffectV3Repository
	calls int
}

func (r *noGenerationReserveCounter) ReserveSlotProviderV3(ctx context.Context, input imageagent.SlotEffectV3Reservation) (imageagent.SlotEffectV3Attempt, bool, error) {
	r.calls++
	return r.SlotExternalEffectV3Repository.ReserveSlotProviderV3(ctx, input)
}
func TestTerminalNoGenerationGuardsExecutionAndRecoveryBeforeAnyReserve(t *testing.T) {
	for _, entry := range []string{"execute", "recover"} {
		t.Run(entry, func(t *testing.T) {
			repo, input := initializedSlotEffectV3Activity(t, "run-terminal-no-generation")
			input.Identity = imageagent.ExecutionIdentity{ScopeProtocol: imageagent.OrganizationScopeProtocol, RunID: input.RunID, TenantID: input.Identity.TenantID, UserID: input.Identity.UserID, MemberID: "grant-1", BusinessTaskID: "task-1"}
			effects := &noGenerationReserveCounter{SlotExternalEffectV3Repository: repo.(imageagent.SlotExternalEffectV3Repository)}
			executor := &recordingStagedExecutor{}
			artifacts := &recordingArtifactStore{}
			a := newV3Activities(t, repo, effects, executor, artifacts)
			a.generationRecovery = &recordingGenerationFinalizer{state: imageagent.GenerationNoEffect}
			var err error
			if entry == "execute" {
				_, err = a.ExecuteSlotV3(context.Background(), input)
			} else {
				_, err = a.RecoverEffectV3(context.Background(), EffectRecoveryWorkflowInput{RunID: input.RunID, Identity: input.Identity, PlanRevision: input.PlanRevision, Slot: input.Slot, Attempt: input.Attempt, AssetCatalog: input.AssetCatalog})
			}
			var known *imageagent.ProviderDispatchError
			require.ErrorAs(t, err, &known)
			require.Equal(t, imageagent.ProviderRejectedBeforeEffect, known.State)
			require.Zero(t, effects.calls)
			require.Zero(t, executor.GenerateCalls())
			require.Zero(t, artifacts.PrepareCalls())
		})
	}
}

func TestGenerationFinalizationPrecedesRevokedExecutionAndRecovery(t *testing.T) {
	for _, entry := range []string{"execute", "recovery"} {
		t.Run(entry, func(t *testing.T) {
			repo, input := initializedSlotEffectV3Activity(t, "run-generation-revoked")
			input.Identity.ScopeProtocol = imageagent.OrganizationScopeProtocol
			input.Identity.RunID = input.RunID
			input.Identity.MemberID = "grant-1"
			input.Identity.BusinessTaskID = "acquisition-1"
			executor := &recordingStagedExecutor{}
			artifacts := &recordingArtifactStore{}
			activities := newV3Activities(t, repo, repo.(imageagent.SlotExternalEffectV3Repository), executor, artifacts)
			finalizer := &recordingGenerationFinalizer{}
			activities.generationRecovery = finalizer
			activities.repository = generationScopedProjectionRepository{Repository: repo, identity: input.Identity}
			authorizer := &revokedGenerationAuthorizer{}
			activities.executionAuthorizer = authorizer
			// The scoped identity cannot pass the live execution boundary. Financial
			// readback must still run, without giving this caller an execution path.
			var err error
			if entry == "execute" {
				_, err = activities.ExecuteSlotV3(context.Background(), input)
			} else {
				_, err = activities.RecoverEffectV3(context.Background(), EffectRecoveryWorkflowInput{RunID: input.RunID, Identity: input.Identity, PlanRevision: input.PlanRevision, Slot: input.Slot, Attempt: input.Attempt, AssetCatalog: input.AssetCatalog})
			}
			require.ErrorIs(t, err, imageagent.ErrIdentityRequired)
			require.Equal(t, 1, finalizer.calls)
			require.Equal(t, 1, authorizer.calls, "live authorization still rejects all new execution")
			require.Equal(t, v3Reservation(input).Identity, finalizer.identity)
			require.Zero(t, executor.GenerateCalls())
			require.Zero(t, artifacts.PrepareCalls())
			require.Zero(t, artifacts.FinalizeCalls())
		})
	}
}
