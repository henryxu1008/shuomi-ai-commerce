package agentconfigpersistence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	reviewproposal "task-processor/internal/product/review/proposal"
)

func TestContentConfigurationReplayBindsLanguageAndFieldSet(t *testing.T) {
	db, store, runs := fixture(t)
	ctx := context.Background()
	scope := agent.Scope{OrganizationID: "content-org", ActorID: "content-actor"}
	enable := command(scope, "enable", 0)
	enable.AgentID, enable.Absent = agentconfig.ContentAgentID, true
	_, err := store.Execute(ctx, enable)
	require.NoError(t, err)
	selection, err := reviewproposal.NewContentSelection("en-US", []string{"title", "description"})
	require.NoError(t, err)
	profile := aicapability.ModelProfile{ClientName: "content", ProviderID: "synthetic", AdapterKind: "openai-compatible", ModelID: "model", EndpointIdentityDigest: "endpoint", CredentialVersion: "credential", RoutingPolicyVersion: "routing", AdapterPolicyVersion: "adapter", PromptVersion: agentconfig.ContentPromptVersion, OutputSchemaVersion: agentconfig.ContentActionVersion, UsageMappingVersion: "usage", CostPricingVersion: "cost", PointTariff: aicapability.ModelPointTariff{PriceVersion: "points", InputPointsPerMillionTokens: 1000000, OutputPointsPerMillionTokens: 1000000}, Currency: "CNY", InputMicrosPerMillion: 1000000, OutputMicrosPerMillion: 1000000, MaximumPromptTokens: 1000, MaximumCompletionTokens: 100, MaximumInputBytes: 4096, MaximumOutputBytes: 4096, DeadlineBound: time.Minute}
	start := agentconfig.StartCommand{Scope: scope, MemberID: "member", ExpectedModeRevision: "0", AgentID: agentconfig.ContentAgentID, AgentVersion: agentconfig.ContentAgentVersion, ExecutionModelProfile: profile,
		Request: agent.Request{ContentSelection: selection, Key: uuid.NewString(), Binding: agent.Binding{ContextKind: "acquisition", ContextID: uuid.NewString(), ProductKey: "product", CatalogVersion: "1", PublicationID: "source-publication", TargetPlatform: "product"}, PolicyVersion: reviewproposal.ContentPolicyVersion, PromptVersion: agentconfig.ContentPromptVersion, Limits: agent.Limits{Steps: 4, ModelCalls: 2, Tokens: 100, CostMicros: 100, Currency: "CNY", Runtime: time.Minute}}}
	frozen, err := store.Prepare(ctx, start)
	require.NoError(t, err)
	require.Equal(t, selection, frozen.Request.ContentSelection)
	ref := agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: frozen.ID, Digest: frozen.Digest}
	for _, changed := range []reviewproposal.ContentSelection{{TargetLanguage: "de", Title: true, Description: true}, {TargetLanguage: "en-US", Title: true}} {
		command := start
		command.Request.ContentSelection = changed
		_, err = store.Prepare(ctx, command)
		require.ErrorIs(t, err, agentconfig.ErrConflict)
		_, err = store.Match(ctx, command, ref)
		require.ErrorIs(t, err, agentconfig.ErrConflict)
	}
	replay, err := store.Prepare(ctx, start)
	require.NoError(t, err)
	require.Equal(t, frozen, replay)
	loaded, err := store.LoadSnapshot(ctx, scope, ref)
	require.NoError(t, err)
	require.Equal(t, frozen, loaded)
	wrong := start
	wrong.AgentID = "product.title.agent"
	_, err = store.Prepare(ctx, wrong)
	require.ErrorIs(t, err, agentconfig.ErrInvalid)
	skillStart := start
	skillStart.Request.Key = uuid.NewString()
	skillStart.Request.ContentSelection = reviewproposal.ContentSelection{TargetLanguage: "en-US", Title: true}
	skillStart.Request.SkillVersionRef = agent.SkillVersionRef{Kind: "skill-version-v1", ID: uuid.NewString(), OwnerKind: "USER", OwnerID: scope.ActorID, Version: 1, Digest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", PolicyVersion: "skill-method-v1"}
	skillSnapshot, err := store.Prepare(ctx, skillStart)
	require.NoError(t, err)
	changedSkill := skillStart
	changedSkill.Request.SkillVersionRef.Version = 2
	_, err = store.Prepare(ctx, changedSkill)
	require.ErrorIs(t, err, agentconfig.ErrConflict)
	skillRef := agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: skillSnapshot.ID, Digest: skillSnapshot.Digest}
	_, err = store.Match(ctx, changedSkill, skillRef)
	require.ErrorIs(t, err, agentconfig.ErrConflict)
	unsupported := skillStart
	unsupported.Request.Key = uuid.NewString()
	unsupported.Request.ContentSelection.Description = true
	_, err = store.Prepare(ctx, unsupported)
	require.ErrorIs(t, err, agentconfig.ErrInvalid)
	guard, err := NewGuard(db, store, runs, func(string, string) (agent.Limits, error) { return skillStart.Request.Limits, nil }, func(context.Context) (agent.ExecutionIdentity, error) {
		return agent.ExecutionIdentity{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: "member"}, nil
	})
	require.NoError(t, err)
	request := skillSnapshot.Request
	request.ConfigurationSnapshotRef = skillRef
	_, acquired, err := guard.Claim(ctx, agent.Record{State: agent.State{Scope: scope, Request: request}}, 0)
	require.ErrorIs(t, err, agentconfig.ErrUnavailable)
	require.False(t, acquired)
}
