package agentconfigpersistence

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	"task-processor/internal/commercetool"
	"testing"
	"time"
)

func TestTitleConfigurationFreezesModeAndRejectsChangedClaim(t *testing.T) {
	db, repo, runs := fixture(t)
	ctx := context.Background()
	identity := agent.ExecutionIdentity{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	scope := agent.Scope{OrganizationID: identity.OrganizationID, ActorID: identity.ActorID}
	enabled := command(scope, "enable", 0)
	enabled.Absent = true
	_, err := repo.Execute(ctx, enabled)
	require.NoError(t, err)
	limits := agent.Limits{Steps: 4, ModelCalls: 2, Tokens: 100, CostMicros: 100, Currency: "USD", Runtime: time.Minute}
	profile := aicapability.ModelProfile{ClientName: "title", ProviderID: "synthetic", AdapterKind: "openai-compatible", ModelID: "model", EndpointIdentityDigest: "endpoint", CredentialVersion: "credential", RoutingPolicyVersion: "routing", AdapterPolicyVersion: "adapter", PromptVersion: "prompt", OutputSchemaVersion: "schema", UsageMappingVersion: "usage", CostPricingVersion: "cost", PointTariff: aicapability.ModelPointTariff{PriceVersion: "points", InputPointsPerMillionTokens: 1000000, OutputPointsPerMillionTokens: 1000000}, Currency: "USD", InputMicrosPerMillion: 1000000, OutputMicrosPerMillion: 1000000, MaximumPromptTokens: 100, MaximumCompletionTokens: 100, MaximumInputBytes: 4096, MaximumOutputBytes: 4096, DeadlineBound: time.Minute}
	start := agentconfig.StartCommand{Scope: scope, MemberID: identity.MemberID, ExpectedModeRevision: "0", AgentID: enabled.AgentID, AgentVersion: "v1.0.0", ExecutionModelProfile: profile, Request: agent.Request{Key: uuid.NewString(), Binding: agent.Binding{ContextKind: "acquisition", ContextID: uuid.NewString(), ProductKey: "p", CatalogVersion: "1", PublicationID: "pub", TargetPlatform: "shein"}, PolicyVersion: "title-review-v1", PromptVersion: "product-title-agent-v1", Limits: limits}}
	snap, err := repo.Prepare(ctx, start)
	require.NoError(t, err)
	require.True(t, snap.Request.ModeSnapshotRef.Valid())
	frozen, err := repo.LoadModeSnapshot(ctx, identity, snap.Request.ModeSnapshotRef)
	require.NoError(t, err)
	require.Equal(t, agent.ModeDefault, frozen.Mode)
	require.Equal(t, "0", frozen.Revision)
	missing := start
	missing.Request.Key = uuid.NewString()
	missing.ExpectedModeRevision = ""
	_, err = repo.Prepare(ctx, missing)
	require.ErrorIs(t, err, agentconfig.ErrInvalid)
	subject := agentconfig.ResolvedModeSubject{Subject: agentconfig.ModeSubject{Kind: agentconfig.RegisteredAgentSubject, ID: enabled.AgentID}, Definition: commercetool.AgentRef{ID: enabled.AgentID, Version: start.AgentVersion}}
	_, err = repo.ExecuteMode(ctx, agentconfig.ModeCommand{Identity: identity, Subject: subject, Key: uuid.NewString(), Mode: agent.ModeFullAccess})
	require.NoError(t, err)
	replay, err := repo.Prepare(ctx, start)
	require.NoError(t, err)
	require.Equal(t, snap, replay)
	changed := start
	changed.ExpectedModeRevision = "1"
	_, err = repo.Prepare(ctx, changed)
	require.ErrorIs(t, err, agentconfig.ErrConflict)
	guard, err := NewGuard(db, repo, runs, func(string, string) (agent.Limits, error) { return limits, nil }, func(context.Context) (agent.ExecutionIdentity, error) { return identity, nil })
	require.NoError(t, err)
	request := snap.Request
	request.ConfigurationSnapshotRef = agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: snap.ID, Digest: snap.Digest}
	_, _, err = guard.Claim(ctx, agent.Record{State: agent.State{Scope: scope, Request: request}}, 0)
	require.ErrorIs(t, err, agentconfig.ErrChanged)
}
