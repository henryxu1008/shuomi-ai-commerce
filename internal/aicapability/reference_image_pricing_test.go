package aicapability

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func referenceImageBillingFixture(t *testing.T) OfficialModelBilling {
	t.Helper()
	var b OfficialModelBilling
	require.NoError(t, json.Unmarshal([]byte(`{"OfficialTariff":{"unitVersion":"ai-micropoint-v1","priceVersion":"reference-v1","fxVersion":"identity-v1","currency":"CNY","cnyMicrosPerCurrencyUnit":1000000,"model":"gpt-image-2.5-flare-2026-09-08","channel":"openai-direct","usageVersion":"reference-images-totals-v1","rates":{"inputText":10000,"inputImage":20000,"outputImage":10000}},"referenceEstimate":{"inputTokens":100,"outputTokens":100,"operationScope":"product-image-effects-v1","size":"1024x1024","quality":"auto","allowAutoSize":true,"maximumPromptBytes":65536,"maximumReferenceImages":3,"maximumReferenceBytes":16777216,"maximumReferencePixels":64000000}}`), &b))
	return b
}

func referenceImageUsageFixture(t *testing.T, raw string) BillableModelUsage {
	t.Helper()
	var u BillableModelUsage
	require.NoError(t, json.Unmarshal([]byte(raw), &u))
	return u
}

func TestReferenceImagesReservationAndD1UseTheOriginalMicropointArithmetic(t *testing.T) {
	b := referenceImageBillingFixture(t)
	r, err := b.ReservationMicropoints(b.OfficialTariff.Model)
	require.NoError(t, err, "a declared reference estimate does not require a fabricated vendor bound")
	require.EqualValues(t, 300, r)
	for _, tc := range []struct {
		name, usage string
		want        int64
	}{
		{"complete partitions", `{"MappingVersion":"reference-images-totals-v1","Complete":true,"InputText":50,"InputImage":50,"OutputImage":50,"referenceImageTotals":{"known":true,"input":100,"output":50,"total":150,"inputTextKnown":true,"inputImageKnown":true}}`, 200},
		{"D1 totals only", `{"MappingVersion":"reference-images-totals-v1","Complete":true,"OutputImage":50,"referenceImageTotals":{"known":true,"input":100,"output":50,"total":150}}`, 250},
		{"above input estimate within reserved money", `{"MappingVersion":"reference-images-totals-v1","Complete":true,"OutputImage":10,"referenceImageTotals":{"known":true,"input":120,"output":10,"total":130}}`, 250},
		{"observed zero", `{"MappingVersion":"reference-images-totals-v1","Complete":true,"referenceImageTotals":{"known":true,"input":0,"output":0,"total":0}}`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := b.ActualMicropoints(b.OfficialTariff.Model, referenceImageUsageFixture(t, tc.usage))
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestReferenceImagesRejectMissingContradictoryAndOverReserveProof(t *testing.T) {
	b := referenceImageBillingFixture(t)
	for _, raw := range []string{
		`{"MappingVersion":"reference-images-totals-v1","Complete":true}`,
		`{"MappingVersion":"reference-images-totals-v1","Complete":true,"OutputImage":10,"referenceImageTotals":{"known":true,"input":100,"output":10,"total":109}}`,
		`{"MappingVersion":"reference-images-totals-v1","Complete":true,"InputText":101,"OutputImage":10,"referenceImageTotals":{"known":true,"input":100,"output":10,"total":110,"inputTextKnown":true}}`,
		`{"MappingVersion":"reference-images-totals-v1","Complete":true,"InputText":60,"InputImage":50,"OutputImage":10,"referenceImageTotals":{"known":true,"input":100,"output":10,"total":110,"inputTextKnown":true,"inputImageKnown":true}}`,
		`{"MappingVersion":"reference-images-totals-v1","Complete":true,"OutputImage":100,"referenceImageTotals":{"known":true,"input":101,"output":100,"total":201}}`,
	} {
		_, err := b.ActualMicropoints(b.OfficialTariff.Model, referenceImageUsageFixture(t, raw))
		require.Error(t, err, raw)
	}
}

func TestReferenceImageProfilePreservesTheOriginalAutoSizeDomain(t *testing.T) {
	b := referenceImageBillingFixture(t)
	data, err := json.Marshal(b)
	require.NoError(t, err)
	require.Contains(t, string(data), `"allowAutoSize":true`, "the shared estimate must not silently discard the original auto request domain")
}
