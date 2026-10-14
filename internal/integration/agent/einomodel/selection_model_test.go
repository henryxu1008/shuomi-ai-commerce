package einomodel

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	"task-processor/internal/authidentity"
	"task-processor/internal/commercetool"
	"task-processor/internal/product/selection"
	"testing"
	"time"
)

func TestSelectionModelUsesExactConsumerAndFreshAllCandidateDispatchGate(t *testing.T) {
	id := authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", UserID: "actor", EffectiveMemberID: "member", TokenExpiresAt: time.Now().Add(time.Hour)}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), id)
	profile := aicapability.ModelProfile{ClientName: "selection", ProviderID: "synthetic", AdapterKind: "openai-compatible", ModelID: "model", EndpointIdentityDigest: "endpoint", CredentialVersion: "credential", RoutingPolicyVersion: "routing", AdapterPolicyVersion: "adapter", PromptVersion: agentconfig.SelectionPromptVersion, OutputSchemaVersion: agentconfig.SelectionActionVersion, UsageMappingVersion: "usage", CostPricingVersion: "cost", PointTariff: aicapability.ModelPointTariff{PriceVersion: "points", InputPointsPerMillionTokens: 1000000, OutputPointsPerMillionTokens: 1000000}, Currency: "CNY", InputMicrosPerMillion: 1000000, OutputMicrosPerMillion: 1000000, MaximumPromptTokens: 16000, MaximumCompletionTokens: 100, MaximumInputBytes: 64 << 10, MaximumOutputBytes: 4096, DeadlineBound: time.Minute}
	mode := agent.ModeSnapshotRef{Kind: agent.ExecutionModeSnapshotKind, ID: uuid.NewString(), Digest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	binding := agent.Binding{ContextKind: agent.SelectionContextKind, ContextID: uuid.NewString(), TargetPlatform: "product"}
	for n := 0; n < 2; n++ {
		binding.Selection[n] = agent.SelectionProductRef{ItemID: uuid.NewString(), ItemRevision: "1", ProductKey: string(rune('a' + n)), CatalogVersion: "1", PublicationID: "source"}
	}
	if binding.Selection[0].ItemID > binding.Selection[1].ItemID {
		binding.Selection[0], binding.Selection[1] = binding.Selection[1], binding.Selection[0]
	}
	snapshot := agentconfig.Snapshot{AgentID: agentconfig.SelectionAgentID, AgentVersion: agentconfig.SelectionAgentVersion, MemberID: "member", ExecutionModelProfile: profile, Request: agent.Request{Binding: binding, ModeSnapshotRef: mode, PolicyVersion: agentconfig.SelectionPolicyVersion, PromptVersion: agentconfig.SelectionPromptVersion, Limits: agent.Limits{Steps: 6, ModelCalls: 3, Tokens: 17000, CostMicros: 17000, Currency: "CNY", Runtime: time.Minute}}}
	executor := &materialsTextFixture{response: `{"Kind":"interrupt","Unresolved":["missing candidate facts"]}`}
	checks := 0
	blocked := false
	m, err := NewSelectionAgentTextModel(executor, testSnapshotReader{snapshot}, func(context.Context, string) (aicapability.ModelProfile, error) { return profile, nil }, func(context.Context, string) TextRouteReadiness { return TextRouteAvailable }, []commercetool.ToolRef{agentconfig.SelectionInspectToolRef()}, func(context.Context) (authidentity.AuthenticatedIdentity, error) { return id, nil }, selection.ModelAdmission{}, func(_ context.Context, in agent.ModelInput) error {
		checks++
		require.Equal(t, binding, in.Binding)
		if blocked {
			return agent.ErrConflict
		}
		return nil
	})
	require.NoError(t, err)
	in := agent.ModelInput{Binding: binding, ModeSnapshotRef: mode, ConfigurationSnapshotRef: agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: uuid.NewString(), Digest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, PolicyVersion: agentconfig.SelectionPolicyVersion, PromptVersion: agentconfig.SelectionPromptVersion, AgentID: agentconfig.SelectionAgentID, AgentVersion: agentconfig.SelectionAgentVersion, AgentRunID: uuid.NewString()}
	quote, err := m.Quote(ctx, in)
	require.NoError(t, err)
	in.UpperBound, in.InvocationID = quote, "selection-invocation"
	_, err = m.Decide(ctx, in)
	require.NoError(t, err)
	require.Equal(t, 1, checks)
	require.Equal(t, aicapability.OperationProductSelectionDecision, executor.input.Operation)
	blocked = true
	_, err = m.Decide(ctx, in)
	require.ErrorIs(t, err, agent.ErrConflict)
	require.Equal(t, 1, executor.dispatched)
	wrong := in
	wrong.AgentID = agentconfig.ContentAgentID
	_, err = m.Quote(ctx, wrong)
	require.Error(t, err)
	blocked = false
	executor.response = `{"Kind":"propose","Candidate":{"Changes":[{"Field":"title","Value":"unauthorized"}]}}`
	_, err = m.Decide(ctx, in)
	require.ErrorIs(t, err, agent.ErrModelInvalidOutput)
}
