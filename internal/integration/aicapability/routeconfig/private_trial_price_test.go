package routeconfiguration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability"
	d "github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/routeconfig"
	governed "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/aicapability/einomodel"
	"github.com/stretchr/testify/require"
)

func TestPrivateTrialPriceQualifiesOnlyTheLocalContentBinding(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	ref := d.ModelBindingReference{ID: "controlled-title", Version: "controlled-v1"}
	endpoint := "http://127.0.0.1:18086/v1"
	operation := string(aicapability.OperationProductContentDecision)
	profile := aicapability.ModelProfile{ClientName: ref.ID, ProviderID: "grsai", AdapterKind: "openai-compatible", ModelID: "synthetic-title",
		RoutingPolicyVersion: "admission", AdapterPolicyVersion: "adapter-v1", PromptVersion: "prompt", OutputSchemaVersion: "schema",
		UsageMappingVersion: "openai-text-v1", CostPricingVersion: "fixture-cost", Currency: "CNY", InputMicrosPerMillion: 300000,
		OutputMicrosPerMillion: 2000000, MaximumPromptTokens: 16000, MaximumCompletionTokens: 512, MaximumInputBytes: 65536,
		MaximumOutputBytes: 16384, DeadlineBound: time.Second}
	def := Definition{ConnectionID: "controlled", ConnectionProvider: "grsai", ConnectionEndpoint: endpoint,
		Region: "global", Manufacturer: "openai", ServiceType: "standard-sync", Channel: "synthetic", PriceID: "synthetic-price",
		Scope: d.Scope{OrganizationID: "trial", Capability: operation}, ReferenceVersion: ref.Version, BoundEvidenceVersion: "fixture-bound",
		Text:              &governed.RoutePolicy{Profile: profile, AdmittedEndpointIdentityDigest: governed.EndpointIdentityDigest(endpoint)},
		MaximumCostMicros: 300000, MaximumMicropoints: 100000000}
	// Decode the exact manifest field so this regression runs against the unmodified candidate.
	require.NoError(t, json.Unmarshal([]byte(`{"privateTrialPrice":{"notBefore":"2026-11-16T11:59:00Z","notAfter":"2026-11-16T12:30:00Z","tariff":{"unitVersion":"ai-micropoint-v1","priceVersion":"trial-price-v1","fxVersion":"trial-cny","currency":"CNY","cnyMicrosPerCurrencyUnit":1000000,"model":"synthetic-title","channel":"synthetic","usageVersion":"openai-text-v1","rates":{"inputText":10000,"outputText":20000}}}}`), &def))
	meta := &sharedMetadataFixture{credential: d.Credential{OrganizationID: "trial", ConnectionID: "controlled", Provider: "grsai", Region: "global", Channel: "synthetic", EndpointDigest: governed.EndpointIdentityDigest(endpoint), Reference: "channel-credential-v1:synthetic", Enabled: true, KeyPresent: true}}
	catalog, err := NewCatalog([]Definition{def})
	require.NoError(t, err)
	shared := &SharedCatalog{Catalog: catalog, Credentials: meta, Now: func() time.Time { return now }}
	q, err := shared.QualifyModel(context.Background(), "trial", d.ModelCategoryText, ref)
	require.NoError(t, err)
	require.True(t, q.Ready, q.Reason)
	require.Equal(t, "trial-price-v1", q.Price.PriceVersion)
	require.Equal(t, q.Price, q.Text.OfficialTariff)
	rejected, err := shared.QualifyModel(context.Background(), "other-org", d.ModelCategoryText, ref)
	require.NoError(t, err)
	require.False(t, rejected.Ready)
	meta.credential.EndpointDigest = governed.EndpointIdentityDigest("https://outside.example/v1")
	rejected, err = shared.QualifyModel(context.Background(), "trial", d.ModelCategoryText, ref)
	require.NoError(t, err)
	require.False(t, rejected.Ready)
	require.Equal(t, "CREDENTIAL_NOT_ADMITTED", rejected.Reason)
	now = now.Add(time.Hour)
	rejected, err = shared.QualifyModel(context.Background(), "trial", d.ModelCategoryText, ref)
	require.NoError(t, err)
	require.False(t, rejected.Ready)
	require.Equal(t, "PRICE_APPLICABILITY_MISSING", rejected.Reason)
	def.ConnectionEndpoint = "https://outside.example/v1"
	_, err = NewCatalog([]Definition{def})
	require.Error(t, err)
}
