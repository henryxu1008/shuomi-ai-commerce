package review

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/product/enrichment"
	"testing"
)

type skillTransactionProbe struct {
	*candidateGateStore
	afterCallback func()
}

func (s skillTransactionProbe) Run(ctx context.Context, op Operation, fn func(Tx) (View, error)) (View, error) {
	v, e := s.candidateGateStore.Run(ctx, op, fn)
	s.afterCallback()
	return v, e
}
func skillCandidateInput(t *testing.T) (*Service, *candidateGateStore, context.Context, CandidateInput) {
	t.Helper()
	service, store, ctx, input := candidateGateFixture(t, true)
	selection := enrichment.ContentSelection{TargetLanguage: "en-US", Title: true}
	input.ContentSelection = selection
	input.Execution.ContentSelection = selection
	input.Candidate.Changes = input.Candidate.Changes[:1]
	input.Execution.SkillVersionRef = agent.SkillVersionRef{Kind: "skill-version-v1", ID: uuid.NewString(), OwnerKind: "USER", OwnerID: input.Execution.Scope.ActorID, Version: 1, Digest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", PolicyVersion: "skill-method-v1"}
	service.WithExecutionPolicy(func(_ context.Context, e ExecutionProvenance, _ enrichment.Candidate, _ bool) (PolicyProvenance, error) {
		return candidateGateProof(e, agent.ModeDefault), nil
	})
	return service, store, ctx, input
}
func TestSkillGateCoversTheEntireCandidateTransaction(t *testing.T) {
	service, store, ctx, input := skillCandidateInput(t)
	held, released := false, false
	service.WithExecutionGate(func(_ context.Context, e ExecutionProvenance) (func(), error) {
		require.Equal(t, *input.Execution, e)
		held = true
		return func() { held = false; released = true }, nil
	})
	service.store = skillTransactionProbe{candidateGateStore: store, afterCallback: func() { require.True(t, held, "gate released before transaction result was determined") }}
	_, e := service.CreateFromCandidate(ctx, uuid.NewString(), input)
	require.NoError(t, e)
	require.True(t, released)
	require.False(t, held)
}
func TestDeniedSkillGateHasNoNewCandidateEffects(t *testing.T) {
	service, store, ctx, input := skillCandidateInput(t)
	service.WithExecutionGate(func(context.Context, ExecutionProvenance) (func(), error) { return nil, ErrForbidden })
	_, e := service.CreateFromCandidate(ctx, uuid.NewString(), input)
	require.ErrorIs(t, e, ErrForbidden)
	require.Zero(t, store.saved)
	require.Zero(t, store.completed)
}
