package agentconfigpersistence

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
)

func materialsCandidateCommand() agentconfig.StartCommand {
	p := aicapability.ModelProfile{ClientName: "materials", ProviderID: "synthetic", AdapterKind: "openai-compatible", ModelID: "model", EndpointIdentityDigest: "endpoint", CredentialVersion: "credential", RoutingPolicyVersion: "routing", AdapterPolicyVersion: "adapter", PromptVersion: agentconfig.MaterialsPromptVersion, OutputSchemaVersion: agentconfig.MaterialsActionVersion, UsageMappingVersion: "usage", CostPricingVersion: "cost", OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "points", FXVersion: "fixture-fx-identity", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: "model", Channel: "fixture-channel", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialTokenRates{InputText: 10000, OutputText: 10000}}, Currency: "CNY", InputMicrosPerMillion: 1000000, OutputMicrosPerMillion: 1000000, MaximumPromptTokens: 1000, MaximumCompletionTokens: 100, MaximumInputBytes: 4096, MaximumOutputBytes: 4096, DeadlineBound: time.Minute}
	return agentconfig.StartCommand{Scope: agent.Scope{OrganizationID: "materials-c32", ActorID: "actor"}, MemberID: "member", ExpectedModeRevision: "0", AgentID: agentconfig.MaterialsAgentID, AgentVersion: agentconfig.MaterialsAgentVersion, ExecutionModelProfile: p, Request: agent.Request{Key: uuid.NewString(), Binding: agent.Binding{ContextKind: "listing-target-record", ContextID: uuid.NewString(), ProductKey: "product", CatalogVersion: "1", PublicationID: "publication", TargetPlatform: "shein"}, PolicyVersion: agentconfig.MaterialsPolicyVersion, PromptVersion: agentconfig.MaterialsPromptVersion, Limits: agent.Limits{Steps: 6, ModelCalls: 3, Tokens: 2000, CostMicros: 2000, Currency: "CNY", Runtime: time.Minute}}}
}

func withCandidateRules(c agentconfig.StartCommand) agentconfig.StartCommand {
	c.Request.Binding.ContextKind = agent.CandidateRulesContextKind
	c.Request.Binding.CandidateRules = agent.CandidateRulesRef{BaseRevision: "1", BaseDigest: strings.Repeat("a", 64), StoreID: uuid.NewString(), Site: "shein-us", MerchantDigest: strings.Repeat("b", 64), CandidateCategoryID: "123", SelectorDigest: strings.Repeat("c", 64)}
	return c
}

func TestMaterialsCandidateCommandIdentityCannotChangePurposeOrBase(t *testing.T) {
	for _, queryFirst := range []bool{false, true} {
		_, s, _ := fixture(t)
		ctx := context.Background()
		ordinary := materialsCandidateCommand()
		enable := command(ordinary.Scope, "enable", 0)
		enable.AgentID, enable.Absent = ordinary.AgentID, true
		_, err := s.Execute(ctx, enable)
		require.NoError(t, err)
		query := withCandidateRules(ordinary)
		first, changed := ordinary, query
		if queryFirst {
			first, changed = query, ordinary
		}
		frozen, err := s.Prepare(ctx, first)
		require.NoError(t, err)
		replay, err := s.Prepare(ctx, first)
		require.NoError(t, err)
		require.Equal(t, frozen, replay)
		_, err = s.Prepare(ctx, changed)
		require.ErrorIs(t, err, agentconfig.ErrConflict, "same public command key cannot silently start the other purpose")
		changed = first
		changed.Request.Binding.ContextID = uuid.NewString()
		_, err = s.Prepare(ctx, changed)
		require.ErrorIs(t, err, agentconfig.ErrConflict, "base record is intent, not a new namespace for the same command")
	}
}

func TestMaterialsCandidateConcurrentPrepareHasOnePublicCommand(t *testing.T) {
	db, s, _ := fixture(t)
	c := materialsCandidateCommand()
	enable := command(c.Scope, "enable", 0)
	enable.AgentID, enable.Absent = c.AgentID, true
	_, err := s.Execute(context.Background(), enable)
	require.NoError(t, err)
	commands := []agentconfig.StartCommand{c, withCandidateRules(c)}
	start := make(chan struct{})
	errors := make([]error, 2)
	var wg sync.WaitGroup
	for n := range commands {
		wg.Add(1)
		go func(n int) { defer wg.Done(); <-start; _, errors[n] = s.Prepare(context.Background(), commands[n]) }(n)
	}
	close(start)
	wg.Wait()
	winners := 0
	for _, err := range errors {
		if err == nil {
			winners++
		} else {
			require.ErrorIs(t, err, agentconfig.ErrConflict)
		}
	}
	require.Equal(t, 1, winners)
	var count int64
	require.NoError(t, db.Model(&snapshotRow{}).Where("agent_id=? AND request_key=?", c.AgentID, c.Request.Key).Count(&count).Error)
	require.EqualValues(t, 1, count)
}
