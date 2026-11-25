package imageagent

import (
	"strings"
	"testing"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability"
	"github.com/stretchr/testify/require"
)

func usageOnlyImageIntent() GenerationIntent {
	digest := strings.Repeat("a", 64)
	return GenerationIntent{
		Identity: SlotExternalEffectIdentity{RunScope: RunScope{TenantID: "org-1", OwnerUserID: "user-1", RunID: "run-1"}, SlotID: "source-main", Attempt: 1, PlanRevision: 1},
		MemberID: "member-1", CatalogHash: "catalog-v1:" + digest, SourceDigest: digest,
		PromptVersion: ImageSetSchema, RouteReference: "route-1", CredentialReference: "credential-1", ConfigurationVersion: "config-1",
		Provider: "grsai", Model: aicapability.GRSaiProductImageModel, Protocol: aicapability.GRSaiProductImageProtocol,
		Resolution: "1024x1024", Quality: "auto", InputProtocol: ImageSetSchema, InputDigest: digest, BillingMode: aicapability.ModelUsageOnly,
		ConnectionID: "connection-1",
	}
}

func TestUsageOnlyImageFactUsesSupplierProofWithoutPointReceipts(t *testing.T) {
	intent := usageOnlyImageIntent()
	fact, err := NewGenerationFact(intent)
	require.NoError(t, err)
	fact, won, err := fact.BeginDispatch()
	require.NoError(t, err)
	require.True(t, won)
	require.Empty(t, fact.TerminalProofDigest())
	acceptedAt := time.Date(2026, 11, 24, 14, 0, 0, 0, time.UTC)
	accepted, err := fact.RecordProviderTask("14-task-1", acceptedAt)
	require.NoError(t, err)
	_, err = accepted.RecordProviderTask("14-other", acceptedAt)
	require.ErrorIs(t, err, ErrRevisionConflict)
	unknown, err := accepted.MarkUnknown()
	require.NoError(t, err)
	usage := GenerationSuccess{ResponseID: "14-task-1", ResultDigest: strings.Repeat("b", 64), ResultURL: "https://images.example.test/result.png", ProviderImageCount: 1}
	success, err := unknown.RecordSuccess(usage)
	require.NoError(t, err)
	require.NoError(t, success.Validate())
	require.Equal(t, GenerationSucceeded, success.State)
	require.Len(t, success.TerminalProofDigest(), 64)
	require.Equal(t, GenerationReservationReceipt{}, success.Reservation)
	require.Equal(t, GenerationSettlementReceipt{}, success.Settlement)
	_, err = success.BindSettlement(GenerationSettlementReceipt{})
	require.ErrorIs(t, err, ErrRevisionConflict)
	_, err = unknown.RecordSuccess(GenerationSuccess{ResponseID: "14-other", ResultDigest: usage.ResultDigest, ResultURL: usage.ResultURL, ProviderImageCount: 1})
	require.Error(t, err)
}

func TestToAPIsUsageOnlyFactBindsHighQualityAndKeepsUnknownDispatch(t *testing.T) {
	intent := usageOnlyImageIntent()
	intent.Provider = aicapability.ToAPIsProductImageProvider
	intent.Model = aicapability.ToAPIsProductImageModel
	intent.Protocol = aicapability.ToAPIsProductImageProtocol
	_, err := NewGenerationFact(intent)
	require.Error(t, err, "GRSai quality must not carry over")
	intent.Quality = "high"
	fact, err := NewGenerationFact(intent)
	require.NoError(t, err)
	fact, won, err := fact.BeginDispatch()
	require.NoError(t, err)
	require.True(t, won)
	fact, err = fact.MarkUnknown()
	require.NoError(t, err)
	_, won, err = fact.BeginDispatch()
	require.NoError(t, err)
	require.False(t, won)
	require.Zero(t, fact.Reservation)
	require.Zero(t, fact.Settlement)
}

func TestUsageOnlyImageFactDistinguishesSupplierFailureFromNeverSent(t *testing.T) {
	intent := usageOnlyImageIntent()
	prepared, err := NewGenerationFact(intent)
	require.NoError(t, err)
	neverSent, err := prepared.RecordNoGeneration()
	require.NoError(t, err)
	active, won, err := prepared.BeginDispatch()
	require.NoError(t, err)
	require.True(t, won)
	active, err = active.RecordProviderTask("14-task-1", time.Date(2026, 11, 24, 14, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	failed, err := active.RecordProviderFailure("14-task-1", "violation", strings.Repeat("c", 64))
	require.NoError(t, err)
	require.NoError(t, failed.Validate())
	require.Equal(t, GenerationSupplierFailed, failed.State)
	require.NotEqual(t, neverSent.TerminalProofDigest(), failed.TerminalProofDigest())
	require.Zero(t, failed.Success.ProviderImageCount)
	intent.Points = 1
	_, err = NewGenerationFact(intent)
	require.Error(t, err)
}
