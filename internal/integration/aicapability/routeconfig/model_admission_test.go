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

func TestEinoSharedCatalogNativeReferencePriceUsesTheSameQualificationOwner(t *testing.T) {
	now := time.Date(2026, 11, 4, 8, 0, 0, 0, time.UTC)
	for _, tc := range []struct{ adapter, provider, manufacturer, policy, usage, model string }{
		{"google-generate-content", "google", "google", "eino-gemini-v0.1.36-text-v1", "gemini-generate-content-usage-v1", "synthetic-gemini"},
		{"qwen-compatible", "DASHSCOPE", "qwen", "eino-qwen-v0.1.9-text-v1", "qwen-text-v1", "synthetic-qwen"},
		{"claude-native", "anthropic", "anthropic", "eino-claude-v0.1.26-text-v1", "claude-text-v1", "synthetic-claude"},
	} {
		t.Run(tc.adapter, func(t *testing.T) {
			ref := d.ModelBindingReference{ID: "configured-model", Version: "fixture-v1"}
			p := aicapability.ModelProfile{ClientName: ref.ID, ProviderID: tc.provider, AdapterKind: tc.adapter, ModelID: tc.model,
				EndpointIdentityDigest: "fixture-endpoint", CredentialVersion: "fixture", RoutingPolicyVersion: "fixture",
				AdapterPolicyVersion: tc.policy, PromptVersion: "prompt", OutputSchemaVersion: "schema", UsageMappingVersion: tc.usage,
				CostPricingVersion: "cost", Currency: "CNY", InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1,
				MaximumPromptTokens: 10000, MaximumCompletionTokens: 64, MaximumInputBytes: 8192, MaximumOutputBytes: 8192, DeadlineBound: time.Second}
			if tc.adapter == "google-generate-content" {
				p.ThinkingMode = "low"
			}
			def := Definition{Manufacturer: tc.manufacturer, Channel: "configured-channel", Region: "global", ServiceType: "standard-sync",
				PriceID: "fixture-price", ReferenceVersion: ref.Version, MaximumCostMicros: 100,
				Scope: d.Scope{OrganizationID: "org", Capability: string(aicapability.OperationProductAgentDecision)},
				Text:  &governed.RoutePolicy{Profile: p, AdmittedEndpointIdentityDigest: p.EndpointIdentityDigest}}
			catalog, err := NewCatalog([]Definition{def})
			require.NoError(t, err)
			prices, err := d.FreezePlatformPrices(d.PlatformPricesInput{Prices: []d.PublishedModelPrice{{
				ID: def.PriceID, Provider: tc.manufacturer, Model: tc.model, Channel: def.Channel, Region: def.Region, ServiceType: def.ServiceType,
				Currency: "CNY", UsageVersion: aicapability.ReferenceTextTotalsV1,
				Rates:     aicapability.OfficialModelRates{InputText: 10000, OutputText: 20000},
				SourceURL: map[string]string{"google": "https://ai.google.dev/gemini-api/docs/pricing", "qwen": "https://help.aliyun.com/zh/model-studio/billing-for-model-studio", "anthropic": "https://www.anthropic.com/pricing"}[tc.manufacturer], VerifiedAt: now, EffectiveFrom: now, EffectiveUntil: now.Add(time.Hour)}}}, "fixture-publication", nil, now)
			require.NoError(t, err)
			metadata := &sharedMetadataFixture{prices: prices, credential: d.Credential{OrganizationID: "org", EndpointDigest: p.EndpointIdentityDigest, Enabled: true, KeyPresent: true}}
			bindTestChannel(catalog, &metadata.credential)
			shared := &SharedCatalog{Catalog: catalog, Credentials: metadata, Prices: &d.PlatformPricesService{Repository: metadata, Now: func() time.Time { return now }}, Now: func() time.Time { return now }}
			q, err := shared.QualifyModel(context.Background(), "org", d.ModelCategoryText, ref)
			require.NoError(t, err)
			require.True(t, q.Ready, q.Reason)
			require.True(t, q.Text.OfficialTariff.IsReferenceEstimate())
			require.Equal(t, tc.model, q.Text.ModelID)
			require.Equal(t, metadata.credential.Reference, q.Text.CredentialVersion)
			scope := def.Scope
			original := catalog.definitions[scope][ref.ID]
			alias := original
			route := *original.Text
			alias.Text = &route
			alias.Text.Profile.ModelID = "vendor/" + tc.model
			if tc.adapter == "google-generate-content" {
				alias.Text.Profile.ModelID = "channel-" + tc.model
			}
			catalog.definitions[scope][ref.ID] = alias
			missing, err := shared.QualifyModel(context.Background(), "org", d.ModelCategoryText, ref)
			require.NoError(t, err)
			require.False(t, missing.Ready, "never guess or strip a channel model prefix to borrow a price")
			alias.ManufacturerModel = tc.model
			catalog.definitions[scope][ref.ID] = alias
			qualified, err := shared.QualifyModel(context.Background(), "org", d.ModelCategoryText, ref)
			require.NoError(t, err)
			require.True(t, qualified.Ready, qualified.Reason)
			require.Equal(t, tc.model, qualified.Price.Model)
			require.Equal(t, alias.Text.Profile.ModelID, qualified.Price.WireModel)
			require.Equal(t, qualified.Price, qualified.Text.OfficialTariff)
			catalog.definitions[scope][ref.ID] = original

			// The same selected model must cover every opened consumer, while
			// keeping each operation's own prompt/output contract.
			other := original
			other.Scope.Capability = string(aicapability.OperationProductContentDecision)
			otherRoute := *original.Text
			other.Text = &otherRoute
			other.Text.Profile.ClientName = "another-reference"
			other.Text.Profile.PromptVersion = "content-prompt"
			other.Text.Profile.OutputSchemaVersion = "content-output"
			complete, err := NewCatalog([]Definition{original, other})
			require.NoError(t, err)
			candidate := *shared
			candidate.Catalog = complete
			missing, err = candidate.QualifyModel(context.Background(), "org", d.ModelCategoryText, ref)
			require.NoError(t, err)
			require.False(t, missing.Ready)
			require.Equal(t, "OPEN_CONSUMER_MISSING", missing.Reason)
			delete(complete.definitions[other.Scope], "another-reference")
			other.Text.Profile.ClientName = ref.ID
			complete.definitions[other.Scope][ref.ID] = other
			qualified, err = candidate.QualifyModel(context.Background(), "org", d.ModelCategoryText, ref)
			require.NoError(t, err)
			require.True(t, qualified.Ready, qualified.Reason)
			require.Len(t, qualified.TextProfiles, 2)
			require.Equal(t, "content-output", qualified.TextProfiles[other.Scope.Capability].OutputSchemaVersion)
			require.NotEqual(t, q.ProfileVersion, qualified.ProfileVersion)
			metadata.credential.OrganizationID = "other-org"
			q, err = shared.QualifyModel(context.Background(), "org", d.ModelCategoryText, ref)
			require.NoError(t, err)
			require.False(t, q.Ready, "shared model support must never borrow another enterprise's connection")
		})
	}
}
