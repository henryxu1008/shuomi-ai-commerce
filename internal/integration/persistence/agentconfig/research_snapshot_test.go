package agentconfigpersistence

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	"task-processor/internal/commercetool"
	"testing"
	"time"
)

func TestResearchConfigurationHasAnExactConsumerAndCannotBorrowSingleProductIdentity(t *testing.T) {
	r := agent.Request{Binding: agent.Binding{ContextKind: agent.ResearchSelectionContextKind, ContextID: uuid.NewString(), TargetPlatform: "product"}, PolicyVersion: agentconfig.ResearchPolicyVersion, PromptVersion: agentconfig.ResearchPromptVersion}
	for n := 0; n < 2; n++ {
		r.Binding.Selection[n] = agent.SelectionProductRef{ItemID: uuid.NewString(), ItemRevision: "1", ProductKey: string(rune('a' + n)), CatalogVersion: "1", PublicationID: "source"}
	}
	if r.Binding.Selection[0].ItemID > r.Binding.Selection[1].ItemID {
		r.Binding.Selection[0], r.Binding.Selection[1] = r.Binding.Selection[1], r.Binding.Selection[0]
	}
	r.Binding.ResearchSelection = researchRefs(2)
	require.True(t, r.Binding.Valid())
	require.True(t, validTextConfigurationRequest("product.research.agent", r))
	for _, id := range []string{"product.title.agent", "product.content.agent", "listing.materials.agent", "unregistered"} {
		require.False(t, validTextConfigurationRequest(id, r), id)
	}
	r.Binding = agent.Binding{ContextKind: "acquisition", ContextID: uuid.NewString(), ProductKey: "product", CatalogVersion: "1", PublicationID: "source", TargetPlatform: "shein"}
	require.False(t, validTextConfigurationRequest("product.research.agent", r))
}

func TestResearchSnapshotsGuardAndReplayUseOriginalModeOwner(t *testing.T) {
	for _, mode := range []agent.ExecutionMode{agent.ModeDefault, agent.ModeReadOnly, agent.ModeAutoApprove, agent.ModeFullAccess} {
		t.Run(string(mode), func(t *testing.T) {
			db, store, runs := fixture(t)
			ctx := context.Background()
			id := agent.ExecutionIdentity{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
			scope := agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}
			subject := agentconfig.ResolvedModeSubject{Subject: agentconfig.ModeSubject{Kind: agentconfig.RegisteredAgentSubject, ID: agentconfig.ResearchAgentID}, Definition: commercetool.AgentRef{ID: agentconfig.ResearchAgentID, Version: agentconfig.ResearchAgentVersion}}
			revision := "0"
			if mode != agent.ModeDefault {
				mr, err := store.ExecuteMode(ctx, agentconfig.ModeCommand{Identity: id, Subject: subject, Key: uuid.NewString(), Mode: mode})
				require.NoError(t, err)
				revision = mr.Revision
			}
			limits := agent.Limits{Steps: 6, ModelCalls: 2, Tokens: 2000, CostMicros: 2000, Currency: "CNY", Runtime: time.Minute}
			profile := aicapability.ModelProfile{ClientName: "research", ProviderID: "synthetic", AdapterKind: "openai-compatible", ModelID: "model", EndpointIdentityDigest: "endpoint", CredentialVersion: "credential", RoutingPolicyVersion: "routing", AdapterPolicyVersion: "adapter", PromptVersion: agentconfig.ResearchPromptVersion, OutputSchemaVersion: agentconfig.ResearchActionVersion, UsageMappingVersion: "usage", CostPricingVersion: "cost", PointTariff: aicapability.ModelPointTariff{PriceVersion: "points", InputPointsPerMillionTokens: 1000000, OutputPointsPerMillionTokens: 1000000}, Currency: "CNY", InputMicrosPerMillion: 1000000, OutputMicrosPerMillion: 1000000, MaximumPromptTokens: 1000, MaximumCompletionTokens: 100, MaximumInputBytes: 4096, MaximumOutputBytes: 4096, DeadlineBound: time.Minute}
			binding := agent.Binding{ContextKind: agent.ResearchSelectionContextKind, ContextID: uuid.NewString(), TargetPlatform: "product"}
			binding.Selection[0] = agent.SelectionProductRef{ItemID: "11111111-1111-4111-8111-111111111111", ItemRevision: "1", ProductKey: "a", CatalogVersion: "1", PublicationID: "source-a"}
			binding.Selection[1] = agent.SelectionProductRef{ItemID: "22222222-2222-4222-8222-222222222222", ItemRevision: "1", ProductKey: "b", CatalogVersion: "1", PublicationID: "source-b"}
			binding.ResearchSelection = researchRefs(2)
			start := agentconfig.StartCommand{Scope: scope, MemberID: id.MemberID, ExpectedModeRevision: revision, AgentID: agentconfig.ResearchAgentID, AgentVersion: agentconfig.ResearchAgentVersion, ExecutionModelProfile: profile, Request: agent.Request{Key: uuid.NewString(), Binding: binding, GoalSummary: "比较真实资料并指出缺失成本", PolicyVersion: agentconfig.ResearchPolicyVersion, PromptVersion: agentconfig.ResearchPromptVersion, Limits: limits}}
			_, err := store.Prepare(ctx, start)
			require.ErrorIs(t, err, agentconfig.ErrNotEnabled)
			_, err = store.Execute(ctx, agentconfig.Command{Scope: scope, Key: uuid.NewString(), AgentID: start.AgentID, Operation: "enable", Absent: true})
			require.NoError(t, err)
			snap, err := store.Prepare(ctx, start)
			require.NoError(t, err)
			ref := agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: snap.ID, Digest: snap.Digest}
			frozen, err := store.CheckRegisteredTextExecutionMode(ctx, id, ref, snap.Request.ModeSnapshotRef)
			require.NoError(t, err)
			require.Equal(t, mode, frozen.Mode)
			changed := start
			changed.Request.Binding.Selection[1].CatalogVersion = "2"
			_, err = store.Prepare(ctx, changed)
			require.ErrorIs(t, err, agentconfig.ErrConflict)
			foreign := id
			foreign.MemberID = "other-member"
			_, err = store.CheckRegisteredTextExecutionMode(ctx, foreign, ref, snap.Request.ModeSnapshotRef)
			require.Error(t, err)
			guard, err := NewGuard(db, store, runs, func(string, string) (agent.Limits, error) { return limits, nil }, func(context.Context) (agent.ExecutionIdentity, error) { return id, nil })
			require.NoError(t, err)
			request := snap.Request
			request.ConfigurationSnapshotRef = ref
			now := time.Now().UTC().Truncate(time.Microsecond)
			initial := agent.Record{State: agent.State{RunID: uuid.NewString(), Scope: scope, Request: request, Fingerprint: strings.Repeat("a", 64), Phase: agent.Running, StartedAt: now, Deadline: now.Add(time.Minute), HumanReviewRequired: true}}
			_, acquired, err := guard.Claim(ctx, initial, 0)
			require.NoError(t, err)
			require.True(t, acquired)
			read, found, err := runs.LookupContext(ctx, scope, agent.ResearchSelectionContextKind, binding.ContextID, request.Key)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, binding, read.State.Request.Binding)
			for _, wrong := range []agent.Scope{{OrganizationID: "other", ActorID: scope.ActorID}, {OrganizationID: scope.OrganizationID, ActorID: "other"}} {
				_, found, err = runs.LookupContext(ctx, wrong, agent.ResearchSelectionContextKind, binding.ContextID, request.Key)
				require.NoError(t, err)
				require.False(t, found)
			}
			_, acquired, err = guard.Claim(ctx, initial, 0)
			require.NoError(t, err)
			require.False(t, acquired, "replay never creates a second dispatch")
			_, err = store.Execute(ctx, agentconfig.Command{Scope: scope, Key: uuid.NewString(), AgentID: start.AgentID, Operation: "disable", Expected: 1})
			require.NoError(t, err)
			_, err = store.CheckRegisteredTextExecutionMode(ctx, id, ref, snap.Request.ModeSnapshotRef)
			require.ErrorIs(t, err, agentconfig.ErrChanged)
			retained, err := store.LoadSnapshot(ctx, scope, ref)
			require.NoError(t, err)
			require.Equal(t, snap, retained, "disable does not rewrite saved history or binding")
		})
	}
}

func researchRefs(count int) agent.ResearchSelectionRef {
	r := agent.ResearchSelectionRef{InspectionDigest: strings.Repeat("a", 64)}
	for n := 0; n < count; n++ {
		r.Candidates[n] = agent.ResearchCandidateRef{CandidateID: uuid.NewString(), CandidateRevision: "1", SelectedLinkID: uuid.NewString(), EvidenceDigest: strings.Repeat("b", 64)}
	}
	return r
}

func researchStart(count int) agentconfig.StartCommand {
	b := agent.Binding{ContextKind: agent.ResearchSelectionContextKind, ContextID: uuid.NewString(), TargetPlatform: "product", ResearchSelection: researchRefs(count)}
	for n := 0; n < count; n++ {
		b.Selection[n] = agent.SelectionProductRef{ItemID: fmt.Sprintf("%08d-1111-4111-8111-111111111111", n+1), ItemRevision: "1", ProductKey: fmt.Sprintf("amazon:US:B%09d", n), CatalogVersion: "1", PublicationID: "data:" + uuid.NewString()}
	}
	profile := aicapability.ModelProfile{ClientName: "research", ProviderID: "synthetic", AdapterKind: "openai-compatible", ModelID: "model", EndpointIdentityDigest: "endpoint", CredentialVersion: "credential", RoutingPolicyVersion: "routing", AdapterPolicyVersion: "adapter", PromptVersion: agentconfig.ResearchPromptVersion, OutputSchemaVersion: agentconfig.ResearchActionVersion, UsageMappingVersion: "usage", CostPricingVersion: "cost", PointTariff: aicapability.ModelPointTariff{PriceVersion: "points", InputPointsPerMillionTokens: 1000000, OutputPointsPerMillionTokens: 1000000}, Currency: "CNY", InputMicrosPerMillion: 1000000, OutputMicrosPerMillion: 1000000, MaximumPromptTokens: 1000, MaximumCompletionTokens: 100, MaximumInputBytes: 4096, MaximumOutputBytes: 4096, DeadlineBound: time.Minute}
	return agentconfig.StartCommand{Scope: agent.Scope{OrganizationID: "org", ActorID: "actor"}, MemberID: "member", ExpectedModeRevision: "0", AgentID: agentconfig.ResearchAgentID, AgentVersion: agentconfig.ResearchAgentVersion, ExecutionModelProfile: profile, Request: agent.Request{Key: uuid.NewString(), Binding: b, GoalSummary: "Compare saved observations", PolicyVersion: agentconfig.ResearchPolicyVersion, PromptVersion: agentconfig.ResearchPromptVersion, Limits: agent.Limits{Steps: 6, ModelCalls: 2, Tokens: 2000, CostMicros: 2000, Currency: "CNY", Runtime: time.Minute}}}
}
func TestResearchSnapshotFullBudgetAndReceiptRecovery(t *testing.T) {
	db, s, _ := fixture(t)
	ctx := context.Background()
	c := researchStart(2)
	_, err := s.Execute(ctx, agentconfig.Command{Scope: c.Scope, Key: uuid.NewString(), AgentID: c.AgentID, Operation: "enable", Absent: true})
	require.NoError(t, err)
	for _, count := range []int{2, 5} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			c := researchStart(count)
			c.Request.GoalSummary = strings.Repeat("g", 512)
			snap, err := s.Prepare(ctx, c)
			require.NoError(t, err)
			raw, err := json.Marshal(snap)
			require.NoError(t, err)
			require.LessOrEqual(t, len(raw), 8192)
			recovered, found, err := s.LoadResearchPreparationByKey(ctx, c.Scope, c.MemberID, c.Request.Binding.ContextID, c.Request.Key)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, snap, recovered)
			replay, err := s.Prepare(ctx, c)
			require.NoError(t, err)
			require.Equal(t, snap, replay)
			_, found, err = s.LoadResearchPreparationByKey(ctx, c.Scope, "replacement-member", c.Request.Binding.ContextID, c.Request.Key)
			require.ErrorIs(t, err, agentconfig.ErrForbidden)
			require.True(t, found)
			_, found, err = s.LoadResearchPreparationByKey(ctx, c.Scope, c.MemberID, uuid.NewString(), c.Request.Key)
			require.NoError(t, err)
			require.False(t, found)
		})
	}
	// Legal individual fields can exceed the shared whole-snapshot budget. No
	// mode/configuration receipt may survive that failed transaction.
	c = researchStart(5)
	c.Request.GoalSummary = strings.Repeat("g", 512)
	for n := range c.Request.Binding.Selection {
		c.Request.Binding.Selection[n].ProductKey = strings.Repeat(string(rune('a'+n)), 128)
		c.Request.Binding.Selection[n].PublicationID = strings.Repeat("p", 128)
	}
	c.ExecutionModelProfile.ClientName = strings.Repeat("x", 128)
	c.ExecutionModelProfile.ProviderID = strings.Repeat("v", 128)
	c.ExecutionModelProfile.PointTariff.PriceVersion = strings.Repeat("t", 128)
	c.ExecutionModelProfile.ModelID = strings.Repeat("m", 128)
	c.ExecutionModelProfile.EndpointIdentityDigest = strings.Repeat("e", 128)
	c.ExecutionModelProfile.CredentialVersion = strings.Repeat("c", 128)
	c.ExecutionModelProfile.RoutingPolicyVersion = strings.Repeat("r", 128)
	c.ExecutionModelProfile.AdapterPolicyVersion = strings.Repeat("a", 128)
	c.ExecutionModelProfile.UsageMappingVersion = strings.Repeat("u", 128)
	c.ExecutionModelProfile.CostPricingVersion = strings.Repeat("p", 128)
	require.True(t, c.Request.Binding.Valid())
	require.NoError(t, c.ExecutionModelProfile.Validate())
	retained, err := s.Prepare(ctx, c)
	require.NoError(t, err)
	raw, err := json.Marshal(retained)
	require.NoError(t, err)
	require.LessOrEqual(t, len(raw), 8192)
	c.Request.Key = uuid.NewString()
	c.Request.GoalSummary = strings.Repeat("<", 512)
	_, err = s.Prepare(ctx, c)
	require.ErrorIs(t, err, agentconfig.ErrInvalid)
	var count int64
	require.NoError(t, db.Table("agent_configuration.start_snapshots").Where("request_key=?", c.Request.Key).Count(&count).Error)
	require.Zero(t, count)
	require.NoError(t, db.Table("agent_configuration.execution_mode_snapshots").Where("request_key=?", c.Request.Key).Count(&count).Error)
	require.Zero(t, count)
}

func TestResearchRecentFiltersProjectAndMemberBeforePagingAndAnchor(t *testing.T) {
	db, s, runs := fixture(t)
	ctx := context.Background()
	base := researchStart(2)
	_, err := s.Execute(ctx, agentconfig.Command{Scope: base.Scope, Key: uuid.NewString(), AgentID: base.AgentID, Operation: "enable", Absent: true})
	require.NoError(t, err)
	project := base.Request.Binding.ContextID
	other := uuid.NewString()
	var foreignCursor string
	for n := 0; n < 4; n++ {
		c := researchStart(2)
		c.Request.Binding.ContextID = project
		if n == 2 {
			c.Request.Binding.ContextID = other
		}
		if n == 3 {
			c.MemberID = "replacement-member"
		}
		snap, err := s.Prepare(ctx, c)
		require.NoError(t, err)
		if n == 2 {
			foreignCursor = snap.ID
		}
		ref := agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: snap.ID, Digest: snap.Digest}
		request := snap.Request
		request.ConfigurationSnapshotRef = ref
		now := time.Now().UTC().Truncate(time.Microsecond)
		guard, err := NewGuard(db, s, runs, func(string, string) (agent.Limits, error) { return request.Limits, nil }, func(context.Context) (agent.ExecutionIdentity, error) {
			return agent.ExecutionIdentity{OrganizationID: base.Scope.OrganizationID, ActorID: base.Scope.ActorID, MemberID: c.MemberID}, nil
		})
		require.NoError(t, err)
		_, acquired, err := guard.Claim(ctx, agent.Record{State: agent.State{RunID: uuid.NewString(), Scope: base.Scope, Request: request, Fingerprint: strings.Repeat("a", 64), Phase: agent.Running, StartedAt: now, Deadline: now.Add(time.Minute), HumanReviewRequired: true}}, 0)
		require.NoError(t, err)
		require.True(t, acquired)
	}
	calls := 0
	authorize := func(b agent.Binding) error { calls++; require.Equal(t, project, b.ContextID); return nil }
	filter := RecentContext{Kind: agent.ResearchSelectionContextKind, ID: project}
	page, cursor, err := s.Recent(ctx, base.Scope, base.MemberID, base.AgentID, "", 1, runs, authorize, filter)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.NotEmpty(t, cursor)
	require.Equal(t, 1, calls)
	page, cursor, err = s.Recent(ctx, base.Scope, base.MemberID, base.AgentID, cursor, 1, runs, authorize, filter)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Empty(t, cursor)
	_, _, err = s.Recent(ctx, base.Scope, base.MemberID, base.AgentID, foreignCursor, 1, runs, authorize, filter)
	require.ErrorIs(t, err, agentconfig.ErrInvalid)
	_, _, err = s.Recent(ctx, base.Scope, base.MemberID, base.AgentID, "", 1, runs, authorize)
	require.ErrorIs(t, err, agentconfig.ErrInvalid, "research discovery must always be project scoped")
}
