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
	"task-processor/internal/product/enrichment"
	d "task-processor/internal/skill"
	"testing"
	"time"
)

type methodFixture struct {
	packageValue    d.Package
	gates, releases int
	deny            bool
}

func (f *methodFixture) ReadMethod(context.Context, agent.ExecutionIdentity, agent.SkillVersionRef) (d.Package, error) {
	return f.packageValue, nil
}
func (f *methodFixture) AcquireMethod(context.Context, agent.ExecutionIdentity, agent.SkillVersionRef) (func(), error) {
	f.gates++
	if f.deny {
		return nil, d.ErrForbidden
	}
	return func() { f.releases++ }, nil
}
func TestSkillMaterialIsInsideTheOriginalQuoteAndDispatchGate(t *testing.T) {
	ctx := context.Background()
	identity := authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", UserID: "actor", EffectiveMemberID: "member", TokenExpiresAt: time.Now().Add(time.Hour)}
	ctx = authidentity.WithAuthenticatedIdentity(ctx, identity)
	profile := aicapability.ModelProfile{ClientName: "content", ProviderID: "synthetic", AdapterKind: "openai-compatible", ModelID: "model", EndpointIdentityDigest: "endpoint", CredentialVersion: "credential", RoutingPolicyVersion: "routing", AdapterPolicyVersion: "adapter", PromptVersion: agentconfig.ContentPromptVersion, OutputSchemaVersion: agentconfig.ContentActionVersion, UsageMappingVersion: "usage", CostPricingVersion: "cost", PointTariff: aicapability.ModelPointTariff{PriceVersion: "points", InputPointsPerMillionTokens: 1000000, OutputPointsPerMillionTokens: 1000000}, Currency: "CNY", InputMicrosPerMillion: 1000000, OutputMicrosPerMillion: 1000000, MaximumPromptTokens: 16000, MaximumCompletionTokens: 100, MaximumInputBytes: 64 << 10, MaximumOutputBytes: 4096, DeadlineBound: time.Minute}
	binding := agent.Binding{ContextKind: "acquisition", ContextID: "context", ProductKey: "product", CatalogVersion: "1", PublicationID: "publication", TargetPlatform: "product"}
	selection := enrichment.ContentSelection{TargetLanguage: "en-US", Title: true}
	mode := agent.ModeSnapshotRef{Kind: agent.ExecutionModeSnapshotKind, ID: uuid.NewString(), Digest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	snapshot := agentconfig.Snapshot{AgentID: agentconfig.ContentAgentID, AgentVersion: agentconfig.ContentAgentVersion, MemberID: "member", ExecutionModelProfile: profile, Request: agent.Request{ContentSelection: selection, ModeSnapshotRef: mode, Binding: binding, PolicyVersion: enrichment.ContentPolicyVersion, PromptVersion: agentconfig.ContentPromptVersion, Limits: agent.Limits{Steps: 4, ModelCalls: 2, Tokens: 17000, CostMicros: 17000, Currency: "CNY", Runtime: time.Minute}}}
	executor := &contentTextFixture{}
	model, e := NewContentAgentTextModel(executor, testSnapshotReader{snapshot}, func(context.Context, string) (aicapability.ModelProfile, error) { return profile, nil }, func(context.Context, string) TextRouteReadiness { return TextRouteAvailable }, []commercetool.ToolRef{{ID: "product.snapshot.read", Version: "v1"}}, func(context.Context) (authidentity.AuthenticatedIdentity, error) { return identity, nil })
	require.NoError(t, e)
	input := agent.ModelInput{ContentSelection: selection, ModeSnapshotRef: mode, ConfigurationSnapshotRef: agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: uuid.NewString(), Digest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, Binding: binding, PolicyVersion: enrichment.ContentPolicyVersion, PromptVersion: agentconfig.ContentPromptVersion, AgentID: agentconfig.ContentAgentID, AgentVersion: agentconfig.ContentAgentVersion, AgentRunID: uuid.NewString()}
	plain, e := model.Quote(ctx, input)
	require.NoError(t, e)
	pack, e := d.ValidatePackage([]d.File{{Path: "SKILL.md", Text: "---\nname: product-title-normalization\ndescription: Normalize titles.\n---\nNever add unverified certification.\n"}, {Path: "references/rules.txt", Text: "Preserve verified brand spelling."}})
	require.NoError(t, e)
	ref := agent.SkillVersionRef{Kind: "skill-version-v1", ID: uuid.NewString(), OwnerKind: "USER", OwnerID: "actor", Version: 1, Digest: pack.Digest, PolicyVersion: d.MethodPolicyVersion}
	input.SkillVersionRef = ref
	snapshot.Request.SkillVersionRef = ref
	model.snapshots = testSnapshotReader{snapshot}
	fixture := &methodFixture{packageValue: pack}
	model, e = model.WithSkillContext(fixture)
	require.NoError(t, e)
	quote, e := model.Quote(ctx, input)
	require.NoError(t, e)
	require.NotEqual(t, plain.Reference, quote.Reference)
	require.Greater(t, quote.Tokens, plain.Tokens)
	input.UpperBound, input.InvocationID = quote, "controlled-invocation"
	_, e = model.Decide(ctx, input)
	require.NoError(t, e)
	require.Contains(t, executor.input.Prompt, "Never add unverified certification.")
	require.Contains(t, executor.input.Prompt, "Preserve verified brand spelling.")
	require.Contains(t, executor.input.System, "skill-method-v1")
	require.Equal(t, 1, fixture.gates)
	require.Equal(t, 1, fixture.releases)
	fixture.deny = true
	_, e = model.Decide(ctx, input)
	require.Error(t, e)
	fixture.packageValue.Files[0].Text += "Different bytes after quote."
	_, e = model.Quote(ctx, input)
	require.Error(t, e)
	wrong := input
	wrong.SkillVersionRef.Version++
	_, e = model.Quote(ctx, wrong)
	require.Error(t, e)
}
