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
	reviewproposal "task-processor/internal/product/review/proposal"
)

// This executor observes only the fixed request/quote and never calls a model.
type contentTextFixture struct {
	input aicapability.TextInputIdentity
	calls int
}

func (e *contentTextFixture) GenerateWithGate(ctx context.Context, input aicapability.TextInputIdentity, quote aicapability.TextQuote, validate func(string) error, gate func(context.Context) (func(), error)) (governed.TextOutput, error) {
	e.input = input
	e.calls++
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

func TestContentTwoCallDecisionInstructionUsesFrozenBudget(t *testing.T) {
	for _, tc := range []struct {
		name      string
		calls     int
		canonical bool
	}{
		{"initial decision", 2, false},
		{"canonical read needs more evidence", 2, true},
		{"wider budget retains optional reads", 3, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			identity := authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", UserID: "actor", EffectiveMemberID: "member", TokenExpiresAt: time.Now().Add(time.Hour)}
			ctx := authidentity.WithAuthenticatedIdentity(context.Background(), identity)
			selection := reviewproposal.ContentSelection{TargetLanguage: "en-US", Title: true}
			profile := aicapability.ModelProfile{ClientName: "content", ProviderID: "synthetic", AdapterKind: "openai-compatible", ModelID: "model", EndpointIdentityDigest: "endpoint", CredentialVersion: "credential", RoutingPolicyVersion: "routing", AdapterPolicyVersion: "adapter", PromptVersion: agentconfig.ContentPromptVersion, OutputSchemaVersion: agentconfig.ContentActionVersion, UsageMappingVersion: "usage", CostPricingVersion: "cost", PointTariff: aicapability.ModelPointTariff{PriceVersion: "points", InputPointsPerMillionTokens: 1000000, OutputPointsPerMillionTokens: 1000000}, Currency: "CNY", InputMicrosPerMillion: 1000000, OutputMicrosPerMillion: 1000000, MaximumPromptTokens: 16000, MaximumCompletionTokens: 100, MaximumInputBytes: 64 << 10, MaximumOutputBytes: 4096, DeadlineBound: time.Minute}
			mode := agent.ModeSnapshotRef{Kind: agent.ExecutionModeSnapshotKind, ID: uuid.NewString(), Digest: strings.Repeat("b", 64)}
			binding := agent.Binding{ContextKind: "acquisition", ContextID: "context", ProductKey: "product", CatalogVersion: "1", PublicationID: "publication", TargetPlatform: "product"}
			limits := agent.Limits{Steps: 12, ModelCalls: tc.calls, Tokens: 17000, CostMicros: 17000, Currency: "CNY", Runtime: time.Minute}
			snapshot := agentconfig.Snapshot{AgentID: agentconfig.ContentAgentID, AgentVersion: agentconfig.ContentAgentVersion, MemberID: identity.EffectiveMemberID, ExecutionModelProfile: profile, Request: agent.Request{ContentSelection: selection, ModeSnapshotRef: mode, Binding: binding, PolicyVersion: reviewproposal.ContentPolicyVersion, PromptVersion: agentconfig.ContentPromptVersion, Limits: limits}}
			executor := &contentTextFixture{}
			definition, _ := contentagent.Registration()
			model, err := NewContentAgentTextModel(executor, testSnapshotReader{snapshot}, func(context.Context, string) (aicapability.ModelProfile, error) { return profile, nil }, func(context.Context, string) TextRouteReadiness { return TextRouteAvailable }, definition.AllowedTools, func(context.Context) (authidentity.AuthenticatedIdentity, error) { return identity, nil })
			require.NoError(t, err)
			input := agent.ModelInput{ContentSelection: selection, ModeSnapshotRef: mode, ConfigurationSnapshotRef: agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: uuid.NewString(), Digest: strings.Repeat("a", 64)}, Binding: binding, PolicyVersion: reviewproposal.ContentPolicyVersion, PromptVersion: agentconfig.ContentPromptVersion, AgentID: agentconfig.ContentAgentID, AgentVersion: agentconfig.ContentAgentVersion, AgentRunID: uuid.NewString()}
			if tc.canonical {
				input.History = []agent.Observation{{Tool: commercetool.ToolRef{ID: "product.canonical.inspect", Version: "v2.0.0"}, Output: json.RawMessage(`{"product_key":"product","catalog_version":"1","catalog_publication_id":"publication","snapshot":{"title":"Card","sources":[]}}`)}}
			}
			quote, err := model.Quote(ctx, input)
			require.NoError(t, err)
			require.Zero(t, executor.calls, "quoting must not dispatch a model")
			input.UpperBound, input.InvocationID = quote, "controlled-invocation"
			result, err := model.Decide(ctx, input)
			require.NoError(t, err)
			require.Equal(t, "interrupt", result.Action.Kind)
			require.Equal(t, limits, snapshot.Request.Limits, "the instruction must not raise a run limit")
			var dispatched struct {
				Input        agent.ModelInput
				AllowedTools []commercetool.ToolRef
			}
			require.NoError(t, json.Unmarshal([]byte(executor.input.Prompt), &dispatched))
			require.Equal(t, definition.AllowedTools, dispatched.AllowedTools, "prompt guidance must not change tool authority")
			if tc.calls == 2 {
				require.Contains(t, executor.input.System, "This frozen run permits at most two model decisions.")
				require.Contains(t, executor.input.System, "After the canonical read, return Kind=propose")
				require.Contains(t, executor.input.System, "Kind=interrupt with the specific missing facts")
				require.Contains(t, executor.input.System, "Do not request another tool after that read.")
				// A quote for the old unconstrained prompt cannot authorize the
				// corrected two-call prompt, even with the same frozen input.
				unconstrained := executor.input
				unconstrained.System, unconstrained.InvocationID = contentTextSystem, ""
				oldQuote, err := aicapability.QuoteText(unconstrained)
				require.NoError(t, err)
				input.UpperBound = agent.Quote{Tokens: oldQuote.MaximumTokens, CostMicros: oldQuote.MaximumCostMicros, Currency: oldQuote.Currency, Known: true, Reference: oldQuote.InputHash}
				_, err = model.Decide(ctx, input)
				require.ErrorIs(t, err, agent.ErrUnavailable)
				require.Equal(t, 1, executor.calls, "a mismatched prompt quote must fail before dispatch")
			} else {
				require.Equal(t, contentTextSystem, executor.input.System, "wider frozen budgets keep the existing prompt")
			}
		})
	}
}

func TestContentModelBindsFixedOperationAndFrozenSelection(t *testing.T) {
	identity := authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", UserID: "actor", EffectiveMemberID: "member", TokenExpiresAt: time.Now().Add(time.Hour)}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), identity)
	selection, err := reviewproposal.NewContentSelection("en-US", []string{"title", "description"})
	require.NoError(t, err)
	profile := aicapability.ModelProfile{ClientName: "content", ProviderID: "synthetic", AdapterKind: "openai-compatible", ModelID: "model", EndpointIdentityDigest: "endpoint", CredentialVersion: "credential", RoutingPolicyVersion: "routing", AdapterPolicyVersion: "adapter", PromptVersion: agentconfig.ContentPromptVersion, OutputSchemaVersion: agentconfig.ContentActionVersion, UsageMappingVersion: "usage", CostPricingVersion: "cost", PointTariff: aicapability.ModelPointTariff{PriceVersion: "points", InputPointsPerMillionTokens: 1000000, OutputPointsPerMillionTokens: 1000000}, Currency: "CNY", InputMicrosPerMillion: 1000000, OutputMicrosPerMillion: 1000000, MaximumPromptTokens: 16000, MaximumCompletionTokens: 100, MaximumInputBytes: 64 << 10, MaximumOutputBytes: 4096, DeadlineBound: time.Minute}
	modeRef := agent.ModeSnapshotRef{Kind: agent.ExecutionModeSnapshotKind, ID: uuid.NewString(), Digest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	binding := agent.Binding{ContextKind: "acquisition", ContextID: "context", ProductKey: "product", CatalogVersion: "1", PublicationID: "publication", TargetPlatform: "product"}
	snapshot := agentconfig.Snapshot{AgentID: agentconfig.ContentAgentID, AgentVersion: agentconfig.ContentAgentVersion, MemberID: identity.EffectiveMemberID, ExecutionModelProfile: profile, Request: agent.Request{ContentSelection: selection, ModeSnapshotRef: modeRef, Binding: binding, PolicyVersion: reviewproposal.ContentPolicyVersion, PromptVersion: agentconfig.ContentPromptVersion, Limits: agent.Limits{Steps: 4, ModelCalls: 2, Tokens: 17000, CostMicros: 17000, Currency: "CNY", Runtime: time.Minute}}}
	executor := &contentTextFixture{}
	definition, _ := contentagent.Registration()
	model, err := NewContentAgentTextModel(executor, testSnapshotReader{snapshot}, func(context.Context, string) (aicapability.ModelProfile, error) { return profile, nil }, func(context.Context, string) TextRouteReadiness { return TextRouteAvailable }, definition.AllowedTools, func(context.Context) (authidentity.AuthenticatedIdentity, error) { return identity, nil })
	require.NoError(t, err)
	input := agent.ModelInput{ContentSelection: selection, ModeSnapshotRef: modeRef, ConfigurationSnapshotRef: agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: uuid.NewString(), Digest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, Binding: binding, PolicyVersion: reviewproposal.ContentPolicyVersion, PromptVersion: agentconfig.ContentPromptVersion, AgentID: agentconfig.ContentAgentID, AgentVersion: agentconfig.ContentAgentVersion, AgentRunID: uuid.NewString()}
	quote, err := model.Quote(ctx, input)
	require.NoError(t, err)
	for _, changed := range []reviewproposal.ContentSelection{{TargetLanguage: "de", Title: true, Description: true}, {TargetLanguage: "en-US", Title: true}} {
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

func TestContentPromptExampleUsesTheActualMultiFieldWireContract(t *testing.T) {
	var proposal agent.Action
	for _, line := range strings.Split(contentTextSystem, "\n") {
		if strings.HasPrefix(line, `{"Kind":"propose"`) {
			require.NoError(t, json.Unmarshal([]byte(line), &proposal))
		}
	}
	fields := map[string]string{}
	for _, change := range proposal.Candidate.Changes {
		fields[change.Field] = change.Value
	}
	require.NotEmpty(t, fields["description"], "the model needs an executable description example")
	for _, field := range []string{"selling_points", "seo_keywords"} {
		var values []string
		require.NoError(t, json.Unmarshal([]byte(fields[field]), &values), "FieldChange.Value is a string containing the complete JSON array for %s", field)
		require.NotEmpty(t, values)
	}
	require.Contains(t, contentTextSystem, "Value is always a JSON string")
}
