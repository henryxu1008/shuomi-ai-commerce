package agent

import (
	"crypto/sha256"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/commercetool"
	"task-processor/internal/product/enrichment"
)

func TestContentSelectionIsPartOfFrozenRequestAndModelInput(t *testing.T) {
	s, err := enrichment.NewContentSelection("en-US", []string{"title", "description"})
	require.NoError(t, err)
	r := Request{ContentSelection: s, PolicyVersion: enrichment.ContentPolicyVersion}
	require.True(t, r.ValidContentSelection())
	input := (State{Request: r}).ModelInput(commercetool.AgentDefinition{ID: "product.content.agent", Version: "v1.0.0"})
	require.Equal(t, s, input.ContentSelection)
	raw, err := json.Marshal(r)
	require.NoError(t, err)
	originalHash := sha256.Sum256(raw)
	changed := r
	changed.ContentSelection.TargetLanguage = "de"
	raw, err = json.Marshal(changed)
	require.NoError(t, err)
	require.NotEqual(t, originalHash, sha256.Sum256(raw))
	changed = r
	changed.ContentSelection.Description = false
	raw, err = json.Marshal(changed)
	require.NoError(t, err)
	require.NotEqual(t, originalHash, sha256.Sum256(raw))
	r.PolicyVersion = "title-review-v1"
	require.False(t, r.ValidContentSelection())
	r.ContentSelection = enrichment.ContentSelection{}
	require.True(t, r.ValidContentSelection())
	r.PolicyVersion = enrichment.ContentPolicyVersion
	require.False(t, r.ValidContentSelection())
}

func TestEmptyContentSelectionPreservesOriginalTitleRequestWire(t *testing.T) {
	r := Request{Key: "original-request", PolicyVersion: "title-review-v1", PromptVersion: "product-title-agent-v1"}
	raw, err := json.Marshal(r)
	require.NoError(t, err)
	// Exact pre-content Request encoding, including omitted zero snapshot refs.
	const original = `{"Key":"original-request","Binding":{"ContextKind":"","ContextID":"","ProductKey":"","CatalogVersion":"","PublicationID":"","TargetPlatform":""},"GoalSummary":"","PolicyVersion":"title-review-v1","PromptVersion":"product-title-agent-v1","Limits":{"Steps":0,"ModelCalls":0,"Tokens":0,"CostMicros":0,"Currency":"","Runtime":0}}`
	require.Equal(t, original, string(raw))
}
