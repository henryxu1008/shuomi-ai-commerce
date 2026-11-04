package routeconfiguration

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
	channel "task-processor/internal/aicapability/channelcredential"
	d "task-processor/internal/aicapability/routeconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"testing"
	"time"
)

type sharedMetadataFixture struct {
	d.Repository
	credential d.Credential
	prices     d.PlatformPricesSnapshot
	reads      int
}

func (f *sharedMetadataFixture) Credential(_ context.Context, org, id string) (d.Credential, error) {
	f.reads++
	if id == "" {
		return d.Credential{}, d.ErrUnavailable
	}
	if f.credential.OrganizationID != org || f.credential.ConnectionID != id {
		return d.Credential{}, d.ErrNotFound
	}
	return f.credential, nil
}
func (f *sharedMetadataFixture) Current(_ context.Context, scope d.Scope) (d.Revision, error) {
	if scope != d.PlatformPricesScope() {
		return d.Revision{}, d.ErrNotFound
	}
	return d.Revision{ID: "prices", Scope: scope, SchemaVersion: 1, Payload: d.Payload{PlatformPrices: &f.prices}}, nil
}
func sharedCatalogImageFixture(t *testing.T) (*SharedCatalog, *sharedMetadataFixture, d.ModelBindingReference) {
	now := time.Date(2026, 11, 3, 8, 0, 0, 0, time.UTC)
	endpoint := "https://workspace.cn-beijing.maas.aliyuncs.com"
	ref := d.ModelBindingReference{ID: "synthetic-image", Version: "synthetic-binding"}
	input := d.PlatformPricesInput{Prices: []d.PublishedModelPrice{{ID: "synthetic-price", Region: "cn-beijing", ServiceType: "image-sync", Image: &d.OfficialImagePriceScope{Width: 1024, Height: 1024, Quality: "auto"}, Provider: "qwen", Model: "qwen-image-3.0", Channel: "bailian-beijing", Currency: "CNY", UsageVersion: "qwen-image-count-v1", Rates: aicapability.OfficialModelRates{OutputImagePerItem: 200000}, SourceURL: "https://help.aliyun.com/zh/model-studio/", VerifiedAt: now, EffectiveFrom: now, EffectiveUntil: now.Add(time.Hour)}}}
	prices, err := d.FreezePlatformPrices(input, "price-publication", nil, now)
	require.NoError(t, err)
	metadata := &sharedMetadataFixture{prices: prices, credential: d.Credential{OrganizationID: "org", EndpointDigest: governed.EndpointIdentityDigest(endpoint), Reference: "synthetic-reference", Enabled: true, KeyPresent: true}}
	def := Definition{Manufacturer: "qwen", Region: "cn-beijing", ServiceType: "image-sync", Scope: d.Scope{OrganizationID: "org", Capability: d.ImageCapability}, Channel: "bailian-beijing", PriceID: "synthetic-price", ReferenceVersion: ref.Version, BoundEvidenceVersion: "synthetic-count-bound", Image: &ImageAdmission{UnitVersion: "ai-micropoint-v1", MaximumMicropoints: 100000000, ClientName: ref.ID, Provider: "bailian", Model: "qwen-image-3.0", Endpoint: endpoint, CredentialVersion: "synthetic-credential", Billing: aicapability.OfficialModelBilling{UsageBound: aicapability.BillableModelUsage{MappingVersion: "qwen-image-count-v1", Complete: true, OutputImageItems: 1}}}}
	catalog, err := NewCatalog([]Definition{def})
	require.NoError(t, err)
	m := (channel.Metadata{OrganizationID: "org", ConnectionID: "shared-image-connection", Provider: "bailian", Channel: "bailian-beijing", Region: "cn-beijing", Account: "fixture", BaseURL: endpoint, Revision: 1, Enabled: true, KeyPresent: true}).WithIdentity()
	metadata.credential.ConnectionID, metadata.credential.Provider, metadata.credential.Channel, metadata.credential.Region = m.ConnectionID, m.Provider, m.Channel, m.Region
	metadata.credential.Reference = m.Reference
	for scope, bindings := range catalog.definitions {
		for id, definition := range bindings {
			definition.ConnectionID = m.ConnectionID
			definition.ConnectionProvider = m.Provider
			definition.Image.DeadlineSeconds = 30
			catalog.definitions[scope][id] = definition
		}
	}
	return &SharedCatalog{Catalog: catalog, Credentials: metadata, Prices: &d.PlatformPricesService{Repository: metadata, Now: func() time.Time { return now }}, Now: func() time.Time { return now }}, metadata, ref
}
func TestSharedImageAdmissionUsesPublishedOfficialPriceAndOriginalCredentialOwner(t *testing.T) {
	catalog, metadata, ref := sharedCatalogImageFixture(t)
	q, err := catalog.QualifyModel(context.Background(), "org", d.ModelCategoryImage, ref)
	require.NoError(t, err)
	require.True(t, q.Ready, q.Reason)
	require.EqualValues(t, 20000000, q.Image.Points)
	require.Equal(t, q.Price, q.Image.Billing.OfficialTariff)
	require.NotEmpty(t, q.ProfileVersion)
	metadata.prices.Prices[0].Rates.OutputImagePerItem = 250000
	metadata.prices.Prices[0].Version = "new-synthetic-price"
	next, err := catalog.QualifyModel(context.Background(), "org", d.ModelCategoryImage, ref)
	require.NoError(t, err)
	require.True(t, next.Ready, next.Reason)
	require.EqualValues(t, 25000000, next.Image.Points)
	require.NotEqual(t, q.ProfileVersion, next.ProfileVersion)
	metadata.credential.Provider = "wrong-channel-provider"
	revoked, err := catalog.QualifyModel(context.Background(), "org", d.ModelCategoryImage, ref)
	require.NoError(t, err)
	require.False(t, revoked.Ready)
	require.Equal(t, "CREDENTIAL_NOT_ADMITTED", revoked.Reason)
}
func TestSharedImageAdmissionCannotBorrowTextCredentialsOrMissingBillingBound(t *testing.T) {
	catalog, _, ref := sharedCatalogImageFixture(t)
	catalog.Credentials = nil
	q, err := catalog.QualifyModel(context.Background(), "org", d.ModelCategoryImage, ref)
	require.NoError(t, err)
	require.False(t, q.Ready)
	require.Equal(t, "CREDENTIAL_READER_MISSING", q.Reason)
	catalog, _, ref = sharedCatalogImageFixture(t)
	for scope, defs := range catalog.Catalog.definitions {
		def := defs[ref.ID]
		def.Image.Billing.UsageBound.Complete = false
		catalog.Catalog.definitions[scope][ref.ID] = def
	}
	q, err = catalog.QualifyModel(context.Background(), "org", d.ModelCategoryImage, ref)
	require.NoError(t, err)
	require.False(t, q.Ready)
	require.Equal(t, "COST_BOUND_MISSING", q.Reason)
}
func TestSharedDefaultFlareAndVisionRemainVisibleWithoutFalseReadiness(t *testing.T) {
	catalog, _, _ := sharedCatalogImageFixture(t)
	rows, err := catalog.Options(context.Background(), "org")
	require.NoError(t, err)
	found := false
	for _, row := range rows {
		if row.Reference.ID == "openai-flare" {
			found = true
			require.False(t, row.Ready)
			require.Equal(t, "gpt-image-2.5-flare-2026-09-08", row.Model)
			require.Equal(t, "PROTOCOL_AND_COST_BOUND_MISSING", row.Reason)
		}
	}
	require.True(t, found)
}
func TestTextAdmissionDescriptorDoesNotRequireOrAuthorizeItsOwnCustomerTariff(t *testing.T) {
	profile := aicapability.ModelProfile{ClientName: "synthetic", ProviderID: "synthetic", AdapterKind: "openai-compatible", ModelID: "synthetic-mini", EndpointIdentityDigest: "endpoint", CredentialVersion: "credential", RoutingPolicyVersion: "admission", AdapterPolicyVersion: "adapter", PromptVersion: "prompt", OutputSchemaVersion: "schema", UsageMappingVersion: "usage", CostPricingVersion: "channel-cost", Currency: "CNY", InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1, MaximumPromptTokens: 100, MaximumCompletionTokens: 100, MaximumInputBytes: 1024, MaximumOutputBytes: 1024, DeadlineBound: time.Second}
	require.NoError(t, profile.ValidateShape())
	require.Error(t, profile.Validate(), "an admission shape is not an execution or charge permission")
	catalog, err := NewCatalog([]Definition{{Scope: d.Scope{OrganizationID: "org", Capability: string(aicapability.OperationProductAgentDecision)}, Text: &governed.RoutePolicy{Profile: profile, AdmittedCredentialVersion: "credential", AdmittedEndpointIdentityDigest: "endpoint"}, MaximumCostMicros: 100}})
	require.NoError(t, err)
	require.NotNil(t, catalog)
}

func TestRemovedBindingRemainsAnExplicitUnreadyReadInsteadOfDisappearingCurrent(t *testing.T) {
	catalog, _, _ := sharedCatalogImageFixture(t)
	q, err := catalog.QualifyModel(context.Background(), "org", d.ModelCategoryText, d.ModelBindingReference{ID: "removed-binding", Version: "retained-v1"})
	require.NoError(t, err)
	require.False(t, q.Ready)
	require.Equal(t, "UNKNOWN", q.Model)
	require.Equal(t, "removed-binding", q.Reference.ID)
	require.Equal(t, "BINDING_NOT_ADMITTED", q.Reason)
}

func TestOfficialPriceCannotBorrowAnotherManufacturerOrImageScope(t *testing.T) {
	for _, kind := range []string{"manufacturer", "region", "service", "image-size"} {
		t.Run(kind, func(t *testing.T) {
			catalog, metadata, ref := sharedCatalogImageFixture(t)
			price := &metadata.prices.Prices[0]
			switch kind {
			case "manufacturer":
				price.Provider = "openai"
				price.SourceURL = "https://developers.openai.com/api/docs/pricing"
			case "region":
				price.Region = "other-region"
			case "service":
				price.ServiceType = "other-service"
			case "image-size":
				price.Image.Width = 512
			}
			q, err := catalog.QualifyModel(context.Background(), "org", d.ModelCategoryImage, ref)
			require.NoError(t, err)
			require.False(t, q.Ready)
			require.Equal(t, "PRICE_APPLICABILITY_MISSING", q.Reason)
		})
	}
}

func TestSharedTextAdmissionRequiresAnImplementedOfficialUsageMapper(t *testing.T) {
	for _, tc := range []struct {
		adapter, mapping string
		ready            bool
	}{{"openai-compatible", "openai-text-v1", true}, {"openai-compatible", "unimplemented-text-v1", false}, {"claude-native", "openai-text-v1", false}, {"google-interactions", "openai-text-v1", false}} {
		t.Run(tc.adapter+"/"+tc.mapping, func(t *testing.T) {
			now := time.Date(2026, 11, 3, 8, 0, 0, 0, time.UTC)
			ref := d.ModelBindingReference{ID: "synthetic-text", Version: "synthetic-binding"}
			p := aicapability.ModelProfile{ClientName: ref.ID, ProviderID: "synthetic", AdapterKind: tc.adapter, ModelID: "synthetic-mini", EndpointIdentityDigest: "synthetic-endpoint", CredentialVersion: "credential", RoutingPolicyVersion: "admission", AdapterPolicyVersion: "adapter", PromptVersion: "prompt", OutputSchemaVersion: "schema", UsageMappingVersion: "cost-usage", CostPricingVersion: "channel-cost", Currency: "CNY", InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1, MaximumPromptTokens: 100, MaximumCompletionTokens: 100, MaximumInputBytes: 1024, MaximumOutputBytes: 1024, DeadlineBound: time.Second}
			manufacturer, source := "openai", "https://developers.openai.com/api/docs/pricing"
			if tc.adapter == "google-interactions" {
				p.ProviderID, p.ModelID = "google", "gemini-3.8-flash"
				p.AdapterPolicyVersion, p.UsageMappingVersion = "google-interactions-v1", "google-interactions-usage-v1"
				manufacturer, source = "google", "https://ai.google.dev/gemini-api/docs/pricing"
			}
			prices, err := d.FreezePlatformPrices(d.PlatformPricesInput{Prices: []d.PublishedModelPrice{{ID: "synthetic-price", Region: "global", ServiceType: "standard-sync", Provider: manufacturer, Channel: "synthetic-channel", Model: p.ModelID, Currency: "CNY", UsageVersion: tc.mapping, Rates: aicapability.OfficialModelRates{InputText: 1000000, OutputText: 1000000}, SourceURL: source, VerifiedAt: now, EffectiveFrom: now, EffectiveUntil: now.Add(time.Hour)}}}, "synthetic-publication", nil, now)
			require.NoError(t, err)
			metadata := &sharedMetadataFixture{prices: prices, credential: d.Credential{OrganizationID: "org", EndpointDigest: p.EndpointIdentityDigest, Reference: "synthetic-reference", Enabled: true, KeyPresent: true}}
			catalog, err := NewCatalog([]Definition{{Manufacturer: manufacturer, Region: "global", ServiceType: "standard-sync", Channel: "synthetic-channel", PriceID: "synthetic-price", ReferenceVersion: ref.Version, BoundEvidenceVersion: "synthetic-bound", Scope: d.Scope{OrganizationID: "org", Capability: string(aicapability.OperationProductAgentDecision)}, Text: &governed.RoutePolicy{Profile: p, AdmittedCredentialVersion: "credential", AdmittedEndpointIdentityDigest: p.EndpointIdentityDigest}, MaximumCostMicros: 100}})
			require.NoError(t, err)
			bindTestChannel(catalog, &metadata.credential)
			shared := &SharedCatalog{Catalog: catalog, Credentials: metadata, Prices: &d.PlatformPricesService{Repository: metadata, Now: func() time.Time { return now }}, Now: func() time.Time { return now }}
			q, err := shared.QualifyModel(context.Background(), "org", d.ModelCategoryText, ref)
			require.NoError(t, err)
			require.Equal(t, tc.ready, q.Ready, q.Reason)
			if !tc.ready {
				require.Equal(t, "METERING_MAPPING_NOT_ADMITTED", q.Reason)
			} else {
				require.Equal(t, q.Price, q.TextProfiles[string(aicapability.OperationProductAgentDecision)].OfficialTariff)
			}
		})
	}
}

func TestDeepSeekReferenceAdmissionUsesFrozenReferenceAndCurrentCredentials(t *testing.T) {
	now := time.Date(2026, 11, 3, 8, 0, 0, 0, time.UTC)
	ref := d.ModelBindingReference{ID: "synthetic-deepseek", Version: "synthetic-binding"}
	p := aicapability.ModelProfile{ClientName: ref.ID, ProviderID: "DASHSCOPE", AdapterKind: "openai-compatible", ModelID: "deepseek-v4.1-flash", EndpointIdentityDigest: "synthetic-endpoint", CredentialVersion: "credential", RoutingPolicyVersion: "admission", AdapterPolicyVersion: governed.DashScopeDeepSeekV41FlashNonThinkingPolicy, PromptVersion: "prompt", OutputSchemaVersion: "schema", UsageMappingVersion: "cost-usage", CostPricingVersion: "channel-cost", Currency: "CNY", InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1, MaximumPromptTokens: 100, MaximumCompletionTokens: 100, MaximumInputBytes: 1024, MaximumOutputBytes: 1024, DeadlineBound: time.Second}
	def := Definition{Manufacturer: "deepseek", Region: "cn-beijing", ServiceType: "standard-sync", Channel: "bailian-beijing", PriceID: "synthetic-price", ReferenceVersion: ref.Version, Scope: d.Scope{OrganizationID: "org", Capability: string(aicapability.OperationProductResearchDecision)}, Text: &governed.RoutePolicy{Profile: p, AdmittedCredentialVersion: "credential", AdmittedEndpointIdentityDigest: p.EndpointIdentityDigest}, MaximumCostMicros: 100, MaximumMicropoints: 100000}
	catalog, err := NewCatalog([]Definition{def})
	require.NoError(t, err)
	prices, err := d.FreezePlatformPrices(d.PlatformPricesInput{Prices: []d.PublishedModelPrice{{ID: def.PriceID, Provider: def.Manufacturer, Model: p.ModelID, Region: def.Region, ServiceType: def.ServiceType, Channel: def.Channel, Currency: "CNY", UsageVersion: aicapability.ReferenceTextTotalsV1, Rates: aicapability.OfficialModelRates{InputText: 2_000_000, OutputText: 8_000_000}, SourceURL: "https://api-docs.deepseek.com/", VerifiedAt: now, EffectiveFrom: now, EffectiveUntil: now.Add(time.Hour)}}}, "reference-v1", nil, now)
	require.NoError(t, err)
	metadata := &sharedMetadataFixture{prices: prices, credential: d.Credential{OrganizationID: "org", EndpointDigest: p.EndpointIdentityDigest, Reference: "synthetic-only", Enabled: true, KeyPresent: true}}
	bindTestChannel(catalog, &metadata.credential)
	shared := &SharedCatalog{Catalog: catalog, Credentials: metadata, Prices: &d.PlatformPricesService{Repository: metadata, Now: func() time.Time { return now }}, Now: func() time.Time { return now }}
	q, err := shared.QualifyModel(context.Background(), "org", d.ModelCategoryText, ref)
	require.NoError(t, err)
	require.True(t, q.Ready, q.Reason)
	require.True(t, q.Price.IsReferenceEstimate())
	require.Equal(t, prices.Prices[0].Version, q.Price.PriceVersion)
	require.Empty(t, q.BoundEvidenceVersion, "a reference strategy does not invent supplier bound evidence")
	// The same channel-cost profile must not borrow a larger retail budget
	// from a changed reference rate. Metadata is the existing Definition.
	for scope, definitions := range catalog.definitions {
		entry := definitions[ref.ID]
		entry.MaximumMicropoints = 0
		definitions[ref.ID] = entry
		q, err = shared.QualifyModel(context.Background(), "org", d.ModelCategoryText, ref)
		require.NoError(t, err)
		require.False(t, q.Ready)
		require.Equal(t, "POINT_BUDGET_MISSING", q.Reason)
		require.NoError(t, json.Unmarshal([]byte(`{"maximumMicropoints":100000}`), &entry))
		definitions[ref.ID] = entry
		q, err = shared.QualifyModel(context.Background(), "org", d.ModelCategoryText, ref)
		require.NoError(t, err)
		require.True(t, q.Ready, "equal to the configured point budget is admissible")
		before := q.ProfileVersion
		entry.MaximumCostMicros = 100 // channel estimate remains unchanged
		require.NoError(t, json.Unmarshal([]byte(`{"maximumMicropoints":99999}`), &entry))
		catalog.definitions[scope][ref.ID] = entry
		q, err = shared.QualifyModel(context.Background(), "org", d.ModelCategoryText, ref)
		require.NoError(t, err)
		require.False(t, q.Ready, "retail point cap is independent of channel cost")
		require.Equal(t, "POINT_BUDGET_EXCEEDED", q.Reason)
		require.NoError(t, json.Unmarshal([]byte(`{"maximumMicropoints":100001}`), &entry))
		catalog.definitions[scope][ref.ID] = entry
		q, err = shared.QualifyModel(context.Background(), "org", d.ModelCategoryText, ref)
		require.NoError(t, err)
		require.True(t, q.Ready)
		require.NotEqual(t, before, q.ProfileVersion, "budget changes invalidate old profile confirmation")
		beforePrice := q.ProfileVersion
		metadata.prices.Prices[0].Rates.InputText *= 2
		metadata.prices.Prices[0].Rates.OutputText *= 2
		metadata.prices.Prices[0].Version = "higher-reference-version"
		q, err = shared.QualifyModel(context.Background(), "org", d.ModelCategoryText, ref)
		require.NoError(t, err)
		require.False(t, q.Ready, "a reference change cannot raise the operation's point budget")
		require.Equal(t, "POINT_BUDGET_EXCEEDED", q.Reason)
		metadata.prices.Prices[0].Rates.InputText /= 2
		metadata.prices.Prices[0].Rates.OutputText /= 2
		q, err = shared.QualifyModel(context.Background(), "org", d.ModelCategoryText, ref)
		require.NoError(t, err)
		require.True(t, q.Ready)
		require.NotEqual(t, beforePrice, q.ProfileVersion, "reference versions are included in confirmation identity")
	}
	metadata.credential.Enabled = false
	q, err = shared.QualifyModel(context.Background(), "org", d.ModelCategoryText, ref)
	require.NoError(t, err)
	require.False(t, q.Ready)
	require.Equal(t, "CREDENTIAL_NOT_ADMITTED", q.Reason)
}

// Explicit fixture admission: models remain in their definitions, the shared
// credential supplies only connection identity and endpoint metadata.
func bindTestChannel(c *Catalog, row *d.Credential) {
	for scope, defs := range c.definitions {
		for id, def := range defs {
			if scope.OrganizationID != row.OrganizationID {
				continue
			}
			def.ConnectionID = "fixture-connection"
			def.ConnectionProvider = "fixture-provider"
			row.ConnectionID = def.ConnectionID
			row.Provider = def.ConnectionProvider
			row.Channel = def.Channel
			row.Region = def.Region
			row.Reference = "channel-credential-v1:fixture-revision-1"
			if def.Image != nil {
				def.Image.DeadlineSeconds = 30
			}
			defs[id] = def
		}
	}
}
