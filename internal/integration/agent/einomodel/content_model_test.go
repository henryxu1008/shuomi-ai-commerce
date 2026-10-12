package einomodel

import (
	"context"
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
	"task-processor/internal/product/enrichment"
)

// This executor observes only the fixed request/quote and never calls a model.
type contentTextFixture struct {
	input aicapability.TextInputIdentity
}

func (e *contentTextFixture) GenerateWithGate(ctx context.Context, input aicapability.TextInputIdentity, quote aicapability.TextQuote, validate func(string) error, gate func(context.Context) (func(), error)) (governed.TextOutput, error) {
	e.input = input
	_, err := gate(ctx)
	if err != nil {
		return governed.TextOutput{}, err
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
	model, err := NewContentAgentTextModel(executor, testSnapshotReader{snapshot}, func(context.Context, string) (aicapability.ModelProfile, error) { return profile, nil }, func(context.Context, string) TextRouteReadiness { return TextRouteAvailable }, []commercetool.ToolRef{{ID: "product.snapshot.read", Version: "v1"}}, func(context.Context) (authidentity.AuthenticatedIdentity, error) { return identity, nil })
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
}
