package aiworkbench

import (
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/product/enrichment"
	"testing"
	"time"
)

func TestContentProjectionUsesWholeSelectedFieldSet(t *testing.T) {
	now := time.Now()
	task := BusinessTask{Kind: "product.content.localize", Scope: Scope{"org", "actor"}, OperationID: "item", ProductKey: "product", TargetPlatform: "product", ExecutionRequestKey: "key"}
	run := agent.Record{State: agent.State{Scope: agent.Scope{OrganizationID: "org", ActorID: "actor"}, Request: agent.Request{Key: "key", Binding: agent.Binding{ContextKind: "own_product", ContextID: "item", ProductKey: "product", TargetPlatform: "product"}, PolicyVersion: enrichment.ContentPolicyVersion, ContentSelection: enrichment.ContentSelection{TargetLanguage: "en-US", Description: true, SellingPoints: true}}, Phase: agent.HumanReviewRequired, Validation: &agent.Validation{Valid: true, PolicyVersion: enrichment.ContentPolicyVersion}, Candidate: enrichment.Candidate{Changes: []enrichment.FieldChange{{Field: "description", Value: "Description"}, {Field: "selling_points", Value: "Point"}}}}}
	got, err := ProjectTask(task, &run, "", now)
	require.NoError(t, err)
	require.True(t, got.CanReview, "valid multi-field content must reach the original Review")
	run.State.Candidate.Changes[1].Field = "title"
	got, err = ProjectTask(task, &run, "", now)
	require.NoError(t, err)
	require.False(t, got.CanReview, "same count with an unselected field is not reviewable")
}
