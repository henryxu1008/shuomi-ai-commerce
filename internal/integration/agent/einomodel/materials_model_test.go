package einomodel

import (
	"context"
	"encoding/json"
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
	"task-processor/internal/listing/materials"
)

type materialsTextFixture struct {
	input      aicapability.TextInputIdentity
	response   string
	dispatched int
}

func (e *materialsTextFixture) GenerateWithGate(ctx context.Context, input aicapability.TextInputIdentity, _ aicapability.TextQuote, validate func(string) error, gate func(context.Context) (func(), error)) (governed.TextOutput, error) {
	_, err := gate(ctx)
	if err != nil {
		return governed.TextOutput{}, err
	}
	e.input, e.dispatched = input, e.dispatched+1
	err = validate(e.response)
	return governed.TextOutput{Usage: governed.ObservedUsage{Known: true, PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}}, err
}

func materialsModelFixture(t *testing.T) (*AgentTextModel, *materialsTextFixture, context.Context, agent.ModelInput) {
	t.Helper()
	id := authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", UserID: "actor", EffectiveMemberID: "member", TokenExpiresAt: time.Now().Add(time.Hour)}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), id)
	profile := aicapability.ModelProfile{ClientName: "materials", ProviderID: "synthetic", AdapterKind: "openai-compatible", ModelID: "model", EndpointIdentityDigest: "endpoint", CredentialVersion: "credential", RoutingPolicyVersion: "routing", AdapterPolicyVersion: "adapter", PromptVersion: materials.PromptVersion, OutputSchemaVersion: materials.ActionVersion, UsageMappingVersion: "usage", CostPricingVersion: "cost", PointTariff: aicapability.ModelPointTariff{PriceVersion: "points", InputPointsPerMillionTokens: 1000000, OutputPointsPerMillionTokens: 1000000}, Currency: "CNY", InputMicrosPerMillion: 1000000, OutputMicrosPerMillion: 1000000, MaximumPromptTokens: 16000, MaximumCompletionTokens: 100, MaximumInputBytes: 64 << 10, MaximumOutputBytes: 4096, DeadlineBound: time.Minute}
	mode := agent.ModeSnapshotRef{Kind: agent.ExecutionModeSnapshotKind, ID: uuid.NewString(), Digest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	binding := agent.Binding{ContextKind: "listing-target-record", ContextID: "record", ProductKey: "product", CatalogVersion: "1", PublicationID: "publication", TargetPlatform: "shein"}
	snapshot := agentconfig.Snapshot{AgentID: materials.AgentID, AgentVersion: materials.AgentVersion, MemberID: id.EffectiveMemberID, ExecutionModelProfile: profile, Request: agent.Request{Binding: binding, ModeSnapshotRef: mode, PolicyVersion: materials.PolicyVersion, PromptVersion: materials.PromptVersion, Limits: agent.Limits{Steps: 6, ModelCalls: 3, Tokens: 17000, CostMicros: 17000, Currency: "CNY", Runtime: time.Minute}}}
	executor := &materialsTextFixture{response: `{"Kind":"interrupt","Unresolved":["controlled missing evidence"]}`}
	model, err := NewMaterialsAgentTextModel(executor, testSnapshotReader{snapshot}, func(context.Context, string) (aicapability.ModelProfile, error) { return profile, nil }, func(context.Context, string) TextRouteReadiness { return TextRouteAvailable }, []commercetool.ToolRef{materials.InspectToolRef()}, func(context.Context) (authidentity.AuthenticatedIdentity, error) { return id, nil }, materials.ModelAdmission{})
	require.NoError(t, err)
	in := agent.ModelInput{Binding: binding, ModeSnapshotRef: mode, ConfigurationSnapshotRef: agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: uuid.NewString(), Digest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, PolicyVersion: materials.PolicyVersion, PromptVersion: materials.PromptVersion, AgentID: materials.AgentID, AgentVersion: materials.AgentVersion, AgentRunID: uuid.NewString()}
	return model, executor, ctx, in
}

func TestMaterialsModelBindsDedicatedOperationAndNoContentMutation(t *testing.T) {
	m, executor, ctx, in := materialsModelFixture(t)
	quote, err := m.Quote(ctx, in)
	require.NoError(t, err)
	in.UpperBound, in.InvocationID = quote, "invocation"
	_, err = m.Decide(ctx, in)
	require.NoError(t, err)
	require.Equal(t, aicapability.OperationListingMaterialsDecision, executor.input.Operation)
	require.Contains(t, executor.input.System, "listing-materials-v1")
	require.Contains(t, executor.input.System, "Never change title")
	for _, response := range []string{
		`{"Kind":"propose","Candidate":{"Changes":[{"Field":"title","Value":"bad"}]}}`,
		`{"Kind":"tool","Tool":{"ID":"publish","Version":"v1.0.0"}}`,
		`{"Kind":"propose","DomainProposal":{"Kind":"listing-materials-v1","Payload":{"brandCode":"brand","evidenceIds":["observed"],"price":1}}}`,
	} {
		executor.response = response
		result, err := m.Decide(ctx, in)
		require.ErrorIs(t, err, agent.ErrModelInvalidOutput)
		require.True(t, result.Usage.Known)
		require.EqualValues(t, 15, result.Usage.Tokens)
	}
}

func TestMaterialsModelRejectsWrongConsumerAndRetainsWholeEvidence(t *testing.T) {
	m, _, ctx, in := materialsModelFixture(t)
	in.AgentID = agentconfig.ContentAgentID
	_, err := m.Quote(ctx, in)
	require.Error(t, err)
	raw, err := json.Marshal(materials.Disclosure{EvidenceID: "listing-evidence:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Binding: in.Binding, Revision: 1, StoreID: "store"})
	require.NoError(t, err)
	projected, err := (materials.ModelAdmission{}).ProjectEvidence(raw)
	require.NoError(t, err)
	require.JSONEq(t, string(raw), string(projected))
	_, err = (materials.ModelAdmission{}).ProjectEvidence(json.RawMessage(string(make([]byte, materials.MaxInspectionBytes+1))))
	require.Error(t, err, "oversized facts must not silently become partial evidence")
}
