package einomodel

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	"task-processor/internal/authidentity"
	"task-processor/internal/commercetool"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/product/contentagent"
	"task-processor/internal/product/enrichment"
)

// This executor observes only the fixed request/quote and never calls a model.
type contentTextFixture struct {
	input aicapability.TextInputIdentity
}

func (e *contentTextFixture) GenerateWithGate(ctx context.Context, input aicapability.TextInputIdentity, quote aicapability.TextQuote, validate func(string) error, gate func(context.Context) (func(), error)) (governed.TextOutput, error) {
	e.input = input
	release, err := gate(ctx)
	if err != nil {
		return governed.TextOutput{}, err
	}
	if release != nil {
		defer release()
	}
	err = validate(`{"Kind":"interrupt","Tool":{"ID":"","Version":""},"Candidate":{"Changes":null},"Unresolved":["controlled missing evidence"],"Confidence":[]}`)
	return governed.TextOutput{Usage: governed.ObservedUsage{Known: true, PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}}, err
}

func TestContentModelBindsFixedOperationAndFrozenSelection(t *testing.T) {
	identity := authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", UserID: "actor", EffectiveMemberID: "member", TokenExpiresAt: time.Now().Add(time.Hour)}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), identity)
	selection, err := enrichment.NewContentSelection("en-US", []string{"title", "description"})
	require.NoError(t, err)
	profile := aicapability.ModelProfile{ClientName: "content", ProviderID: "synthetic", AdapterKind: "openai-compatible", ModelID: "model", EndpointIdentityDigest: "endpoint", CredentialVersion: "credential", RoutingPolicyVersion: "routing", AdapterPolicyVersion: "adapter", PromptVersion: agentconfig.ContentPromptVersion, OutputSchemaVersion: agentconfig.ContentActionVersion, UsageMappingVersion: "usage", CostPricingVersion: "cost", PointTariff: aicapability.ModelPointTariff{PriceVersion: "points", InputPointsPerMillionTokens: 1000000, OutputPointsPerMillionTokens: 1000000}, Currency: "CNY", InputMicrosPerMillion: 1000000, OutputMicrosPerMillion: 1000000, MaximumPromptTokens: 16000, MaximumCompletionTokens: 100, MaximumInputBytes: 64 << 10, MaximumOutputBytes: 4096, DeadlineBound: time.Minute}
	modeRef := agent.ModeSnapshotRef{Kind: agent.ExecutionModeSnapshotKind, ID: uuid.NewString(), Digest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	binding := agent.Binding{ContextKind: "acquisition", ContextID: "context", ProductKey: "product", CatalogVersion: "1", PublicationID: "publication", TargetPlatform: "product"}
	snapshot := agentconfig.Snapshot{AgentID: agentconfig.ContentAgentID, AgentVersion: agentconfig.ContentAgentVersion, MemberID: identity.EffectiveMemberID, ExecutionModelProfile: profile, Request: agent.Request{ContentSelection: selection, ModeSnapshotRef: modeRef, Binding: binding, PolicyVersion: enrichment.ContentPolicyVersion, PromptVersion: agentconfig.ContentPromptVersion, Limits: agent.Limits{Steps: 4, ModelCalls: 2, Tokens: 17000, CostMicros: 17000, Currency: "CNY", Runtime: time.Minute}}}
	executor := &contentTextFixture{}
	definition, _ := contentagent.Registration()
	model, err := NewContentAgentTextModel(executor, testSnapshotReader{snapshot}, func(context.Context, string) (aicapability.ModelProfile, error) { return profile, nil }, func(context.Context, string) TextRouteReadiness { return TextRouteAvailable }, definition.AllowedTools, func(context.Context) (authidentity.AuthenticatedIdentity, error) { return identity, nil })
	require.NoError(t, err)
	input := agent.ModelInput{ContentSelection: selection, ModeSnapshotRef: modeRef, ConfigurationSnapshotRef: agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: uuid.NewString(), Digest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, Binding: binding, PolicyVersion: enrichment.ContentPolicyVersion, PromptVersion: agentconfig.ContentPromptVersion, AgentID: agentconfig.ContentAgentID, AgentVersion: agentconfig.ContentAgentVersion, AgentRunID: uuid.NewString()}
	quote, err := model.Quote(ctx, input)
	require.NoError(t, err)
	for _, changed := range []enrichment.ContentSelection{{TargetLanguage: "de", Title: true, Description: true}, {TargetLanguage: "en-US", Title: true}} {
		bad := input
		bad.ContentSelection = changed
		_, err = model.Quote(ctx, bad)
		require.Error(t, err)
	}
	input.UpperBound, input.InvocationID = quote, "controlled-invocation"
	result, err := model.Decide(ctx, input)
	require.NoError(t, err)
	require.Equal(t, "interrupt", result.Action.Kind)
	require.Equal(t, aicapability.OperationProductContentDecision, executor.input.Operation)
	require.Contains(t, executor.input.Prompt, `"targetLanguage":"en-US"`)
	require.Contains(t, executor.input.System, "ONLY the explicitly selected content fields")
	require.NotContains(t, executor.input.System, "ONLY a title change")
	require.Contains(t, executor.input.System, "An empty Input.History means evidence has not been read yet, not that the saved product lacks facts.")
	require.Contains(t, executor.input.System, "Before proposing or interrupting for missing product facts, request product.canonical.inspect")
	var dispatched struct {
		Input        agent.ModelInput
		AllowedTools []commercetool.ToolRef
	}
	require.NoError(t, json.Unmarshal([]byte(executor.input.Prompt), &dispatched))
	require.Empty(t, dispatched.Input.History, "initial facts must still be fetched by the authorized tool")
	require.Equal(t, input.Binding, dispatched.Input.Binding)
	require.Equal(t, definition.AllowedTools, dispatched.AllowedTools)
}

// Prompt examples are executable protocol examples: each must name one real
// action, with no competing Tool/Candidate fields. This does not pretend to
// predict a provider's response; the real model path is verified separately.
func TestContentPromptActionExamplesAreUnambiguous(t *testing.T) {
	definition, _ := contentagent.Registration()
	examples := map[string]agent.Action{}
	for _, line := range strings.Split(contentTextSystem, "\n") {
		if !strings.HasPrefix(line, `{"Kind":`) {
			continue
		}
		var action agent.Action
		require.NoError(t, json.Unmarshal([]byte(line), &action))
		require.NotContains(t, examples, action.Kind)
		examples[action.Kind] = action
	}
	require.Len(t, examples, 3, "show separate tool, propose and interrupt examples instead of a mixed action")
	read, ok := examples["tool"]
	require.True(t, ok)
	require.Contains(t, definition.AllowedTools, read.Tool)
	require.Equal(t, "product.canonical.inspect", read.Tool.ID)
	require.Empty(t, read.Candidate.Changes)
	require.Empty(t, read.Unresolved)
	proposal, ok := examples["propose"]
	require.True(t, ok)
	require.Empty(t, proposal.Tool)
	require.NotEmpty(t, proposal.Candidate.Changes)
	missing, ok := examples["interrupt"]
	require.True(t, ok)
	require.Empty(t, missing.Tool)
	require.Empty(t, missing.Candidate.Changes)
	require.NotEmpty(t, missing.Unresolved)
}
