package aiworkbenchpersistence

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aiworkbench"
	reviewproposal "task-processor/internal/product/review/proposal"
	"testing"
	"time"
)

func TestContentMessageFingerprintStaysStableAcrossOwnerExtraction(t *testing.T) {
	input := aiworkbench.MessageInput{Content: "Localize", TaskSelection: &aiworkbench.TaskSelection{Subject: aiworkbench.Subject{Kind: "own_product", ID: "fa5a9456-c8cb-43bf-ad1c-57a3e9f5199a", CatalogVersion: "3", PublicationID: "current-pub"}, Capability: aiworkbench.ContentTaskKind, Parameters: reviewproposal.ContentSelection{TargetLanguage: "en-US", Description: true}}}
	require.True(t, validMessage(input))
	require.Equal(t, "d034f715f401089f760b245255e78377d26931bf9e13ba08fa9c58f7fbaf2506", messageFingerprint("conversation", input))
}

func TestContentMessageFreezesSourceNeutralSelectionAndKeepsTitleFingerprint(t *testing.T) {
	input := aiworkbench.MessageInput{Content: "Localize", TaskSelection: &aiworkbench.TaskSelection{Subject: aiworkbench.Subject{Kind: "own_product", ID: "fa5a9456-c8cb-43bf-ad1c-57a3e9f5199a", CatalogVersion: "3", PublicationID: "current-pub"}, Capability: aiworkbench.ContentTaskKind, Parameters: reviewproposal.ContentSelection{TargetLanguage: "en-US", Description: true}}}
	require.True(t, validMessage(input))
	first := messageFingerprint("conversation", input)
	input.TaskSelection.Parameters.SellingPoints = true
	require.NotEqual(t, first, messageFingerprint("conversation", input))
	input.OperationID = "fake-acquisition"
	require.False(t, validMessage(input), "mixed contracts must be rejected")
	title := aiworkbench.MessageInput{Content: "Title", OperationID: "op", TargetPlatform: "shein"}
	expected := digest(struct {
		Operation      string
		ConversationID string
		Input          struct{ Content, OperationID, TargetPlatform, TemplateID, TemplateRevision, KnowledgeBaseID string }
	}{messageOperation, "conversation", struct{ Content, OperationID, TargetPlatform, TemplateID, TemplateRevision, KnowledgeBaseID string }{Content: "Title", OperationID: "op", TargetPlatform: "shein"}})
	require.Equal(t, expected, messageFingerprint("conversation", title), "current title command receipts keep their exact fingerprint")
	raw, _ := json.Marshal(input.WorkScope())
	require.LessOrEqual(t, len(raw), 1024)
}

func TestContentConfirmRejectsChangedLanguageFieldsOwnerAndLimits(t *testing.T) {
	selection := reviewproposal.ContentSelection{TargetLanguage: "en-US", Description: true, SellingPoints: true}
	limits := agent.Limits{Steps: 8, ModelCalls: 3, Tokens: 10000, CostMicros: 10000, Currency: "CNY", Runtime: time.Minute}
	kind := "own_product"
	row := proposalRow{Kind: aiworkbench.ContentTaskKind, Digest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ContextKind: &kind, ContentSelection: &selection, ExecutionLimits: &limits, GoalSummary: "Localize", OperationID: "fa5a9456-c8cb-43bf-ad1c-57a3e9f5199a", ProductKey: "product", CatalogVersion: "3", PublicationID: "current-pub", TargetPlatform: "product", AgentID: agentconfig.ContentAgentID, AgentVersion: agentconfig.ContentAgentVersion}
	config := agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: row.OperationID, Digest: row.Digest}
	request := agent.Request{Key: row.OperationID, Binding: agent.Binding{ContextKind: kind, ContextID: row.OperationID, ProductKey: row.ProductKey, CatalogVersion: row.CatalogVersion, PublicationID: row.PublicationID, TargetPlatform: "product"}, ContentSelection: selection, Limits: limits, GoalSummary: row.GoalSummary, PolicyVersion: reviewproposal.ContentPolicyVersion, PromptVersion: agentconfig.ContentPromptVersion, ConfigurationSnapshotRef: config}
	prepared := func(r agent.Request) aiworkbench.PreparedTask {
		raw, err := json.Marshal(r)
		require.NoError(t, err)
		return aiworkbench.PreparedTask{ProposalDigest: row.Digest, ConfigurationSnapshotRef: config, ExecutionRequest: raw}
	}
	_, _, err := validatePreparedTask(prepared(request), row, request.Key)
	require.NoError(t, err)
	for name, change := range map[string]func(*agent.Request){"language": func(r *agent.Request) { r.ContentSelection.TargetLanguage = "de" }, "fields": func(r *agent.Request) { r.ContentSelection.Title = true }, "owner": func(r *agent.Request) { r.Binding.ContextKind = "acquisition" }, "budget": func(r *agent.Request) { r.Limits.CostMicros++ }, "policy": func(r *agent.Request) { r.PolicyVersion = "title-review-v1" }} {
		t.Run(name, func(t *testing.T) {
			bad := request
			change(&bad)
			_, _, e := validatePreparedTask(prepared(bad), row, request.Key)
			require.ErrorIs(t, e, aiworkbench.ErrInvalid)
		})
	}
}
