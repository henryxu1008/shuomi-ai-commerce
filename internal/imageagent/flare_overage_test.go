package imageagent

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
	"testing"
)

func flareFactFixture(t *testing.T) GenerationFact {
	t.Helper()
	intent := generationTestIntent()
	intent.Provider, intent.Model, intent.Protocol = "openai", aicapability.FlareImageModel, aicapability.FlareImageProtocol
	intent.PriceVersion, intent.Points = "reference-v1", 300
	intent.Billing = ImageBilling{}
	require.NoError(t, json.Unmarshal([]byte(`{"OfficialTariff":{"unitVersion":"ai-micropoint-v1","priceVersion":"reference-v1","fxVersion":"identity-v1","currency":"CNY","cnyMicrosPerCurrencyUnit":1000000,"model":"gpt-image-2.5-flare-2026-09-08","channel":"openai-direct","usageVersion":"reference-images-totals-v1","rates":{"inputText":10000,"inputImage":20000,"outputImage":10000}},"referenceEstimate":{"inputTokens":100,"outputTokens":100,"operationScope":"product-image-effects-v1","size":"1024x1024","quality":"auto","allowAutoSize":true,"maximumPromptBytes":65536,"maximumReferenceImages":3,"maximumReferenceBytes":16777216,"maximumReferencePixels":64000000}}`), &intent.Billing))
	fact, err := NewGenerationFact(intent)
	require.NoError(t, err)
	fact, err = fact.BindReservation(generationTestReceipt(fact))
	require.NoError(t, err)
	fact, _, err = fact.BeginDispatch()
	require.NoError(t, err)
	return fact
}

func flareProof(input, output int64) GenerationSuccess {
	return GenerationSuccess{ResponseID: "response-1", ResultDigest: generationHash("result"), ResultUnavailable: "invalid_result", UsageKnown: true, InputTokens: input, OutputTokens: output, TotalTokens: input + output,
		BillableUsage: aicapability.BillableModelUsage{MappingVersion: aicapability.ReferenceImagesTotalsV1, Complete: true, OutputImage: output, ReferenceImageTotals: aicapability.ReferenceImageTotals{Known: true, Input: input, Output: output, Total: input + output}}}
}

func TestFlareOverageRetainsEvidenceWithoutTerminalProofOrSettlement(t *testing.T) {
	f := flareFactFixture(t)
	proof := flareProof(101, 100) // A=302 > original R=300
	unknown, err := f.RecordSuccess(proof)
	require.NoError(t, err)
	require.Equal(t, GenerationUnknown, unknown.State)
	require.Empty(t, unknown.TerminalProofDigest())
	require.Equal(t, GenerationSuccess{}, unknown.Success)
	encoded, err := json.Marshal(unknown)
	require.NoError(t, err)
	require.Contains(t, string(encoded), "ObservedOverage")
	require.NoError(t, unknown.Validate())
	replay, err := unknown.RecordSuccess(proof)
	require.NoError(t, err)
	require.Equal(t, unknown, replay)
	kept, err := unknown.MarkUnknown()
	require.NoError(t, err)
	require.Equal(t, unknown, kept)
	_, err = unknown.RecordSuccess(flareProof(1, 1))
	require.ErrorIs(t, err, ErrRevisionConflict)
	_, err = unknown.RecordRejection(GenerationRejection{Kind: "source_edit_preflight", Digest: generationHash("no-send")})
	require.Error(t, err)
}

func TestFlareBundleLocatorDoesNotInventPublicURL(t *testing.T) {
	f := flareFactFixture(t)
	proof := flareProof(100, 50)
	proof.ResultUnavailable = ""
	raw, err := json.Marshal(proof)
	require.NoError(t, err)
	var fields map[string]any
	require.NoError(t, json.Unmarshal(raw, &fields))
	fields["ResultLocatorKind"] = "artifact_bundle"
	raw, err = json.Marshal(fields)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &proof))
	got, err := f.RecordSuccess(proof)
	require.NoError(t, err)
	require.Equal(t, GenerationSucceeded, got.State)
	require.Empty(t, got.Success.ResultURL)
}

func TestFlareStopIdentityIsTenantAndFrozenBindingNotRunOrActor(t *testing.T) {
	in := flareFactFixture(t).Intent
	binding := in.ReferenceImageBindingDigest()
	other := in
	other.Identity.RunID = "another-run"
	other.Identity.OwnerUserID = "another-actor"
	other.MemberID = "another-member"
	other.Identity.Attempt++
	require.Equal(t, binding, other.ReferenceImageBindingDigest())
	other = in
	other.Identity.TenantID = "another-tenant"
	require.NotEqual(t, binding, other.ReferenceImageBindingDigest())
	other = in
	other.Billing.ReferenceEstimate.InputTokens++
	require.NotEqual(t, binding, other.ReferenceImageBindingDigest())
	other = in
	other.ConfigurationVersion = "another-version"
	require.NotEqual(t, binding, other.ReferenceImageBindingDigest())
}

func TestReferenceUsageCannotBecomeALegacySuccess(t *testing.T) {
	f, err := NewGenerationFact(generationTestIntent())
	require.NoError(t, err)
	f, err = f.BindReservation(generationTestReceipt(f))
	require.NoError(t, err)
	f, _, err = f.BeginDispatch()
	require.NoError(t, err)
	_, err = f.RecordSuccess(flareProof(0, 0))
	require.ErrorIs(t, err, ErrValidation)
}
