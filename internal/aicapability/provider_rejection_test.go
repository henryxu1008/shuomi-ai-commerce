package aicapability

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func rejectionFixture(t *testing.T) InvocationRecord {
	t.Helper()
	start := time.Date(2026, 11, 8, 7, 56, 27, 0, time.UTC)
	fact := InvocationRecord{InvocationID: "original:step:1", AgentRunID: "original", TenantID: "org", UserID: "actor", MemberID: "member", InputHash: "input", ProviderID: "openai", ModelID: SingleImageMiniModel, Operation: OperationOwnProductIntake, Capability: CapabilityProductEnrichText, StartedAt: start, Outcome: InvocationDispatched, MaximumPromptTokens: 6660, MaximumCompletionTokens: 4000, Currency: "CNY", CacheStatus: CacheStatusNotApplicable,
		OfficialTariff: OfficialModelTariff{UnitVersion: AIPointMicrounitV1, PriceVersion: "price", FXVersion: "fx", Currency: "USD", CNYMicrosPerCurrencyUnit: 6716800, Model: SingleImageMiniModel, Channel: "openai-direct", UsageVersion: "openai-text-v1", Rates: OfficialModelRates{InputText: 750000, CachedInputText: 75000, OutputText: 4500000}}}
	p := &ProviderRejectionProof{Version: ProviderRejectionProofV1, BindingDigest: InvocationBindingDigest(fact), DispatchFingerprint: strings.Repeat("a", 64), ReservationID: "reservation", ReservedMicropoints: 15445282, MemberMonth: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), OperatorID: "operator", ApprovalEvidenceID: "approval", OperationKey: ProviderRejectionOperationKey(fact.InvocationID), ObservedAt: start.Add(time.Hour), Evidence: ProviderRejectionEvidence{Source: "provider_request_receipt", Reason: "billing_inactive", HTTPStatus: 429, RejectedAt: start.Add(2 * time.Second), RequestIDHash: strings.Repeat("b", 64), ProviderRecordDigest: strings.Repeat("c", 64), NoChargeReceiptHash: strings.Repeat("d", 64), NoChargeEvidenceDigest: strings.Repeat("e", 64), ProjectIdentityHash: strings.Repeat("f", 64), KeyIdentityHash: strings.Repeat("1", 64), MatchingRequests: 1, Handoffs: 1}}
	p.Digest = ProviderRejectionDigest(*p)
	fact.ProviderRejectionProof = p
	fact.Outcome, fact.FinishedAt = InvocationFailed, p.Evidence.RejectedAt
	fact.EstimatedCostKnown = true
	fact.ErrorCategory, fact.ErrorCode = ErrorProviderRejected, ProviderRejectedUnbilledCode
	return fact
}

func TestProviderRejectionRequiresOriginalBindingAndExplicitNoChargeEvidence(t *testing.T) {
	require.NoError(t, ValidateProviderRejection(rejectionFixture(t)))
	for _, change := range []struct {
		name  string
		apply func(*InvocationRecord)
	}{
		{"tenant", func(f *InvocationRecord) { f.TenantID = "other" }},
		{"member", func(f *InvocationRecord) { f.MemberID = "other" }},
		{"input", func(f *InvocationRecord) { f.InputHash = "other" }},
		{"model", func(f *InvocationRecord) { f.ModelID = "other" }},
		{"rate", func(f *InvocationRecord) { f.OfficialTariff.Rates.InputText++ }},
		{"missing no charge receipt", func(f *InvocationRecord) { f.ProviderRejectionProof.Evidence.NoChargeReceiptHash = "" }},
		{"only aggregate evidence", func(f *InvocationRecord) { f.ProviderRejectionProof.Evidence.Source = "aggregate_usage" }},
		{"ambiguous requests", func(f *InvocationRecord) { f.ProviderRejectionProof.Evidence.MatchingRequests = 2 }},
		{"wrong reservation amount", func(f *InvocationRecord) { f.ProviderRejectionProof.ReservedMicropoints++ }},
		{"wrong month", func(f *InvocationRecord) {
			f.ProviderRejectionProof.MemberMonth = f.ProviderRejectionProof.MemberMonth.AddDate(0, 1, 0)
		}},
		{"claims token observation", func(f *InvocationRecord) { f.UsageKnown = true }},
		{"no dispatch disguise", func(f *InvocationRecord) { f.ErrorCode = "rejected_before_dispatch" }},
	} {
		t.Run(change.name, func(t *testing.T) {
			fact := rejectionFixture(t)
			change.apply(&fact)
			fact.ProviderRejectionProof.Digest = ProviderRejectionDigest(*fact.ProviderRejectionProof)
			require.Error(t, ValidateProviderRejection(fact))
		})
	}
}

func TestProviderRejectionDoesNotChangeOriginalDispatchEncoding(t *testing.T) {
	fact := rejectionFixture(t)
	original := InvocationDispatchIdentity(fact)
	require.Nil(t, original.ProviderRejectionProof)
	raw, err := json.Marshal(original)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "providerRejectionProof")
	require.Equal(t, fact.ProviderRejectionProof.BindingDigest, InvocationBindingDigest(fact))
}
