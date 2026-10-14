package shopdiagnosis

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/commercetool"
	"testing"
)

func TestDiagnosisModelNotesFitTheExistingPublicResponse(t *testing.T) {
	for _, notes := range [][]string{make([]string, 4096), {strings.Repeat("x", 8193)}} {
		require.Error(t, (ModelAdmission{}).ValidateAction(agent.Action{Kind: "interrupt", Unresolved: notes}))
	}
	notes := make([]string, 4095)
	for n := range notes {
		notes[n] = "missing fact"
	}
	require.NoError(t, (ModelAdmission{}).ValidateAction(agent.Action{Kind: "interrupt", Unresolved: notes}))
	validation, err := (Validator{}).ValidateDomain(context.Background(), agent.Request{}, agent.DomainProposal{}, nil)
	require.NoError(t, err)
	require.LessOrEqual(t, len(notes)+len(validation.Unresolved), 4096)
}

func TestDiagnosisOnlyAcceptsActualCitedAdvisoryResults(t *testing.T) {
	_, i := fixture()
	binding, out, e := i.Prepare(context.Background(), contextID, shopID, testPeriod)
	require.NoError(t, e)
	raw, _ := json.Marshal(out)
	parsed, e := ParseInspection(raw)
	require.NoError(t, e)
	require.Equal(t, out.Ref, parsed.Ref)
	proposal := Proposal{Overview: "核对已录入事实", Recommendations: []Recommendation{{Advice: "核对净利润与成本", EvidenceIDs: []string{out.Facts[0].ID}}}}
	require.NoError(t, ValidateProposal(out, proposal))
	proposal.Recommendations[0].EvidenceIDs = []string{"fact:foreign"}
	require.Error(t, ValidateProposal(out, proposal))
	_, e = ParseProposal(json.RawMessage(`{"overview":"建议","recommendations":[],"newRevenue":999}`))
	require.Error(t, e)
	tampered := out
	tampered.Facts = append([]Fact{}, out.Facts...)
	tampered.Facts[0].Value = "other shop"
	raw, _ = json.Marshal(tampered)
	_, e = ParseInspection(raw)
	require.Error(t, e)
	proposal.Recommendations[0].EvidenceIDs = []string{out.Facts[0].ID}
	payload, _ := json.Marshal(proposal)
	candidate := &agent.DomainProposal{Kind: agentconfig.DiagnosisProposalKind, Payload: payload}
	state := agent.State{Phase: agent.HumanReviewRequired, Request: agent.Request{Binding: binding, PolicyVersion: agentconfig.DiagnosisPolicyVersion, PromptVersion: agentconfig.DiagnosisPromptVersion}, DomainProposal: candidate, Validation: &agent.Validation{Valid: true, PolicyVersion: agentconfig.DiagnosisPolicyVersion, CandidateHash: digest(candidate)}, History: []agent.Observation{{Tool: agentconfig.DiagnosisInspectToolRef(), CallID: "call", AuditStatus: commercetool.AuditStatusRecorded, Output: mustJSON(out)}}}
	_, _, valid := ValidatedResult(state)
	require.True(t, valid)
	state.StopReason = agent.StopRepairLimit
	_, _, valid = ValidatedResult(state)
	require.False(t, valid, "repair exhaustion is not a successful recommendation")
}
func mustJSON(v any) json.RawMessage { raw, _ := json.Marshal(v); return raw }
