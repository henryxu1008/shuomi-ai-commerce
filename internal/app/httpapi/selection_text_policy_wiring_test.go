package httpapi

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	credential "task-processor/internal/aicapability/channelcredential"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	texteino "task-processor/internal/integration/agent/einomodel"
	governed "task-processor/internal/integration/aicapability/einomodel"
	qualified "task-processor/internal/integration/aicapability/routeconfig"
	runstore "task-processor/internal/integration/persistence/agent"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/research"
)

// This fixture exercises the stable reader port without a provider connection.
type selectionCurrentTextPolicies struct {
	policies map[governed.RouteKey]governed.RoutePolicy
	keys     []governed.RouteKey
	err      error
}

func (p *selectionCurrentTextPolicies) ReadPolicy(_ context.Context, key governed.RouteKey) (governed.RoutePolicy, error) {
	p.keys = append(p.keys, key)
	if policy, ok := p.policies[key]; ok && p.err == nil {
		return policy, nil
	}
	return governed.RoutePolicy{}, governed.ErrNotDispatched
}

type selectionChannelSecrets struct{ secret credential.Secret }

func (f selectionChannelSecrets) Resolve(_ context.Context, org, id, reference string) (credential.Secret, error) {
	if f.secret.Metadata.OrganizationID != org || f.secret.Metadata.ConnectionID != id || f.secret.Metadata.Reference != reference {
		return credential.Secret{}, credential.ErrChanged
	}
	return f.secret, nil
}

func selectionTextWiringFixture(t *testing.T) (*productAgentApplication, sqlmock.Sqlmock, httpModelFixture, *selectionCurrentTextPolicies) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, sqlDB.Close())
	})
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	configuration, err := configstore.New(db)
	require.NoError(t, err)
	store, err := runstore.New(db)
	require.NoError(t, err)
	current := &selectionCurrentTextPolicies{policies: map[governed.RouteKey]governed.RoutePolicy{}}
	parent := &productAgentApplication{configuration: configuration, store: store, textAdmission: governed.NewBoundedAdmission(), config: ProductAgentDependencies{RunDB: db, CurrentTextPolicies: current, TextTransport: appHTTPRoundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("configuration reads must never dispatch a provider request")
		return nil, errors.New("unexpected dispatch")
	})}}
	row := httpModelFixture{TenantID: "fixture-org", ClientName: "shared-text", APIKey: "synthetic-only", BaseURL: "https://provider.example.test/v1", Model: "deepseek-v4.1-flash", APIStyle: "openai-compatible", TimeoutSecond: 5, Enabled: true}
	metadata := (credential.Metadata{OrganizationID: row.TenantID, ConnectionID: row.ClientName, Provider: "DASHSCOPE", Channel: "fixture-channel", Account: "synthetic", Region: "global", BaseURL: row.BaseURL, Revision: 1, Enabled: true, KeyPresent: true}).WithIdentity()
	row.reference = metadata.Reference
	parent.config.ChannelCredentials = selectionChannelSecrets{credential.Secret{Metadata: metadata, APIKey: row.APIKey}}
	t.Cleanup(func() { require.NoError(t, mock.ExpectationsWereMet()) })
	return parent, mock, row, current
}

func selectionTextWiringPolicy(row httpModelFixture, researchOperation bool) governed.RoutePolicy {
	prompt, schema := agentconfig.SelectionPromptVersion, agentconfig.SelectionActionVersion
	if researchOperation {
		prompt, schema = agentconfig.ResearchPromptVersion, agentconfig.ResearchActionVersion
	}
	return governed.RoutePolicy{AdmittedCredentialVersion: row.reference, AdmittedEndpointIdentityDigest: governed.EndpointIdentityDigest(row.BaseURL), Profile: aicapability.ModelProfile{
		ConnectionID: row.ClientName, ClientName: row.ClientName, ModelID: row.Model, ProviderID: "DASHSCOPE", AdapterKind: "openai-compatible",
		RoutingPolicyVersion: "fixture-route-v1", AdapterPolicyVersion: governed.DashScopeDeepSeekV41FlashNonThinkingPolicy, PromptVersion: prompt, OutputSchemaVersion: schema,
		UsageMappingVersion: "fixture-usage-v1", CostPricingVersion: "fixture-cost-v1", Currency: "CNY", InputMicrosPerMillion: 2000000, OutputMicrosPerMillion: 8000000,
		// A synthetic frozen official-price snapshot, never a published price.
		OfficialTariff:      aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "fixture-only", FXVersion: "fixture-fx-identity", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: row.Model, Channel: "fixture-channel", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 2000000, OutputText: 8000000}},
		MaximumPromptTokens: 32000, MaximumCompletionTokens: 4000, MaximumInputBytes: 32 << 10, MaximumOutputBytes: 16 << 10, DeadlineBound: 5 * time.Second,
	}}
}

func buildSelectionTextConsumer(t *testing.T, parent *productAgentApplication, researchOperation bool) (*texteino.AgentTextModel, func(context.Context, string) (aicapability.ModelProfile, error)) {
	t.Helper()
	auth, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	events := []string{}
	resolver := mountedOrganizationResolverStub{events: &events}
	if researchOperation {
		app, err := buildMarketResearchApplication(context.Background(), parent, &collection.Service{}, &research.Service{}, nil, resolver, auth)
		require.NoError(t, err)
		if app.model != nil {
			require.NotNil(t, app.runtime)
		}
		return app.model, app.profile
	}
	app, err := buildProductSelectionApplication(context.Background(), parent, &collection.Service{}, nil, resolver, auth)
	require.NoError(t, err)
	if app.model != nil {
		require.NotNil(t, app.runtime)
	}
	return app.model, app.profile
}

func TestResearchAndSelectionUseCurrentTextPoliciesWithoutStaticMaps(t *testing.T) {
	parent, _, row, current := selectionTextWiringFixture(t)
	for _, researchOperation := range []bool{true, false} {
		operation := aicapability.OperationProductSelectionDecision
		if researchOperation {
			operation = aicapability.OperationProductResearchDecision
		}
		t.Run(string(operation), func(t *testing.T) {
			key := governed.RouteKey{OrganizationID: row.TenantID, Operation: operation}
			policy := selectionTextWiringPolicy(row, researchOperation)
			current.policies[key] = policy
			model, profile := buildSelectionTextConsumer(t, parent, researchOperation)
			require.NotNil(t, model, "an injected current reader must not require a second static policy")
			require.NotNil(t, profile)
			actual, err := profile(context.Background(), row.TenantID)
			require.NoError(t, err)
			require.Equal(t, policy.ShapeProfile(), actual, "preserve operation prompt/schema and smaller limits")
			ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{UserID: "fixture-actor", TenantID: row.TenantID, EffectiveOrganizationID: row.TenantID, TokenExpiresAt: time.Now().Add(time.Minute)})
			require.Equal(t, texteino.TextRouteAvailable, model.RouteReadinessForVerifiedOrganization(ctx, row.TenantID))
			require.Equal(t, []governed.RouteKey{key, key}, current.keys)
			current.keys = nil
			_, err = profile(context.Background(), "other-org")
			require.ErrorIs(t, err, governed.ErrNotDispatched)
			require.Equal(t, []governed.RouteKey{{OrganizationID: "other-org", Operation: operation}}, current.keys)
			current.keys = nil
		})
	}
}

func TestResearchAndSelectionNeverFallBackWhenCurrentTextPoliciesReject(t *testing.T) {
	for _, researchOperation := range []bool{true, false} {
		for _, missing := range []bool{false, true} {
			parent, _, row, current := selectionTextWiringFixture(t)
			policy := selectionTextWiringPolicy(row, researchOperation)
			operation := aicapability.OperationProductSelectionDecision
			parent.config.SelectionTextPolicies = map[string]governed.RoutePolicy{row.TenantID: policy}
			if researchOperation {
				operation = aicapability.OperationProductResearchDecision
				parent.config.ResearchTextPolicies = map[string]governed.RoutePolicy{row.TenantID: policy}
			}
			key := governed.RouteKey{OrganizationID: row.TenantID, Operation: operation}
			if !missing {
				current.policies[key], current.err = policy, errors.New("current configuration unavailable")
			}
			model, profile := buildSelectionTextConsumer(t, parent, researchOperation)
			require.NotNil(t, model)
			_, err := profile(context.Background(), row.TenantID)
			require.ErrorIs(t, err, governed.ErrNotDispatched)
			ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{UserID: "fixture-actor", TenantID: row.TenantID, EffectiveOrganizationID: row.TenantID, TokenExpiresAt: time.Now().Add(time.Minute)})
			require.Equal(t, texteino.TextRouteUnavailable, model.RouteReadinessForVerifiedOrganization(ctx, row.TenantID))
			require.Equal(t, []governed.RouteKey{key, key}, current.keys)
		}
	}
}

func TestResearchAndSelectionKeepFactsAvailableWithoutTextPolicies(t *testing.T) {
	parent, _, _, _ := selectionTextWiringFixture(t)
	parent.config.CurrentTextPolicies = nil
	for _, researchOperation := range []bool{true, false} {
		model, profile := buildSelectionTextConsumer(t, parent, researchOperation)
		require.Nil(t, model)
		require.Nil(t, profile)
	}
}

func TestResearchAndSelectionShareCategoryAndFrozenOfficialPrice(t *testing.T) {
	parent, _, row, _ := selectionTextWiringFixture(t)
	selectionPolicy, researchPolicy := selectionTextWiringPolicy(row, false), selectionTextWiringPolicy(row, true)
	shared := ProductAgentDependencies{
		ResearchTextPolicies:  map[string]governed.RoutePolicy{row.TenantID: researchPolicy},
		SelectionTextPolicies: map[string]governed.RoutePolicy{row.TenantID: selectionPolicy},
	}
	parent.config.CurrentTextPolicies = qualified.SharedTextReader{Service: syntheticSelectionSharedCategory{settings: &shared}}
	var profiles []aicapability.ModelProfile
	for _, researchOperation := range []bool{true, false} {
		model, profile := buildSelectionTextConsumer(t, parent, researchOperation)
		require.NotNil(t, model)
		p, err := profile(context.Background(), row.TenantID)
		require.NoError(t, err)
		profiles = append(profiles, p)
		retail, err := p.OfficialTariff.ReserveTextMicropoints(p.MaximumPromptTokens, p.MaximumCompletionTokens)
		require.NoError(t, err)
		require.EqualValues(t, 19200000, 2*retail)
		cost, err := p.MaximumCost()
		require.NoError(t, err)
		require.LessOrEqual(t, 2*cost, int64(300000))
	}
	require.Equal(t, agentconfig.ResearchPromptVersion, profiles[0].PromptVersion)
	require.Equal(t, agentconfig.SelectionPromptVersion, profiles[1].PromptVersion)
	require.Equal(t, agentconfig.ResearchActionVersion, profiles[0].OutputSchemaVersion)
	require.Equal(t, agentconfig.SelectionActionVersion, profiles[1].OutputSchemaVersion)
	require.Equal(t, profiles[0].ClientName, profiles[1].ClientName)
	require.Equal(t, profiles[0].CredentialVersion, profiles[1].CredentialVersion)
	require.Equal(t, profiles[0].RoutingPolicyVersion, profiles[1].RoutingPolicyVersion)
	require.Equal(t, profiles[0].OfficialTariff, profiles[1].OfficialTariff)
	require.Equal(t, aicapability.AIPointMicrounitV1, profiles[0].OfficialTariff.UnitVersion)
	shared.ResearchTextPolicies, shared.SelectionTextPolicies = nil, nil
	for _, researchOperation := range []bool{true, false} {
		_, profile := buildSelectionTextConsumer(t, parent, researchOperation)
		_, err := profile(context.Background(), row.TenantID)
		require.ErrorIs(t, err, governed.ErrNotDispatched, "a missing shared category must remain unavailable")
	}
}
