package httpapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/listing/materials"
	"task-processor/internal/product/collection"
)

func TestListingMaterialAdoptionRequiresExactReviewableDomainResult(t *testing.T) {
	state := agent.State{RunID: "run", Scope: agent.Scope{OrganizationID: "org", ActorID: "actor"}, Phase: agent.HumanReviewRequired, Request: agent.Request{Binding: agent.Binding{ContextKind: "listing-target-record", ContextID: "record", ProductKey: "product", CatalogVersion: "1", PublicationID: "publication", TargetPlatform: "shein"}, PolicyVersion: materials.PolicyVersion, PromptVersion: materials.PromptVersion}, DomainProposal: &agent.DomainProposal{Kind: materials.ProposalKind, Payload: json.RawMessage(`{"brandCode":"brand","evidenceIds":["observed"]}`)}}
	state.Validation = &agent.Validation{Valid: true, PolicyVersion: materials.PolicyVersion, CandidateHash: collection.Digest(state.DomainProposal)}
	require.True(t, materialsRunReviewable(state))
	key := materialsAdoptionKey(state)
	require.Equal(t, key, materialsAdoptionKey(state), "same run/candidate never creates another Target command")
	for _, change := range []func(*agent.State){
		func(s *agent.State) { s.Phase = agent.Interrupted },
		func(s *agent.State) { s.StopReason = agent.StopUsageUnknown },
		func(s *agent.State) { v := *s.Validation; v.CandidateHash = "changed"; s.Validation = &v },
		func(s *agent.State) { s.Request.Binding.ContextKind = "acquisition" },
	} {
		copied := state
		change(&copied)
		require.False(t, materialsRunReviewable(copied))
	}
}
