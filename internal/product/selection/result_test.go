package selection

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/commercetool"
	"task-processor/internal/product/collection"
	"testing"
)

func TestSelectionResultNeverCallsRepairExhaustionASuccess(t *testing.T) {
	inspector, ctx, choices, _ := inspectionFixture(t)
	binding, out, err := inspector.Prepare(ctx, uuid.NewString(), choices)
	require.NoError(t, err)
	proposal := Proposal{Overview: "进一步核对供货条件", Assessments: []Assessment{{ItemID: out.Products[0].Ref.ItemID, Advice: "核实成本", EvidenceIDs: []string{out.Products[0].Facts[0].ID}}}}
	payload, err := json.Marshal(proposal)
	require.NoError(t, err)
	wire, err := json.Marshal(out)
	require.NoError(t, err)
	state := agent.State{Phase: agent.HumanReviewRequired, Request: agent.Request{Binding: binding, PolicyVersion: agentconfig.SelectionPolicyVersion, PromptVersion: agentconfig.SelectionPromptVersion}, DomainProposal: &agent.DomainProposal{Kind: agentconfig.SelectionProposalKind, Payload: payload}, History: []agent.Observation{{Tool: agentconfig.SelectionInspectToolRef(), CallID: "actual-read", AuditStatus: commercetool.AuditStatusRecorded, Output: wire}}}
	state.Validation = &agent.Validation{Valid: true, PolicyVersion: agentconfig.SelectionPolicyVersion, CandidateHash: collection.Digest(state.DomainProposal)}
	_, _, ok := ValidatedResult(state)
	require.True(t, ok)
	invalid := state
	invalid.StopReason = agent.StopRepairLimit
	_, _, ok = ValidatedResult(invalid)
	require.False(t, ok)
	invalid = state
	invalid.Validation = &agent.Validation{Valid: false, PolicyVersion: agentconfig.SelectionPolicyVersion, CandidateHash: state.Validation.CandidateHash}
	_, _, ok = ValidatedResult(invalid)
	require.False(t, ok)
	invalid = state
	invalid.History = nil
	_, _, ok = ValidatedResult(invalid)
	require.False(t, ok)
	invalid = state
	invalid.Validation = &agent.Validation{Valid: true, PolicyVersion: agentconfig.SelectionPolicyVersion, CandidateHash: "wrong"}
	_, _, ok = ValidatedResult(invalid)
	require.False(t, ok)
}

func TestSelectionAdmissionRejectsUnsupportedMetricsAndTamperedFacts(t *testing.T) {
	_, err := ParseProposal(json.RawMessage(`{"overview":"made up","assessments":[],"profit":100}`))
	require.Error(t, err)
	inspector, ctx, choices, _ := inspectionFixture(t)
	_, out, err := inspector.Prepare(ctx, uuid.NewString(), choices)
	require.NoError(t, err)
	raw, err := json.Marshal(out)
	require.NoError(t, err)
	_, err = (ModelAdmission{}).ProjectEvidence(raw)
	require.NoError(t, err)
	out.Products[0].Facts[0].Value = json.RawMessage(`"invented title"`)
	raw, err = json.Marshal(out)
	require.NoError(t, err)
	_, err = (ModelAdmission{}).ProjectEvidence(raw)
	require.Error(t, err)
}
