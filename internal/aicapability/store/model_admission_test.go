package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
)

func TestEinoAliasedSingleImageKeepsOriginalLedgerAndManufacturerPrice(t *testing.T) {
	l := NewGormInvocationRecorder(newInvocationLedgerDB(t))
	f := aicapability.InvocationRecord{InvocationID: "image", TenantID: "org", UserID: "actor", MemberID: "member", AgentRunID: "run",
		InputHash: "frozen", ModelID: "vendor/model", Capability: aicapability.CapabilityProductImageAnalysis,
		Operation: aicapability.OperationProductImageAnalysis, Outcome: aicapability.InvocationDispatched, StartedAt: time.Now().UTC(),
		MaximumPromptTokens: 5000, MaximumCompletionTokens: 64, OfficialTariff: aicapability.OfficialModelTariff{
			UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "published", FXVersion: "fx", Currency: "CNY",
			CNYMicrosPerCurrencyUnit: 1000000, Model: "model", WireModel: "vendor/model", Channel: "channel",
			UsageVersion: aicapability.ReferenceTextTotalsV1, Rates: aicapability.OfficialModelRates{InputText: 10000, OutputText: 20000}}}
	won, err := l.ClaimInvocation(context.Background(), f)
	require.NoError(t, err)
	require.True(t, won)
	dispatch := f
	f.Outcome, f.FinishedAt, f.UsageKnown = aicapability.InvocationSucceeded, time.Now().UTC(), true
	f.PromptTokens, f.CompletionTokens, f.TotalTokens = 30, 5, 35
	f.BillableUsage = &aicapability.BillableModelUsage{MappingVersion: aicapability.ReferenceTextTotalsV1, Complete: true, InputText: 30, OutputText: 5}
	require.NoError(t, l.RecordInvocation(context.Background(), f))
	read, err := l.ReadModelInvocation(context.Background(), "org", f.InvocationID)
	require.NoError(t, err)
	require.Equal(t, "model", read.OfficialTariff.Model)
	require.Equal(t, "vendor/model", read.ModelID)
	require.Equal(t, f.OfficialTariff, read.OfficialTariff)
	won, err = l.ClaimInvocation(context.Background(), dispatch)
	require.NoError(t, err)
	require.False(t, won)
	_, err = l.ReadModelInvocation(context.Background(), "other-org", f.InvocationID)
	require.Error(t, err)
	dispatch.InvocationID = "different-image"
	dispatch.ModelID = "other/model"
	_, err = l.ClaimInvocation(context.Background(), dispatch)
	require.Error(t, err)
}
