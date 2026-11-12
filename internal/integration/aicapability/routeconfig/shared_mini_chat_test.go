package routeconfiguration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
	d "task-processor/internal/aicapability/routeconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
)

func TestMiniChatAndContentQualifyTogetherWithOnePublishedPrice(t *testing.T) {
	now := time.Date(2026, 11, 11, 8, 0, 0, 0, time.UTC)
	ref := d.ModelBindingReference{ID: "mini", Version: "fixture-binding"}
	p := aicapability.ModelProfile{ClientName: ref.ID, ProviderID: "openai", AdapterKind: "openai-compatible", ModelID: aicapability.SingleImageMiniModel, EndpointIdentityDigest: governed.EndpointIdentityDigest("https://api.openai.com/v1"), CredentialVersion: "fixture", RoutingPolicyVersion: "admission", AdapterPolicyVersion: governed.OpenAIMiniTextAdapterPolicy, PromptVersion: "prompt", OutputSchemaVersion: "schema", UsageMappingVersion: "openai-text-v1", CostPricingVersion: "channel-cost", Currency: "CNY", InputMicrosPerMillion: 6000000, OutputMicrosPerMillion: 36000000, MaximumPromptTokens: 4096, MaximumCompletionTokens: 512, MaximumInputBytes: 65536, MaximumOutputBytes: 16384, DeadlineBound: time.Second}
	defs := []Definition{}
	for _, op := range []aicapability.Operation{aicapability.OperationAIWorkbenchChatPlan, aicapability.OperationProductContentDecision} {
		profile := p
		if op == aicapability.OperationProductContentDecision {
			profile.MaximumPromptTokens = 16000
			profile.PromptVersion = "content-prompt"
		}
		defs = append(defs, Definition{Manufacturer: "openai", Region: "global", ServiceType: "standard-sync", Channel: "openai-direct", PriceID: "fixture-price", ReferenceVersion: ref.Version, BoundEvidenceVersion: "fixture-bound", Scope: d.Scope{OrganizationID: "org", Capability: string(op)}, Text: &governed.RoutePolicy{Profile: profile, AdmittedCredentialVersion: "fixture", AdmittedEndpointIdentityDigest: p.EndpointIdentityDigest}, MaximumCostMicros: 300000})
	}
	catalog, err := NewCatalog(defs)
	require.NoError(t, err)
	prices, err := d.FreezePlatformPrices(d.PlatformPricesInput{Prices: []d.PublishedModelPrice{{ID: "fixture-price", Provider: "openai", Model: p.ModelID, Channel: "openai-direct", Region: "global", ServiceType: "standard-sync", Currency: "USD", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 750000, CachedInputText: 75000, OutputText: 4500000}, SourceURL: "https://developers.openai.com/api/docs/pricing", VerifiedAt: now, EffectiveFrom: now, EffectiveUntil: now.Add(time.Hour)}}, FX: []d.PublishedFXRate{{Currency: "USD", CNYMicrosPerCurrencyUnit: 6719100, SourceURL: "https://www.chinamoney.com.cn/", RateDate: "2026-11-11", VerifiedAt: now, EffectiveFrom: now, EffectiveUntil: now.Add(time.Hour)}}}, "fixture-publication", nil, now)
	require.NoError(t, err)
	metadata := &sharedMetadataFixture{prices: prices, credential: d.Credential{OrganizationID: "org", EndpointDigest: p.EndpointIdentityDigest, Reference: "fixture", Enabled: true, KeyPresent: true}}
	bindTestChannel(catalog, &metadata.credential)
	shared := &SharedCatalog{Catalog: catalog, Credentials: metadata, Prices: &d.PlatformPricesService{Repository: metadata, Now: func() time.Time { return now }}, Now: func() time.Time { return now }}
	q, err := shared.QualifyModel(context.Background(), "org", d.ModelCategoryText, ref)
	require.NoError(t, err)
	require.True(t, q.Ready, q.Reason)
	require.Len(t, q.TextProfiles, 2)
	for _, op := range []aicapability.Operation{aicapability.OperationAIWorkbenchChatPlan, aicapability.OperationProductContentDecision} {
		profile := q.TextProfiles[string(op)]
		require.True(t, governed.ValidRouteProfile(profile, op))
		require.Equal(t, q.Price, profile.OfficialTariff)
		require.Equal(t, metadata.credential.Reference, profile.CredentialVersion)
	}
	// A changed unified price or FX must not silently exceed the agent's CNY cost bound.
	for scope, bindings := range catalog.definitions {
		def := bindings[ref.ID]
		profile := *def.Text
		profile.Profile.InputMicrosPerMillion = 1
		def.Text = &profile
		catalog.definitions[scope][ref.ID] = def
	}
	q, err = shared.QualifyModel(context.Background(), "org", d.ModelCategoryText, ref)
	require.NoError(t, err)
	require.False(t, q.Ready)
	require.Equal(t, "COST_BOUND_MISSING", q.Reason)
}
