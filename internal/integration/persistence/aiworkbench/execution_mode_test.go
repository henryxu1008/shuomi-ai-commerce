package aiworkbenchpersistence

import (
	"github.com/stretchr/testify/require"
	"task-processor/internal/aiworkbench"
	"testing"
)

func TestNewProposalRequiresObservedModeRevision(t *testing.T) {
	p := aiworkbench.ExecutionProposal{Kind: "product.title.optimize", GoalSummary: "Improve title", OperationID: "op", ProductKey: "product", CatalogVersion: "1", PublicationID: "publication", TargetPlatform: "shein", AgentID: "product.title.agent", AgentVersion: "v1.0.0", ObservedAgentRevision: "1", ObservedActivationEpoch: "1", ExecutionModelProfile: []byte(`{"profile_id":"controlled"}`)}
	for _, revision := range []string{"", "00", "-1", "9223372036854775808"} {
		p.ObservedModeRevision = revision
		require.False(t, validProposal(&p), "revision %q must not authorize a new proposal", revision)
	}
	for _, revision := range []string{"0", "1", "9223372036854775807"} {
		p.ObservedModeRevision = revision
		require.True(t, validProposal(&p), "canonical revision %q", revision)
	}
}
