package agentconfigpersistence

import (
	"context"
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

func TestSelectionConfigurationHasAnExactConsumerAndCannotBorrowSingleProductIdentity(t *testing.T) {
	r := agent.Request{Binding: agent.Binding{ContextKind: agent.SelectionContextKind, ContextID: uuid.NewString(), TargetPlatform: "product"}, PolicyVersion: "product-selection-policy-v1", PromptVersion: "product-selection-prompt-v1"}
	for n := 0; n < 2; n++ {
		r.Binding.Selection[n] = agent.SelectionProductRef{ItemID: uuid.NewString(), ItemRevision: "1", ProductKey: string(rune('a' + n)), CatalogVersion: "1", PublicationID: "source"}
	}
	if r.Binding.Selection[0].ItemID > r.Binding.Selection[1].ItemID {
		r.Binding.Selection[0], r.Binding.Selection[1] = r.Binding.Selection[1], r.Binding.Selection[0]
	}
	require.True(t, r.Binding.Valid())
	require.True(t, validTextConfigurationRequest("product.selection.agent", r))
	for _, id := range []string{"product.title.agent", "product.content.agent", "listing.materials.agent", "unregistered"} {
		require.False(t, validTextConfigurationRequest(id, r), id)
	}
	r.Binding = agent.Binding{ContextKind: "acquisition", ContextID: uuid.NewString(), ProductKey: "product", CatalogVersion: "1", PublicationID: "source", TargetPlatform: "shein"}
	require.False(t, validTextConfigurationRequest("product.selection.agent", r))
}

func TestSelectionSnapshotsGuardAndReplayUseOriginalModeOwner(t *testing.T) {
	for _, mode := range []agent.ExecutionMode{agent.ModeDefault, agent.ModeReadOnly, agent.ModeAutoApprove, agent.ModeFullAccess} {
		t.Run(string(mode), func(t *testing.T) {
			db, store, runs := fixture(t)
			ctx := context.Background()
			id := agent.ExecutionIdentity{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
			scope := agent.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}
			subject := agentconfig.ResolvedModeSubject{Subject: agentconfig.ModeSubject{Kind: agentconfig.RegisteredAgentSubject, ID: agentconfig.SelectionAgentID}, Definition: commercetool.AgentRef{ID: agentconfig.SelectionAgentID, Version: agentconfig.SelectionAgentVersion}}
			revision := "0"
			if mode != agent.ModeDefault {
				mr, err := store.ExecuteMode(ctx, agentconfig.ModeCommand{Identity: id, Subject: subject, Key: uuid.NewString(), Mode: mode})
				require.NoError(t, err)
				revision = mr.Revision
			}
			limits := agent.Limits{Steps: 6, ModelCalls: 3, Tokens: 2000, CostMicros: 2000, Currency: "CNY", Runtime: time.Minute}
			profile := aicapability.ModelProfile{ClientName: "selection", ProviderID: "synthetic", AdapterKind: "openai-compatible", ModelID: "model", EndpointIdentityDigest: "endpoint", CredentialVersion: "credential", RoutingPolicyVersion: "routing", AdapterPolicyVersion: "adapter", PromptVersion: agentconfig.SelectionPromptVersion, OutputSchemaVersion: agentconfig.SelectionActionVersion, UsageMappingVersion: "usage", CostPricingVersion: "cost", PointTariff: aicapability.ModelPointTariff{PriceVersion: "points", InputPointsPerMillionTokens: 1000000, OutputPointsPerMillionTokens: 1000000}, Currency: "CNY", InputMicrosPerMillion: 1000000, OutputMicrosPerMillion: 1000000, MaximumPromptTokens: 1000, MaximumCompletionTokens: 100, MaximumInputBytes: 4096, MaximumOutputBytes: 4096, DeadlineBound: time.Minute}
			binding := agent.Binding{ContextKind: agent.SelectionContextKind, ContextID: uuid.NewString(), TargetPlatform: "product"}
			binding.Selection[0] = agent.SelectionProductRef{ItemID: "11111111-1111-4111-8111-111111111111", ItemRevision: "1", ProductKey: "a", CatalogVersion: "1", PublicationID: "source-a"}
			binding.Selection[1] = agent.SelectionProductRef{ItemID: "22222222-2222-4222-8222-222222222222", ItemRevision: "1", ProductKey: "b", CatalogVersion: "1", PublicationID: "source-b"}
			start := agentconfig.StartCommand{Scope: scope, MemberID: id.MemberID, ExpectedModeRevision: revision, AgentID: agentconfig.SelectionAgentID, AgentVersion: agentconfig.SelectionAgentVersion, ExecutionModelProfile: profile, Request: agent.Request{Key: uuid.NewString(), Binding: binding, GoalSummary: "比较真实资料并指出缺失成本", PolicyVersion: agentconfig.SelectionPolicyVersion, PromptVersion: agentconfig.SelectionPromptVersion, Limits: limits}}
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
			read, found, err := runs.LookupContext(ctx, scope, agent.SelectionContextKind, binding.ContextID, request.Key)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, binding, read.State.Request.Binding)
			for _, wrong := range []agent.Scope{{OrganizationID: "other", ActorID: scope.ActorID}, {OrganizationID: scope.OrganizationID, ActorID: "other"}} {
				_, found, err = runs.LookupContext(ctx, wrong, agent.SelectionContextKind, binding.ContextID, request.Key)
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
