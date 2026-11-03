package agentconfigpersistence

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	"task-processor/internal/commercetool"
)

func analysisStart(t *testing.T, s *Store) agentconfig.StartCommand {
	t.Helper()
	id := agent.ExecutionIdentity{OrganizationID: "analysis-org", ActorID: "actor", MemberID: "member"}
	scope := agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}
	_, err := s.Execute(context.Background(), agentconfig.Command{Scope: scope, Key: uuid.NewString(), AgentID: agentconfig.ImageAgentID, Operation: "enable", Absent: true})
	require.NoError(t, err)
	subject := agentconfig.ResolvedModeSubject{Subject: agentconfig.ModeSubject{Kind: agentconfig.RegisteredAgentSubject, ID: agentconfig.ImageAgentID}, Definition: commercetool.AgentRef{ID: agentconfig.ImageAgentID, Version: agentconfig.ImageAgentVersion}}
	mr, err := s.ExecuteMode(context.Background(), agentconfig.ModeCommand{Identity: id, Subject: subject, Key: uuid.NewString(), Mode: agent.ModeReadOnly})
	require.NoError(t, err)
	p := aicapability.ModelProfile{ClientName: "openai-mini", ProviderID: "openai", AdapterKind: "openai-compatible", ModelID: aicapability.SingleImageMiniModel, EndpointIdentityDigest: strings.Repeat("a", 64), CredentialVersion: "credential", RoutingPolicyVersion: "shared-choice", AdapterPolicyVersion: aicapability.SingleImageMiniAdapterPolicy, PromptVersion: agentconfig.ImageAnalysisPromptVersion, OutputSchemaVersion: agentconfig.ImageAnalysisResultVersion, UsageMappingVersion: "openai-text-v1", CostPricingVersion: "budget", Currency: "CNY", InputMicrosPerMillion: 1000000, OutputMicrosPerMillion: 1000000, MaximumPromptTokens: 8000, MaximumCompletionTokens: 1024, MaximumInputBytes: 8192, MaximumOutputBytes: 16384, DeadlineBound: time.Minute,
		OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "synthetic-price", FXVersion: "fixture-fx", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: aicapability.SingleImageMiniModel, Channel: "openai-direct", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 10000, CachedInputText: 1000, OutputText: 20000}}}
	c := agentconfig.StartCommand{Scope: scope, MemberID: id.MemberID, ExpectedModeRevision: mr.Revision, AgentID: agentconfig.ImageAgentID, AgentVersion: agentconfig.ImageAgentVersion, ExecutionModelProfile: p, Request: agent.Request{Key: uuid.NewString(), Binding: agent.Binding{ContextKind: "own-product", ContextID: uuid.NewString(), ProductKey: "product", CatalogVersion: "1", PublicationID: "publication", TargetPlatform: "product"}, PolicyVersion: agentconfig.ImageAnalysisPolicyVersion, PromptVersion: agentconfig.ImageAnalysisPromptVersion, Limits: agent.Limits{Steps: 6, ModelCalls: 1, Tokens: 10000, CostMicros: 10000, Currency: "CNY", Runtime: 2 * time.Minute}}}
	a, err := agentconfig.FreezeImageAnalysis(c, "source", "catalog-image-1", "", "", aicapability.SingleImageInput{ContentSHA256: strings.Repeat("b", 64), MediaType: "image/png", ByteLength: 256, Width: 64, Height: 64, Detail: "high", MeteringVersion: aicapability.SingleImageMiniMetering})
	require.NoError(t, err)
	c.ImageAnalysis = &a
	return c
}

func TestImageAnalysisPreparationUsesOriginalImageActivationAndMode(t *testing.T) {
	db, s, runs := fixture(t)
	c := analysisStart(t, s)
	ctx := context.Background()
	snap, err := s.Prepare(ctx, c)
	require.NoError(t, err)
	require.Nil(t, snap.Template, "readonly analysis needs no generation template")
	require.Equal(t, agentconfig.ImageAgentID, snap.AgentID)
	require.Equal(t, agentconfig.ImageAnalysisAgentID, snap.ImageAnalysis.ExecutionID)
	require.Equal(t, agentconfig.ImageAnalysisVersion, snap.ImageAnalysis.ExecutionVersion)
	require.Positive(t, snap.ImageAnalysis.MaximumMicropoints)
	ref := agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: snap.ID, Digest: snap.Digest}
	identity := agent.ExecutionIdentity{OrganizationID: c.Scope.OrganizationID, ActorID: c.Scope.ActorID, MemberID: c.MemberID}
	mode, err := s.CheckRegisteredTextExecutionMode(ctx, identity, ref, snap.Request.ModeSnapshotRef)
	require.NoError(t, err)
	require.Equal(t, agent.ModeReadOnly, mode.Mode)
	require.Equal(t, agentconfig.ImageAgentID, mode.Subject.Subject.ID)
	require.Equal(t, agentconfig.ImageAgentID, mode.Subject.Definition.ID)
	loaded, err := s.LoadSnapshot(ctx, c.Scope, ref)
	require.NoError(t, err)
	require.True(t, loaded.Request == snap.Request, "the existing comparable Request stays unchanged")
	raw, err := json.Marshal(snap)
	require.NoError(t, err)
	require.LessOrEqual(t, len(raw), 8192)
	require.NotContains(t, string(raw), "base64")
	guard, err := NewGuard(db, s, runs, func(id, version string) (agent.Limits, error) {
		require.Equal(t, agentconfig.ImageAgentID, id)
		require.Equal(t, agentconfig.ImageAgentVersion, version)
		return c.Request.Limits, nil
	}, func(context.Context) (agent.ExecutionIdentity, error) { return identity, nil })
	require.NoError(t, err)
	request := snap.Request
	request.ConfigurationSnapshotRef = ref
	now := time.Now().UTC().Truncate(time.Microsecond)
	initial := agent.Record{State: agent.State{RunID: uuid.NewString(), Scope: c.Scope, Request: request, Fingerprint: strings.Repeat("a", 64), Phase: agent.Running, StartedAt: now, Deadline: now.Add(request.Limits.Runtime), HumanReviewRequired: true}}
	_, acquired, err := guard.Claim(ctx, initial, 0)
	require.NoError(t, err)
	require.True(t, acquired)
	_, acquired, err = guard.Claim(ctx, initial, 0)
	require.NoError(t, err)
	require.False(t, acquired)
	var activationCount int64
	require.NoError(t, db.Table("agent_configuration.organization_agents").Where("organization_id=?", c.Scope.OrganizationID).Count(&activationCount).Error)
	require.EqualValues(t, 1, activationCount, "no second analysis activation or mode subject")
}

func TestImageAnalysisPreparationBindsCrossContextKeyAndRetainsOriginalReceipt(t *testing.T) {
	_, s, _ := fixture(t)
	c := analysisStart(t, s)
	ctx := context.Background()
	snap, err := s.Prepare(ctx, c)
	require.NoError(t, err)
	other := c
	other.Request.Binding.ContextKind, other.Request.Binding.ContextID = "acquisition", uuid.NewString()
	a, err := agentconfig.FreezeImageAnalysis(other, "source", "catalog-image-1", "", "", c.ImageAnalysis.Image)
	require.NoError(t, err)
	other.ImageAnalysis = &a
	_, err = s.Prepare(ctx, other)
	require.ErrorIs(t, err, agentconfig.ErrConflict, "a new context cannot create another operation for the original analysis key")
	_, err = s.Execute(ctx, agentconfig.Command{Scope: c.Scope, Key: uuid.NewString(), AgentID: agentconfig.ImageAgentID, Operation: "disable", Expected: 1})
	require.NoError(t, err)
	replay, err := s.Prepare(ctx, c)
	require.NoError(t, err, "an original receipt remains readable, not executable")
	require.Equal(t, snap.ID, replay.ID)
	require.Equal(t, snap.Digest, replay.Digest)
	ref := agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: snap.ID, Digest: snap.Digest}
	_, err = s.CheckRegisteredTextExecutionMode(ctx, agent.ExecutionIdentity{OrganizationID: c.Scope.OrganizationID, ActorID: c.Scope.ActorID, MemberID: c.MemberID}, ref, snap.Request.ModeSnapshotRef)
	require.ErrorIs(t, err, agentconfig.ErrChanged)
	_, err = s.LoadSnapshot(ctx, agent.Scope{OrganizationID: "foreign", ActorID: c.Scope.ActorID}, ref)
	require.Error(t, err)
	newIntent := c
	newIntent.Request.Key = uuid.NewString()
	a, err = agentconfig.FreezeImageAnalysis(newIntent, "source", "catalog-image-1", "", "", c.ImageAnalysis.Image)
	require.NoError(t, err)
	newIntent.ImageAnalysis = &a
	_, err = s.Prepare(ctx, newIntent)
	require.ErrorIs(t, err, agentconfig.ErrNotEnabled)
}

func TestImageAnalysisPreparationAndClaimCoexistWithImageFence(t *testing.T) {
	db, s, runs := fixture(t)
	c := analysisStart(t, s)
	holdImageSkillFence(t, db, c.Scope.OrganizationID)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	snap, err := s.Prepare(ctx, c)
	require.NoError(t, err, "analysis must reuse the image preparation lock order and shared activation fence")
	id := agent.ExecutionIdentity{OrganizationID: c.Scope.OrganizationID, ActorID: c.Scope.ActorID, MemberID: c.MemberID}
	g, err := NewGuard(db, s, runs, func(string, string) (agent.Limits, error) { return c.Request.Limits, nil }, func(context.Context) (agent.ExecutionIdentity, error) { return id, nil })
	require.NoError(t, err)
	request := snap.Request
	request.ConfigurationSnapshotRef = agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: snap.ID, Digest: snap.Digest}
	now := time.Now().UTC()
	_, acquired, err := g.Claim(ctx, agent.Record{State: agent.State{RunID: uuid.NewString(), Scope: c.Scope, Request: request, Fingerprint: strings.Repeat("a", 64), Phase: agent.Running, StartedAt: now, Deadline: now.Add(request.Limits.Runtime), HumanReviewRequired: true}}, 0)
	require.NoError(t, err)
	require.True(t, acquired)
}

func TestImageAnalysisPreparationConcurrentOriginalKey(t *testing.T) {
	db, s, _ := fixture(t)
	c := analysisStart(t, s)
	var wg sync.WaitGroup
	type result struct {
		snap agentconfig.Snapshot
		err  error
	}
	results := make(chan result, 6)
	for range 6 {
		wg.Go(func() { snap, err := s.Prepare(context.Background(), c); results <- result{snap, err} })
	}
	wg.Wait()
	close(results)
	var original agentconfig.Snapshot
	for r := range results {
		require.NoError(t, r.err)
		if original.ID == "" {
			original = r.snap
		} else {
			require.Equal(t, original, r.snap)
		}
	}
	var n int64
	require.NoError(t, db.Table("agent_configuration.start_snapshots").Where("organization_id=? AND actor_id=? AND request_key=?", c.Scope.OrganizationID, c.Scope.ActorID, c.Request.Key).Count(&n).Error)
	require.EqualValues(t, 1, n)
}
