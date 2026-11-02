package httpapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/listing/materials"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/product/collection"
)

func materialV1HistoricalState() agent.State {
	s := agent.State{RunID: "historical-run", Scope: agent.Scope{OrganizationID: "org", ActorID: "actor"}, Phase: agent.HumanReviewRequired,
		Request:        agent.Request{Binding: agent.Binding{ContextKind: "listing-target-record", ContextID: "record", ProductKey: "product", CatalogVersion: "1", PublicationID: "publication", TargetPlatform: "shein"}, PolicyVersion: "listing-materials-policy-v1", PromptVersion: "listing-materials-prompt-v1"},
		DomainProposal: &agent.DomainProposal{Kind: "listing-materials-v1", Payload: json.RawMessage(`{"brandCode":"brand","evidenceIds":["original-evidence"]}`)}}
	s.Validation = &agent.Validation{Valid: true, PolicyVersion: "listing-materials-policy-v1", CandidateHash: collection.Digest(s.DomainProposal)}
	return s
}

func TestMaterialsV1CommittedReceiptKeepsOriginalIntegrityWithoutCurrentRules(t *testing.T) {
	s := materialV1HistoricalState()
	require.True(t, materialsHistoricalReviewable(s))
	base := record.TargetRecord{ID: "record", TargetID: "target", Revision: 3, EffectiveVersion: 1}
	p, err := materials.ParseProposal(s.DomainProposal.Payload)
	require.NoError(t, err)
	input, err := materials.ApplyProposal(base, p)
	require.NoError(t, err)
	prior := record.TargetReceipt{Record: record.TargetRecord{TargetID: base.TargetID, Revision: 4, EffectiveVersion: 1, Input: input}}
	require.NoError(t, verifyMaterialsAdoptionReceipt(base, s, prior))
	prior.Record.Input.Draft.Product.BrandCode = "tampered"
	require.Error(t, verifyMaterialsAdoptionReceipt(base, s, prior))
	s.Validation.CandidateHash = "tampered"
	require.False(t, materialsHistoricalReviewable(s))
}

func TestMaterialsV1HistoricalProposalCannotEnterNewAdoption(t *testing.T) {
	s := materialV1HistoricalState()
	before, err := json.Marshal(s)
	require.NoError(t, err)
	require.False(t, materialsRunReviewable(s), "D1 forbids new adoption of a v1 proposal after the v2 cutover")
	after, err := json.Marshal(s)
	require.NoError(t, err)
	require.Equal(t, before, after, "an execution refusal must not rewrite the historical run or reservation")
}
