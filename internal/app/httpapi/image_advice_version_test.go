package httpapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/commercetool"
	"task-processor/internal/listing/materials"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/catalog/tools/canonicalinspect"
	"task-processor/internal/product/collection"
	proposal "task-processor/internal/product/review/proposal"
)

func TestPreviousMaterialsVersionIsHistoricalButNotAdoptable(t *testing.T) {
	s := agent.State{Phase: agent.HumanReviewRequired, Request: agent.Request{Binding: agent.Binding{ContextKind: "listing-target-record", ContextID: "record", ProductKey: "product", CatalogVersion: "1", PublicationID: "publication", TargetPlatform: "shein"}, PolicyVersion: "listing-materials-policy-v2", PromptVersion: "listing-materials-prompt-v2"}, DomainProposal: &agent.DomainProposal{Kind: materials.ProposalKind, Payload: json.RawMessage(`{"brandCode":"brand","evidenceIds":["observed"]}`)}}
	s.Validation = &agent.Validation{Valid: true, PolicyVersion: s.Request.PolicyVersion, CandidateHash: collection.Digest(s.DomainProposal)}
	require.True(t, materialsHistoricalReviewable(s), "read the exact previous candidate/committed receipt")
	require.False(t, materialsRunReviewable(s), "D2 forbids a previous definition from becoming a new adoption")
}

func TestPreviousTitleVersionCannotBecomeNewReview(t *testing.T) {
	b := agent.Binding{ProductKey: "product", CatalogVersion: "1", PublicationID: "publication"}
	output := canonicalinspect.Output{ProductKey: b.ProductKey, CatalogVersion: b.CatalogVersion, CatalogPublicationID: b.PublicationID, Snapshot: catalog.ProductSnapshot{Title: "Saved", Sources: []catalog.SourceRecord{{Detail: "source"}}}}
	raw, err := json.Marshal(output)
	require.NoError(t, err)
	s := agent.State{Phase: agent.HumanReviewRequired, Request: agent.Request{Binding: b, PolicyVersion: "title-review-v1", PromptVersion: "product-title-agent-v1"}, Validation: &agent.Validation{Valid: true}, Candidate: proposal.Candidate{Changes: []proposal.FieldChange{{Field: "title", Value: "Saved title", EvidenceIDs: []string{"source"}}}}, History: []agent.Observation{{Tool: canonicalinspect.Definition().Ref, CallID: "call", AuditStatus: commercetool.AuditStatusRecorded, Output: raw}}}
	require.True(t, agentCandidateEvidenceObserved(b, s.Candidate, s.History), "the old exact observation itself remains valid")
	require.False(t, agentRunReviewable(s), "D2 denies new review/adoption despite valid historical evidence")
}
