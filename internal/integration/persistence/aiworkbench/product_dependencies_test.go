package aiworkbenchpersistence

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aiworkbench"
	reviewproposal "task-processor/internal/product/review/proposal"
)

func TestProductDependenciesRejectMissingOrAmbiguousMessageOwners(t *testing.T) {
	selection := &aiworkbench.TaskSelection{Subject: aiworkbench.Subject{Kind: "own_product", ID: "11111111-1111-4111-8111-111111111111", CatalogVersion: "1", PublicationID: "publication-1"}, Capability: aiworkbench.ContentTaskKind, Parameters: reviewproposal.ContentSelection{TargetLanguage: "en-US", Description: true}}
	work, err := json.Marshal(aiworkbench.WorkScope{TaskSelection: selection})
	require.NoError(t, err)
	id := "assistant-1"
	messages := []messageRow{{ID: id, AuthorKind: "ASSISTANT", Sequence: 2}}
	cmd := commandRow{Operation: messageOperation, AssistantMessageID: &id, WorkScopeJSON: work, PlannerModelProfile: json.RawMessage(`{"PromptVersion":"ai-workbench-chat-plan-v4"}`)}
	result, err := planningProductDependencies(messages, []commandRow{cmd})
	require.NoError(t, err)
	require.Equal(t, []*aiworkbench.TaskSelection{selection}, result)
	_, err = planningProductDependencies(messages, []commandRow{cmd, cmd})
	require.Error(t, err)
	_, err = planningProductDependencies(append(messages, messageRow{ID: "missing", AuthorKind: "ASSISTANT", Sequence: 3}), []commandRow{cmd})
	require.Error(t, err)
	cmd.WorkScopeJSON = json.RawMessage(`{}`)
	_, err = planningProductDependencies(messages, []commandRow{cmd})
	require.Error(t, err)
}
