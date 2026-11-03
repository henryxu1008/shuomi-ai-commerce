package routeconfig

import (
	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
	"testing"
	"time"
)

func TestDecimalRatesAreExactBoundedIntegers(t *testing.T) {
	for _, tc := range []struct {
		input string
		scale int
		want  int64
	}{{"1.000001", 6, 1000001}, {"0.000001", 6, 1}, {"12", 0, 12}, {"1.5", 6, 1500000}} {
		got, err := ParseRate(tc.input, tc.scale)
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
	}
	for _, value := range []string{"0", "-1", "1e3", " 1", "1 ", "NaN", "1.0000001", "999999999999999999999999"} {
		_, err := ParseRate(value, 6)
		require.Error(t, err, value)
	}
	_, err := ParseRate("1.5", 0)
	require.Error(t, err)
}

func testBinding() Binding {
	p := aicapability.ModelProfile{ClientName: "binding-a", ProviderID: "fixture", ModelID: "fixture-model", AdapterKind: "openai-compatible", EndpointIdentityDigest: "endpoint-a", CredentialVersion: "credential-a", RoutingPolicyVersion: "route-a", AdapterPolicyVersion: "adapter-v1", PromptVersion: "prompt-v1", OutputSchemaVersion: "schema-v1", UsageMappingVersion: "usage-v1", CostPricingVersion: "price-a", OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "point-a", FXVersion: "fixture-fx-identity", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: "fixture-model", Channel: "fixture-channel", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialTokenRates{InputText: 1, OutputText: 1}}, Currency: "CNY", InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1, MaximumPromptTokens: 1000, MaximumCompletionTokens: 100, MaximumInputBytes: 4096, MaximumOutputBytes: 4096, DeadlineBound: time.Second}
	return Binding{ID: "binding-a", CredentialVersion: "credential-a", Text: &p, MaximumCostMicros: 1000000}
}

func TestPublicationVersionsFreezeRatesAndPreventRollbackABA(t *testing.T) {
	b := testBinding()
	r := Rates{InputCost: "1.000001", OutputCost: "2"}
	first, err := Freeze(b, r, "revision-a")
	require.NoError(t, err)
	second, err := Freeze(b, r, "revision-b")
	require.NoError(t, err)
	require.NotEqual(t, *first.Text, *second.Text)
	require.EqualValues(t, 1000001, first.Text.InputMicrosPerMillion)
	require.EqualValues(t, 1, first.Text.OfficialTariff.Rates.InputText)
	require.Equal(t, "route-a", b.Text.RoutingPolicyVersion, "candidate descriptor must remain immutable")
	require.Equal(t, "revision-a", first.Text.RoutingPolicyVersion)
	require.Equal(t, "revision-a", first.Text.CostPricingVersion)
	require.Equal(t, "point-a", first.Text.OfficialTariff.PriceVersion)
	b.MaximumCostMicros = 1
	_, err = Freeze(b, r, "revision-c")
	require.Error(t, err, "rates cannot bypass existing cost ceiling")
}

func TestImagePriceDoesNotBecomeTokenTariffOrProviderCost(t *testing.T) {
	b := Binding{ID: "image-a", CredentialVersion: "credential-a", MaximumImagePoints: 100, Image: &ImageProfile{Provider: "fixture", Model: "image-model", Protocol: "fixture-sync", EndpointDigest: "endpoint", CredentialReference: "db:1:1:image-a"}}
	p, err := Freeze(b, Rates{PointsPerImage: "10"}, "image-r1")
	require.NoError(t, err)
	require.Nil(t, p.Text)
	require.EqualValues(t, 10, p.Image.Points)
	require.Equal(t, "image-r1", p.Image.PriceVersion)
	_, err = Freeze(b, Rates{PointsPerImage: "10", InputCost: "1"}, "image-r2")
	require.Error(t, err)
}
