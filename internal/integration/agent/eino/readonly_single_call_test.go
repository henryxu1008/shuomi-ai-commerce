package einoruntime

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/commercetool"
	"task-processor/internal/product/catalog/tools/canonicalinspect"
)

func TestObservedInvalidKeepsBoundedReasonsOnOriginalReadback(t *testing.T) {
	for _, oversized := range []bool{false, true} {
		r, request, model, _, _, _ := readonlyRuntime(t)
		model.observedInvalid = true
		model.observedUnresolved = []string{`own-intake-validation:{"field":"rows[0].stock","code":"stock_evidence"}`}
		if oversized {
			model.observedUnresolved = []string{strings.Repeat("x", agent.MaxModelOutputBytes)}
		}
		record, err := r.Start(context.Background(), request)
		require.NoError(t, err)
		require.Equal(t, agent.StopInvalidOutput, record.State.StopReason)
		if oversized {
			require.Empty(t, record.State.Unresolved)
		} else {
			require.Equal(t, model.observedUnresolved, record.State.Unresolved)
		}
		replay, err := r.Start(context.Background(), request)
		require.NoError(t, err)
		require.Equal(t, record.State.Unresolved, replay.State.Unresolved)
		require.Equal(t, 1, model.calls)
		require.EqualValues(t, 2, record.State.Usage.Tokens)
		require.Nil(t, record.State.DomainProposal)
	}
}

type readonlyValidator struct {
	valid bool
	calls int
}

func (v *readonlyValidator) ValidateDomain(_ context.Context, r agent.Request, _ agent.DomainProposal, _ []agent.Observation) (agent.Validation, error) {
	v.calls++
	return agent.Validation{Valid: v.valid, PolicyVersion: r.PolicyVersion}, nil
}

func readonlyRuntime(t *testing.T) (*Runtime, agent.Request, *fakeModel, *fakeTools, *readonlyValidator, *fakeStore) {
	t.Helper()
	r, request, m, _, store := listingDomainFixture(t, domainProposalFixture())
	cfg := r.config
	cfg.SingleCallReadOnly = true
	cfg.PreReadTools = []commercetool.ToolRef{canonicalinspect.Definition().Ref}
	v := &readonlyValidator{valid: true}
	cfg.DomainValidator = v
	r, err := New(cfg)
	require.NoError(t, err)
	request.Limits.ModelCalls = 1
	return r, request, m, cfg.Gateway.(*fakeTools), v, store
}

func TestReadonlySingleCallReadsActualEvidenceBeforeModelAndReplaysOnly(t *testing.T) {
	r, request, model, tools, validator, _ := readonlyRuntime(t)
	record, err := r.Start(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, 1, tools.calls)
	require.Equal(t, 1, model.calls)
	require.Equal(t, 1, validator.calls)
	require.Len(t, model.lastInput.History, 1, "fixed pre-read is real Tool evidence, not a paid model decision")
	require.Equal(t, canonicalinspect.Definition().Ref, model.lastInput.History[0].Tool)
	require.Equal(t, commercetool.AuditStatusRecorded, model.lastInput.History[0].AuditStatus)
	require.Empty(t, model.lastInput.History[0].InvocationID)
	require.Equal(t, 1, record.State.Usage.ModelCalls)
	require.Zero(t, record.State.Repairs)
	replay, err := r.Start(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, record.State.RunID, replay.State.RunID)
	require.Equal(t, 1, model.calls)
	require.Equal(t, 1, tools.calls)
	_, err = r.Resume(context.Background(), request, record.State.Revision, "try again")
	require.ErrorIs(t, err, agent.ErrInvalid, "readonly analysis has no resume path that can send again")
}

func TestReadonlySingleCallAuditFailurePreventsAnyModelCall(t *testing.T) {
	r, request, model, tools, validator, _ := readonlyRuntime(t)
	tools.auditFail = true
	record, err := r.Start(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, agent.StopAudit, record.State.StopReason)
	require.Zero(t, model.calls)
	require.Zero(t, validator.calls)
}

func TestReadonlySingleCallInvalidResultDoesNotRepair(t *testing.T) {
	r, request, model, _, validator, _ := readonlyRuntime(t)
	validator.valid = false
	record, err := r.Start(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, 1, model.calls)
	require.Equal(t, 1, validator.calls)
	require.Zero(t, record.State.Repairs)
	require.Equal(t, agent.StopInvalidOutput, record.State.StopReason)
	require.Equal(t, agent.Stopped, record.State.Phase)
}

func TestReadonlySingleCallRejectsExpandedBudgetBeforeClaim(t *testing.T) {
	r, request, model, _, _, store := readonlyRuntime(t)
	request.Limits.ModelCalls = 2
	_, err := r.Start(context.Background(), request)
	require.ErrorIs(t, err, agent.ErrInvalid)
	require.False(t, store.exists)
	require.Zero(t, model.calls)
}
