package temporal

import (
	"github.com/stretchr/testify/require"
	"task-processor/internal/imageagent"
	"testing"
)

func TestArtworkPendingPublicationRetainsExactResultDigestWithoutImageSetSelection(t *testing.T) {
	s := &workflowUpdateState{input: &WorkflowInput{Plan: imageagent.Plan{Revision: 1, Artwork: &imageagent.ArtworkPlan{Schema: imageagent.ArtworkSchema}}}}
	record := &workflowUpdateRecord{kind: "approve_results", phase: updatePhaseApprovalPublishStarted, command: ApproveResultsSignal{RunID: "artwork-run", PlanRevision: 1, ResultDigest: "original-artwork-result", ActorID: "actor", ActionID: "save"}}
	pending := s.pendingReceipt("save", record)
	require.Equal(t, "original-artwork-result", pending.ResultDigest)
	require.Empty(t, pending.SelectionDigest, "saving an artwork does not invent a product-image set approval")
}
